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

package storage

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	agentmetrics "github.com/hawkular/hawkular-openshift-agent/emitter/metrics"
	"github.com/hawkular/hawkular-openshift-agent/k8s"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

type MetricsStorageManager struct {
	MetricsChannel              chan []hmetrics.MetricHeader
	MetricDefinitionsChannel    chan []hmetrics.MetricDefinition
	hawkClientMetrics           *hmetrics.Client
	hawkClientMetricDefinitions *hmetrics.Client
	globalConfig                *config.Config
}

func NewMetricsStorageManager(conf *config.Config) (ms *MetricsStorageManager, err error) {

	hawkularServer, err := processHawkularMetricsConfig(conf)
	if err != nil {
		return nil, err
	}

	// create one client for metrics and another for definitions - this way no concurrency issues to worry about
	clientMetrics, err := getHawkularMetricsClient(hawkularServer)
	if err != nil {
		return nil, err
	}
	clientMetricDefs, err := getHawkularMetricsClient(hawkularServer)
	if err != nil {
		return nil, err
	}

	ms = &MetricsStorageManager{
		MetricsChannel:              make(chan []hmetrics.MetricHeader, 100),
		MetricDefinitionsChannel:    make(chan []hmetrics.MetricDefinition, 100),
		hawkClientMetrics:           clientMetrics,
		hawkClientMetricDefinitions: clientMetricDefs,
		globalConfig:                conf,
	}
	return
}

func (ms *MetricsStorageManager) StartStoringMetrics() {
	log.Info("START storing metrics definitions and data")
	go ms.consumeMetricDefinitions()
	go ms.consumeMetrics()
}

func (ms *MetricsStorageManager) StopStoringMetrics() {
	log.Info("STOP storing metrics definitions and data")
	close(ms.MetricsChannel)
	close(ms.MetricDefinitionsChannel)
}

func (ms *MetricsStorageManager) consumeMetricDefinitions() {
	for metricDefs := range ms.MetricDefinitionsChannel {
		if len(metricDefs) == 0 {
			continue
		}

		// If a tenant is provided, use it. Otherwise, use the global tenant.
		// This assumes all metric defs in the given array are associated with the same tenant.
		var tenant string
		if metricDefs[0].Tenant != "" {
			tenant = metricDefs[0].Tenant
		} else {
			tenant = ms.globalConfig.Hawkular_Server.Tenant
		}

		modifier := hmetrics.Tenant(tenant)

		// Store the metric definitions to H-Metrics.
		for _, md := range metricDefs {
			log.Tracef("Asked to store metric definition: %#v", md)

			existing, err := ms.hawkClientMetricDefinitions.Definition(md.Type, md.ID, modifier)
			if existing == nil {
				if err == nil {
					// doesn't exist - create it
					ok, createErr := ms.hawkClientMetricDefinitions.Create(md, modifier)
					if !ok {
						log.Warningf("Failed to create new metric definition [%v] of type [%v] in tenant [%v]. err=%v", md.ID, md.Type, tenant, createErr)
					}
				} else {
					log.Warningf("Failed to determine if metric definition [%v] of type [%v] in tenant [%v] exists. err=%v", md.ID, md.Type, tenant, err)
				}
			} else {
				// metric def exists, we just need to update it if it needs to be
				if !reflect.DeepEqual(md.Tags, existing.Tags) {
					log.Debugf("Deleting obsolete tags from metric definition [%v] of type [%v] in tenant [%v]", md.ID, md.Type, tenant)
					// if there are tags that currently exist but no longer should, delete them
					for existingTagName, existingTagValue := range existing.Tags {
						if _, ok := md.Tags[existingTagName]; !ok {
							log.Tracef("Deleting obsolete tag [%v] from metric definition [%v] of type [%v] in tenant [%v].", existingTagName, md.ID, md.Type, tenant)
							err := ms.hawkClientMetricDefinitions.DeleteTags(md.Type, md.ID, map[string]string{existingTagName: existingTagValue}, modifier)
							if err != nil {
								log.Warningf("Failed to delete obsolete tag [%v=%v] from metric definition [%v] of type [%v] in tenant [%v]. err=%v", existingTagName, existingTagValue, md.ID, md.Type, tenant, err)
							}
						}
					}

					// now update any existing/new ones
					log.Debugf("Updating tags for metric definition [%v] of type [%v] in tenant [%v]", md.ID, md.Type, tenant)
					err := ms.hawkClientMetricDefinitions.UpdateTags(md.Type, md.ID, md.Tags, modifier)
					if err != nil {
						log.Warningf("Failed to update tags for metric definition [%v] of type [%v] in tenant [%v]. err=%v", md.ID, md.Type, tenant, err)
					}
				}
			}
		}
	}
}

func (ms *MetricsStorageManager) consumeMetrics() {
	for metrics := range ms.MetricsChannel {
		if len(metrics) == 0 {
			continue
		}

		// If a tenant is provided, use it. Otherwise, use the global tenant.
		// This assumes all metrics in the given array are associated with the same tenant.
		var tenant string
		if metrics[0].Tenant != "" {
			tenant = metrics[0].Tenant
		} else {
			tenant = ms.globalConfig.Hawkular_Server.Tenant
		}

		// Store the metrics to H-Metrics.
		err := ms.hawkClientMetrics.Write(metrics, hmetrics.Tenant(tenant))

		if err != nil {
			log.Warningf("Failed to store metrics. err=%v", err)
			log.Tracef("These metrics failed to be stored: %v", metrics)
		} else {
			log.Debugf("Stored datapoints for [%v] metrics", len(metrics))
			for _, m := range metrics {
				agentmetrics.Metrics.DataPointsStored.Add(float64(len(m.Data)))
				log.Tracef("Stored [%v] [%v] datapoints for metric named [%v]: %v", len(m.Data), m.Type, m.ID, m.Data)
			}
		}
	}
}

func processHawkularMetricsConfig(conf *config.Config) (config.Hawkular_Server, error) {
	if !k8s.IsConfiguredForKubernetes(conf) {
		return conf.Hawkular_Server, nil
	}

	client, clientErr := k8s.GetKubernetesClient(conf)
	if clientErr != nil {
		log.Errorf("Error trying to get Kubernetes client: %v", clientErr)
		return conf.Hawkular_Server, clientErr
	}

	waitForSecret := func(namespace, secretName, key string) ([]byte, error) {

		timeout := time.Now().Add(5 * time.Minute)

		messaged := false
		var lastError error
		for {
			if time.Now().After(timeout) {
				return []byte{}, fmt.Errorf("Could not fetch the required secrets after 5 minutes. This may mean the secrets have not yet been created yet or the agent does not have the required permission to access the secret.")
			}

			secret, err := client.Secrets(namespace).Get(secretName)
			if err != nil {
				if lastError != err {
					log.Errorf("Error trying to get Secret named [%v] from namespace [%v]: err=%v", secretName, namespace, err)
					lastError = err
				}
			} else {
				value, found := secret.Data[key]
				if !found {
					if lastError != err {
						log.Errorf("Secret [%v] does not contain key [%v] in namespace [%v]", secretName, key, namespace)
						lastError = err
					}
				} else {
					return value, nil
				}
			}

			if !messaged {
				log.Errorf("Could not get the requested secret. This may mean the secret has not yet been created yet or the agent does not have the required permission to access the secret. Will attempt again every 5 seconds for the next 5 minutes")
			}
			messaged = true
			time.Sleep(5 * time.Second)
		}
	}

	// Function that will extract a credential string based on the given value.
	// If the given value string is prefixed with "secret:" it is assumed to be a
	// key/value from a k8s secret. The value string can have an optional namespace
	// such as "secret:my-project/my-secret/username". If the string is not prefixed, it is used as-is.
	// The file parameter, if true, will tell this function to write the credential value
	// to a file and return the filename; otherwise, the credential value is returned directly.
	// The file parameter, when true, is strictly for the CA cert file.
	f := func(v string, file bool) (string, error) {
		if strings.HasPrefix(v, "secret:") {
			v = strings.TrimLeft(v, "secret:")
			splits := strings.SplitN(v, "/", -1)
			if len(splits) != 2 && len(splits) != 3 {
				log.Errorf("Hawkular Metrics secret specifier is invalid: [%v]", v)
				return "", fmt.Errorf("Invalid Hawkular Metrics secret specifier: [%v]", v)
			}

			var namespace string
			var secretName string
			var secretKey string
			if len(splits) == 2 {
				namespace = conf.Kubernetes.Pod_Namespace
				secretName = splits[0]
				secretKey = splits[1]
			} else {
				namespace = splits[0]
				secretName = splits[1]
				secretKey = splits[2]
			}
			bytes, err := waitForSecret(namespace, secretName, secretKey)
			if err != nil {
				return "", err
			}

			if !file {
				return strings.TrimSpace(string(bytes)), nil
			} else {
				dirName := os.TempDir() + string(os.PathSeparator) + secretName
				os.MkdirAll(dirName, 0744)
				fileName := dirName + string(os.PathSeparator) + secretKey
				fileErr := ioutil.WriteFile(fileName, bytes, 0644)
				if fileErr != nil {
					log.Errorf("Could not write the Hawkular Metrics CA to file: err=%v", fileErr)
					return "", fmt.Errorf("Could not write the Hawkular Metrics CA to file: err=%v", fileErr)
				}
				return fileName, nil
			}
		} else {
			return v, nil
		}
	}

	var url, caCertFile, username, password, token, tenant string
	var err error

	if url, err = f(conf.Hawkular_Server.URL, false); err != nil {
		return conf.Hawkular_Server, err
	}
	if caCertFile, err = f(conf.Hawkular_Server.CA_Cert_File, true); err != nil {
		return conf.Hawkular_Server, err
	}
	if username, err = f(conf.Hawkular_Server.Credentials.Username, false); err != nil {
		return conf.Hawkular_Server, err
	}
	if password, err = f(conf.Hawkular_Server.Credentials.Password, false); err != nil {
		return conf.Hawkular_Server, err
	}
	if token, err = f(conf.Hawkular_Server.Credentials.Token, false); err != nil {
		return conf.Hawkular_Server, err
	}
	if tenant, err = f(conf.Hawkular_Server.Tenant, false); err != nil {
		return conf.Hawkular_Server, err
	}

	hawkularCredentials := config.Hawkular_Server{
		URL:          url,
		CA_Cert_File: caCertFile,
		Credentials: security.Credentials{
			Username: username,
			Password: password,
			Token:    token,
		},
		Tenant: tenant,
	}

	return hawkularCredentials, nil
}

func getHawkularMetricsClient(conf config.Hawkular_Server) (*hmetrics.Client, error) {
	tlsConfig := &tls.Config{}

	if conf.CA_Cert_File != "" {
		certs := x509.NewCertPool()

		cert, err := ioutil.ReadFile(conf.CA_Cert_File)
		if err != nil {
			log.Warningf("Failed to load the CA file for Hawkular Metrics. You may not be able to properly connect to the Hawkular Metrics server. err=%v", err)
		}

		certs.AppendCertsFromPEM(cert)
		tlsConfig.RootCAs = certs
	}

	params := hmetrics.Parameters{
		Tenant:    conf.Tenant,
		Url:       conf.URL,
		Username:  conf.Credentials.Username,
		Password:  conf.Credentials.Password,
		Token:     conf.Credentials.Token,
		TLSConfig: tlsConfig,
	}

	return hmetrics.NewHawkularClient(params)
}
