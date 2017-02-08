/*
   Copyright 2016-2017 Red Hat, Inc. and/or its affiliates
   and other contributors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/hawkular/hawkular-openshift-agent/collector/manager"
	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/emitter"
	"github.com/hawkular/hawkular-openshift-agent/emitter/status"
	"github.com/hawkular/hawkular-openshift-agent/k8s"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/storage"
)

// Identifies the build. These are set via ldflags during the build (see Makefile).
var (
	version    = "unknown"
	commitHash = "unknown"
)

// Command line arguments
var (
	argConfigFile = flag.String("config", "", "Path to the YAML configuration file. If not specified, environment variables will be used to configure the agent.")
)

// Configuration is the configuration for the agent itself
var Configuration *config.Config

// K8SNodeEventConsumer monitors for changes to a Kubernetes environment
var K8SNodeEventConsumer *k8s.NodeEventConsumer

func init() {
	// log everything to stderr so that it can be easily gathered by logs, separate log files are problematic with containers
	flag.Set("logtostderr", "true")
}

func main() {
	defer glog.Flush()

	// process command line
	flag.Parse()
	validateFlags()

	// log startup information
	log.Infof("Hawkular OpenShift Agent: Version: %v, Commit: %v\n", version, commitHash)
	log.Debugf("Hawkular OpenShift Agent Command line: [%v]", strings.Join(os.Args, " "))

	// load config file if specified, otherwise, rely on environment variables to configure us
	if *argConfigFile != "" {
		c, err := config.LoadFromFile(*argConfigFile)
		if err != nil {
			glog.Fatal(err)
		}
		Configuration = c
	} else {
		log.Infof("No configuration file specified. Will rely on environment for configuration.")
		Configuration = config.NewConfig()
	}
	log.Tracef("Hawkular OpenShift Agent Configuration:\n%s", Configuration)

	if err := validateConfig(); err != nil {
		glog.Fatal(err)
	}

	// prepare our own emitter endpoint - the agent emits status and its own metrics so it can monitor itself
	status.InitStatusReport("Hawkular OpenShift Agent", version, commitHash, Configuration.Emitter.Status_Log_Size)
	status.StatusReport.AddLogMessage("Agent Started")
	emitter.StartEmitter(Configuration)

	// prepare the storage manager and start storing metrics as they come in
	storageManager, err := storage.NewMetricsStorageManager(Configuration)
	if err != nil {
		glog.Fatal("Cannot create storage manager. err=%v", err)
	}
	storageManager.StartStoringMetrics()

	// prepare the collector manager and start monitoring the pre-configured endpoints
	collectorManager := manager.NewMetricsCollectorManager(Configuration,
		storageManager.MetricsChannel, storageManager.MetricDefinitionsChannel)
	collectorManager.StartCollectingEndpoints(Configuration.Endpoints)

	// Start monitoring the node, if any
	K8SNodeEventConsumer := k8s.NewNodeEventConsumer(Configuration, collectorManager)
	K8SNodeEventConsumer.Start()

	// wait forever, or at least until we are told to exit
	waitForTermination()

	// Shutdown internal components
	K8SNodeEventConsumer.Stop()
	collectorManager.StopCollectingAll()
	storageManager.StopStoringMetrics()

}

func waitForTermination() {
	// Channel that is notified when we are done and should exit
	// TODO: may want to make this a package variable - other things might want to tell us to exit
	var doneChan = make(chan bool)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			log.Info("Termination Signal Received")
			doneChan <- true
		}
	}()

	<-doneChan
}

func validateConfig() error {
	var minInterval time.Duration
	if Configuration.Collector.Minimum_Collection_Interval == "" {
		Configuration.Collector.Minimum_Collection_Interval = "5s"
		minInterval = time.Second * 5
	} else {
		var err error
		if minInterval, err = time.ParseDuration(Configuration.Collector.Minimum_Collection_Interval); err != nil {
			return fmt.Errorf("Invalid minimum collection interval. err=%v", err)
		} else if minInterval < (time.Second * 5) {
			return fmt.Errorf("Configured minimum collection interval is too low: %v", Configuration.Collector.Minimum_Collection_Interval)
		}
	}

	if Configuration.Collector.Default_Collection_Interval == "" {
		Configuration.Collector.Default_Collection_Interval = "5m"
	}

	if defaultInterval, err := time.ParseDuration(Configuration.Collector.Default_Collection_Interval); err != nil {
		return fmt.Errorf("Invalid default collection interval. err=%v", err)
	} else if defaultInterval < minInterval {
		return fmt.Errorf("Configured default collection interval [%v] is less than the minimum collection interval [%v]", Configuration.Collector.Default_Collection_Interval, Configuration.Collector.Minimum_Collection_Interval)
	}

	if err := Configuration.Hawkular_Server.Credentials.ValidateCredentials(); err != nil {
		return fmt.Errorf("Hawkular Server configuration is invalid: %v", err)
	}

	for _, e := range Configuration.Endpoints {
		if err := e.ValidateEndpoint(); err != nil {
			return fmt.Errorf("Hawkular Server configuration is invalid: %v", err)
		}
	}

	if Configuration.Emitter.Status_Log_Size < 10 || Configuration.Emitter.Status_Log_Size > 1000 {
		return fmt.Errorf("Emitter Status Log Size must be between 10 and 1000. It was [%v]",
			Configuration.Emitter.Status_Log_Size)
	}

	if Configuration.Emitter.Metrics_Enabled == "true" {
		if err := Configuration.Emitter.Metrics_Credentials.ValidateCredentials(); err != nil {
			return fmt.Errorf("Emitter metrics credentials are invalid: %v", err)
		}
		if Configuration.Emitter.Metrics_Credentials.Token != "" {
			return fmt.Errorf("Token is not supported for emitter metrics credentials")
		}
	}

	if Configuration.Emitter.Status_Enabled == "true" {
		if err := Configuration.Emitter.Status_Credentials.ValidateCredentials(); err != nil {
			return fmt.Errorf("Emitter status credentials are invalid: %v", err)
		}
		if Configuration.Emitter.Status_Credentials.Token != "" {
			return fmt.Errorf("Token is not supported for emitter status credentials")
		}
		if Configuration.Emitter.Status_Credentials.Username == "" || Configuration.Emitter.Status_Credentials.Password == "" {
			log.Warning("The status emitter is not secure. It is recommended you secure the status emitter with credentials.")
		}
	}

	return nil
}

func validateFlags() {
	if *argConfigFile != "" {
		if _, err := os.Stat(*argConfigFile); err != nil {
			if os.IsNotExist(err) {
				log.Debugf("Configuration file [%v] does not exist.", *argConfigFile)
			}
		}
	}
}
