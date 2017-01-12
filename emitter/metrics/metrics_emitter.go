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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
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

func RegisterMetrics() {
	prometheus.MustRegister(Metrics.DataPointsCollected)
}
