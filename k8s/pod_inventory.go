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
	"fmt"
)

type PodInventory struct {
	Node           Node
	DiscoveredPods map[string]*Pod // key is a pod identifier
}

func NewPodInventory(node Node) PodInventory {
	return PodInventory{
		Node:           node,
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
	return fmt.Sprintf("PodInventory: node-name=[%v], pods=[%v]", pi.Node.Name, pi.DiscoveredPods)
}
