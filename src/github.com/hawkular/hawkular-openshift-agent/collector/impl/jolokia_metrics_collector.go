package impl

import (
	"fmt"

	//"github.com/golang/glog"
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

type JolokiaMetricsCollector struct {
	Id       string
	Endpoint *collector.Endpoint
}

func NewJolokiaMetricsCollector(id string, endpoint *collector.Endpoint) (mc *JolokiaMetricsCollector) {
	mc = &JolokiaMetricsCollector{
		Id:       id,
		Endpoint: endpoint,
	}
	return
}

// GetId implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) GetId() string {
	return jc.Id
}

// GetEndpoint implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) GetEndpoint() *collector.Endpoint {
	return jc.Endpoint
}

// CollectMetrics does the real work of actually connecting to a remote Jolokia endpoint,
// collects all metrics it find there, and returns those metrics.
// CollectMetrics implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) CollectMetrics() (metrics []hmetrics.MetricHeader, err error) {
	log.Debugf("Told to collect Jolokia metrics from [%v]", jc.Endpoint.Url)

	_, err = http.GetHttpClient("", "")
	if err != nil {
		err = fmt.Errorf("Failed to connect to Jolokia endpoint [%v]. err=%v", jc.Endpoint.Url, err)
		return
	}

	url := jc.Endpoint.Url

	// TODO
	err = fmt.Errorf("Jolokia Endpoints are not yet supported: %v", url)

	return
}
