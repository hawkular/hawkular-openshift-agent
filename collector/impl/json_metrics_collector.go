/*
   Copyright 2017 Red Hat, Inc. and/or its affiliates
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
	"sort"
	"strconv"
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/config/tags"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/json"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/util/math"
)

const (
	LABEL_PREFIX      = "label"      // prefix to the tags that will be used to distinguish metrics within a metric family
	ARRAY_STAT_PREFIX = "array_stat" // prefix to the tags for each array statistic metric
)

type JSONMetricsCollector struct {
	ID            collector.CollectorID
	Identity      *security.Identity
	Endpoint      *collector.Endpoint
	Environment   map[string]string
	metricNameMap map[string]collector.MonitoredMetric
}

func NewJSONMetricsCollector(id collector.CollectorID, identity security.Identity, endpoint collector.Endpoint, env map[string]string) (mc *JSONMetricsCollector) {
	mc = &JSONMetricsCollector{
		ID:          id,
		Identity:    &identity,
		Endpoint:    &endpoint,
		Environment: env,
	}

	// Put all metrics in a map so we can quickly look them up to know which metrics should be stored and which are to be ignored.
	mc.metricNameMap = make(map[string]collector.MonitoredMetric, len(endpoint.Metrics))
	for _, m := range endpoint.Metrics {
		mc.metricNameMap[m.Name] = m
	}

	return
}

// GetId implements a method from MetricsCollector interface
func (mc *JSONMetricsCollector) GetID() collector.CollectorID {
	return mc.ID
}

// GetEndpoint implements a method from MetricsCollector interface
func (mc *JSONMetricsCollector) GetEndpoint() *collector.Endpoint {
	return mc.Endpoint
}

// GetAdditionalEnvironment implements a method from MetricsCollector interface
func (mc *JSONMetricsCollector) GetAdditionalEnvironment() map[string]string {
	return mc.Environment
}

// CollectMetrics does the real work of actually connecting to a remote JSON endpoint (like Golang Expvar)
// and collects all metrics it find there, and returns those metrics.
// CollectMetrics implements a method from MetricsCollector interface
func (mc *JSONMetricsCollector) CollectMetrics() (metrics []hmetrics.MetricHeader, err error) {

	url := mc.Endpoint.URL

	httpConfig := http.HttpClientConfig{
		Identity: mc.Identity,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: mc.Endpoint.TLS.Skip_Certificate_Validation,
		},
	}
	httpClient, err := httpConfig.BuildHttpClient()
	if err != nil {
		err = fmt.Errorf("Failed to create http client for JSON endpoint [%v]. err=%v", url, err)
		return
	}

	// Get the JSON endpoint data
	jsonData, err := json.Scrape(url, &mc.Endpoint.Credentials, httpClient)
	if err != nil {
		return
	}

	// Listen to a channel that will receive all the metrics so we can store them in our metrics array.
	// We only ever want to collect one metric per "id+tags" combination. If we see multiple metrics
	// with the same "id+tags" combination, we'll collect the first one of them but ignore all the
	// rest (metricExistenceMap is used to make sure we eliminate these duplicate metrics).
	// We also ignore metrics that have no datapoints (should never happen but we check anyway).
	metrics = make([]hmetrics.MetricHeader, 0)
	metricExistenceMap := make(map[string]bool, 0)
	metricsChan := make(chan hmetrics.MetricHeader)
	go func() {
		for m := range metricsChan {
			if len(m.Data) > 0 {
				// build the unique key for this metric to compare with what is in existence map
				buf := bytes.NewBufferString(m.ID)
				if len(m.Data[0].Tags) > 0 {
					tagKeys := make([]string, len(m.Data[0].Tags))
					for k, _ := range m.Data[0].Tags {
						tagKeys = append(tagKeys, k)
					}
					sort.Strings(tagKeys)
					for _, k := range tagKeys {
						buf.WriteString(k)
						buf.WriteString(m.Data[0].Tags[k])
					}
				}
				uniqueKey := buf.String()

				// if we have not yet seen this metric, put it in our metrics array; otherwise skip it
				if _, ok := metricExistenceMap[uniqueKey]; !ok {
					metrics = append(metrics, m)
					metricExistenceMap[uniqueKey] = true
				}
			}
		}
	}()

	// Walk the JSON data and send all the valid metrics found there to the channel
	context := processingContext{
		metricsChan: metricsChan,
		now:         time.Now(),
	}

	for k, v := range jsonData {
		context.currentMetricId = ""
		context.currentMetricTags = []tags.Tags{}
		mc.processJSON(context, k, v)
	}

	close(metricsChan)

	if log.IsTrace() {
		var buffer bytes.Buffer
		n := 0
		buffer.WriteString(fmt.Sprintf("JSON metrics collected from endpoint [%v]:\n", url))
		for _, m := range metrics {
			buffer.WriteString(fmt.Sprintf("%v\n", m))
			n += len(m.Data)
		}
		buffer.WriteString(fmt.Sprintf("==TOTAL EXPVAR METRICS COLLECTED=%v\n", n))
		log.Trace(buffer.String())
	}

	return
}

// CollectMetricDetails implements a method from MetricsCollector interface
func (mc *JSONMetricsCollector) CollectMetricDetails(metricNames []string) ([]collector.MetricDetails, error) {
	// json does not provide this information
	return make([]collector.MetricDetails, 0), nil
}

func (mc *JSONMetricsCollector) processJSON(context processingContext, jsonName string, jsonValue interface{}) {
	defer context.reset(context.currentMetricId)

	url := mc.Endpoint.URL
	metricId := context.buildMetricId(jsonName)
	if context.currentMetricId != "" {
		context.currentMetricTags = append(context.currentMetricTags, tags.Tags{fmt.Sprintf("%v%v", LABEL_PREFIX, len(context.currentMetricTags)+1): jsonName})
	}

	metricType := hmetrics.Gauge
	if len(mc.metricNameMap) > 0 {
		if monitoredMetric, ok := mc.metricNameMap[metricId]; ok {
			metricType = monitoredMetric.Type
		} else {
			log.Tracef("Told not to collect JSON metric [%v] from endpoint [%v]", metricId, url)
			return
		}
	}

	switch vv := jsonValue.(type) {

	case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// terminal json element - metric has a numeric value
		val, err := strconv.ParseFloat(fmt.Sprint(jsonValue), 64)
		if err == nil {
			currentMetric := hmetrics.MetricHeader{
				ID:     metricId,
				Tenant: mc.Endpoint.Tenant,
				Type:   metricType,
				Data: []hmetrics.Datapoint{
					hmetrics.Datapoint{
						Timestamp: context.now,
						Value:     val,
						Tags:      context.tagsClone(),
					},
				},
			}

			context.metricsChan <- currentMetric
		} else {
			log.Warningf("Failed to convert value to float [%v=%v] from URL=[%v]. err=%v", metricId, jsonValue, url, err)
		}

	case string:
		// terminal json element - metric has a string value
		log.Debugf("JSON string metrics not supported yet. [%v%v] from url [%v]", metricId, context.tagsString(), url)

	case bool:
		// terminal json element - metric is a boolean that we will convert to a number. false=0.0, true=1.0
		var val float64
		if jsonValue.(bool) {
			val = 1.0
		} else {
			val = 0.0
		}
		currentMetric := hmetrics.MetricHeader{
			ID:     metricId,
			Tenant: mc.Endpoint.Tenant,
			Type:   metricType,
			Data: []hmetrics.Datapoint{
				hmetrics.Datapoint{
					Timestamp: context.now,
					Value:     val,
					Tags:      context.tagsClone(),
				},
			},
		}
		context.metricsChan <- currentMetric

	case []interface{}:
		if len(vv) == 0 {
			break // nothing to do, array is empty
		}

		switch vvv := vv[0].(type) {
		case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			{
				// terminal json element - this is a flat array of numeric values
				// We don't know anything about the data, but we'll assume calculating
				// some statistics (min, max, avg, stddev) would be useful.
				tagName := ARRAY_STAT_PREFIX

				minMetric := hmetrics.MetricHeader{
					ID:     metricId,
					Tenant: mc.Endpoint.Tenant,
					Type:   hmetrics.Gauge,
					Data: []hmetrics.Datapoint{
						hmetrics.Datapoint{
							Timestamp: context.now,
							Tags:      context.tagsClone(),
						},
					},
				}
				minMetric.Data[0].Tags[tagName] = "min"

				maxMetric := hmetrics.MetricHeader{
					ID:     metricId,
					Tenant: mc.Endpoint.Tenant,
					Type:   hmetrics.Gauge,
					Data: []hmetrics.Datapoint{
						hmetrics.Datapoint{
							Timestamp: context.now,
							Tags:      context.tagsClone(),
						},
					},
				}
				maxMetric.Data[0].Tags[tagName] = "max"

				avgMetric := hmetrics.MetricHeader{
					ID:     metricId,
					Tenant: mc.Endpoint.Tenant,
					Type:   hmetrics.Gauge,
					Data: []hmetrics.Datapoint{
						hmetrics.Datapoint{
							Timestamp: context.now,
							Tags:      context.tagsClone(),
						},
					},
				}
				avgMetric.Data[0].Tags[tagName] = "avg"

				stddevMetric := hmetrics.MetricHeader{
					ID:     metricId,
					Tenant: mc.Endpoint.Tenant,
					Type:   hmetrics.Gauge,
					Data: []hmetrics.Datapoint{
						hmetrics.Datapoint{
							Timestamp: context.now,
							Tags:      context.tagsClone(),
						},
					},
				}
				stddevMetric.Data[0].Tags[tagName] = "stddev"

				isValid := true
				floatArr := make([]float64, len(vv))
				for i, metricArrItem := range vv {
					val, err := strconv.ParseFloat(fmt.Sprint(metricArrItem), 64)
					if err == nil {
						floatArr[i] = val
					} else {
						log.Warningf("Failed to convert value to float [%v=%v] from URL=[%v]. err=%v", metricId, metricArrItem, url, err)
						isValid = false
						break
					}
				}

				if isValid {
					avgValue := math.Avg(floatArr)
					minMetric.Data[0].Value = math.Min(floatArr)
					maxMetric.Data[0].Value = math.Max(floatArr)
					avgMetric.Data[0].Value = avgValue
					stddevMetric.Data[0].Value = math.Stddev(floatArr, avgValue)
					context.metricsChan <- minMetric
					context.metricsChan <- maxMetric
					context.metricsChan <- avgMetric
					context.metricsChan <- stddevMetric
				}
			}
		case map[string]interface{}:
			{
				// recursively process the values of the array which are maps
				for _, metricArrItem := range vv {
					// each map entry is a separate metric
					for metricKey, metricValue := range metricArrItem.(map[string]interface{}) {
						context.currentMetricId = metricId
						mc.processJSON(context, metricKey, metricValue)
					}
				}
			}
		default:
			{
				log.Debugf("JSON collector cannot process the type [%T] in array for key [%v%v]. url=[%v]", vvv, metricId, context.tagsString(), url)
			}
		}

	case map[string]interface{}:
		// recursively process the values of the map to get the individual metrics
		for metricKey, metricValue := range vv {
			context.currentMetricId = metricId
			mc.processJSON(context, metricKey, metricValue)
		}

	default:
		log.Debugf("JSON collector cannot process the type [%T] for key [%v%v]. url=[%v]", vv, metricId, context.tagsString(), url)
	}
}

// Context struct and methods

type processingContext struct {
	now               time.Time
	metricsChan       chan hmetrics.MetricHeader
	currentMetricId   string
	currentMetricTags []tags.Tags
}

func (c *processingContext) buildMetricId(jsonName string) string {
	if c.currentMetricId != "" {
		return c.currentMetricId
	} else {
		return jsonName
	}
}

func (c *processingContext) reset(metricId string) {
	c.currentMetricId = metricId
	if len(c.currentMetricTags) > 0 {
		c.currentMetricTags = c.currentMetricTags[:len(c.currentMetricTags)-1] // pop the end of the array off
	}
}

func (c *processingContext) tagsClone() tags.Tags {
	newTags := tags.Tags{}
	for _, t := range c.currentMetricTags {
		newTags.AppendTags(t)
	}
	return newTags
}

func (c *processingContext) tagsString() string {
	var buffer bytes.Buffer
	comma := ""

	buffer.WriteString("")
	for _, t := range c.currentMetricTags {
		for k, v := range t {
			buffer.WriteString(fmt.Sprintf("%v%v=%v", comma, k, v))
			comma = ","
		}
	}

	str := buffer.String()
	if str != "" {
		str = fmt.Sprintf("{%v}", str)
	}

	return str
}
