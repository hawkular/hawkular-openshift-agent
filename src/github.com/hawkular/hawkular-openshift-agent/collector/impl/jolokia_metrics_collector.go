package impl

import (
	"bytes"
	"fmt"
	"time"

	"github.com/golang/glog"
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/jolokia"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

type JolokiaMetricsCollector struct {
	Id       string
	Endpoint *collector.Endpoint
}

func NewJolokiaMetricsCollector(id string, endpoint collector.Endpoint) (mc *JolokiaMetricsCollector) {
	mc = &JolokiaMetricsCollector{
		Id:       id,
		Endpoint: &endpoint,
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

	url := jc.Endpoint.Url
	now := time.Now()

	if len(jc.Endpoint.Metrics) == 0 {
		log.Debugf("There are no metrics defined for Jolokia endpoint [%v]", url)
		metrics = make([]hmetrics.MetricHeader, 0)
		return
	}

	log.Debugf("Told to collect [%v] Jolokia metrics from [%v]", len(jc.Endpoint.Metrics), url)

	httpClient, err := http.GetHttpClient("", "")
	if err != nil {
		err = fmt.Errorf("Failed to create http client for Jolokia endpoint [%v]. err=%v", url, err)
		return
	}

	// build up the bulk request with all the metrics we need to collect
	requests := jolokia.NewJolokiaRequests()
	for _, m := range jc.Endpoint.Metrics {
		req := &jolokia.JolokiaRequest{
			Type: jolokia.RequestTypeRead,
		}
		jolokia.ParseMetricName(m.Name, req)
		requests.AddRequest(*req)
	}
	log.Tracef("Making bulk Jolokia request from [%v]:\n%v", url, requests)

	// send the request to the Jolokia endpoint
	responses, err := requests.SendRequests(url, httpClient)
	if err != nil {
		err = fmt.Errorf("Failed to collect metrics from Jolokia endpoint [%v]. err=%v", url, err)
		return
	}

	// convert the metric data we got from Jolokia into our Hawkular-Metrics data format
	metrics = make([]hmetrics.MetricHeader, 0)

	for i, resp := range responses.Responses {
		if resp.IsSuccess() {
			data := make([]hmetrics.Datapoint, 1)
			data[0] = hmetrics.Datapoint{
				Timestamp: now,
				Value:     resp.GetValueAsFloat(),
			}

			id := jc.Endpoint.Metrics[i].Id
			if id == "" {
				id = jc.Endpoint.Metrics[i].Name
			}

			metric := hmetrics.MetricHeader{
				Type:   jc.Endpoint.Metrics[i].Type,
				ID:     id,
				Tenant: jc.Endpoint.Tenant,
				Data:   data,
			}

			metrics = append(metrics, metric)

		} else {
			glog.Warningf("Failed to collect metric [%v] from Jolokia endpoint [%v]. err=%v",
				jc.Endpoint.Metrics[i].Name, url, err)
		}
	}

	if log.IsTrace() {
		var buffer bytes.Buffer
		n := 0
		buffer.WriteString(fmt.Sprintf("Jolokia metrics collected from endpoint [%v]:\n", url))
		for _, m := range metrics {
			buffer.WriteString(fmt.Sprintf("%v\n", m))
			n += len(m.Data)
		}
		buffer.WriteString(fmt.Sprintf("==TOTAL JOLOKIA METRICS COLLECTED=%v\n", n))
		log.Trace(buffer.String())
	}

	return
}
