package collector

import (
	"github.com/hawkular/hawkular-client-go/metrics"
)

type EndpointType string

const (
	ENDPOINT_TYPE_PROMETHEUS EndpointType = "prometheus"
	ENDPOINT_TYPE_JOLOKIA                 = "jolokia"
)

// MonitoredMetric provides information about a specific metric that is to be collected.
type MonitoredMetric struct {
	Type metrics.MetricType
	Name string
}

// Endpoint provides information about how to connect to a particular endpoint in order
// to collect metrics from it.
// If tenant is not supplied, the global tenant ID defined
// in the global agent configuration file should be used.
// USED FOR YAML (see agent config file)
type Endpoint struct {
	Type                     EndpointType
	Url                      string
	Collection_Interval_Secs int
	Tenant                   string
	Metrics                  []MonitoredMetric
}
