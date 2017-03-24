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
)

type Pod struct {
	Node             Node
	Namespace        Namespace
	Name             string
	UID              string
	PodIP            string
	HostIP           string
	Hostname         string
	Subdomain        string
	Labels           map[string]string
	Annotations      map[string]string
	ConfigMapVolumes map[string]string
	ClusterName      string
	ResourceVersion  string
	SelfLink         string
}

// Identifier returns a string smaller than String() but can still uniquely identify the pod
func (p *Pod) GetIdentifier() string {
	return fmt.Sprintf("%v/%v/%v/%v", p.Node.Name, p.Namespace.Name, p.Name, p.UID)
}

func (p *Pod) String() string {
	return fmt.Sprintf("Pod: [%v], pod-ip=[%v], host-ip=[%v], subdomain=[%v], hostname=[%v], labels=[%v], annotations=[%v], config-map-volumes=[%v]",
		p.GetIdentifier(), p.PodIP, p.HostIP, p.Subdomain, p.Hostname, p.Labels, p.Annotations, p.ConfigMapVolumes)
}
