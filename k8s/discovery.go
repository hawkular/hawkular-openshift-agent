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
	"fmt"
	"sync"

	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/fields"
	"k8s.io/client-go/1.4/pkg/watch"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

const (
	// the volume name that is used to link to the ConfigMap that a pod wants to use for its hawkular configuration
	HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME = "hawkular-openshift-agent"

	// the name of the YAML entry in a namespace's ConfigMap that contains information on a single pod's endpoints to be monitored
	HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME = "hawkular-openshift-agent"
)

var watcherTimeout int64 = 3600 // number of seconds all watchers will timeout and refresh

// watchProcessor has one job - keep the watcher up and running. OpenShift watchers always have a timeout,
// so they will always periodically shutdown. The agent, however, always wants to watch, so it needs these
// watchers. So watchProcessor will go into an infinite loop - when a watcher shuts down,
// the watchProcessor immediately creates a new watcher and goes back to processing events. Only when
// the watchProcessor is told to abort will it exit that loop and stop watching.
type watchProcessor struct {
	createWatchFunc func() watch.Interface  // creates the watch object
	processFunc     func(w watch.Interface) // processes all watch events that come from the watch object
	abort           bool                    // true when the processing should stop
	watcher         watch.Interface         // if started, this is the watcher whose events are being processed
	lock            sync.Mutex
}

func newWatchProcessor(f1 func() watch.Interface, f2 func(w watch.Interface)) *watchProcessor {
	return &watchProcessor{
		createWatchFunc: f1,
		processFunc:     f2,
		abort:           false,
		lock:            sync.Mutex{},
	}
}

func (w *watchProcessor) start() {
	go func() {
		for !w.shouldStop() {
			if w.watcher != nil {
				w.watcher.Stop()
			}
			w.watcher = w.createWatchFunc()
			w.processFunc(w.watcher)
		}
	}()
}

func (w *watchProcessor) stop() {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.abort = true
	if w.watcher != nil {
		w.watcher.Stop()
		w.watcher = nil
	}
}

func (w *watchProcessor) shouldStop() bool {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.abort
}

type Discovery struct {
	AgentConfig       *config.Config
	Client            *core.CoreClient
	PodWatcher        *watchProcessor
	ConfigMapWatchers map[string]*watchProcessor
	NodeEventChannel  chan *NodeEvent
	Inventory         *Inventory
}

func NewDiscovery(conf *config.Config, client *core.CoreClient, node Node) *Discovery {
	d := Discovery{
		AgentConfig:       conf,
		Client:            client,
		ConfigMapWatchers: make(map[string]*watchProcessor),
		Inventory:         NewInventory(node),
	}
	return &d
}

func (d *Discovery) start() chan *NodeEvent {
	log.Debugf("Starting Kubernetes Discovery")
	d.NodeEventChannel = make(chan *NodeEvent)
	d.watchPods()
	return d.NodeEventChannel
}

func (d *Discovery) stop() {
	log.Debugf("Stopping Kubernetes Discovery")
	d.unwatchPods()
	for k, _ := range d.ConfigMapWatchers {
		d.unwatchConfigMap(k)
	}
	close(d.NodeEventChannel)
}

func (d *Discovery) sendNodeEventDueToChangedPod(pod *Pod, trigger Trigger) {
	var cme *ConfigMapEntry

	configMapName, hasVol := pod.ConfigMapVolumes[HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME]
	if hasVol == true {
		cm, ok := d.Inventory.ConfigMaps.GetEntry(pod.Namespace.Name, configMapName)
		if ok == true {
			cme = cm.Entry
			log.Debugf("Changed pod [%v] with volume [%v=%v] has a config map",
				pod.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME, configMapName)
		} else {
			log.Debugf("Changed pod [%v] with volume [%v=%v] does not have a config map",
				pod.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME, configMapName)
		}
	} else {
		log.Debugf("Changed pod [%v] does not have volume [%v]",
			pod.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME)
	}

	ne := NodeEvent{
		Trigger:        trigger,
		Pod:            pod,
		ConfigMapEntry: cme,
	}

	d.NodeEventChannel <- &ne
}

func (d *Discovery) sendNodeEventDueToChangedConfigMap(namespace string, name string, cm *ConfigMap, trigger Trigger) {
	d.Inventory.Pods.ForEachPod(func(p *Pod) bool {
		// if the pod isn't in the namespace whose config map changed, then skip it and go on to the next
		if p.Namespace.Name != namespace {
			return true
		}

		var cme *ConfigMapEntry

		if cm != nil {
			// the config map changed
			configMapName, hasVol := p.ConfigMapVolumes[HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME]
			if hasVol == true && configMapName == cm.Name {
				cme = cm.Entry
				log.Debugf("Changed configmap [%v] for namespace [%v] affects pod [%v] with volume: [%v=%v]",
					cm.Name, namespace, p.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME, configMapName)
			} else {
				log.Debugf("Changed configmap [%v] for namespace [%v] does not affect pod [%v]",
					cm.Name, namespace, p.GetIdentifier())
				return true
			}
		} else {
			// the config map was deleted
			configMapName, hasVol := p.ConfigMapVolumes[HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME]
			if hasVol == true && configMapName == name {
				log.Debugf("Deleted configmap [%v] for namespace [%v] affects pod [%v] with volume: [%v=%v]",
					name, namespace, p.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_VOLUME_NAME, configMapName)
			} else {
				log.Debugf("Deleted configmap [%v] for namespace [%v] does not affect pod [%v]",
					name, namespace, p.GetIdentifier())
				return true
			}
		}

		ne := NodeEvent{
			Trigger:        trigger,
			Pod:            p,
			ConfigMapEntry: cme,
		}

		d.NodeEventChannel <- &ne
		return true
	})
}

func (d *Discovery) watchPods() {
	createFunc := func() watch.Interface {
		// we only want to listen to pods on our own node
		fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName==%v", d.Inventory.Node.Name))
		if err != nil {
			log.Error(err)
			return nil
		}

		listOptions := api.ListOptions{
			Watch:          true,
			FieldSelector:  fieldSelector,
			TimeoutSeconds: &watcherTimeout,
		}

		watcher, err := d.Client.Pods(v1.NamespaceAll).Watch(listOptions)
		if err != nil {
			log.Error(err)
			return nil
		}

		return watcher
	}

	processFunc := func(watcher watch.Interface) {
		for event := range watcher.ResultChan() {
			podFromEvent := event.Object.(*v1.Pod)
			namespaceFromEvent, err := d.Client.Namespaces().Get(podFromEvent.GetNamespace())
			var namespaceUID string
			if err != nil {
				log.Warning("Failed to obtain UID of namespace [%v]. err=%v", podFromEvent.GetNamespace(), err)
			} else {
				namespaceUID = string(namespaceFromEvent.GetUID())
			}

			cmVols := make(map[string]string, 0)
			for _, vol := range podFromEvent.Spec.Volumes {
				if vol.ConfigMap != nil {
					cmVols[vol.Name] = vol.ConfigMap.Name
				}
			}

			pod := &Pod{
				Node: d.Inventory.Node,
				Namespace: Namespace{
					Name: podFromEvent.GetNamespace(),
					UID:  namespaceUID,
				},
				Name:             podFromEvent.GetName(),
				UID:              string(podFromEvent.GetUID()),
				PodIP:            podFromEvent.Status.PodIP,
				HostIP:           podFromEvent.Status.HostIP,
				Hostname:         podFromEvent.Spec.Hostname,
				Subdomain:        podFromEvent.Spec.Subdomain,
				Labels:           podFromEvent.GetLabels(),
				Annotations:      podFromEvent.GetAnnotations(),
				ConfigMapVolumes: cmVols,
			}

			switch event.Type {
			case watch.Added:
				{
					log.Debugf("Detected a new pod that was added: %v", pod.GetIdentifier())
					d.Inventory.Pods.AddPod(pod)

					// tell the channel about the change
					d.sendNodeEventDueToChangedPod(pod, POD_ADDED)

					d.watchConfigMap(pod.Namespace.Name)
				}
			case watch.Deleted:
				{
					log.Debugf("Detected an old pod that was deleted: %v", pod.GetIdentifier())
					d.Inventory.Pods.RemovePod(pod)

					// tell the channel about the change
					d.sendNodeEventDueToChangedPod(pod, POD_DELETED)

					// if there are no more pods in the namespace, no more need to keep watching for configmaps
					stopWatcher := true
					d.Inventory.Pods.ForEachPod(func(p *Pod) bool {
						if pod.Namespace.Name == p.Namespace.Name {
							stopWatcher = false
						}
						return stopWatcher // if we know already we should not stop the watcher, we can abort the iteration now, too
					})
					if stopWatcher == true {
						log.Debugf("No more pods in namespace [%v], unwatching configmaps", pod.Namespace.Name)
						d.unwatchConfigMap(pod.Namespace.Name)
					}
				}
			case watch.Modified:
				{
					log.Debugf("Detected a modified pod: %v", pod.GetIdentifier())
					d.Inventory.Pods.ReplacePod(pod)

					// tell the channel about the change
					d.sendNodeEventDueToChangedPod(pod, POD_MODIFIED)

					d.watchConfigMap(pod.Namespace.Name) // in case for some reason we aren't watching it, watch it now
				}
			default:
				{
					log.Debugf("Ignoring event [%v] on pod [%v]", event.Type, pod.GetIdentifier())
				}
			}

			log.Tracef("PodInventory has been updated: %v", d.Inventory.Pods)
		}
		log.Debugf("Watcher has disconnected (was watching for pod changes in node [%v])", d.Inventory.Node.Name)
	}

	d.PodWatcher = newWatchProcessor(createFunc, processFunc)
	d.PodWatcher.start()
}

func (d *Discovery) unwatchPods() {
	if d.PodWatcher != nil {
		log.Infof("Stopping the pod watcher for node [%v]", d.Inventory.Node.Name)
		d.PodWatcher.stop()
		d.PodWatcher = nil
	}
}

func (d *Discovery) watchConfigMap(namespace string) {
	if _, ok := d.ConfigMapWatchers[namespace]; ok == true {
		return // we are already watching this namespace's configmap
	}

	createFunc := func() watch.Interface {
		// pods are free to use any name for their ConfigMap - so we need to get all of them
		fieldSelector := fields.Everything()

		listOptions := api.ListOptions{
			Watch:          true,
			FieldSelector:  fieldSelector,
			TimeoutSeconds: &watcherTimeout,
		}

		watcher, err := d.Client.ConfigMaps(namespace).Watch(listOptions)
		if err != nil {
			log.Error(err)
			return nil
		}

		return watcher
	}

	processFunc := func(watcher watch.Interface) {
		for event := range watcher.ResultChan() {
			configMapFromEvent := event.Object.(*v1.ConfigMap)
			configMapName := configMapFromEvent.Name

			switch event.Type {
			case watch.Added:
				{
					var cm *ConfigMap

					yaml, ok := configMapFromEvent.Data[HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME]
					if !ok {
						log.Debugf("Detected a new configmap [%v] for namespace [%v] but has no entry named [%v]. Ignoring.", configMapName, namespace, HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME)
						continue
					}
					log.Debugf("Detected a new configmap [%v] for namespace [%v]", configMapName, namespace)
					cme, err := UnmarshalConfigMapEntry(yaml)
					if err == nil {
						cm = NewConfigMap(namespace, configMapName, cme)
						d.Inventory.ConfigMaps.AddEntry(cm)
					} else {
						log.Warningf("Cannot use new configmap [%v] for namespace [%v]. err=%v", configMapName, namespace, err)
						continue
					}

					log.Tracef("Added configmap [%v] for namespace [%v]=%v", configMapName, namespace, cm)

					// tell the channel about the change
					d.sendNodeEventDueToChangedConfigMap(namespace, configMapName, cm, CONFIG_MAP_ADDED)
				}
			case watch.Deleted:
				{
					_, ok := configMapFromEvent.Data[HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME]
					if !ok {
						log.Debugf("Detected a deleted configmap [%v] for namespace [%v] but has no entry named [%v]. Ignoring.", configMapName, namespace, HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME)
						continue
					}

					log.Debugf("Detected an old configmap [%v] that was deleted from namespace [%v]", configMapName, namespace)
					d.Inventory.ConfigMaps.RemoveEntry(namespace, configMapName)

					// tell the channel about the change
					d.sendNodeEventDueToChangedConfigMap(namespace, configMapName, nil, CONFIG_MAP_DELETED)
				}
			case watch.Modified:
				{
					var cm *ConfigMap

					yaml, ok := configMapFromEvent.Data[HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME]
					if !ok {
						log.Debugf("Detected a modified configmap [%v] for namespace [%v] but has no entry named [%v]. Ignoring.", configMapName, namespace, HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME)
						continue
					}
					log.Debugf("Detected a modified configmap [%v] for namespace [%v]", configMapName, namespace)
					cme, err := UnmarshalConfigMapEntry(yaml)
					if err == nil {
						d.Inventory.ConfigMaps.RemoveEntry(namespace, configMapName)
						cm = NewConfigMap(namespace, configMapName, cme)
						d.Inventory.ConfigMaps.AddEntry(cm)
					} else {
						log.Warningf("Cannot use modified configmap [%v] for namespace [%v]. err=%v", configMapName, namespace, err)
						continue
					}

					log.Tracef("Modified configmap [%v] for namespace [%v]=%v", configMapName, namespace, cm)

					// tell the channel about the change
					d.sendNodeEventDueToChangedConfigMap(namespace, configMapName, cm, CONFIG_MAP_MODIFIED)
				}
			default:
				{
					log.Debugf("Ignoring event [%v] on configmap [%v] in namespace [%v]", event.Type, configMapName, namespace)
				}
			}
		}
		log.Debugf("Watcher has disconnected (was watching for configmap changes in namespace [%v])", namespace)
	}

	d.ConfigMapWatchers[namespace] = newWatchProcessor(createFunc, processFunc)
	d.ConfigMapWatchers[namespace].start()
}

func (d *Discovery) unwatchConfigMap(namespace string) {
	doomedWatcher, ok := d.ConfigMapWatchers[namespace]
	if ok == true {
		log.Infof("Stopping the configmap watcher for namespace [%v]", namespace)
		doomedWatcher.stop()
		delete(d.ConfigMapWatchers, namespace)
	}

	// remove the config maps for the namespace if we cached them before
	d.Inventory.ConfigMaps.ClearNamespace(namespace)
}
