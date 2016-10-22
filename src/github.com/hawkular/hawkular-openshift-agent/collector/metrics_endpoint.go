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
// The "Id" is the metric ID as it will be stored in Hawkular Metrics - it may or may not
// be identical to the actual metric name. The "Name" is the name of the metric as it is
// found in the endpoint. This is the true name of the metric as it is exposed from the system
// from where it came from.
// USED FOR YAML
type MonitoredMetric struct {
	Id   string
	Name string
	Type metrics.MetricType
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
