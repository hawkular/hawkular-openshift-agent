/*
   Copyright 2016 Red Hat, Inc. and/or its affiliates
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

package k8s

import (
	"testing"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/tags"
)

func TestConfigMapEntryYamlNilTags(t *testing.T) {
	yaml1 := `
endpoints:
- type: prometheus
  protocol: https
  port: 8888
  path: /the/path
  collection_interval_secs: 12345
  metrics:
  - id: metric1id
    name: metric1
    type: gauge
`
	cme, err := UnmarshalConfigMapEntry(yaml1)
	if err != nil {
		t.Fatalf("Could not unmarshal ConfigMapEntry yaml. err=%v", err)
	}

	if cme.Endpoints[0].Type != collector.ENDPOINT_TYPE_PROMETHEUS {
		t.Fatalf("Endpoint.Type is wrong")
	}
	if cme.Endpoints[0].Tags == nil {
		t.Fatalf("Endpoint tags should not be nil")
	}
	if len(cme.Endpoints[0].Tags) != 0 {
		t.Fatalf("Endpoint tags should be empty but not nil")
	}
	if cme.Endpoints[0].Metrics[0].Tags == nil {
		t.Fatalf("Metric tags should not be nil")
	}
	if len(cme.Endpoints[0].Metrics[0].Tags) != 0 {
		t.Fatalf("Metric tags should be empty but not nil")
	}

	yaml2, err := MarshalConfigMapEntry(cme)
	if err != nil {
		t.Fatalf("Could not marshal ConfigMapEntry. err=%v", err)
	}

	cme2, err := UnmarshalConfigMapEntry(yaml2)
	if err != nil {
		t.Fatalf("Could not unmarshal ConfigMapEntry yaml. err=%v", err)
	}
	if cme2.Endpoints[0].Tags == nil || len(cme2.Endpoints[0].Tags) != 0 {
		t.Fatalf("Endpoint tags should be empty but not nil")
	}
	if cme2.Endpoints[0].Metrics[0].Tags == nil || len(cme2.Endpoints[0].Metrics[0].Tags) != 0 {
		t.Fatalf("Metric tags should be empty but not nil")
	}

}

func TestYamlText(t *testing.T) {
	cme := NewConfigMapEntry()
	cme.Endpoints = append(cme.Endpoints, K8SEndpoint{
		Collection_Interval_Secs: 123,
		Type:     collector.ENDPOINT_TYPE_PROMETHEUS,
		Protocol: K8S_ENDPOINT_PROTOCOL_HTTP,
		Port:     1111,
		Path:     "/1111",
		Metrics: []collector.MonitoredMetric{
			collector.MonitoredMetric{
				Id:   "metric1id",
				Type: hmetrics.Gauge,
				Name: "metric1",
				Tags: tags.Tags{
					"tag1": "tag1value",
				},
			},
			collector.MonitoredMetric{
				Id:   "metric2id",
				Type: hmetrics.Counter,
				Name: "metric2",
			},
		},
	})
	cme.Endpoints = append(cme.Endpoints, K8SEndpoint{
		Type:     collector.ENDPOINT_TYPE_JOLOKIA,
		Protocol: K8S_ENDPOINT_PROTOCOL_HTTPS,
		Port:     2222,
		Path:     "/2222",
		Collection_Interval_Secs: 123,
	})

	// I just want to see what happens if you don't specify the metrics slice in the second endpoint
	if len(cme.Endpoints[0].Metrics) != 2 {
		t.Fatalf("Should have two metrics in first endpoint: %v", cme)
	}
	if len(cme.Endpoints[1].Metrics) != 0 {
		t.Fatalf("Should have zero metrics in second endpoint: %v", cme)
	}

	// see that we can convert to YAML
	yaml, err := MarshalConfigMapEntry(cme)
	if err != nil {
		t.Fatalf("Could not marshal ConfigMapEntry. err=%v", err)
	}

	t.Logf("ConfigMapEntry YAML:\n%v\n", yaml)
}

func TestConfigMapEntryYaml(t *testing.T) {
	yaml1 := `
endpoints:
- type: prometheus
  protocol: https
  port: 8888
  path: /the/path
  collection_interval_secs: 12345
  tags:
    endpointtagname1: endpointtag1
    endpointtagname2: endpointtag2
    endpointtagname3: endpointtag3
  metrics:
  - id: metric1id
    name: metric1
    type: gauge
    tags:
      tagname1: tagvalue1
      tagname2: ${POD:name}
      tagname3: $HOSTNAME
  - id: metric2id
    name: metric2
    type: counter
`
	cme, err := UnmarshalConfigMapEntry(yaml1)
	if err != nil {
		t.Fatalf("Could not unmarshal ConfigMapEntry yaml. err=%v", err)
	}

	if cme.Endpoints[0].Type != collector.ENDPOINT_TYPE_PROMETHEUS {
		t.Fatalf("Endpoint.Type is wrong")
	}
	if cme.Endpoints[0].Protocol != K8S_ENDPOINT_PROTOCOL_HTTPS {
		t.Fatalf("Endpoint.Protocol is wrong")
	}
	if cme.Endpoints[0].Port != 8888 {
		t.Fatalf("Endpoint.Port is wrong")
	}
	if cme.Endpoints[0].Path != "/the/path" {
		t.Fatalf("Endpoint.Path is wrong")
	}
	if cme.Endpoints[0].Collection_Interval_Secs != 12345 {
		t.Fatalf("Endpoint.Collection_Interval is wrong")
	}
	if len(cme.Endpoints[0].Metrics) != 2 {
		t.Fatalf("Endpoint.Metrics length is wrong")
	}
	if cme.Endpoints[0].Metrics[0].Id != "metric1id" {
		t.Fatalf("Endpoint.Metrics[0] id is wrong")
	}
	if cme.Endpoints[0].Metrics[0].Name != "metric1" {
		t.Fatalf("Endpoint.Metrics[0] name is wrong")
	}
	if cme.Endpoints[0].Metrics[0].Type != hmetrics.Gauge {
		t.Fatalf("Endpoint.Metrics[0] type is wrong")
	}
	if cme.Endpoints[0].Tags["endpointtagname1"] != "endpointtag1" {
		t.Fatalf("Endpoint tag 1 is wrong")
	}
	if cme.Endpoints[0].Tags["endpointtagname2"] != "endpointtag2" {
		t.Fatalf("Endpoint tag 2 is wrong")
	}
	if cme.Endpoints[0].Tags["endpointtagname3"] != "endpointtag3" {
		t.Fatalf("Endpoint tag 3 is wrong")
	}
	if cme.Endpoints[0].Metrics[0].Tags["tagname1"] != "tagvalue1" {
		t.Fatalf("Endpoint.Metrics[0] tag 1 is wrong")
	}
	if cme.Endpoints[0].Metrics[0].Tags["tagname2"] != "${POD:name}" {
		t.Fatalf("Endpoint.Metrics[0] tag 2 is wrong")
	}
	if cme.Endpoints[0].Metrics[0].Tags["tagname3"] != "$HOSTNAME" {
		t.Fatalf("Endpoint.Metrics[0] tag 3 is wrong")
	}

	yaml2, err := MarshalConfigMapEntry(cme)
	if err != nil {
		t.Fatalf("Could not marshal ConfigMapEntry. err=%v", err)
	}

	cme2, err := UnmarshalConfigMapEntry(yaml2)
	if err != nil {
		t.Fatalf("Could not unmarshal ConfigMapEntry yaml. err=%v", err)
	}
	if cme.Endpoints[0].Collection_Interval_Secs != cme2.Endpoints[0].Collection_Interval_Secs {
		t.Fatalf("Marshalling did not produce expected yaml. [%v] != [%v]", cme, cme2)
	}
}

func TestConfigMap(t *testing.T) {
	yaml1 := `
endpoints:
- type: jolokia
  protocol: https
  port: 8888
  collection_interval_secs: 12345
  path: /the/path
  metrics: []
`
	cme1, err := UnmarshalConfigMapEntry(yaml1)
	if err != nil {
		t.Fatalf("Could not unmarshal ConfigMapEntry yaml. err=%v", err)
	}

	cm := NewConfigMap("the ns", "the name", cme1)

	if cm.Entry != cme1 {
		t.Fatalf("Config map entry wasn't saved correctly")
	}
}
