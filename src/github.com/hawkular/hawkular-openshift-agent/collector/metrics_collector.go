package collector

import (
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
)

// MetricsCollector provides the method used to collect metrics for a given endpoint.
// All endpoint types (e.g. Prometheus, Jolokia) must have a MetricsCollector implementation.
type MetricsCollector interface {
	GetId() string
	GetEndpoint() *Endpoint
	CollectMetrics() ([]hmetrics.MetricHeader, error)
}
