package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

// Environment vars can define some default values
const (
	ENV_HS_URL      = "HAWKULAR_SERVER_URL"
	ENV_HS_TENANT   = "HAWKULAR_SERVER_TENANT"
	ENV_HS_USERNAME = "HAWKULAR_SERVER_USERNAME"
	ENV_HS_PASSWORD = "HAWKULAR_SERVER_PASSWORD"
	ENV_HS_TOKEN    = "HAWKULAR_SERVER_TOKEN"

	ENV_IDENTITY_CERT_FILE        = "HAWKULAR_OPENSHIFT_AGENT_CERT_FILE"
	ENV_IDENTITY_PRIVATE_KEY_FILE = "HAWKULAR_OPENSHIFT_AGENT_PRIVATE_KEY_FILE"

	ENV_K8S_MASTER_URL      = "K8S_MASTER_URL"
	ENV_K8S_POD_NAMESPACE   = "K8S_POD_NAMESPACE"
	ENV_K8S_POD_NAME        = "K8S_POD_NAME"
	ENV_K8S_TOKEN           = "K8S_TOKEN"
	ENV_K8S_CA_CERT_FILE    = "K8S_CA_CERT_FILE"
	ENV_K8S_AUTHORIZED_PODS = "K8S_AUTHORIZED_PODS"
)

// Identity provides information about the identity of this agent.
// USED FOR YAML
type Identity struct {
	Cert_File        string
	Private_Key_File string
}

// Hawkular_Server defines where the Hawkular Server is. This is where metrics are stored.
// The agent can identify with the Hawkular Server in one of two ways: either through
// basic authentication (with Username and Password) or with a bearer Token. Only
// one may be configured.
// USED FOR YAML
type Hawkular_Server struct {
	Url      string
	Tenant   string
	Username string ",omitempty"
	Password string ",omitempty"
	Token    string ",omitempty"
}

// Collector provides information about collecting metrics from monitored endpoints.
// USED FOR YAML
type Collector struct {
	Minimum_Collection_Interval_Secs int
}

// Kubernetes provides all the details necessary to communicate with the Kubernetes system.
// USED FOR YAML
type Kubernetes struct {
	Master_Url      string   ",omitempty"
	Token           string   ",omitempty"
	CA_Cert_File    string   ",omitempty"
	Pod_Namespace   string   ",omitempty"
	Pod_Name        string   ",omitempty"
	Authorized_Pods []string ",omitempty"
}

// Config defines the agent's full YAML configuration.
// USED FOR YAML
type Config struct {
	Identity ",omitempty"
	Hawkular_Server
	Collector  ",omitempty"
	Kubernetes ",omitempty"
	Endpoints  []collector.Endpoint ",omitempty"
}

func NewConfig() (c *Config) {
	c = new(Config)

	c.Identity.Cert_File = getDefaultString(ENV_IDENTITY_CERT_FILE, "")
	c.Identity.Private_Key_File = getDefaultString(ENV_IDENTITY_PRIVATE_KEY_FILE, "")

	c.Hawkular_Server.Url = getDefaultString(ENV_HS_URL, "http://127.0.0.1:8080")
	c.Hawkular_Server.Tenant = getDefaultString(ENV_HS_TENANT, "hawkular")
	c.Hawkular_Server.Username = getDefaultString(ENV_HS_USERNAME, "")
	c.Hawkular_Server.Password = getDefaultString(ENV_HS_PASSWORD, "")
	c.Hawkular_Server.Token = getDefaultString(ENV_HS_TOKEN, "")

	c.Collector.Minimum_Collection_Interval_Secs = 10

	c.Kubernetes.Master_Url = getDefaultString(ENV_K8S_MASTER_URL, "")
	c.Kubernetes.Pod_Namespace = getDefaultString(ENV_K8S_POD_NAMESPACE, "")
	c.Kubernetes.Pod_Name = getDefaultString(ENV_K8S_POD_NAME, "")
	c.Kubernetes.Token = getDefaultString(ENV_K8S_TOKEN, "")
	c.Kubernetes.CA_Cert_File = getDefaultString(ENV_K8S_CA_CERT_FILE, "")
	c.Kubernetes.Authorized_Pods = convertCsvToArray(getDefaultString(ENV_K8S_AUTHORIZED_PODS, ""))

	return
}

func getDefaultString(envvar string, defaultValue string) (retVal string) {
	retVal = os.Getenv(envvar)
	if retVal == "" {
		retVal = defaultValue
	}
	return
}

func convertCsvToArray(csv string) []string {
	if csv == "" {
		return []string{}
	}
	return strings.Split(csv, ",")
}

// String marshals the given Config into a YAML string
func (conf Config) String() (str string) {
	str, err := Marshal(&conf)
	if err != nil {
		str = fmt.Sprintf("Failed to marshal config to string. err=%v", err)
		log.Debugf(str)
	}

	return
}

// Unmarshal parses the given YAML string and returns its Config object representation.
func Unmarshal(yamlString string) (conf *Config, err error) {
	conf = NewConfig()
	err = yaml.Unmarshal([]byte(yamlString), &conf)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse yaml data. error=%v", err)
	}

	return
}

// Marshal converts the Config object and returns its YAML string.
func Marshal(conf *Config) (yamlString string, err error) {
	yamlBytes, err := yaml.Marshal(&conf)
	if err != nil {
		return "", fmt.Errorf("Failed to produce yaml. error=%v", err)
	}

	yamlString = string(yamlBytes)
	return
}

// LoadFromFile reads the YAML from the given file, parses the content, and returns its Config object representation.
func LoadFromFile(filename string) (conf *Config, err error) {
	log.Debugf("Reading YAML config from [%s]", filename)
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to load config file [%v]. error=%v", filename, err)
	}

	return Unmarshal(string(fileContent))
}

// SaveToFile converts the Config object and stores its YAML string into the given file, overwriting any data that is in the file.
func SaveToFile(filename string, conf *Config) (err error) {
	fileContent, err := Marshal(conf)
	if err != nil {
		return fmt.Errorf("Failed to save config file [%v]. error=%v", filename, err)
	}

	log.Debugf("Writing YAML config to [%s]", filename)
	err = ioutil.WriteFile(filename, []byte(fileContent), 0640)
	return
}
