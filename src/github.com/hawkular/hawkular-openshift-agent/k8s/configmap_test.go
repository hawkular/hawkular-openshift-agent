package k8s

import (
	"testing"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
)

func TestYamlText(t *testing.T) {
	cme := NewConfigMapEntry()
	cme.Endpoints = append(cme.Endpoints, K8SEndpoint{
		Collection_Interval_Secs: 123,
		Type:     collector.ENDPOINT_TYPE_PROMETHEUS,
		Protocol: K8S_ENDPOINT_PROTOCOL_HTTP,
		Port:     1111,
		Path:     "/1111",
		Metrics: []K8SMetric{
			K8SMetric{
				Type: hmetrics.Gauge,
				Name: "metric1",
			},
			K8SMetric{
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

func aTestConfigMapEntryYaml(t *testing.T) {
	yaml1 := `
collection_interval_secs: 12345
endpoints:
  -type: prometheus
   protocol: https
   port: 8888
   path: /the/path
   metrics: []
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

func aTestConfigMap(t *testing.T) {
	yaml1 := `
collection_interval_secs: 12345
endpoints:
  -type: prometheus
   protocol: https
   port: 8888
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
