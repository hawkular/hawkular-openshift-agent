package impl

import (
	"bytes"
	"fmt"
	"time"

	//"github.com/golang/glog"
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
	prom "github.com/prometheus/client_model/go"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/prometheus"
)

type PrometheusMetricsCollector struct {
	Id             string
	Endpoint       *collector.Endpoint
	metricNamesMap map[string]bool
}

func NewPrometheusMetricsCollector(id string, endpoint *collector.Endpoint) (mc *PrometheusMetricsCollector) {
	mc = &PrometheusMetricsCollector{
		Id:       id,
		Endpoint: endpoint,
	}

	// put all metric names in a set so we can quickly look them up to know which metrics should be stored and which are to be ignored
	mc.metricNamesMap = make(map[string]bool, len(endpoint.Metrics))
	for _, m := range endpoint.Metrics {
		mc.metricNamesMap[m.Name] = true
	}

	return
}

// GetId implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) GetId() string {
	return pc.Id
}

// GetEndpoint implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) GetEndpoint() *collector.Endpoint {
	return pc.Endpoint
}

// CollectMetrics does the real work of actually connecting to a remote Prometheus endpoint,
// collects all metrics it find there, and returns those metrics.
// CollectMetrics implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) CollectMetrics() (metrics []hmetrics.MetricHeader, err error) {
	log.Debugf("Told to collect Prometheus metrics from [%v]", pc.Endpoint.Url)

	client, err := http.GetHttpClient("", "")
	if err != nil {
		err = fmt.Errorf("Failed to create http client for Prometheus endpoint [%v]. err=%v", pc.Endpoint.Url, err)
		return
	}

	url := pc.Endpoint.Url
	now := time.Now()

	metricFamilies, err := prometheus.Scrape(url, client)
	if err != nil {
		err = fmt.Errorf("Failed to collect Prometheus metrics from [%v]. err=%v", pc.Endpoint.Url, err)
		return
	}

	metrics = make([]hmetrics.MetricHeader, 0)

	for _, metricFamily := range metricFamilies {

		// if the endpoint was given a list of metrics to collect but the current metric isn't in the list, skip it
		if len(pc.metricNamesMap) > 0 {
			if _, ok := pc.metricNamesMap[metricFamily.GetName()]; ok == false {
				continue
			}
		}

		// convert the prometheus metric into a hawkular metrics object
		switch metricFamily.GetType() {
		case prom.MetricType_GAUGE:
			{
				metrics = append(metrics, pc.convertGauge(metricFamily, now))
			}
		case prom.MetricType_COUNTER:
			{
				metrics = append(metrics, pc.convertCounter(metricFamily, now))
			}
		case prom.MetricType_SUMMARY,
			prom.MetricType_HISTOGRAM,
			prom.MetricType_UNTYPED:
			fallthrough
		default:
			{
				log.Tracef("Skipping unsupported Prometheus metric [%v] of type [%v]", metricFamily.GetName(), metricFamily.GetType())
				continue
			}
		}
	}

	if log.IsTrace() {
		var buffer bytes.Buffer
		n := 0
		buffer.WriteString(fmt.Sprintf("Metrics collected from endpoint [%v]:\n", url))
		for _, m := range metrics {
			buffer.WriteString(fmt.Sprintf("%v\n", m))
			n += len(m.Data)
		}
		buffer.WriteString(fmt.Sprintf("==TOTAL METRICS COLLECTED=%v\n", n))
		log.Trace(buffer.String())
	}

	return
}

func (pc *PrometheusMetricsCollector) convertGauge(promMetricFamily *prom.MetricFamily, now time.Time) (metric hmetrics.MetricHeader) {
	metric = hmetrics.MetricHeader{
		Type:   hmetrics.Gauge,
		ID:     promMetricFamily.GetName(),
		Tenant: pc.Endpoint.Tenant,
		Data:   make([]hmetrics.Datapoint, len(promMetricFamily.GetMetric())),
	}

	for i, m := range promMetricFamily.GetMetric() {
		g := m.GetGauge()
		metric.Data[i] = hmetrics.Datapoint{
			Timestamp: now,
			Value:     g.GetValue(),
			Tags:      pc.convertLabelsMap(m.GetLabel()),
		}
	}

	return
}

func (pc *PrometheusMetricsCollector) convertCounter(promMetricFamily *prom.MetricFamily, now time.Time) (metric hmetrics.MetricHeader) {
	metric = hmetrics.MetricHeader{
		Type:   hmetrics.Counter,
		ID:     promMetricFamily.GetName(),
		Tenant: pc.Endpoint.Tenant,
		Data:   make([]hmetrics.Datapoint, len(promMetricFamily.GetMetric())),
	}

	for i, m := range promMetricFamily.GetMetric() {
		g := m.GetCounter()
		metric.Data[i] = hmetrics.Datapoint{
			Timestamp: now,
			Value:     g.GetValue(),
			Tags:      pc.convertLabelsMap(m.GetLabel()),
		}
	}

	return
}

func (pc *PrometheusMetricsCollector) convertLabelsMap(promLabels []*prom.LabelPair) (hmetricsLabels map[string]string) {
	hmetricsLabels = make(map[string]string, len(promLabels))
	for _, l := range promLabels {
		hmetricsLabels[l.GetName()] = l.GetValue()
	}
	return
}
