package k8s

import (
	"fmt"
)

type Pod struct {
	Node        Node
	Namespace   Namespace
	Name        string
	UID         string
	PodIP       string
	HostIP      string
	Labels      map[string]string
	Annotations map[string]string
}

// Identifier returns a string smaller than String() but can still uniquely identify the pod
func (p *Pod) GetIdentifier() string {
	return fmt.Sprintf("%v/%v/%v/%v", p.Node.Name, p.Namespace.Name, p.Name, p.UID)
}

func (p *Pod) String() string {
	return fmt.Sprintf("Pod: [%v], pod-ip=[%v], host-ip=[%v], labels=[%v], annotations=[%v]",
		p.GetIdentifier(), p.PodIP, p.HostIP, p.Labels, p.Annotations)
}
