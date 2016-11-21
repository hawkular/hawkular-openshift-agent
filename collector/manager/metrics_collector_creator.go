package manager

import (
	"fmt"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/impl"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
)

func CreateMetricsCollector(id string, identity security.Identity, endpoint collector.Endpoint, env map[string]string) (theCollector collector.MetricsCollector, err error) {
	switch endpoint.Type {
	case collector.ENDPOINT_TYPE_PROMETHEUS:
		{
			theCollector = impl.NewPrometheusMetricsCollector(id, identity, endpoint, env)
		}
	case collector.ENDPOINT_TYPE_JOLOKIA:
		{
			theCollector = impl.NewJolokiaMetricsCollector(id, identity, endpoint, env)
		}
	default:
		{
			err = fmt.Errorf("Unknown endpoint type [%v]", endpoint.Type)
		}
	}
	return
}
