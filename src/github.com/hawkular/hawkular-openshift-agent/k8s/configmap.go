package k8s

import (
	"fmt"
)

type ConfigMap struct {
	Namespace string
	Name      string
	Entry     *ConfigMapEntry
}

func NewConfigMap(namespace string, name string, cme *ConfigMapEntry) (cm *ConfigMap) {
	cm = &ConfigMap{
		Namespace: namespace,
		Name:      name,
		Entry:     cme,
	}
	return
}

func (cm *ConfigMap) String() string {
	return fmt.Sprintf("ConfigMap[%v:%v]: %v", cm.Namespace, cm.Name, cm.Entry)
}
