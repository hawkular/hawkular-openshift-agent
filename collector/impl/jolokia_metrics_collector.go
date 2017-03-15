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

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/http"
	"github.com/hawkular/hawkular-openshift-agent/jolokia"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

type JolokiaMetricsCollector struct {
	ID          collector.CollectorID
	Identity    *security.Identity
	Endpoint    *collector.Endpoint
	Environment map[string]string
}

func NewJolokiaMetricsCollector(id collector.CollectorID, identity security.Identity, endpoint collector.Endpoint, env map[string]string) (mc *JolokiaMetricsCollector) {
	mc = &JolokiaMetricsCollector{
		ID:          id,
		Identity:    &identity,
		Endpoint:    &endpoint,
		Environment: env,
	}

	return
}

// GetId implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) GetID() collector.CollectorID {
	return jc.ID
}

// GetEndpoint implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) GetEndpoint() *collector.Endpoint {
	return jc.Endpoint
}

// GetAdditionalEnvironment implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) GetAdditionalEnvironment() map[string]string {
	return jc.Environment
}

// CollectMetrics does the real work of actually connecting to a remote Jolokia endpoint,
// collects all metrics it find there, and returns those metrics.
// CollectMetrics implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) CollectMetrics() (metrics []hmetrics.MetricHeader, err error) {

	url := jc.Endpoint.URL
	now := time.Now()

	if len(jc.Endpoint.Metrics) == 0 {
		log.Debugf("There are no metrics defined for Jolokia endpoint [%v]", url)
		metrics = make([]hmetrics.MetricHeader, 0)
		return
	}

	log.Debugf("Told to collect [%v] Jolokia metrics from [%v]", len(jc.Endpoint.Metrics), url)

	httpConfig := http.HttpClientConfig{
		Identity: jc.Identity,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: jc.Endpoint.TLS.Skip_Certificate_Validation,
		},
	}
	httpClient, err := httpConfig.BuildHttpClient()
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
	responses, err := requests.SendRequests(url, &jc.Endpoint.Credentials, httpClient)
	if err != nil {
		err = fmt.Errorf("Failed to collect metrics from Jolokia endpoint [%v]. err=%v", url, err)
		return
	}

	// convert the metric data we got from Jolokia into our Hawkular-Metrics data format
	metrics = make([]hmetrics.MetricHeader, 0)

	for i, resp := range responses.Responses {
		if resp.IsSuccess() {
			if respValue, ok := resp.Value.(float64); ok {
				data := make([]hmetrics.Datapoint, 1)
				data[0] = hmetrics.Datapoint{
					Timestamp: now,
					Value:     respValue,
				}

				metric := hmetrics.MetricHeader{
					Type:   jc.Endpoint.Metrics[i].Type,
					ID:     jc.Endpoint.Metrics[i].Name, // the caller (collector manager) will determine the real ID
					Tenant: jc.Endpoint.Tenant,
					Data:   data,
				}

				metrics = append(metrics, metric)
			} else {
				log.Debugf("Received non-float value [%v] for metric [%v] from Jolokia endpoint [%v].",
					resp.Value, jc.Endpoint.Metrics[i].Name, url)

			}
		} else {
			log.Warningf("Failed to collect metric [%v] from Jolokia endpoint [%v]. err=%v",
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

// CollectMetricDetails implements a method from MetricsCollector interface
func (jc *JolokiaMetricsCollector) CollectMetricDetails(metricNames []string) ([]collector.MetricDetails, error) {
	// TODO: can we get information like metric type and description from JMX?
	return make([]collector.MetricDetails, 0), nil
}
