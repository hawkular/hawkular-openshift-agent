package config

import (
	"os"
	"testing"

	"github.com/hawkular/hawkular-openshift-agent/collector"
)

func TestEnvVar(t *testing.T) {
	defer os.Setenv(ENV_HS_URL, os.Getenv(ENV_HS_URL))
	defer os.Setenv(ENV_HS_TOKEN, os.Getenv(ENV_HS_TOKEN))
	defer os.Setenv(ENV_K8S_POD_NAMESPACE, os.Getenv(ENV_K8S_POD_NAMESPACE))
	defer os.Setenv(ENV_K8S_POD_NAME, os.Getenv(ENV_K8S_POD_NAME))
	os.Setenv(ENV_HS_URL, "http://TestEnvVar:9090")
	os.Setenv(ENV_HS_TOKEN, "abc123")
	os.Setenv(ENV_K8S_POD_NAMESPACE, "TestEnvVar pod namespace")
	os.Setenv(ENV_K8S_POD_NAME, "TestEnvVar pod name")

	conf := NewConfig()

	if conf.Hawkular_Server.Url != "http://TestEnvVar:9090" {
		t.Error("Hawkular Server URL is wrong")
	}
	if conf.Hawkular_Server.Credentials.Token != "abc123" {
		t.Error("Hawkular Server Token is wrong")
	}
	if conf.Kubernetes.Pod_Namespace != "TestEnvVar pod namespace" {
		t.Error("Pod namespace is wrong")
	}
	if conf.Kubernetes.Pod_Name != "TestEnvVar pod name" {
		t.Error("Pod name is wrong")
	}
}

func TestDefaults(t *testing.T) {
	conf := NewConfig()

	if conf.Collector.Minimum_Collection_Interval_Secs != 10 {
		t.Error("Minimum collection interval default is wrong")
	}
	if conf.Hawkular_Server.Url != "http://127.0.0.1:8080" {
		t.Error("Hawkular Server URL is wrong")
	}
	if conf.Hawkular_Server.Tenant != "hawkular" {
		t.Error("Hawkular Server Tenant is wrong")
	}
	if conf.Hawkular_Server.Credentials.Username != "" {
		t.Error("Hawkular Server Username should be empty by default")
	}
	if conf.Hawkular_Server.Credentials.Password != "" {
		t.Error("Hawkular Server Password should be empty by default")
	}
	if conf.Hawkular_Server.Credentials.Token != "" {
		t.Error("Hawkular Server Token should be empty by default")
	}
	if conf.Kubernetes.Pod_Namespace != "" {
		t.Error("Pod namespace is wrong")
	}
	if conf.Kubernetes.Pod_Name != "" {
		t.Error("Pod name is wrong")
	}
	if len(conf.Endpoints) != 0 {
		t.Error("There should be no endpoints by default")
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	testConf := Config{
		Collector: Collector{
			Minimum_Collection_Interval_Secs: 12345,
		},
		Hawkular_Server: Hawkular_Server{
			Url: "http://server:80",
		},
		Kubernetes: Kubernetes{
			Pod_Namespace: "TestMarshalUnmarshal namespace",
			Pod_Name:      "TestMarshalUnmarshal name",
		},
		Endpoints: []collector.Endpoint{
			{
				Url:  "http://host:1111/metrics",
				Type: collector.ENDPOINT_TYPE_PROMETHEUS,
				Collection_Interval_Secs: 123,
			},
			{
				Url:  "http://host:2222/jolokia",
				Type: collector.ENDPOINT_TYPE_JOLOKIA,
				Collection_Interval_Secs: 234,
			},
		},
	}

	yamlString, err := Marshal(&testConf)
	if err != nil {
		t.Errorf("Failed to marshal: %v", err)
	}
	if yamlString == "" {
		t.Errorf("Failed to marshal - empty yaml string")
	}

	conf, err := Unmarshal(yamlString)
	if err != nil {
		t.Errorf("Failed to unmarshal: %v", err)
	}

	if conf.Collector.Minimum_Collection_Interval_Secs != 12345 {
		t.Errorf("Failed to unmarshal collection interval:\n%v", conf)
	}
	if conf.Hawkular_Server.Url != "http://server:80" {
		t.Errorf("Failed to unmarshal server url:\n%v", conf)
	}
	if conf.Kubernetes.Pod_Namespace != "TestMarshalUnmarshal namespace" {
		t.Error("Pod namespace is wrong")
	}
	if conf.Kubernetes.Pod_Name != "TestMarshalUnmarshal name" {
		t.Error("Pod name is wrong")
	}
	if conf.Endpoints[0].Collection_Interval_Secs != 123 {
		t.Error("First endpoint is not correct")
	}
	if conf.Endpoints[1].Collection_Interval_Secs != 234 {
		t.Error("Second endpoint is not correct")
	}
}

func TestLoadSave(t *testing.T) {
	testConf := Config{
		Collector: Collector{
			Minimum_Collection_Interval_Secs: 12345,
		},
		Hawkular_Server: Hawkular_Server{
			Url: "http://TestLoadSave:80",
		},
		Kubernetes: Kubernetes{
			Pod_Namespace: "TestLoadSave namespace",
			Pod_Name:      "TestLoadSave name",
		},
		Endpoints: []collector.Endpoint{
			{
				Url:  "http://host:1111/metrics",
				Type: collector.ENDPOINT_TYPE_PROMETHEUS,
				Collection_Interval_Secs: 123,
			},
			{
				Url:  "http://host:2222/jolokia",
				Type: collector.ENDPOINT_TYPE_JOLOKIA,
				Collection_Interval_Secs: 234,
			},
		},
	}

	filename := "/tmp/config_test.yaml"
	defer os.Remove(filename)

	err := SaveToFile(filename, &testConf)
	if err != nil {
		t.Errorf("Failed to save to file: %v", err)
	}

	conf, err := LoadFromFile(filename)
	if err != nil {
		t.Errorf("Failed to load from file: %v", err)
	}

	if conf.Collector.Minimum_Collection_Interval_Secs != 12345 {
		t.Errorf("Failed to unmarshal collection interval:\n%v", conf)
	}
	if conf.Hawkular_Server.Url != "http://TestLoadSave:80" {
		t.Errorf("Failed to unmarshal server url:\n%v", conf)
	}
	if conf.Kubernetes.Pod_Namespace != "TestLoadSave namespace" {
		t.Error("Pod namespace is wrong")
	}
	if conf.Kubernetes.Pod_Name != "TestLoadSave name" {
		t.Error("Pod name is wrong")
	}
	if conf.Endpoints[0].Collection_Interval_Secs != 123 {
		t.Error("First endpoint is not correct")
	}
	if conf.Endpoints[1].Collection_Interval_Secs != 234 {
		t.Error("Second endpoint is not correct")
	}

}

func TestError(t *testing.T) {
	_, err := Unmarshal("bogus-yaml")
	if err == nil {
		t.Errorf("Unmarshal should have failed")
	}

	_, err = LoadFromFile("bogus-file-name")
	if err == nil {
		t.Errorf("Load should have failed")
	}
}
