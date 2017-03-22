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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	hawkhttp "github.com/hawkular/hawkular-openshift-agent/http"
)

/*
   Responses to all the different combinations of Jolokia read requests the agent will submit.
   The agent submits POST requests, but POST or GET doesn't matter because the responses are the same.
*/

// 1. Request all attributes for a single MBean: read/java.lang:type=Memory
var singleMBeanMultpleAttributes = `
  [{
    "request":{
      "mbean":"java.lang:type=Memory",
      "type":"read"
     },
     "value":{
       "HeapMemoryUsage":{
         "committed":123,
         "init":456,
         "max":789,
         "used":12345
       },
       "NonHeapMemoryUsage":{
         "committed":321,
         "init":654,
         "max":987,
         "used":54321
       },
       "ObjectName":{
         "objectName":"java.lang:type=Memory"
       },
       "ObjectPendingFinalizationCount":0,
       "Verbose":false
     },
     "timestamp":1490064914,
     "status":200
   }]
`

// 2. Request one non-composite attribute from one MBean: read/java.lang:type=Memory/ObjectPendingFinalizationCount
var singleMBeanSingleBasicAttribute = `
  [{
    "request":{
      "mbean":"java.lang:type=Memory",
      "attribute":"ObjectPendingFinalizationCount",
      "type":"read"
    },
    "value":123456,
    "timestamp":1490064943,
    "status":200
  }]
`

// 3. Request one composite attribute from one MBean: read/java.lang:type=Memory/HeapMemoryUsage
var singleMBeanSingleCompositeAttribute = `
   [{
     "request":{
       "mbean":"java.lang:type=Memory",
       "attribute":"HeapMemoryUsage",
       "type":"read"
     },
     "value":{
       "committed":123,
       "init":456,
       "max":789,
       "used":12345
     },
     "timestamp":1490064906,
     "status":200
   }]
`

// 4. Request one part of a composite attribute from one MBean: read/java.lang:type=Memory/HeapMemoryUsage/init
var singleMBeanSingleCompositeAttributeSinglePart = `
  [{
    "request":{
      "path":"init",
      "mbean":"java.lang:type=Memory",
      "attribute":"HeapMemoryUsage",
      "type":"read"
    },
    "value":12345,
    "timestamp":1490065093,
    "status":200
  }]
`

// 5. Request one non-composite attribute from multiple MBeans: read/java.lang:type=* /ObjectPendingFinalizationCount
var multipleMBeansSingleBasicAttribute = `
  [{
    "request":{
      "mbean":"java.lang:type=*",
      "attribute":"ObjectPendingFinalizationCount",
      "type":"read"
    },
    "value":{
      "java.lang:type=Memory":{
        "ObjectPendingFinalizationCount":5
      }
    },
    "timestamp":1490064984,
    "status":200
  }]
`

// 6. Request one composite attribute from multiple MBeans: read/java.lang:type=* /HeapMemoryUsage
var multipleMBeansSingleCompositeAttribute = `
  [{
    "request":{
      "mbean":"java.lang:type=*",
      "attribute":"HeapMemoryUsage",
      "type":"read"
    },
    "value":{
      "java.lang:type=Memory":{
        "HeapMemoryUsage":{
          "committed":123,
          "init":456,
          "max":789,
          "used":12345
        }
      }
    },
    "timestamp":1490065033,
    "status":200
  }]
`

func TestSingleMBeanMultipleAttributes(t *testing.T) {

	// setup our mock jolokia server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponseJson := singleMBeanMultpleAttributes
		fmt.Fprintln(w, mockResponseJson)
	}))
	defer ts.Close()

	httpConfig := hawkhttp.HttpClientConfig{}
	httpClient, err := httpConfig.BuildHttpClient()

	monitoredMetricName := "java.lang:type=Memory#*"
	reqs := NewJolokiaRequests()
	req := JolokiaRequest{}
	metricNameParts, _ := ParseMetricNameForJolokiaRequest(monitoredMetricName, &req)
	reqs.AddRequest(req)

	resp, err := reqs.SendRequests(ts.URL, &security.Credentials{}, httpClient)
	if err != nil {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	url := ts.URL
	now := time.Now()
	tenantId := "hawkular"
	respValue := resp.Responses[0].Value
	monitoredMetric := collector.MonitoredMetric{
		Name: monitoredMetricName,
	}

	metrics := ProcessJolokiaResponseValueObject(url, now, tenantId, respValue, monitoredMetric, metricNameParts)

	if !(len(metrics) == 10) {
		t.Fatalf("Missing some metric data: %v", metrics)
	}

	var metricToTest hmetrics.MetricHeader
	var dataptToTest hmetrics.Datapoint

	// metric 1
	metricToTest = metrics[0]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 123) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "committed") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 2
	metricToTest = metrics[1]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 456) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "init") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 3
	metricToTest = metrics[2]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 789) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "max") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 4
	metricToTest = metrics[3]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 12345) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "used") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 5
	metricToTest = metrics[4]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 321) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "NonHeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "committed") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 6
	metricToTest = metrics[5]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 654) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "NonHeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "init") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 7
	metricToTest = metrics[6]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 987) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "NonHeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "max") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 8
	metricToTest = metrics[7]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 54321) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 2) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "NonHeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "used") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 9
	metricToTest = metrics[8]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 0) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 1) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "ObjectPendingFinalizationCount") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}

	// metric 10
	metricToTest = metrics[9]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 0) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 1) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "Verbose") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
}

func TestSingleMBeanSingleBasicAttribute(t *testing.T) {

	// setup our mock jolokia server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponseJson := singleMBeanSingleBasicAttribute
		fmt.Fprintln(w, mockResponseJson)
	}))
	defer ts.Close()

	httpConfig := hawkhttp.HttpClientConfig{}
	httpClient, err := httpConfig.BuildHttpClient()

	monitoredMetricName := "java.lang:type=Memory#ObjectPendingFinalizationCount"
	reqs := NewJolokiaRequests()
	req := JolokiaRequest{}
	metricNameParts, _ := ParseMetricNameForJolokiaRequest(monitoredMetricName, &req)
	reqs.AddRequest(req)

	resp, err := reqs.SendRequests(ts.URL, &security.Credentials{}, httpClient)
	if err != nil {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	url := ts.URL
	now := time.Now()
	tenantId := "hawkular"
	respValue := resp.Responses[0].Value
	monitoredMetric := collector.MonitoredMetric{
		Name: monitoredMetricName,
	}

	metrics := ProcessJolokiaResponseValueObject(url, now, tenantId, respValue, monitoredMetric, metricNameParts)

	if !(len(metrics) == 1) {
		t.Fatalf("Missing some metric data: %v", metrics)
	}

	var metricToTest hmetrics.MetricHeader
	var dataptToTest hmetrics.Datapoint

	// metric 1
	metricToTest = metrics[0]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 123456) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 0) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
}

func TestSingleMBeanSingleCompositeAttribute(t *testing.T) {

	// setup our mock jolokia server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponseJson := singleMBeanSingleCompositeAttribute
		fmt.Fprintln(w, mockResponseJson)
	}))
	defer ts.Close()

	httpConfig := hawkhttp.HttpClientConfig{}
	httpClient, err := httpConfig.BuildHttpClient()

	monitoredMetricName := "java.lang:type=Memory#HeapMemoryUsage#*"
	reqs := NewJolokiaRequests()
	req := JolokiaRequest{}
	metricNameParts, _ := ParseMetricNameForJolokiaRequest(monitoredMetricName, &req)
	reqs.AddRequest(req)

	resp, err := reqs.SendRequests(ts.URL, &security.Credentials{}, httpClient)
	if err != nil {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	url := ts.URL
	now := time.Now()
	tenantId := "hawkular"
	respValue := resp.Responses[0].Value
	monitoredMetric := collector.MonitoredMetric{
		Name: monitoredMetricName,
	}

	metrics := ProcessJolokiaResponseValueObject(url, now, tenantId, respValue, monitoredMetric, metricNameParts)

	if !(len(metrics) == 4) {
		t.Fatalf("Missing some metric data: %v", metrics)
	}

	var metricToTest hmetrics.MetricHeader
	var dataptToTest hmetrics.Datapoint

	// metric 1
	metricToTest = metrics[0]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 123) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 1) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "committed") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 2
	metricToTest = metrics[1]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 456) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 1) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "init") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 3
	metricToTest = metrics[2]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 789) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 1) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "max") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}

	// metric 4
	metricToTest = metrics[3]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 12345) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 1) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "used") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}
}

func TestSingleMBeanSingleCompositeAttributeSinglePart(t *testing.T) {

	// setup our mock jolokia server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponseJson := singleMBeanSingleCompositeAttributeSinglePart
		fmt.Fprintln(w, mockResponseJson)
	}))
	defer ts.Close()

	httpConfig := hawkhttp.HttpClientConfig{}
	httpClient, err := httpConfig.BuildHttpClient()

	monitoredMetricName := "java.lang:type=Memory#HeapMemoryUsage#init"
	reqs := NewJolokiaRequests()
	req := JolokiaRequest{}
	metricNameParts, _ := ParseMetricNameForJolokiaRequest(monitoredMetricName, &req)
	reqs.AddRequest(req)

	resp, err := reqs.SendRequests(ts.URL, &security.Credentials{}, httpClient)
	if err != nil {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	url := ts.URL
	now := time.Now()
	tenantId := "hawkular"
	respValue := resp.Responses[0].Value
	monitoredMetric := collector.MonitoredMetric{
		Name: monitoredMetricName,
	}

	metrics := ProcessJolokiaResponseValueObject(url, now, tenantId, respValue, monitoredMetric, metricNameParts)

	if !(len(metrics) == 1) {
		t.Fatalf("Missing some metric data: %v", metrics)
	}

	var metricToTest hmetrics.MetricHeader
	var dataptToTest hmetrics.Datapoint

	// metric 1
	metricToTest = metrics[0]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 12345) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 0) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
}

func TestMultipleMBeansSingleBasicAttribute(t *testing.T) {
	// setup our mock jolokia server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponseJson := multipleMBeansSingleBasicAttribute
		fmt.Fprintln(w, mockResponseJson)
	}))
	defer ts.Close()

	httpConfig := hawkhttp.HttpClientConfig{}
	httpClient, err := httpConfig.BuildHttpClient()

	monitoredMetricName := "java.lang:type=*#ObjectPendingFinalizationCount"
	reqs := NewJolokiaRequests()
	req := JolokiaRequest{}
	metricNameParts, _ := ParseMetricNameForJolokiaRequest(monitoredMetricName, &req)
	reqs.AddRequest(req)

	resp, err := reqs.SendRequests(ts.URL, &security.Credentials{}, httpClient)
	if err != nil {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	url := ts.URL
	now := time.Now()
	tenantId := "hawkular"
	respValue := resp.Responses[0].Value
	monitoredMetric := collector.MonitoredMetric{
		Name: monitoredMetricName,
	}

	metrics := ProcessJolokiaResponseValueObject(url, now, tenantId, respValue, monitoredMetric, metricNameParts)

	if !(len(metrics) == 1) {
		t.Fatalf("Missing some metric data: %v", metrics)
	}

	var metricToTest hmetrics.MetricHeader
	var dataptToTest hmetrics.Datapoint

	// metric 1
	metricToTest = metrics[0]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 5) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 1) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["type"] == "Memory") {
		t.Fatalf("Bad key tag: %v", metricToTest)
	}
}

func TestMultipleMBeansSingleCompositeAttribute(t *testing.T) {
	// setup our mock jolokia server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponseJson := multipleMBeansSingleCompositeAttribute
		fmt.Fprintln(w, mockResponseJson)
	}))
	defer ts.Close()

	httpConfig := hawkhttp.HttpClientConfig{}
	httpClient, err := httpConfig.BuildHttpClient()

	monitoredMetricName := "java.lang:type=*#HeapMemoryUsage#*"
	reqs := NewJolokiaRequests()
	req := JolokiaRequest{}
	metricNameParts, _ := ParseMetricNameForJolokiaRequest(monitoredMetricName, &req)
	reqs.AddRequest(req)

	resp, err := reqs.SendRequests(ts.URL, &security.Credentials{}, httpClient)
	if err != nil {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	url := ts.URL
	now := time.Now()
	tenantId := "hawkular"
	respValue := resp.Responses[0].Value
	monitoredMetric := collector.MonitoredMetric{
		Name: monitoredMetricName,
	}

	metrics := ProcessJolokiaResponseValueObject(url, now, tenantId, respValue, monitoredMetric, metricNameParts)

	if !(len(metrics) == 4) {
		t.Fatalf("Missing some metric data: %v", metrics)
	}

	var metricToTest hmetrics.MetricHeader
	var dataptToTest hmetrics.Datapoint

	// metric 1
	metricToTest = metrics[0]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 123) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 3) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "committed") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["type"] == "Memory") {
		t.Fatalf("Bad key tag: %v", metricToTest)
	}

	// metric 2
	metricToTest = metrics[1]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 456) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 3) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "init") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["type"] == "Memory") {
		t.Fatalf("Bad key tag: %v", metricToTest)
	}

	// metric 3
	metricToTest = metrics[2]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 789) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 3) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "max") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["type"] == "Memory") {
		t.Fatalf("Bad key tag: %v", metricToTest)
	}

	// metric 4
	metricToTest = metrics[3]
	dataptToTest = metricToTest.Data[0]

	if !(metricToTest.ID == monitoredMetricName) {
		t.Fatalf("Bad id: %v", metricToTest.ID)
	}
	if !(dataptToTest.Value.(float64) == 12345) {
		t.Fatalf("Bad value: %v", metricToTest)
	}
	if !(len(dataptToTest.Tags) == 3) {
		t.Fatalf("Bad tags: %v", metricToTest)
	}
	if !(dataptToTest.Tags["1"] == "HeapMemoryUsage") {
		t.Fatalf("Bad $1 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["2"] == "used") {
		t.Fatalf("Bad $2 tag: %v", metricToTest)
	}
	if !(dataptToTest.Tags["type"] == "Memory") {
		t.Fatalf("Bad key tag: %v", metricToTest)
	}
}
