package collector

import (
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
)

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
}
