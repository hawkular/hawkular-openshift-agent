package k8s

import ()

type Inventory struct {
	Node       Node
	Pods       PodInventory
	ConfigMaps *ConfigMaps
}

func NewInventory(node Node) *Inventory {
	i := Inventory{
		Node:       node,
		Pods:       NewPodInventory(node),
		ConfigMaps: NewConfigMaps(),
	}

	return &i
}
