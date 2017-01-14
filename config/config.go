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

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/config/tags"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

// Environment vars can define some default values
const (
	ENV_HS_URL          = "HAWKULAR_SERVER_URL"
	ENV_HS_TENANT       = "HAWKULAR_SERVER_TENANT"
	ENV_HS_USERNAME     = "HAWKULAR_SERVER_USERNAME"
	ENV_HS_PASSWORD     = "HAWKULAR_SERVER_PASSWORD"
	ENV_HS_TOKEN        = "HAWKULAR_SERVER_TOKEN"
	ENV_HS_CA_CERT_FILE = "HAWKULAR_SERVER_CA_CERT_FILE"

	ENV_IDENTITY_CERT_FILE        = "HAWKULAR_OPENSHIFT_AGENT_CERT_FILE"
	ENV_IDENTITY_PRIVATE_KEY_FILE = "HAWKULAR_OPENSHIFT_AGENT_PRIVATE_KEY_FILE"

	ENV_K8S_MASTER_URL    = "K8S_MASTER_URL"
	ENV_K8S_POD_NAMESPACE = "K8S_POD_NAMESPACE"
	ENV_K8S_POD_NAME      = "K8S_POD_NAME"
	ENV_K8S_TOKEN         = "K8S_TOKEN"
	ENV_K8S_CA_CERT_FILE  = "K8S_CA_CERT_FILE"
	ENV_K8S_TENANT        = "K8S_TENANT"

	ENV_EMITTER_ADDRESS         = "EMITTER_ADDRESS"
	ENV_EMITTER_METRICS_ENABLED = "EMITTER_METRICS_ENABLED"
	ENV_EMITTER_STATUS_ENABLED  = "EMITTER_STATUS_ENABLED"
	ENV_EMITTER_HEALTH_ENABLED  = "EMITTER_HEALTH_ENABLED"
	ENV_EMITTER_STATUS_LOG_SIZE = "EMITTER_STATUS_LOG_SIZE"

	ENV_COLLECTOR_MINIMUM_COLL_INTERVAL = "COLLECTOR_MINIMUM_COLLECTION_INTERVAL"
	ENV_COLLECTOR_DEFAULT_COLL_INTERVAL = "COLLECTOR_DEFAULT_COLLECTION_INTERVAL"
)

// Hawkular_Server defines where the Hawkular Server is. This is where metrics are stored.
// The agent can identify with the Hawkular Server in one of two ways: either through
// basic authentication (with Username and Password) or with a bearer Token. Only
// one may be configured.
// USED FOR YAML
type Hawkular_Server struct {
	URL          string
	Tenant       string
	Credentials  security.Credentials ",omitempty"
	CA_Cert_File string               ",omitempty"
}

// Collector provides information about collecting metrics from monitored endpoints.
// Tags specified here will be attached to all metrics this agent collects and stores.
// ID_Prefix is a string (with potential ${env} tokens) that is prepended to all IDs of
// all metrics collected by the agent.
// USED FOR YAML
type Collector struct {
	Minimum_Collection_Interval string
	Default_Collection_Interval string
	Tags                        tags.Tags ",omitempty"
	Metric_ID_Prefix            string
}

// Kubernetes provides all the details necessary to communicate with the Kubernetes system.
// Master_Url should be an empty string if the agent is deployed in OpenShift.
// Pod_Namespace and Pod_Name should identify the pod where the agent is running (if it is
// running in OpenShift) or should identify any pod in the node to be monitored by the agent
// (if the agent is not running in OpenShift). Pod_Namespace should be empty if you do not wish
// for the agent to monitor anything in OpenShift.
// If Tenant is supplied, all metrics collected from all pods will have this tenant.
// You can specify ${x} tokens in the value for Tenant, such as ${some_env} or one of the POD
// tokens such as ${POD:namespace_name} which means all metrics will be stored under a tenant
// that is the same name of the pod namespace where the metric was collected.
// If Tenant is not supplied, the default is the Tenant defined in the Hawkular_Server section.
// USED FOR YAML
type Kubernetes struct {
	Master_URL    string ",omitempty"
	Token         string ",omitempty"
	CA_Cert_File  string ",omitempty"
	Pod_Namespace string ",omitempty"
	Pod_Name      string ",omitempty"
	Tenant        string ",omitempty"
}

// Emitter defines the behavior of the emitter which is responsible for
// emitting the agent's own metric data, a status report, and/or the health probe.
// USED FOR YAML
type Emitter struct {
	Metrics_Enabled string ",omitempty"
	Status_Enabled  string ",omitempty"
	Health_Enabled  string ",omitempty"
	Address         string ",omitempty"
	Status_Log_Size int    ",omitempty"
}

// Config defines the agent's full YAML configuration.
// USED FOR YAML
type Config struct {
	Identity        security.Identity ",omitempty"
	Hawkular_Server Hawkular_Server
	Emitter         Emitter              ",omitempty"
	Collector       Collector            ",omitempty"
	Kubernetes      Kubernetes           ",omitempty"
	Endpoints       []collector.Endpoint ",omitempty"
}

func NewConfig() (c *Config) {
	c = new(Config)

	c.Identity.Cert_File = getDefaultString(ENV_IDENTITY_CERT_FILE, "")
	c.Identity.Private_Key_File = getDefaultString(ENV_IDENTITY_PRIVATE_KEY_FILE, "")

	c.Hawkular_Server.URL = getDefaultString(ENV_HS_URL, "http://127.0.0.1:8080")
	c.Hawkular_Server.Tenant = getDefaultString(ENV_HS_TENANT, "hawkular")
	// If we are passing the username/password/token via an environment variable from a secret, we need to trim
	c.Hawkular_Server.Credentials.Username = strings.TrimSpace(getDefaultString(ENV_HS_USERNAME, ""))
	c.Hawkular_Server.Credentials.Password = strings.TrimSpace(getDefaultString(ENV_HS_PASSWORD, ""))
	c.Hawkular_Server.Credentials.Token = strings.TrimSpace(getDefaultString(ENV_HS_TOKEN, ""))
	c.Hawkular_Server.CA_Cert_File = getDefaultString(ENV_HS_CA_CERT_FILE, "")

	c.Collector.Minimum_Collection_Interval = getDefaultString(ENV_COLLECTOR_MINIMUM_COLL_INTERVAL, "10s")
	c.Collector.Default_Collection_Interval = getDefaultString(ENV_COLLECTOR_DEFAULT_COLL_INTERVAL, "5m")

	c.Kubernetes.Master_URL = getDefaultString(ENV_K8S_MASTER_URL, "")
	c.Kubernetes.Pod_Namespace = getDefaultString(ENV_K8S_POD_NAMESPACE, "")
	c.Kubernetes.Pod_Name = getDefaultString(ENV_K8S_POD_NAME, "")
	c.Kubernetes.Token = getDefaultString(ENV_K8S_TOKEN, "")
	c.Kubernetes.CA_Cert_File = getDefaultString(ENV_K8S_CA_CERT_FILE, "")
	c.Kubernetes.Tenant = getDefaultString(ENV_K8S_TENANT, "")

	c.Emitter.Metrics_Enabled = getDefaultString(ENV_EMITTER_METRICS_ENABLED, "true")
	c.Emitter.Status_Enabled = getDefaultString(ENV_EMITTER_STATUS_ENABLED, "true")
	c.Emitter.Health_Enabled = getDefaultString(ENV_EMITTER_HEALTH_ENABLED, "true")
	c.Emitter.Address = getDefaultString(ENV_EMITTER_ADDRESS, "")
	c.Emitter.Status_Log_Size = getDefaultInt(ENV_EMITTER_STATUS_LOG_SIZE, 10)

	return
}

func getDefaultString(envvar string, defaultValue string) (retVal string) {
	retVal = os.Getenv(envvar)
	if retVal == "" {
		retVal = defaultValue
	}
	return
}

func getDefaultInt(envvar string, defaultValue int) (retVal int) {
	retValString := os.Getenv(envvar)
	if retValString == "" {
		retVal = defaultValue
	} else {
		if num, err := strconv.Atoi(retValString); err != nil {
			log.Warningf("Invalid number for envvar [%v]. err=%v", envvar, err)
			retVal = defaultValue
		} else {
			retVal = num
		}
	}
	return
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

	// yaml unmarshalling leaves empty tags as nil - we want empty but non-nil
	if conf.Collector.Tags == nil {
		conf.Collector.Tags = tags.Tags{}
	}

	for i, e := range conf.Endpoints {
		if e.Tags == nil {
			conf.Endpoints[i].Tags = tags.Tags{}
		}
		for j, m := range e.Metrics {
			if m.Tags == nil {
				conf.Endpoints[i].Metrics[j].Tags = tags.Tags{}
			}
		}
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
