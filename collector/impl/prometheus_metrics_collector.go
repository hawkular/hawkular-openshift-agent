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
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"
	prom "github.com/prometheus/client_model/go"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/prometheus"
)

type PrometheusMetricsCollector struct {
	ID            collector.CollectorID
	Identity      *security.Identity
	Endpoint      *collector.Endpoint
	Environment   map[string]string
	metricNameMap map[string]bool
}

func NewPrometheusMetricsCollector(id collector.CollectorID, identity security.Identity, endpoint collector.Endpoint, env map[string]string) (mc *PrometheusMetricsCollector) {
	mc = &PrometheusMetricsCollector{
		ID:          id,
		Identity:    &identity,
		Endpoint:    &endpoint,
		Environment: env,
	}

	// Put all metric names in a map so we can quickly look them up to know which metrics should be stored and which are to be ignored.
	mc.metricNameMap = make(map[string]bool, len(endpoint.Metrics))
	for _, m := range endpoint.Metrics {
		mc.metricNameMap[m.Name] = true
	}

	return
}

// GetId implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) GetID() collector.CollectorID {
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
		log.Debugf("Told to collect all Prometheus metrics from [%v]", url)
	} else {
		log.Debugf("Told to collect [%v] Prometheus metrics from [%v]", len(pc.Endpoint.Metrics), url)
	}

	metricFamilies, err := prometheus.Scrape(url, &pc.Endpoint.Credentials, client)
	if err != nil {
		err = fmt.Errorf("Failed to collect Prometheus metrics from [%v]. err=%v", url, err)
		return
	}

	metrics = make([]hmetrics.MetricHeader, 0)

	for _, metricFamily := range metricFamilies {

		// If the endpoint was given a list of metrics to collect but the current metric isn't in the list, skip it.
		// If the metric was in the list, use its ID when storing to H-Metrics.
		if len(pc.metricNameMap) > 0 && pc.metricNameMap[metricFamily.GetName()] == false {
			log.Tracef("Told not to collect metric [%v] from endpoint [%v]", metricFamily.GetName(), url)
			continue
		}

		// by default the metric id is the metric name - we'll let the caller (the collector manager) determine the real ID
		metricId := metricFamily.GetName()

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
			Tags:      pc.prepareTagsMap(m.GetLabel()),
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
			Tags:      pc.prepareTagsMap(m.GetLabel()),
		}
	}

	return
}

func (pc *PrometheusMetricsCollector) prepareTagsMap(promLabels []*prom.LabelPair) (hmetricsTags map[string]string) {
	totalTags := len(promLabels)
	hmetricsTags = make(map[string]string, totalTags)

	// Prometheus endpoints indicate different times series data by attaching labels to data points within a metric family.
	// All Prometheus labels are added as tags to the Hawkular Metric datapoint to indicate there are different time series data.
	// This tells the collector manager to split this one metric into several metrics, which is what they really are.
	for _, l := range promLabels {
		hmetricsTags[l.GetName()] = l.GetValue()
	}

	return
}

// CollectMetricDetails implements a method from MetricsCollector interface
func (pc *PrometheusMetricsCollector) CollectMetricDetails(metricNames []string) (metricDetails []collector.MetricDetails, err error) {

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

	if len(metricNames) == 0 {
		metricDetails = make([]collector.MetricDetails, 0)
		return
	}

	log.Debugf("Told to collect details on [%v] Prometheus metrics from [%v]", len(metricNames), url)

	metricFamilies, err := prometheus.Scrape(url, &pc.Endpoint.Credentials, client)
	if err != nil {
		err = fmt.Errorf("Failed to collect details on Prometheus metrics from [%v]. err=%v", url, err)
		return
	}

	metricDetails = make([]collector.MetricDetails, 0)

	for _, metricFamily := range metricFamilies {

		// if this isn't a metric we are looking for, skip it
		doIt := false
		for _, metricToLookFor := range metricNames {
			if metricToLookFor == metricFamily.GetName() {
				doIt = true
				break
			}
		}
		if !doIt {
			continue
		}

		singleMetricDetails := collector.MetricDetails{
			Name:        metricFamily.GetName(),
			Description: metricFamily.GetHelp(),
		}

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

		metricDetails = append(metricDetails, singleMetricDetails)
	}

	// NOTE: the returned details might NOT be in the same order as their names in the metricNames input parameter!
	return

}
