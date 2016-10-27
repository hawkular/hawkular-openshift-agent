package k8s

import (
	"github.com/golang/glog"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/impl"
	"github.com/hawkular/hawkular-openshift-agent/collector/manager"
	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

// NodeEventConsumer will process our node events that are emitted from our
// Kubernetes watcher client. Node events tell us when pods are added/deleted/modified
// and when namespace configmaps are added/deleted/modified. Based on these events
// the NodeEventConsumer will start and stop collecting metrics for the affected
// monitored endpoints.
type NodeEventConsumer struct {
	Config                  *config.Config
	Discovery               *Discovery
	CollectorIds            map[string][]string // key: pod identifier; value: collector IDs
	MetricsCollectorManager *manager.MetricsCollectorManager
}

func NewNodeEventConsumer(conf *config.Config, mcm *manager.MetricsCollectorManager) *NodeEventConsumer {
	m := NodeEventConsumer{
		Config:                  conf,
		MetricsCollectorManager: mcm,
	}
	return &m
}

// Start will begin watching/discovery of pods and configmaps and will
// process node events as they are received.
func (nec *NodeEventConsumer) Start() {
	conf := nec.Config

	if conf.Kubernetes.Pod_Namespace == "" {
		log.Debug("Not configured to monitor within a Kubernetes environment")
		return
	}

	client, err := GetKubernetesClient(conf)
	if err != nil {
		glog.Errorf("Error trying to get the Kubernetes Client: err=%v", err)
		return
	}

	nodeName, err := GetNodeName(conf, client)
	if err != nil {
		glog.Error(err)
		return
	}

	nec.CollectorIds = make(map[string][]string)

	nec.Discovery = NewDiscovery(conf, client, nodeName)
	nec.Discovery.start()
	go nec.consumeNodeEvents()
}

// Stop will halt all watching/discovery of changes to pods/configmaps,
// stop all collections of all monitored endpoints, and will no longer
// process node events.
func (nec *NodeEventConsumer) Stop() {
	if nec.Discovery != nil {
		nec.Discovery.stop()
	}

	// stop all our collectors
	for _, ids := range nec.CollectorIds {
		for _, id := range ids {
			nec.MetricsCollectorManager.StopCollecting(id)
		}
	}
}

// consumeNodeEvents listens to the node event channel and will process
// all node events as they come in. Depending on the node event, this
// will start or stop collecting metrics for one or more monitored endpoints.
func (nec *NodeEventConsumer) consumeNodeEvents() {
	for ne := range nec.Discovery.NodeEventChannel {
		switch ne.Trigger {
		case POD_ADDED:
			{
				// If we do not have the new pod's config yet, there's nothing to do.
				// (we will wait for the config to come in later)
				if ne.ConfigMapEntry != nil {
					nec.startCollecting(ne)
				}
			}
		case POD_MODIFIED:
			{
				// a modified pod might have changed the configmap it is using which means it might
				// have completely changed the endpoints its collecting. So we need to stop collecting
				// everything so we start collecting only those endpoints in the potentially new and different configmap.
				nec.stopCollecting(ne)

				// There is no config, it must mean the config was deleted so there is nothing to collect now
				if ne.ConfigMapEntry != nil {
					nec.startCollecting(ne)
				}
			}
		case POD_DELETED:
			{
				nec.stopCollecting(ne)
			}
		case CONFIG_MAP_ADDED:
			{
				if ne.ConfigMapEntry != nil {
					nec.startCollecting(ne)
				}
			}
		case CONFIG_MAP_MODIFIED:
			{
				// A modified configmap might have changed the endpoints completely. So we need to stop collecting
				// everything so we start collecting only those endpoints in the modified configmap.
				nec.stopCollecting(ne)

				if ne.ConfigMapEntry != nil {
					nec.startCollecting(ne)
				}
			}
		case CONFIG_MAP_DELETED:
			{
				nec.stopCollecting(ne)
			}
		default:
			{
				glog.Warningf("Ignoring unknown trigger [%v]", ne.Trigger)
			}
		}
	}
}

// startCollecting will be called when one or more monitored endpoints for a pod
// need to start getting their metrics collected.
func (nec *NodeEventConsumer) startCollecting(ne *NodeEvent) {
	for _, cmeEndpoint := range ne.ConfigMapEntry.Endpoints {
		url, err := cmeEndpoint.GetUrl(ne.Pod.IPAddress)
		if err != nil {
			glog.Warningf("Will not start collecting for endpoint in pod [%v] - cannot build URL. err=%v", ne.Pod.GetIdentifier(), err)
			continue
		}

		// We need to convert the k8s endpoint to the generic endpoint struct.
		// Note that the tenant for all metrics collected from this endpoint
		// must be the same as the namespace of the pod where the endpoint is located
		newEndpoint := &collector.Endpoint{
			Url:                      url.String(),
			Type:                     cmeEndpoint.Type,
			Tenant:                   ne.Namespace,
			Credentials:              cmeEndpoint.Credentials,
			Collection_Interval_Secs: cmeEndpoint.Collection_Interval_Secs,
			Metrics:                  make([]collector.MonitoredMetric, len(cmeEndpoint.Metrics)),
		}

		for _, k8sMetric := range cmeEndpoint.Metrics {
			mm := collector.MonitoredMetric{
				Type: k8sMetric.Type,
				Name: k8sMetric.Name,
			}
			newEndpoint.Metrics = append(newEndpoint.Metrics, mm)
		}

		id, err := getIdForEndpoint(ne.Pod, cmeEndpoint)
		if err != nil {
			glog.Warningf("Will not start collecting for endpoint in pod [%v] - cannot get ID. err=%v", ne.Pod.GetIdentifier(), err)
			continue
		}

		var theCollector collector.MetricsCollector
		switch cmeEndpoint.Type {
		case collector.ENDPOINT_TYPE_PROMETHEUS:
			{
				theCollector = impl.NewPrometheusMetricsCollector(id, nec.Config.Identity, *newEndpoint)
			}
		case collector.ENDPOINT_TYPE_JOLOKIA:
			{
				theCollector = impl.NewJolokiaMetricsCollector(id, nec.Config.Identity, *newEndpoint)
			}
		default:
			{
				glog.Warningf("Will not start collecting for endpoint in pod [%v] - unknown endpoint type [%v]",
					ne.Pod.GetIdentifier(), cmeEndpoint.Type)
				return
			}
		}

		nec.MetricsCollectorManager.StartCollecting(theCollector)

		// keep track of each pod's collector IDs in case we need to stop them later on
		ids, ok := nec.CollectorIds[ne.Pod.GetIdentifier()]
		if ok {
			nec.CollectorIds[ne.Pod.GetIdentifier()] = append(ids, id)
		} else {
			nec.CollectorIds[ne.Pod.GetIdentifier()] = []string{id}
		}

	}
}

// stopCollecting will stop collecting from all endpoints for the pod in the given node event
func (nec *NodeEventConsumer) stopCollecting(ne *NodeEvent) {
	ids, ok := nec.CollectorIds[ne.Pod.GetIdentifier()]
	if ok {
		for _, id := range ids {
			nec.MetricsCollectorManager.StopCollecting(id)
		}
	}
}

func getIdForEndpoint(p *Pod, e K8SEndpoint) (id string, err error) {
	url, err := e.GetUrl(p.IPAddress)
	if err != nil {
		return
	}
	id = url.String()
	return
}
