package k8s

import (
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/hawkular/hawkular-client-go/metrics"
	"gopkg.in/yaml.v2"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

type K8SEndpointProtocol string

const (
	K8S_ENDPOINT_PROTOCOL_HTTP  K8SEndpointProtocol = "http"
	K8S_ENDPOINT_PROTOCOL_HTTPS                     = "https"
)

// Endpoint describes a place where and what metrics are exposed.
// Type indicates the kind of metric endpoint (e.g. Prometheus or Jolokia).
// Protocol defines the communications protocol (e.g. http or https).
// Notice that Host is not defined - it will be determined at runtime via the pod configuration.
// If tenant is not supplied, the global tenant ID defined
// in the global agent configuration file should be used.
// USED FOR YAML
type K8SEndpoint struct {
	Type                     collector.EndpointType
	Protocol                 K8SEndpointProtocol
	Port                     int
	Path                     string
	Tenant                   string
	Collection_Interval_Secs int
	Metrics                  []K8SMetric
}

// Metric describes a metric that is to be collected.
// USED FOR YAML
type K8SMetric struct {
	Type metrics.MetricType
	Name string
}

// ConfigMapEntry describes one of the YAML configurations provided in a Pod's configmap.
// USED FOR YAML
type ConfigMapEntry struct {
	Endpoints []K8SEndpoint
}

// GetUrl returns a URL for the endpoint given a host string that is needed to complete the URL
func (e K8SEndpoint) GetUrl(host string) (u *url.URL, err error) {
	leadingSlash := "/"
	if e.Path[0] == '/' {
		leadingSlash = ""
	}
	u, err = url.Parse(fmt.Sprintf("%v://%v:%v%v%v)", e.Protocol, host, e.Port, leadingSlash, e.Path))
	return
}

func NewConfigMapEntry() (c *ConfigMapEntry) {
	c = new(ConfigMapEntry)
	c.Endpoints = make([]K8SEndpoint, 0)
	return
}

// String marshals the given ConfigMapEntry into a YAML string
func (cme *ConfigMapEntry) String() (str string) {
	str, err := MarshalConfigMapEntry(cme)
	if err != nil {
		str = fmt.Sprintf("Failed to marshal config map entry to string. err=%v", err)
		log.Debugf(str)
	}

	return
}

// Unmarshal parses the given YAML string and returns its ConfigMapEntry object representation.
func UnmarshalConfigMapEntry(yamlString string) (cme *ConfigMapEntry, err error) {
	cme = NewConfigMapEntry()
	err = yaml.Unmarshal([]byte(yamlString), &cme)
	if err != nil {
		glog.Errorf("Failed to parse yaml data for config map entry. error=%v", err)
	}

	return
}

// Marshal converts the ConfigMapEntry object and returns its YAML string.
func MarshalConfigMapEntry(cme *ConfigMapEntry) (yamlString string, err error) {
	yamlBytes, err := yaml.Marshal(&cme)
	if err != nil {
		glog.Errorf("Failed to produce yaml for config map entry. error=%v", err)
	}

	yamlString = string(yamlBytes)
	return
}
