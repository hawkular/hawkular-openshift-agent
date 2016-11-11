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

package config

import (
	"os"
	"testing"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/config/tags"
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

	if conf.Hawkular_Server.URL != "http://TestEnvVar:9090" {
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
	if conf.Hawkular_Server.URL != "http://127.0.0.1:8080" {
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
			URL: "http://server:80",
		},
		Kubernetes: Kubernetes{
			Pod_Namespace: "TestMarshalUnmarshal namespace",
			Pod_Name:      "TestMarshalUnmarshal name",
		},
		Endpoints: []collector.Endpoint{
			{
				URL:  "http://host:1111/metrics",
				Type: collector.ENDPOINT_TYPE_PROMETHEUS,
				Collection_Interval_Secs: 123,
			},
			{
				URL:  "http://host:2222/jolokia",
				Type: collector.ENDPOINT_TYPE_JOLOKIA,
				Collection_Interval_Secs: 234,
			},
		},
	}

	testConf.Endpoints[0].Metrics = make([]collector.MonitoredMetric, 1)
	testConf.Endpoints[0].Metrics[0] = collector.MonitoredMetric{
		Name: "a:b=c",
		Type: "gauge",
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
	if conf.Collector.Metric_ID_Prefix != "" {
		t.Errorf("Failed to unmarshal empty metric ID prefix:\n%v", conf)
	}
	if conf.Hawkular_Server.URL != "http://server:80" {
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
	if conf.Collector.Tags == nil || len(conf.Collector.Tags) > 0 {
		t.Error("Global tags should be empty but not nil")
	}
	if conf.Endpoints[0].Tags == nil || len(conf.Endpoints[0].Tags) > 0 {
		t.Error("Endpoint tags should be empty but not nil")
	}
	if conf.Endpoints[0].Metrics[0].Tags == nil || len(conf.Endpoints[0].Metrics[0].Tags) > 0 {
		t.Error("Metric tags should be empty but not nil")
	}
}

func TestLoadSave(t *testing.T) {
	testConf := Config{
		Identity: security.Identity{
			Cert_File:        "/my/cert",
			Private_Key_File: "/my/key",
		},
		Collector: Collector{
			Minimum_Collection_Interval_Secs: 12345,
			Metric_ID_Prefix:                 "prefix",
			Tags: tags.Tags{
				"tag1": "tagvalue1",
				"tag2": "tagvalue2",
			},
		},
		Hawkular_Server: Hawkular_Server{
			URL: "http://TestLoadSave:80",
		},
		Kubernetes: Kubernetes{
			Pod_Namespace: "TestLoadSave namespace",
			Pod_Name:      "TestLoadSave name",
		},
		Endpoints: []collector.Endpoint{
			{
				URL:  "http://host:1111/metrics",
				Type: collector.ENDPOINT_TYPE_PROMETHEUS,
				Collection_Interval_Secs: 123,
			},
			{
				URL:  "http://host:2222/jolokia",
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

	t.Logf("Config from file\n%v", conf)

	if conf.Identity.Cert_File != "/my/cert" {
		t.Errorf("Failed to unmarshal identity:\n%v", conf)
	}
	if conf.Identity.Private_Key_File != "/my/key" {
		t.Errorf("Failed to unmarshal identity:\n%v", conf)
	}
	if conf.Collector.Minimum_Collection_Interval_Secs != 12345 {
		t.Errorf("Failed to unmarshal collection interval:\n%v", conf)
	}
	if conf.Collector.Metric_ID_Prefix != "prefix" {
		t.Errorf("Failed to unmarshal metric ID prefix:\n%v", conf)
	}
	if conf.Hawkular_Server.URL != "http://TestLoadSave:80" {
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
	if conf.Collector.Tags["tag1"] != "tagvalue1" {
		t.Error("Tag1 is not correct")
	}
	if conf.Collector.Tags["tag2"] != "tagvalue2" {
		t.Error("Tag2 is not correct")
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
