package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/golang/glog"

	"github.com/hawkular/hawkular-openshift-agent/collector/manager"
	"github.com/hawkular/hawkular-openshift-agent/config"
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
	glog.Infof("Hawkular OpenShift Agent: Version: %v, Commit: %v\n", version, commitHash)
	log.Debugf("Hawkular OpenShift Agent Command line: [%v]", strings.Join(os.Args, " "))

	// load config file if specified, otherwise, rely on environment variables to configure us
	if *argConfigFile != "" {
		c, err := config.LoadFromFile(*argConfigFile)
		if err != nil {
			glog.Fatal(err)
		}
		Configuration = c
	} else {
		glog.Infof("No configuration file specified. Will rely on environment for configuration.")
		Configuration = config.NewConfig()
	}
	log.Tracef("Hawkular OpenShift Agent Configuration:\n%s", Configuration)

	if err := validateConfig(); err != nil {
		glog.Fatal(err)
	}

	// prepare the storage manager and start storing metrics as they come in
	storageManager, err := storage.NewMetricsStorageManager(Configuration)
	if err != nil {
		glog.Fatal("Cannot create storage manager. err=%v", err)
	}
	storageManager.StartStoringMetrics()

	// prepare the collector manager and start monitoring the pre-configured endpoints
	collectorManager := manager.NewMetricsCollectorManager(Configuration, storageManager.MetricsChannel)
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
			glog.Info("Termination Signal Received")
			doneChan <- true
		}
	}()

	<-doneChan
}

func validateConfig() error {
	if Configuration.Collector.Minimum_Collection_Interval_Secs < 5 {
		return fmt.Errorf("Configured minimum collection interval is too low: %v", Configuration.Collector.Minimum_Collection_Interval_Secs)
	}

	err := Configuration.Hawkular_Server.Credentials.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("Hawkular Server configuration is invalid: %v", err)
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
