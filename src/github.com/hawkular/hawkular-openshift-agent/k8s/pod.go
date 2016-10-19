package k8s

import (
	"fmt"
)

type Pod struct {
	NodeName    string
	Namespace   string
	Name        string
	Uid         string
	IPAddress   string
	Labels      map[string]string
	Annotations map[string]string
}

// Identifier returns a string smaller than String() but can still uniquely identify the pod
func (p *Pod) GetIdentifier() string {
	return fmt.Sprintf("%v/%v/%v/%v", p.NodeName, p.Namespace, p.Name, p.Uid)
}

func (p *Pod) String() string {
	return fmt.Sprintf("Pod: [%v], ip=[%v], labels=[%v], annotations=[%v]",
		p.GetIdentifier(), p.IPAddress, p.Labels, p.Annotations)
}
