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

package collector

import (
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
)

type MetricDetails struct {
	Name        string
	MetricType  hmetrics.MetricType
	Description string
}

// CollectorID identifies a specific endpoint collector.
// PodID is not the k8s pod ID but rather is the pod identifier the agent builds.
// The PodID will be an empty string if the endpoint is external and not within a k8s pod.
// EndpointID is implemented as being unique across all pods.
type CollectorID struct {
	PodID      string
	EndpointID string
}

func (c CollectorID) String() string {
	return c.EndpointID // endpoint is always unique and has pod info encoded in it, this is all we need
}

// MetricsCollector provides the method used to collect metrics for a given endpoint.
// All endpoint types (e.g. Prometheus, Jolokia) must have a MetricsCollector implementation.
type MetricsCollector interface {
	// GetId returns a string identifier for this collector.
	GetID() CollectorID

	// GetEndpoint returns information that describes the remote endpoint.
	GetEndpoint() *Endpoint

	// GetAdditionalEnvironment provides a map of additional name/value pairs used to expand tokens within tags defined for endpoint metrics.
	// These are extra name/value pairs that do not include the OS environment which will always be used in addition to the returned map.
	GetAdditionalEnvironment() map[string]string

	// CollectMetrics connects to the remote endpoint and collects the metrics it finds there.
	// The returned metric headers' IDs should be set to the metric NAMEs as found in the endpoint metrics config;
	// the collector manager will determine the actual ID to use.
	// If a particular metric collected within a single MetricHeader has multiple datapoints that should actually
	// be stored as separate time series data (that is, as separate metrics) then attach tags to those datapoints
	// where each unique combination of tags represents a single time series metric. This supports, for example,
	// Prometheus endpoints where a metric family (with a single metric name) can actually represent multiple
	// time series metrics through tags (what Prometheus calls labels). But this support isn't limited to
	// Prometheus endpoints. If any collector wants to represent multiple time series within a single collected
	// metric, simply tag the datapoints with unique combinations of tags and they will be split out into
	// multiple metric definitions and data.
	CollectMetrics() ([]hmetrics.MetricHeader, error)

	// CollectMetricDetails connects to the remote endpoint and collects details about the given metrics.
	CollectMetricDetails(metricNames []string) ([]MetricDetails, error)
}
