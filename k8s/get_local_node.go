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

	k8score "k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	k8sapi "k8s.io/client-go/1.4/pkg/api/v1"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

// GetLocalNode returns information on the local OpenShift node where the agent is running.
func GetLocalNode(conf *config.Config, client *k8score.CoreClient) (node *k8sapi.Node, err error) {

	podNamespace := conf.Kubernetes.Pod_Namespace
	podName := conf.Kubernetes.Pod_Name

	if podNamespace == "" {
		return nil, fmt.Errorf("Pod namespace was not configured.")
	}

	if podName == "" {
		return nil, fmt.Errorf("Pod name was not configured.")
	}

	pod, err := client.Pods(podNamespace).Get(podName)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining information about the agent pod [%v/%v]. err=%v", podNamespace, podName, err)
	}

	nodeName := pod.Spec.NodeName
	log.Debugf("Agent pod [%v/%v] has node name of [%v]", podNamespace, podName, nodeName)

	node, err = client.Nodes().Get(nodeName)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining information about node [%v] from agent pod [%v/%v]. err=%v", nodeName, podNamespace, podName, err)
	}

	return
}
