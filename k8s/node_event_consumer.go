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
	"os"
	"strings"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/manager"
	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/util/expand"
)

// NodeEventConsumer will process our node events that are emitted from our
// Kubernetes watcher client. Node events tell us when pods are added/deleted/modified
// and when namespace configmaps are added/deleted/modified. Based on these events
// the NodeEventConsumer will start and stop collecting metrics for the affected
// monitored endpoints.
type NodeEventConsumer struct {
	Config                  *config.Config
	Discovery               *Discovery
	CollectorIds            map[string][]collector.CollectorID // key: pod identifier
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
		log.Errorf("Error trying to get the Kubernetes Client: err=%v", err)
		return
	}

	k8sNode, err := GetLocalNode(conf, client)
	if err != nil {
		log.Error(err)
		return
	}

	node := Node{
		Name: k8sNode.GetName(),
		UID:  string(k8sNode.GetUID()),
	}

	nec.CollectorIds = make(map[string][]collector.CollectorID)

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

	// stop all our collectors for all pods
	for podId, ids := range nec.CollectorIds {
		for _, id := range ids {
			nec.MetricsCollectorManager.StopCollecting(id)
		}
		delete(nec.CollectorIds, podId)
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
				log.Warningf("Ignoring unknown trigger [%v]", ne.Trigger)
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
			log.Warningf("Will not start collecting for endpoint in pod [%v] - cannot build URL. err=%v", ne.Pod.GetIdentifier(), err)
			continue
		}

		// get an ID to be used for the collector
		endpointId, err := getIdForEndpoint(ne.Pod, cmeEndpoint)
		if err != nil {
			log.Warningf("Will not start collecting for endpoint in pod [%v] - cannot get ID. err=%v", ne.Pod.GetIdentifier(), err)
			continue
		}
		id := collector.CollectorID{
			PodID:      ne.Pod.GetIdentifier(),
			EndpointID: endpointId,
		}

		// keep track of each pod's collector IDs in case we need to stop them later on
		ids, ok := nec.CollectorIds[ne.Pod.GetIdentifier()]
		if ok {
			alreadyThere := false
			for _, i := range ids {
				if i == id {
					alreadyThere = true
					break
				}
			}
			if !alreadyThere {
				nec.CollectorIds[ne.Pod.GetIdentifier()] = append(ids, id)
			}
		} else {
			nec.CollectorIds[ne.Pod.GetIdentifier()] = []collector.CollectorID{id}
		}

		if cmeEndpoint.IsEnabled() == false {
			m := fmt.Sprintf("Will not start collecting for endpoint [%v] in pod [%v] - it has been disabled.", url, ne.Pod.GetIdentifier())
			log.Info(m)
			nec.MetricsCollectorManager.NotCollecting(id, m)
			continue
		}

		// Define additional envvars with pod specific data for use in replacing ${env} tokens.
		// These tokens are used in tags and in the Tenant field.
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

		// support ${POD:label[<label-name>]}
		for n, v := range ne.Pod.Labels {
			additionalEnv[fmt.Sprintf("POD:label[%v]", n)] = v
		}

		endpointTenant := nec.Config.Hawkular_Server.Tenant
		if nec.Config.Kubernetes.Tenant != "" {
			mappingFunc := expand.MappingFunc(expand.MappingFuncConfig{
				Env:      additionalEnv,
				UseOSEnv: true,
			})
			endpointTenant = os.Expand(nec.Config.Kubernetes.Tenant, mappingFunc)
		}

		endpointCredentials, err := nec.determineCredentials(ne.Pod, cmeEndpoint.Credentials)
		if err != nil {
			m := fmt.Sprintf("Will not start collecting for endpoint in pod [%v] - cannot determine credentials. err=%v", ne.Pod.GetIdentifier(), err)
			log.Warning(m)
			nec.MetricsCollectorManager.NotCollecting(id, m)
			continue
		}

		// We need to convert the k8s endpoint to the generic endpoint struct.
		newEndpoint := &collector.Endpoint{
			URL:                 url.String(),
			Type:                cmeEndpoint.Type,
			Enabled:             cmeEndpoint.Enabled,
			Tenant:              endpointTenant,
			TLS:                 cmeEndpoint.TLS,
			Credentials:         endpointCredentials,
			Collection_Interval: cmeEndpoint.Collection_Interval,
			Metrics:             cmeEndpoint.Metrics,
			Tags:                cmeEndpoint.Tags,
		}

		// make sure the endpoint is configured correctly
		if err := newEndpoint.ValidateEndpoint(); err != nil {
			m := fmt.Sprintf("Will not start collecting for endpoint in pod [%v] - invalid endpoint. err=%v", ne.Pod.GetIdentifier(), err)
			log.Warning(m)
			nec.MetricsCollectorManager.NotCollecting(id, m)
			continue
		}

		if c, err := manager.CreateMetricsCollector(id, nec.Config.Identity, *newEndpoint, additionalEnv); err != nil {
			m := fmt.Sprintf("Will not start collecting for endpoint in pod [%v] - cannot create collector. err=%v", ne.Pod.GetIdentifier(), err)
			log.Warning(m)
			nec.MetricsCollectorManager.NotCollecting(id, m)
			continue
		} else {
			nec.MetricsCollectorManager.StartCollecting(c)
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
		delete(nec.CollectorIds, ne.Pod.GetIdentifier())
	}
}

// determineCredentials will build a Credentials object that contains the credentials needed to
// communicate with the endpoint.
func (nec *NodeEventConsumer) determineCredentials(p *Pod, cmeCredentials security.Credentials) (creds security.Credentials, err error) {
	// function that will extract a credential string based on its value.
	// If the string is prefixed with "secret:" it is assumed to be a key/value from a k8s secret.
	// If the string is not prefixed, it is used as-is.
	f := func(v string) string {
		if strings.HasPrefix(v, "secret:") {
			v = strings.TrimLeft(v, "secret:")
			pair := strings.SplitN(v, "/", 2)
			if len(pair) != 2 {
				err = fmt.Errorf("Secret credentials are invalid for pod [%v]", p.GetIdentifier())
				return ""
			}
			secret, e := nec.Discovery.Client.Secrets(p.Namespace.Name).Get(pair[0])
			if e != nil {
				err = fmt.Errorf("There is no secret named [%v] - credentials are invalid for pod [%v]. err=%v",
					pair[0], p.GetIdentifier(), e)
				return ""
			}
			secretValue, ok := secret.Data[pair[1]]
			if !ok {
				err = fmt.Errorf("There is no key named [%v] in secret named [%v] - credentials are invalid for pod [%v]",
					pair[1], pair[0], p.GetIdentifier())
				return ""
			}
			log.Debugf("Credentials obtained from secret [%v/%v] for pod [%v]", pair[0], pair[1], p.GetIdentifier())
			return string(secretValue)
		} else {
			return v
		}
	}

	creds = security.Credentials{
		Username: f(cmeCredentials.Username),
		Password: f(cmeCredentials.Password),
		Token:    f(cmeCredentials.Token),
	}

	return
}

func getIdForEndpoint(p *Pod, e K8SEndpoint) (id string, err error) {
	url, err := e.GetUrl(p.PodIP)
	if err != nil {
		return
	}
	id = fmt.Sprintf("%v|%v|%v|%v", p.Namespace.Name, p.Name, e.Type, url.String())
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
