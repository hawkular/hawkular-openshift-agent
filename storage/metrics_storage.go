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

package storage

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"reflect"

	"github.com/golang/glog"
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/config"
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
	// create one client for metrics and another for definitions - this way no concurrency issues to worry about
	clientMetrics, err := getHawkularMetricsClient(conf)
	if err != nil {
		return nil, err
	}
	clientMetricDefs, err := getHawkularMetricsClient(conf)
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
	glog.Info("START storing metrics definitions and data")
	go ms.consumeMetricDefinitions()
	go ms.consumeMetrics()
}

func (ms *MetricsStorageManager) StopStoringMetrics() {
	glog.Info("STOP storing metrics definitions and data")
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
			existing, err := ms.hawkClientMetricDefinitions.Definition(md.Type, md.ID, modifier)
			if existing == nil {
				if err == nil {
					// doesn't exist - create it
					ok, createErr := ms.hawkClientMetricDefinitions.Create(md, modifier)
					if !ok {
						glog.Warningf("Failed to create new metric definition [%v] of type [%v] in tenant [%v]. err=%v", md.ID, md.Type, tenant, createErr)
					}
				} else {
					glog.Warningf("Failed to determine if metric definition [%v] of type [%v] in tenant [%v] exists. err=%v", md.ID, md.Type, tenant, err)
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
								glog.Warningf("Failed to delete obsolete tag [%v=%v] from metric definition [%v] of type [%v] in tenant [%v]. err=%v", existingTagName, existingTagValue, md.ID, md.Type, tenant, err)
							}
						}
					}

					// now update any existing/new ones
					log.Debugf("Updating tags for metric definition [%v] of type [%v] in tenant [%v]", md.ID, md.Type, tenant)
					err := ms.hawkClientMetricDefinitions.UpdateTags(md.Type, md.ID, md.Tags, modifier)
					if err != nil {
						glog.Warningf("Failed to update tags for metric definition [%v] of type [%v] in tenant [%v]. err=%v", md.ID, md.Type, tenant, err)
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
			glog.Warningf("Failed to store metrics. err=%v", err)
		} else {
			log.Debugf("Stored datapoints for [%v] metrics", len(metrics))
			if log.IsTrace() {
				for _, m := range metrics {
					log.Tracef("Stored [%v] [%v] datapoints for metric named [%v]: %v", len(m.Data), m.Type, m.ID, m.Data)
				}
			}
		}
	}
}

func getHawkularMetricsClient(conf *config.Config) (*hmetrics.Client, error) {

	tlsConfig := &tls.Config{}

	if conf.Hawkular_Server.CA_Cert_File != "" {
		certs := x509.NewCertPool()

		cert, err := ioutil.ReadFile(conf.Hawkular_Server.CA_Cert_File)
		if err != nil {
			glog.Warningf("Failed to load the CA file for Hawkular Metrics. You may not be able to properly connect to the Hawkular Metrics server. err=%v", err)
		}

		certs.AppendCertsFromPEM(cert)
		tlsConfig.RootCAs = certs
	}

	params := hmetrics.Parameters{
		Tenant:    conf.Hawkular_Server.Tenant,
		Url:       conf.Hawkular_Server.URL,
		Username:  conf.Hawkular_Server.Credentials.Username,
		Password:  conf.Hawkular_Server.Credentials.Password,
		Token:     conf.Hawkular_Server.Credentials.Token,
		TLSConfig: tlsConfig,
	}

	return hmetrics.NewHawkularClient(params)
}
