package k8s

import (
	"github.com/golang/glog"
	core "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/fields"
	"k8s.io/client-go/1.4/pkg/watch"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

const (
	// the annotation name whose value is the name of the ConfigMap that a pod wants to use for its hawkular configuration
	HAWKULAR_OPENSHIFT_AGENT_ANNOTATION_NAME = "hawkular-openshift-agent"

	// the name of the YAML entry in a namespace's ConfigMap that contains information on a single pod's endpoints to be monitored
	HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME = "hawkular-openshift-agent"
)

type Discovery struct {
	AgentConfig       *config.Config
	Client            *core.CoreClient
	PodWatcher        watch.Interface
	ConfigMapWatchers map[string]watch.Interface
	NodeEventChannel  chan *NodeEvent
	Inventory         *Inventory
}

func NewDiscovery(conf *config.Config, client *core.CoreClient, node Node) *Discovery {
	d := Discovery{
		AgentConfig:       conf,
		Client:            client,
		ConfigMapWatchers: make(map[string]watch.Interface),
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

	configMapName, hasAnno := pod.Annotations[HAWKULAR_OPENSHIFT_AGENT_ANNOTATION_NAME]
	if hasAnno == true {
		cm, ok := d.Inventory.ConfigMaps.GetEntry(pod.Namespace.Name, configMapName)
		if ok == true {
			cme = cm.Entry
			log.Debugf("Changed pod [%v] with annotation [%v=%v] has a config map",
				pod.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_ANNOTATION_NAME, configMapName)
		} else {
			log.Debugf("Changed pod [%v] with annotation [%v=%v] does not have a config map",
				pod.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_ANNOTATION_NAME, configMapName)
		}
	} else {
		log.Debugf("Changed pod [%v] does not have annotation [%v]",
			pod.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_ANNOTATION_NAME)
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
			configMapName, hasAnno := p.Annotations[HAWKULAR_OPENSHIFT_AGENT_ANNOTATION_NAME]
			if hasAnno == true && configMapName == cm.Name {
				cme = cm.Entry
				log.Debugf("Configmap [%v] for namespace [%v] affects pod [%v] with annotation: [%v=%v]",
					cm.Name, namespace, p.GetIdentifier(), HAWKULAR_OPENSHIFT_AGENT_ANNOTATION_NAME, configMapName)
			} else {
				log.Debugf("Configmap [%v] for namespace [%v] does not affect pod [%v]",
					cm.Name, namespace, p.GetIdentifier())
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
	// we only want to listen to pods on our own node
	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", d.Inventory.Node.Name)

	listOptions := api.ListOptions{
		Watch:         true,
		FieldSelector: fieldSelector,
	}

	watcher, err := d.Client.Pods(v1.NamespaceAll).Watch(listOptions)
	if err != nil {
		glog.Fatal(err)
	}

	d.PodWatcher = watcher

	go func() {
		for event := range watcher.ResultChan() {
			podFromEvent := event.Object.(*v1.Pod)
			namespaceFromEvent, err := d.Client.Namespaces().Get(podFromEvent.GetNamespace())
			var namespaceUID string
			if err != nil {
				glog.Warning("Failed to obtain UID of namespace [%v]. err=%v", podFromEvent.GetNamespace(), err)
			} else {
				namespaceUID = string(namespaceFromEvent.GetUID())
			}

			pod := &Pod{
				Node: d.Inventory.Node,
				Namespace: Namespace{
					Name: podFromEvent.GetNamespace(),
					UID:  namespaceUID,
				},
				Name:        podFromEvent.GetName(),
				UID:         string(podFromEvent.GetUID()),
				PodIP:       podFromEvent.Status.PodIP,
				HostIP:      podFromEvent.Status.HostIP,
				Labels:      podFromEvent.GetLabels(),
				Annotations: podFromEvent.GetAnnotations(),
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
	}()
}

func (d *Discovery) unwatchPods() {
	if d.PodWatcher != nil {
		d.PodWatcher.Stop()
		d.PodWatcher = nil
	}
}

func (d *Discovery) watchConfigMap(namespace string) {
	if _, ok := d.ConfigMapWatchers[namespace]; ok == true {
		return // we are already watching this namespace's configmap
	}

	// pods are free to use any name for their ConfigMap - so we need to get all of them
	fieldSelector := fields.Everything()

	listOptions := api.ListOptions{
		Watch:         true,
		FieldSelector: fieldSelector,
	}

	watcher, err := d.Client.ConfigMaps(namespace).Watch(listOptions)
	if err != nil {
		glog.Fatal(err)
	}

	d.ConfigMapWatchers[namespace] = watcher

	go func() {
		for event := range watcher.ResultChan() {
			configMapFromEvent := event.Object.(*v1.ConfigMap)
			configMapName := configMapFromEvent.Name

			switch event.Type {
			case watch.Added:
				{
					if len(configMapFromEvent.Data) != 1 {
						log.Debugf("Detected a new configmap [%v] for namespace [%v] but it doesn't have one and only one entry. Ignoring.", configMapName, namespace)
						continue
					}

					var cm *ConfigMap

					yaml, ok := configMapFromEvent.Data[HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME]
					if !ok {
						log.Debugf("Detected a new configmap [%v] for namespace [%v] but its entry is not named [%v]. Ignoring.", configMapName, namespace, HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME)
						continue
					}
					log.Debugf("Detected a new configmap [%v] for namespace [%v]", configMapName, namespace)
					cme, err := UnmarshalConfigMapEntry(yaml)
					if err == nil {
						cm = NewConfigMap(namespace, configMapName, cme)
						d.Inventory.ConfigMaps.AddEntry(cm)
					} else {
						glog.Warningf("Cannot use new configmap [%v] for namespace [%v]. err=%v", configMapName, namespace, err)
						continue
					}

					log.Tracef("Added configmap [%v] for namespace [%v]=%v", configMapName, namespace, cm)

					// tell the channel about the change
					d.sendNodeEventDueToChangedConfigMap(namespace, configMapName, cm, CONFIG_MAP_ADDED)
				}
			case watch.Deleted:
				{
					if len(configMapFromEvent.Data) != 1 {
						log.Debugf("Detected a deleted configmap [%v] for namespace [%v] but it doesn't have one and only one entry. Ignoring.", configMapName, namespace)
						continue
					}

					_, ok := configMapFromEvent.Data[HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME]
					if !ok {
						log.Debugf("Detected a deleted configmap [%v] for namespace [%v] but its entry is not named [%v]. Ignoring.", configMapName, namespace, HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME)
						continue
					}

					log.Debugf("Detected an old configmap [%v] that was deleted from namespace [%v]", configMapName, namespace)
					d.Inventory.ConfigMaps.RemoveEntry(namespace, configMapName)

					// tell the channel about the change
					d.sendNodeEventDueToChangedConfigMap(namespace, configMapName, nil, CONFIG_MAP_DELETED)
				}
			case watch.Modified:
				{
					if len(configMapFromEvent.Data) != 1 {
						log.Debugf("Detected a modified configmap [%v] for namespace [%v] but it doesn't have one and only one entry. Ignoring.", configMapName, namespace)
						continue
					}

					var cm *ConfigMap

					yaml, ok := configMapFromEvent.Data[HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME]
					if !ok {
						log.Debugf("Detected a modified configmap [%v] for namespace [%v] but its entry is not named [%v]. Ignoring.", configMapName, namespace, HAWKULAR_OPENSHIFT_AGENT_CONFIG_MAP_ENTRY_NAME)
						continue
					}
					log.Debugf("Detected a modified configmap [%v] for namespace [%v]", configMapName, namespace)
					cme, err := UnmarshalConfigMapEntry(yaml)
					if err == nil {
						d.Inventory.ConfigMaps.RemoveEntry(namespace, configMapName)
						cm = NewConfigMap(namespace, configMapName, cme)
						d.Inventory.ConfigMaps.AddEntry(cm)
					} else {
						glog.Warningf("Cannot use modified configmap [%v] for namespace [%v]. err=%v", configMapName, namespace, err)
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
	}()
}

func (d *Discovery) unwatchConfigMap(namespace string) {
	doomedWatcher, ok := d.ConfigMapWatchers[namespace]
	if ok == true {
		doomedWatcher.Stop()
		delete(d.ConfigMapWatchers, namespace)
	}

	// remove the config maps for the namespace if we cached them before
	d.Inventory.ConfigMaps.ClearNamespace(namespace)
}
