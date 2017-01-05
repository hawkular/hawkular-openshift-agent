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

package emitter

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

type MetricsType struct {
	DataPointsCollected prometheus.Counter
}

var Metrics = MetricsType{
	DataPointsCollected: prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hawkular_openshift_agent_metric_data_points_collected",
			Help: "The total number of individual metric data points collected from all endpoints.",
		},
	),
}

func init() {
	// Register the metrics with Prometheus's default registry.
	prometheus.MustRegister(Metrics.DataPointsCollected)

	log.Debugf("Registered agent metrics with prometheus")
}

func EmitMetrics(conf *config.Config) {
	if conf.Emitter.Enabled != "true" {
		glog.Info("Agent emitter has been disabled - the agent will not emit any metrics")
		return
	}

	http.Handle("/metrics", promhttp.Handler())

	secure := conf.Identity.Cert_File != "" && conf.Identity.Private_Key_File != ""
	addr := conf.Emitter.Address

	if addr == "" {
		if secure {
			addr = ":8443"
		} else {
			addr = ":8080"
		}
	}

	server := &http.Server{
		Addr: addr,
	}

	glog.Infof("Agent will start emitting its own metrics at [%v]", server.Addr)
	go func() {
		var err error
		if secure {
			err = server.ListenAndServeTLS(conf.Identity.Cert_File, conf.Identity.Private_Key_File)
		} else {
			err = server.ListenAndServe()
		}
		glog.Warning(err)
	}()
}
