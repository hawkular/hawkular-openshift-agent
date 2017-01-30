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

package impl

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
	prom "github.com/prometheus/client_model/go"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/prometheus"
	"github.com/hawkular/hawkular-openshift-agent/util/expand"
)

type PrometheusMetricsCollector struct {
	ID              string
	Identity        *security.Identity
	Endpoint        *collector.Endpoint
	Environment     map[string]string
	metricNameIdMap map[string]string
}

func NewPrometheusMetricsCollector(id string, identity security.Identity, endpoint collector.Endpoint, env map[string]string) (mc *PrometheusMetricsCollector) {
	mc = &PrometheusMetricsCollector{
		ID:          id,
		Identity:    &identity,
		Endpoint:    &endpoint,
		Environment: env,
	}

	// Put all metric names in a map so we can quickly look them up to know which metrics should be stored and which are to be ignored.
	// Notice the value of the map is the metric ID - this will be the Hawkular Metrics ID when the metric is stored.
	mc.metricNameIdMap = make(map[string]string, len(endpoint.Metrics))
	for _, m := range endpoint.Metrics {
		mc.metricNameIdMap[m.Name] = m.ID
	}

	return
}

// GetId implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) GetId() string {
	return pc.ID
}

// GetEndpoint implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) GetEndpoint() *collector.Endpoint {
	return pc.Endpoint
}

// GetAdditionalEnvironment implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) GetAdditionalEnvironment() map[string]string {
	return pc.Environment
}

// CollectMetrics does the real work of actually connecting to a remote Prometheus endpoint,
// collects all metrics it find there, and returns those metrics.
// CollectMetrics implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) CollectMetrics() (metrics []hmetrics.MetricHeader, err error) {

	httpConfig := http.HttpClientConfig{
		Identity: pc.Identity,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: pc.Endpoint.TLS.Skip_Certificate_Validation,
		},
	}
	client, err := httpConfig.BuildHttpClient()
	if err != nil {
		err = fmt.Errorf("Failed to create http client for Prometheus endpoint [%v]. err=%v", pc.Endpoint.URL, err)
		return
	}

	url := pc.Endpoint.URL
	now := time.Now()

	if len(pc.Endpoint.Metrics) == 0 {
		log.Debugf("There are no metrics defined for Prometheus endpoint [%v]", url)
		metrics = make([]hmetrics.MetricHeader, 0)
		return
	}

	log.Debugf("Told to collect [%v] Prometheus metrics from [%v]", len(pc.Endpoint.Metrics), url)

	metricFamilies, err := prometheus.Scrape(url, &pc.Endpoint.Credentials, client)
	if err != nil {
		err = fmt.Errorf("Failed to collect Prometheus metrics from [%v]. err=%v", pc.Endpoint.URL, err)
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
				metrics = append(metrics, pc.convertGauge(metricFamily, metricId, now)...)
			}
		case prom.MetricType_COUNTER:
			{
				metrics = append(metrics, pc.convertCounter(metricFamily, metricId, now)...)
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

func (pc *PrometheusMetricsCollector) convertGauge(promMetricFamily *prom.MetricFamily, id string, now time.Time) (metricsCreated [] hmetrics.MetricHeader) {
	metricsCreated = make([]hmetrics.MetricHeader, 0)
	if strings.Contains(id, "$") {
		for _, m := range promMetricFamily.GetMetric() {
			labelPairMap := pc.prepareTagsMap(m.GetLabel())
			idMappingFunc := expand.MappingFunc(false, labelPairMap)

			// We still tag the data-points as not all labels have to be used on the metric name replacement
			g := m.GetGauge()
			data := make([]hmetrics.Datapoint, 1)
			data[0] = hmetrics.Datapoint{
				Timestamp:      now,
				Value:          g.GetValue(),
				Tags:           labelPairMap,
			}

			metric := hmetrics.MetricHeader{
				Type:   hmetrics.Gauge,
				ID:     os.Expand(id, idMappingFunc),
				Tenant: pc.Endpoint.Tenant,
				Data:   data,
			}

			metricsCreated = append(metricsCreated, metric)
		}

	} else {
		metric := hmetrics.MetricHeader{
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
				Tags:      pc.prepareTagsMap(m.GetLabel()),
			}
		}

		metricsCreated = append(metricsCreated, metric)
	}

	return
}

func (pc *PrometheusMetricsCollector) convertCounter(promMetricFamily *prom.MetricFamily, id string, now time.Time) (metricsCreated []hmetrics.MetricHeader) {
	metricsCreated = make([]hmetrics.MetricHeader, 0)
	if strings.Contains(id, "$") {
		for _, m := range promMetricFamily.GetMetric() {
			labelPairMap := pc.prepareTagsMap(m.GetLabel())
			idMappingFunc := expand.MappingFunc(false, labelPairMap)

			// We still tag the data-points as not all labels have to be used on the metric name replacement
			g := m.GetCounter()
			data := make([]hmetrics.Datapoint, 1)
			data[0] = hmetrics.Datapoint{
				Timestamp:      now,
				Value:          g.GetValue(),
				Tags:           labelPairMap,
			}

			metric := hmetrics.MetricHeader{
				Type:   hmetrics.Counter,
				ID:     os.Expand(id, idMappingFunc),
				Tenant: pc.Endpoint.Tenant,
				Data:   data,
			}

			metricsCreated = append(metricsCreated, metric)
		}

	} else {
		metric := hmetrics.MetricHeader{
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
				Tags:      pc.prepareTagsMap(m.GetLabel()),
			}
		}
	}

	return
}

func (pc *PrometheusMetricsCollector) prepareTagsMap(promLabels []*prom.LabelPair) (hmetricsTags map[string]string) {
	totalTags := len(promLabels)
	hmetricsTags = make(map[string]string, totalTags)

	// all Prometheus labels are added as tags to the metric datapoint
	for _, l := range promLabels {
		hmetricsTags[l.GetName()] = l.GetValue()
	}

	return
}

// CollectMetricDetails implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) CollectMetricDetails() (metricDetails []collector.MetricDetails, err error) {

	httpConfig := http.HttpClientConfig{
		Identity: pc.Identity,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: pc.Endpoint.TLS.Skip_Certificate_Validation,
		},
	}
	client, err := httpConfig.BuildHttpClient()
	if err != nil {
		err = fmt.Errorf("Failed to create http client for Prometheus endpoint [%v]. err=%v", pc.Endpoint.URL, err)
		return
	}

	url := pc.Endpoint.URL

	if len(pc.Endpoint.Metrics) == 0 {
		log.Debugf("There are no metrics defined for Prometheus endpoint [%v]", url)
		metricDetails = make([]collector.MetricDetails, 0)
		return
	}

	log.Debugf("Told to collect details on [%v] Prometheus metrics from [%v]", len(pc.Endpoint.Metrics), url)

	metricFamilies, err := prometheus.Scrape(url, &pc.Endpoint.Credentials, client)
	if err != nil {
		err = fmt.Errorf("Failed to collect details on Prometheus metrics from [%v]. err=%v", pc.Endpoint.URL, err)
		return
	}

	metricDetails = make([]collector.MetricDetails, 0)

	for _, metricFamily := range metricFamilies {

		// by default the metric Id is the metric name
		metricId := metricFamily.GetName()

		// If the endpoint was given a list of metrics to collect but the current metric isn't in the list, skip it.
		// If the metric was in the list, use its ID.
		if len(pc.metricNameIdMap) > 0 {
			var ok bool
			metricId, ok = pc.metricNameIdMap[metricFamily.GetName()]
			if !ok {
				continue
			}
		}

		singleMetricDetails := collector.MetricDetails{}

		switch metricFamily.GetType() {
		case prom.MetricType_GAUGE:
			{
				singleMetricDetails.MetricType = hmetrics.Gauge
			}
		case prom.MetricType_COUNTER:
			{
				singleMetricDetails.MetricType = hmetrics.Counter
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

		singleMetricDetails.ID = metricId
		singleMetricDetails.Description = metricFamily.GetHelp()

		metricDetails = append(metricDetails, singleMetricDetails)
	}

	return

}
