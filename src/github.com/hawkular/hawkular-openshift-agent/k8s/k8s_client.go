package k8s

import (
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/kubernetes/typed/core/v1"
	"k8s.io/client-go/1.4/rest"

	"github.com/hawkular/hawkular-openshift-agent/config"
)

const userAgent string = "Hawkular/Hawkular-OpenShift-Agent"

func GetKubernetesClient(conf *config.Config) (coreClient *v1.CoreClient, err error) {

	var restConfig *rest.Config

	url := conf.Kubernetes.Master_Url
	token := conf.Kubernetes.Token
	caCertFile := conf.Kubernetes.CA_Cert_File

	// if no values are passed, assume that we are running within the container within the Kubernetes cluster
	if url == "" || token == "" {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		c := rest.Config{
			Host:        url,
			BearerToken: token,
		}

		if caCertFile != "" {
			tlsConfig := rest.TLSClientConfig{}
			tlsConfig.CAFile = caCertFile

			c.TLSClientConfig = tlsConfig
		}

		restConfig = &c

	}

	restConfig.UserAgent = userAgent

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	coreClient = client.CoreClient
	return
}
