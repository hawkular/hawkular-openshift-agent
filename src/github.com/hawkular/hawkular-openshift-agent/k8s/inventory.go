package k8s

import ()

type Inventory struct {
	NodeName   string
	Pods       PodInventory
	ConfigMaps *ConfigMaps
}

func NewInventory(nodeName string) *Inventory {
	i := Inventory{
		NodeName:   nodeName,
		Pods:       NewPodInventory(nodeName),
		ConfigMaps: NewConfigMaps(),
	}

	return &i
}
