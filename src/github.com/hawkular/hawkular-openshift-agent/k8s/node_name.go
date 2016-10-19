package k8s

import (
	"fmt"

	"k8s.io/client-go/1.4/kubernetes/typed/core/v1"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

func GetNodeName(conf *config.Config, client *v1.CoreClient) (nodeName string, err error) {

	podNamespace := conf.Kubernetes.Pod_Namespace
	podName := conf.Kubernetes.Pod_Name

	if podNamespace == "" {
		return "", fmt.Errorf("Pod namespace was not configured.")
	}

	if podName == "" {
		return "", fmt.Errorf("Pod name was not configured.")
	}

	pod, err := client.Pods(podNamespace).Get(podName)
	if err != nil {
		return "", fmt.Errorf("Error obtaining information about the pod [%v/%v]. err=%v", podNamespace, podName, err)
	}

	nodeName = pod.Spec.NodeName
	log.Debugf("Pod [%v/%v] has node name of [%v]", podNamespace, podName, nodeName)
	return
}
