/*
   Copyright 2016 Red Hat, Inc. and/or its affiliates
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

package collector

import (
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
)

type MetricDetails struct {
	ID          string
	MetricType  hmetrics.MetricType
	Description string
}

// MetricsCollector provides the method used to collect metrics for a given endpoint.
// All endpoint types (e.g. Prometheus, Jolokia) must have a MetricsCollector implementation.
type MetricsCollector interface {
	// GetId returns a string identifier for this collector.
	GetId() string

	// GetEndpoint returns information that describes the remote endpoint.
	GetEndpoint() *Endpoint

	// GetAdditionalEnvironment provides a map of additional name/value pairs used to expand tokens within tags defined for endpoint metrics.
	// These are extra name/value pairs that do not include the OS environment which will always be used in addition to the returned map.
	GetAdditionalEnvironment() map[string]string

	// CollectMetrics connects to the remote endpoint and collects the metrics it finds there.
	CollectMetrics() ([]hmetrics.MetricHeader, error)

	// CollectMetricDetails connects to the remote endpoint and collects details about the metrics it finds there.
	CollectMetricDetails() ([]MetricDetails, error)
}
