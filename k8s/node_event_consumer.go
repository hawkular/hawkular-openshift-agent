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
	"strings"

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

	k8sNode, err := GetLocalNode(conf, client)
	if err != nil {
		glog.Error(err)
		return
	}

	node := Node{
		Name: k8sNode.GetName(),
		UID:  string(k8sNode.GetUID()),
	}
	nec.CollectorIds = make(map[string][]string)

	nec.Discovery = NewDiscovery(conf, client, node)
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
		url, err := cmeEndpoint.GetUrl(ne.Pod.PodIP)
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
			Tenant:                   ne.Pod.Namespace.Name,
			Credentials:              cmeEndpoint.Credentials,
			Collection_Interval_Secs: cmeEndpoint.Collection_Interval_Secs,
			Metrics:                  cmeEndpoint.Metrics,
			Tags:                     cmeEndpoint.Tags,
		}

		// if a pod has labels, add them to the endpoint tags so they go on all metrics
		newEndpoint.Tags.AppendTags(ne.Pod.Labels)

		// make sure the endpoint is configured correctly
		if err := newEndpoint.ValidateEndpoint(); err != nil {
			glog.Warningf("Will not start collecting for endpoint in pod [%v] - invalid endpoint. err=%v", ne.Pod.GetIdentifier(), err)
			continue
		}

		// get an ID to be used for the collector
		id, err := getIdForEndpoint(ne.Pod, cmeEndpoint)
		if err != nil {
			glog.Warningf("Will not start collecting for endpoint in pod [%v] - cannot get ID. err=%v", ne.Pod.GetIdentifier(), err)
			continue
		}

		// Define additional envvars with pod specific data for use in replacing ${env} tokens in tags.
		additionalEnv := map[string]string{
			"POD:node_name":      ne.Pod.Node.Name,
			"POD:node_uid":       ne.Pod.Node.UID,
			"POD:namespace_name": ne.Pod.Namespace.Name,
			"POD:namespace_uid":  ne.Pod.Namespace.UID,
			"POD:name":           ne.Pod.Name,
			"POD:ip":             ne.Pod.PodIP,
			"POD:host_ip":        ne.Pod.HostIP,
			"POD:uid":            ne.Pod.UID,
			"POD:hostname":       ne.Pod.Hostname,
			"POD:subdomain":      ne.Pod.Subdomain,
			"POD:labels":         joinMap(ne.Pod.Labels),
		}

		var theCollector collector.MetricsCollector
		switch cmeEndpoint.Type {
		case collector.ENDPOINT_TYPE_PROMETHEUS:
			{
				theCollector = impl.NewPrometheusMetricsCollector(id, nec.Config.Identity, *newEndpoint, additionalEnv)
			}
		case collector.ENDPOINT_TYPE_JOLOKIA:
			{
				theCollector = impl.NewJolokiaMetricsCollector(id, nec.Config.Identity, *newEndpoint, additionalEnv)
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
	url, err := e.GetUrl(p.PodIP)
	if err != nil {
		return
	}
	id = url.String()
	return
}

func joinMap(m map[string]string) string {
	s := make([]string, len(m))
	i := 0
	for k, v := range m {
		s[i] = fmt.Sprintf("%v:%v", k, v)
		i++
	}
	return strings.Join(s, ",")
}
