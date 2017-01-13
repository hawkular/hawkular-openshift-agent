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

package k8s

import (
	"testing"

	"github.com/hawkular/hawkular-openshift-agent/collector"
)

func TestConfigMaps(t *testing.T) {
	cme1 := NewConfigMapEntry()
	cme1.Endpoints = append(cme1.Endpoints, K8SEndpoint{
		Type:                collector.ENDPOINT_TYPE_JOLOKIA,
		Protocol:            K8S_ENDPOINT_PROTOCOL_HTTPS,
		Port:                1111,
		Path:                "/1111",
		Collection_Interval: "123s",
	})

	cme2 := NewConfigMapEntry()
	cme2.Endpoints = append(cme2.Endpoints, K8SEndpoint{
		Type:                collector.ENDPOINT_TYPE_PROMETHEUS,
		Protocol:            K8S_ENDPOINT_PROTOCOL_HTTP,
		Port:                2222,
		Path:                "/2222",
		Collection_Interval: "987s",
	})

	// put config maps with different names in the same namespace
	cm1 := NewConfigMap("the ns", "the name1", cme1)
	cm2 := NewConfigMap("the ns", "the name2", cme2)

	cms := NewConfigMaps()

	if _, ok := cms.GetEntry("the ns", "the name1"); ok {
		t.Fatalf("Should not have any entries yet")
	}
	cms.AddEntry(cm1)

	if _, ok := cms.GetEntry("the ns", "the name1"); !ok {
		t.Fatalf("Failed to get the entry 1")
	}

	cms.AddEntry(cm2)
	if _, ok := cms.GetEntry("the ns", "the name1"); !ok {
		t.Fatalf("Failed to get the entry 1")
	}
	if _, ok := cms.GetEntry("the ns", "the name2"); !ok {
		t.Fatalf("Failed to get the entry 2")
	}

	cms.RemoveEntry("the ns", "the name1")
	if _, ok := cms.GetEntry("the ns", "the name1"); ok {
		t.Fatalf("Entry 1 should have been removed")
	}
	if _, ok := cms.GetEntry("the ns", "the name2"); !ok {
		t.Fatalf("Failed to get the entry 2")
	}

	cms.RemoveEntry("the ns", "the name2")
	if _, ok := cms.GetEntry("the ns", "the name1"); ok {
		t.Fatalf("Entry 1 should have been removed")
	}
	if _, ok := cms.GetEntry("the ns", "the name2"); ok {
		t.Fatalf("Entry 2 should have been removed")
	}

	// notice that we expect that names can be the same across namespaces
	cm1 = NewConfigMap("the ns1", "the name", cme1)
	cm2 = NewConfigMap("the ns2", "the name", cme2)
	if _, ok := cms.GetEntry("the ns", "the name1"); ok {
		t.Fatalf("Should not have any entries yet")
	}
	cms.AddEntry(cm1)

	if _, ok := cms.GetEntry("the ns1", "the name"); !ok {
		t.Fatalf("Failed to get the entry 1")
	}

	cms.AddEntry(cm2)
	if _, ok := cms.GetEntry("the ns1", "the name"); !ok {
		t.Fatalf("Failed to get the entry 1")
	}
	if _, ok := cms.GetEntry("the ns2", "the name"); !ok {
		t.Fatalf("Failed to get the entry 2")
	}

	cms.RemoveEntry("the ns1", "the name")
	if _, ok := cms.GetEntry("the ns1", "the name"); ok {
		t.Fatalf("Entry 1 should have been removed")
	}
	if _, ok := cms.GetEntry("the ns2", "the name"); !ok {
		t.Fatalf("Failed to get the entry 2")
	}

	cms.RemoveEntry("the ns2", "the name")
	if _, ok := cms.GetEntry("the ns1", "the name"); ok {
		t.Fatalf("Entry 1 should have been removed")
	}
	if _, ok := cms.GetEntry("the ns2", "the name"); ok {
		t.Fatalf("Entry 2 should have been removed")
	}

	// test ClearNamespace
	cms.AddEntry(cm1)
	cms.AddEntry(cm2)

	cms.ClearNamespace("the ns1")
	if _, ok := cms.GetEntry("the ns1", "the name"); ok {
		t.Fatalf("Namespace 1 entries should have been removed")
	}
	if _, ok := cms.GetEntry("the ns2", "the name"); !ok {
		t.Fatalf("Failed to get the entry 2")
	}

	cms.ClearNamespace("the ns2")
	if _, ok := cms.GetEntry("the ns1", "the name"); ok {
		t.Fatalf("Namespace 1 entries should have been removed")
	}
	if _, ok := cms.GetEntry("the ns2", "the name"); ok {
		t.Fatalf("Namespace 2 entries should have been removed")
	}

	// test ClearAll
	cms.AddEntry(cm1)
	cms.AddEntry(cm2)

	cms.ClearAll()

	if _, ok := cms.GetEntry("the ns1", "the name"); ok {
		t.Fatalf("Namespace 1 entries should have been removed")
	}
	if _, ok := cms.GetEntry("the ns2", "the name"); ok {
		t.Fatalf("Namespace 2 entries should have been removed")
	}
}
