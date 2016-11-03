package k8s

import ()

type ConfigMaps struct {
	Entries map[string]map[string]*ConfigMap // first key is namespace, inner map keyed on configmap name
}

func NewConfigMaps() (cms *ConfigMaps) {
	cms = new(ConfigMaps)
	cms.Entries = make(map[string]map[string]*ConfigMap)
	return
}

func (cms *ConfigMaps) AddEntry(cm *ConfigMap) {
	namespaceMaps, ok := cms.Entries[cm.Namespace]
	if !ok {
		cms.Entries[cm.Namespace] = make(map[string]*ConfigMap)
		namespaceMaps = cms.Entries[cm.Namespace]
	}
	namespaceMaps[cm.Name] = cm
}

func (cms *ConfigMaps) GetEntry(namespace string, name string) (cm *ConfigMap, ok bool) {
	namespaceMaps, ok := cms.Entries[namespace]
	if !ok {
		return nil, false
	}
	cm, ok = namespaceMaps[name]
	return
}

func (cms *ConfigMaps) RemoveEntry(namespace string, name string) {
	namespaceMaps, ok := cms.Entries[namespace]
	if ok {
		delete(namespaceMaps, name)
	}
}

func (cms *ConfigMaps) ClearNamespace(namespace string) {
	delete(cms.Entries, namespace)
}

func (cms *ConfigMaps) ClearAll() {
	cms.Entries = make(map[string]map[string]*ConfigMap)
}
