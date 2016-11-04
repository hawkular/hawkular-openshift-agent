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
)

func Test(t *testing.T) {
	p1 := &Pod{
		Node: Node{
			Name: "node-name",
			UID:  "abcxyz",
		},
		Namespace: Namespace{
			Name: "pod1-namespace",
			UID:  "123abc",
		},
		Name: "pod1-name",
		UID:  "pod1-uuid",
	}

	p2 := &Pod{
		Node: Node{
			Name: "node-name",
			UID:  "abcxyz",
		},
		Namespace: Namespace{
			Name: "pod2-namespace",
			UID:  "321cba",
		},
		Name: "pod2-name",
		UID:  "pod2-uuid",
	}

	pi := NewPodInventory(Node{
		Name: "node-name",
		UID:  "abcxyz",
	})

	if len(pi.DiscoveredPods) != 0 {
		t.Fatalf("Should have 0 pods: %v", pi)
	}

	if pi.HasPod(p1) {
		t.Fatalf("Should not have pod: %v", p1)
	}
	if pi.HasPod(p2) {
		t.Fatalf("Should not have pod: %v", p2)
	}

	pi.AddPod(p1)

	if len(pi.DiscoveredPods) != 1 {
		t.Fatalf("Should have 1 pod: %v", pi)
	}

	if !pi.HasPod(p1) {
		t.Fatalf("Should have pod: %v", p1)
	}
	if pi.HasPod(p2) {
		t.Fatalf("Should not have pod: %v", p2)
	}

	pi.AddPod(p2)

	if len(pi.DiscoveredPods) != 2 {
		t.Fatalf("Should have 2 pod: %v", pi)
	}

	if !pi.HasPod(p1) {
		t.Fatalf("Should have pod: %v", p1)
	}
	if !pi.HasPod(p2) {
		t.Fatalf("Should have pod: %v", p2)
	}

	// closure to test forEachPod method
	count := 0
	forEach := func(p *Pod) bool {
		count++
		return true
	}
	pi.ForEachPod(forEach)
	if count != 2 {
		t.Fatalf("ForEachPod did not iterate over 2 pods")
	}

	// see that the for each function can abort the iteration
	count = 0
	pi.ForEachPod(func(p *Pod) bool {
		count++
		return false
	})
	if count != 1 {
		t.Fatalf("ForEachPod did not abort iteration")
	}

	pi.RemovePod(p1)

	if pi.HasPod(p1) {
		t.Fatalf("Should not have pod: %v", p1)
	}
	if !pi.HasPod(p2) {
		t.Fatalf("Should have pod: %v", p2)
	}

	if len(pi.DiscoveredPods) != 1 {
		t.Fatalf("Should have 1 pod: %v", pi)
	}

	pi.ForEachPod(func(p *Pod) bool {
		if p.Name != "pod2-name" {
			t.Fatalf("Wrong pod in inventory: %v", p)
		}
		return true
	})

	anno := make(map[string]string)
	anno["one"] = "1"

	p2modified := &Pod{
		Node:        p2.Node,
		Namespace:   p2.Namespace,
		Name:        p2.Name,
		UID:         p2.UID,
		Annotations: anno,
	}

	pi.ReplacePod(p2modified)

	if len(pi.DiscoveredPods) != 1 {
		t.Fatalf("Should have 1 pod: %v", pi)
	}

	if pi.HasPod(p1) {
		t.Fatalf("Should not have pod: %v", p1)
	}
	if !pi.HasPod(p2) {
		t.Fatalf("Should have pod: %v", p2)
	}

	pi.ForEachPod(func(p *Pod) bool {
		if p.Name != "pod2-name" || p.Annotations["one"] != "1" {
			t.Fatalf("Wrong pod in inventory: %v", p)
		}
		return true
	})
}
