package k8s

import (
	"testing"
)

func Test(t *testing.T) {
	p1 := &Pod{
		NodeName:  "node-name",
		Name:      "pod1-name",
		Namespace: "pod1-namespace",
		Uid:       "pod1-uuid",
	}

	p2 := &Pod{
		NodeName:  "node-name",
		Name:      "pod2-name",
		Namespace: "pod2-namespace",
		Uid:       "pod2-uuid",
	}

	pi := NewPodInventory("pod_inventory_test")

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
		NodeName:    p2.NodeName,
		Name:        p2.Name,
		Namespace:   p2.Namespace,
		Uid:         p2.Uid,
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
