package k8s

import (
	"fmt"
)

type PodInventory struct {
	NodeName       string
	DiscoveredPods map[string]*Pod // key is a pod identifier
}

func NewPodInventory(nodeName string) PodInventory {
	return PodInventory{
		NodeName:       nodeName,
		DiscoveredPods: make(map[string]*Pod),
	}
}

func (pi *PodInventory) AddPod(p *Pod) {
	pi.DiscoveredPods[p.GetIdentifier()] = p
}

func (pi *PodInventory) RemovePod(p *Pod) {
	delete(pi.DiscoveredPods, p.GetIdentifier())
}

func (pi *PodInventory) ReplacePod(p *Pod) {
	pi.AddPod(p)
}

func (pi *PodInventory) HasPod(p *Pod) (found bool) {
	_, found = pi.DiscoveredPods[p.GetIdentifier()]
	return
}

// ForEachPod invokes the given function once for each pod in inventory. If the function returns false, the iteration aborts.
func (pi *PodInventory) ForEachPod(f func(*Pod) bool) {
	for _, v := range pi.DiscoveredPods {
		if keepGoing := f(v); keepGoing == false {
			break
		}
	}
}

func (pi *PodInventory) String() string {
	return fmt.Sprintf("PodInventory: node-name=[%v], pods=[%v]", pi.NodeName, pi.DiscoveredPods)
}
