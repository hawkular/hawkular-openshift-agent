package impl

import (
	"bytes"
	"fmt"
	"time"

	//"github.com/golang/glog"
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
	prom "github.com/prometheus/client_model/go"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/prometheus"
)

type PrometheusMetricsCollector struct {
	Id              string
	Identity        *security.Identity
	Endpoint        *collector.Endpoint
	metricNameIdMap map[string]string
}

func NewPrometheusMetricsCollector(id string, identity security.Identity, endpoint collector.Endpoint) (mc *PrometheusMetricsCollector) {
	mc = &PrometheusMetricsCollector{
		Id:       id,
		Identity: &identity,
		Endpoint: &endpoint,
	}

	// Put all metric names in a map so we can quickly look them up to know which metrics should be stored and which are to be ignored.
	// Notice the value of the map is the metric ID - this will be the Hawkular Metrics ID when the metric is stored
	mc.metricNameIdMap = make(map[string]string, len(endpoint.Metrics))
	for _, m := range endpoint.Metrics {
		id := m.Id
		if id == "" {
			id = m.Name
		}
		mc.metricNameIdMap[m.Name] = id
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

	client, err := http.GetHttpClient(pc.Identity)
	if err != nil {
		err = fmt.Errorf("Failed to create http client for Prometheus endpoint [%v]. err=%v", pc.Endpoint.Url, err)
		return
	}

	url := pc.Endpoint.Url
	now := time.Now()

	metricFamilies, err := prometheus.Scrape(url, &pc.Endpoint.Credentials, client)
	if err != nil {
		err = fmt.Errorf("Failed to collect Prometheus metrics from [%v]. err=%v", pc.Endpoint.Url, err)
		return
	}

	metrics = make([]hmetrics.MetricHeader, 0)

	for _, metricFamily := range metricFamilies {

		// by default the metric Id is the metric name
		metricId := metricFamily.GetName()

		// If the endpoint was given a list of metrics to collect but the current metric isn't in the list, skip it.
		// If the metric was in the list, use its ID when storing to H-Metrics.
		if len(pc.metricNameIdMap) > 0 {
			var ok bool
			metricId, ok = pc.metricNameIdMap[metricFamily.GetName()]
			if !ok {
				continue
			}
		}

		// convert the prometheus metric into a hawkular metrics object
		switch metricFamily.GetType() {
		case prom.MetricType_GAUGE:
			{
				metrics = append(metrics, pc.convertGauge(metricFamily, metricId, now))
			}
		case prom.MetricType_COUNTER:
			{
				metrics = append(metrics, pc.convertCounter(metricFamily, metricId, now))
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
		buffer.WriteString(fmt.Sprintf("Prometheus metrics collected from endpoint [%v]:\n", url))
		for _, m := range metrics {
			buffer.WriteString(fmt.Sprintf("%v\n", m))
			n += len(m.Data)
		}
		buffer.WriteString(fmt.Sprintf("==TOTAL PROMETHEUS METRICS COLLECTED=%v\n", n))
		log.Trace(buffer.String())
	}

	return
}

func (pc *PrometheusMetricsCollector) convertGauge(promMetricFamily *prom.MetricFamily, id string, now time.Time) (metric hmetrics.MetricHeader) {
	metric = hmetrics.MetricHeader{
		Type:   hmetrics.Gauge,
		ID:     id,
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

func (pc *PrometheusMetricsCollector) convertCounter(promMetricFamily *prom.MetricFamily, id string, now time.Time) (metric hmetrics.MetricHeader) {
	metric = hmetrics.MetricHeader{
		Type:   hmetrics.Counter,
		ID:     id,
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
