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

package jolokia

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

// ProcessJolokiaResponseValueObject takes a value object from a response and processes it, extracting
// out metric data and returning that metric data.
//
// The parameters to this method are passed to the other methods is the processing chain.
//
// url: the jolokia endpoint where the data came from
// now: the time when the collection occurred
// tenantId: metric data is to be assigned to this tenant
// respValueObject: the value as received in the response - it has all the data
// monitoredMetric: defines the metric that was collected
// metricNameParts: the parts of the metric split out
func ProcessJolokiaResponseValueObject(
	url string,
	now time.Time,
	tenantId string,
	respValueObject interface{},
	monitoredMetric collector.MonitoredMetric,
	metricNameParts MetricNameParts) []hmetrics.MetricHeader {

	var metrics []hmetrics.MetricHeader

	if metricNameParts.IsMBeanQuery() {
		metrics = processMultipleMBeans(url, now, tenantId, respValueObject, monitoredMetric, metricNameParts, monitoredMetric.Name)
	} else {
		metrics = processSingleMBean(url, now, tenantId, respValueObject, monitoredMetric, metricNameParts, monitoredMetric.Name)
	}

	return metrics
}

func processMultipleMBeans(
	url string,
	now time.Time,
	tenantId string,
	respValueObject interface{},
	monitoredMetric collector.MonitoredMetric,
	metricNameParts MetricNameParts,
	specificMetricName string) []hmetrics.MetricHeader {

	metrics := make([]hmetrics.MetricHeader, 0)

	if respValue, err := returnValueAsMap(respValueObject); err == nil {
		// Loop over each MBean found in the response object.
		// We want the array is some repeatable order (mostly so we can easily write our tests), so sort the keys.
		var keys []string
		for k := range respValue {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := respValue[k]

			// Set the specific name to the current MBean that is being processed. We build this up as we go,
			// setting it more and more specifically as we know. This is used mainly for log messages,
			// but not always (has important usage when processing composite attributes).
			specificMetricName = k
			if metricNameParts.HasAttribute() {
				specificMetricName += "#" + metricNameParts.Attribute
				if metricNameParts.HasPath() {
					specificMetricName += "#" + metricNameParts.Path
				}
			}

			mbeanMetrics := processSingleMBean(url, now, tenantId, v, monitoredMetric, metricNameParts, specificMetricName)

			// since we know we queried for multiple MBeans,
			// put tags on all datapoints so user can use ${x} tokens in ID
			for _, mm := range mbeanMetrics {
				for i, _ := range mm.Data {
					if currentMBeanParts, err := NewMetricNameParts(k); err == nil {
						if mm.Data[i].Tags == nil {
							mm.Data[i].Tags = make(map[string]string, len(metricNameParts.KeyWildcards))
						}
						for mbeanNameKey, _ := range metricNameParts.KeyWildcards {
							if currentMBeanKeyValue, ok := currentMBeanParts.KeyValues[mbeanNameKey]; ok {
								mm.Data[i].Tags[mbeanNameKey] = currentMBeanKeyValue
							} else {
								log.Warningf("The name of MBean [%v] does not have key [%v] from Jolokia endpoint [%v]", k, mbeanNameKey, url)
							}
						}
					} else {
						log.Warningf("Cannot set tags for metric [%v] from Jolokia endpoint [%v]. The queried name [%v] is invalid. err=%v",
							specificMetricName, url, k, err)
					}
				}
			}

			metrics = append(metrics, mbeanMetrics...)
		}
	} else {
		// if the value was nil, it just means there was no metric data yet so no need to warn about that
		if respValueObject != nil {
			log.Warningf("Cannot process data for metric [%v] from Jolokia endpoint [%v]. err=%v",
				specificMetricName, url, err)
		}
	}

	return metrics
}

func processSingleMBean(
	url string,
	now time.Time,
	tenantId string,
	respValueObject interface{},
	monitoredMetric collector.MonitoredMetric,
	metricNameParts MetricNameParts,
	specificMetricName string) []hmetrics.MetricHeader {

	var metrics []hmetrics.MetricHeader

	if metricNameParts.IsAllAttributes() {
		metrics = make([]hmetrics.MetricHeader, 0)

		// we were asked to collect multiple attributes
		if respValue, err := returnValueAsMap(respValueObject); err == nil {
			// just get the MBean name - if we still have "#*#*" or "#*" then strip them
			justTheMBeanName := specificMetricName
			justTheMBeanName = strings.TrimSuffix(justTheMBeanName, "#*") // first one
			justTheMBeanName = strings.TrimSuffix(justTheMBeanName, "#*") // second one if exists

			// We want the array is some repeatable order (mostly so we can easily write our tests), so sort the keys.
			var keys []string
			for k := range respValue {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := respValue[k]

				// Set the name to the current MBean + attribute that is being processed.
				specificMetricName = justTheMBeanName + "#" + k

				mbeanMetrics := processSingleAttribute(url, now, tenantId, v, monitoredMetric, metricNameParts, specificMetricName)

				// since metricNameParts.IsAllAttributes(), we know we queried all attributes,
				// so put a tag on all datapoints so user can use $1 token in ID
				for _, mm := range mbeanMetrics {
					for i, _ := range mm.Data {
						if mm.Data[i].Tags == nil {
							mm.Data[i].Tags = make(map[string]string, 1)
						}
						mm.Data[i].Tags["1"] = k
					}
				}

				metrics = append(metrics, mbeanMetrics...)
			}
		} else {
			// if the value was nil, it just means there was no metric data yet so no need to warn about that
			if respValueObject != nil {
				log.Warningf("Cannot process metrics from single MBean for metric [%v] from Jolokia endpoint [%v]. err=%v",
					specificMetricName, url, err)
			}
		}
	} else {
		// we were asked to collect a single attribute
		metrics = processSingleAttribute(url, now, tenantId, respValueObject, monitoredMetric, metricNameParts, specificMetricName)
	}

	return metrics
}

func processSingleAttribute(
	url string,
	now time.Time,
	tenantId string,
	respValueObject interface{},
	monitoredMetric collector.MonitoredMetric,
	metricNameParts MetricNameParts,
	specificMetricName string) []hmetrics.MetricHeader {

	var metrics []hmetrics.MetricHeader

	// the response value object may be a basic attribute with a single float value or a composite attribute with multiple values
	switch respValueObject.(type) {
	case map[string]interface{}:
		metrics = processSingleCompositeAttribute(url, now, tenantId, respValueObject, monitoredMetric, metricNameParts, specificMetricName)
	default:
		metrics = processSingleBasicAttribute(url, now, tenantId, respValueObject, monitoredMetric, metricNameParts, specificMetricName)
	}

	return metrics
}

func processSingleCompositeAttribute(
	url string,
	now time.Time,
	tenantId string,
	respValueObject interface{},
	monitoredMetric collector.MonitoredMetric,
	metricNameParts MetricNameParts,
	specificMetricName string) []hmetrics.MetricHeader {

	metrics := make([]hmetrics.MetricHeader, 0)

	if respValue, err := returnValueAsMap(respValueObject); err == nil {
		// Loop over each path in the composite attribute.
		// We want the array is some repeatable order (mostly so we can easily write our tests), so sort the keys.
		var keys []string
		for k := range respValue {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := respValue[k]
			if floatVal, err := returnValueAsFloat(v); err == nil {
				data := make([]hmetrics.Datapoint, 1)
				data[0] = hmetrics.Datapoint{
					Timestamp: now,
					Value:     floatVal,
				}

				// If this was to query all paths, put a tag so user can use $2 token in ID.
				// We also want to put the inner path at the end of our generated ID.
				if metricNameParts.IsAllPaths() {
					data[0].Tags = map[string]string{"2": k}
					specificMetricName = strings.TrimSuffix(specificMetricName, "#*") + "#" + k
				}

				metric := hmetrics.MetricHeader{
					Type:   monitoredMetric.Type,
					ID:     monitoredMetric.Name, // the caller (collector manager) may override when it determines the real ID
					Tenant: tenantId,
					Data:   data,
				}

				metrics = append(metrics, metric)
			} else if innerCompositeMap, err := returnValueAsMap(v); err == nil {
				// in certain cases, Jolokia returns the composite data in an inner map. The key of the outer map
				// referring to this inner map is the name of the composite attribute.
				metrics = processSingleCompositeAttribute(url, now, tenantId, innerCompositeMap, monitoredMetric, metricNameParts, specificMetricName)

				// since we know we queried multiple attributes (we never get this outer-inner map stuff without that)
				// put a tag on all datapoints so user can use $1 token in ID
				for _, m := range metrics {
					for i, _ := range m.Data {
						if m.Data[i].Tags == nil {
							m.Data[i].Tags = make(map[string]string, 1)
						}
						m.Data[i].Tags["1"] = k
					}
				}
			} else {
				// if the value was nil, it just means there was no metric data yet so no need to warn about that
				if v != nil {
					log.Warningf("Cannot process [%v] from composite attribute for metric [%v] from Jolokia endpoint [%v]. err=%v",
						k, specificMetricName, url, err)
				}
			}
		}
	} else {
		// if the value was nil, it just means there was no metric data yet so no need to warn about that
		if respValueObject != nil {
			log.Warningf("Cannot process composite attribute for metric [%v] from Jolokia endpoint [%v]. err=%v",
				specificMetricName, url, err)
		}
	}

	return metrics
}

func processSingleBasicAttribute(
	url string,
	now time.Time,
	tenantId string,
	respValueObject interface{},
	monitoredMetric collector.MonitoredMetric,
	metricNameParts MetricNameParts,
	specificMetricName string) []hmetrics.MetricHeader {

	metrics := make([]hmetrics.MetricHeader, 0)

	if respValue, err := returnValueAsFloat(respValueObject); err == nil {
		data := make([]hmetrics.Datapoint, 1)
		data[0] = hmetrics.Datapoint{
			Timestamp: now,
			Value:     respValue,
		}

		metric := hmetrics.MetricHeader{
			Type:   monitoredMetric.Type,
			ID:     monitoredMetric.Name, // the caller (collector manager) may override when it determines the real ID
			Tenant: tenantId,
			Data:   data,
		}

		metrics = append(metrics, metric)
	} else {
		// if the value was nil, it just means there was no metric data yet so no need to warn about that
		if respValueObject != nil {
			log.Warningf("Cannot process value of metric [%v] from Jolokia endpoint [%v]. err=%v",
				specificMetricName, url, err)
		}
	}

	return metrics
}

// ReturnValueAsMap can be used to return a response value as a map.
// This can be used also to return a composite attribute value as well.
func returnValueAsMap(value interface{}) (map[string]interface{}, error) {
	var theMap map[string]interface{}
	var err error

	switch value.(type) {
	case map[string]interface{}:
		theMap = value.(map[string]interface{})
	default:
		err = fmt.Errorf("Value is not a map but is of type [%T] and will not be processed", value)
	}

	return theMap, err
}

// ReturnValueAsFloat can be used to return a response value as a float.
// This can be used also to return the value of a specific path in a composite attribute.
func returnValueAsFloat(value interface{}) (float64, error) {
	var theFloat float64
	var err error

	switch value.(type) {
	case float64:
		theFloat = value.(float64)
	case float32:
		theFloat = float64(value.(float32))
	case string:
		theFloat, err = strconv.ParseFloat(value.(string), 64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		theFloat, err = strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
	case bool:
		if value.(bool) {
			theFloat = 1.0
		} else {
			theFloat = 0.0
		}
	default:
		err = fmt.Errorf("Metric value [%v] is of an incompatible non-float type [%T] and will not be processed", value, value)
	}

	return theFloat, err
}
