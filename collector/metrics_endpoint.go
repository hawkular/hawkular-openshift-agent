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

package collector

import (
	"fmt"
	"time"

	"github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/config/tags"
)

type EndpointType string

const (
	ENDPOINT_TYPE_PROMETHEUS EndpointType = "prometheus"
	ENDPOINT_TYPE_JOLOKIA                 = "jolokia"
)

// MonitoredMetric provides information about a specific metric that is to be collected.
// The "Id" is the metric ID as it will be stored in Hawkular Metrics - it may or may not
// be identical to the actual metric name. The "Name" is the name of the metric as it is
// found in the endpoint. This is the true name of the metric as it is exposed from the system
// from where it came from.
// Tags specified here will be attached to the metric when stored to Hawkular Metrics.
// USED FOR YAML
type MonitoredMetric struct {
	ID          string ",omitempty"
	Name        string
	Type        metrics.MetricType
	Units       string    ",omitempty"
	Description string    ",omitempty"
	Tags        tags.Tags ",omitempty"
}

// Endpoint provides information about how to connect to a particular endpoint in order
// to collect metrics from it.
// If tenant is not supplied, the global tenant ID defined
// in the global agent configuration file should be used.
// Tags specified here will be attached to all metrics coming from this endpoint.
// TLS configures transport layer security if the URL connection uses TLS.
// USED FOR YAML (see agent config file)
type Endpoint struct {
	Type                EndpointType
	Enabled             string
	URL                 string
	TLS                 security.TLS
	Credentials         security.Credentials
	Collection_Interval string
	Tenant              string
	Tags                tags.Tags ",omitempty"
	Metrics             []MonitoredMetric
}

func (m *MonitoredMetric) String() string {
	return fmt.Sprintf("Metric: id=[%v], name=[%v], type=[%v], units=[%v], tags=[%v]", m.ID, m.Name, m.Type, m.Units, m.Tags)
}

func (m *MonitoredMetric) Clone() MonitoredMetric {
	cloneObj := MonitoredMetric{
		ID:          m.ID,
		Name:        m.Name,
		Type:        m.Type,
		Units:       m.Units,
		Description: m.Description,
	}

	if len(m.Tags) > 0 {
		cloneObj.Tags = tags.Tags{}
		cloneObj.Tags.AppendTags(m.Tags)
	}

	return cloneObj
}

// IsEnabled returns true if this endpoint has been enabled; false otherwise.
func (e *Endpoint) IsEnabled() bool {
	if e.Enabled == "" || e.Enabled == "true" {
		return true
	}
	return false
}

func (e *Endpoint) String() string {
	if e == nil {
		return ""
	}
	metricStrings := make([]string, len(e.Metrics))
	for i, m := range e.Metrics {
		metricStrings[i] = m.String()
	}
	return fmt.Sprintf("Endpoint: type=[%v], enabled=[%v], url=[%v], coll_int=[%v], tenant=[%v], tags=[%v], metrics=[%v]",
		e.Type, e.Enabled, e.URL, e.Collection_Interval, e.Tenant, e.Tags, metricStrings)
}

// ValidateEndpoint will check the endpoint configuration for correctness.
// If things are missing but can be corrected with defaults, that will be done.
// If something is wrong that cannot be corrected, a non-nil error is returned.
func (e *Endpoint) ValidateEndpoint() error {
	if err := e.Credentials.ValidateCredentials(); err != nil {
		return err
	}

	if e.URL == "" {
		return fmt.Errorf("Endpoint is missing URL")
	}

	if e.Type == "" {
		return fmt.Errorf("Endpoint [%v] is missing a valid type", e.URL)
	} else {
		if e.Type != ENDPOINT_TYPE_JOLOKIA && e.Type != ENDPOINT_TYPE_PROMETHEUS {
			return fmt.Errorf("Endpoint [%v] has invalid type [%v]", e.URL, e.Type)
		}
	}

	if e.Enabled != "" && e.Enabled != "true" && e.Enabled != "false" {
		return fmt.Errorf("Endpoint [%v] has invalid enabled flag [%v] - must be 'true' or 'false'", e.URL, e.Enabled)
	}

	for i, m := range e.Metrics {
		if m.Name == "" {
			return fmt.Errorf("Endpoint [%v] has a metric without a name", e.URL)
		}

		if m.Type == "" {
			// no need to define metric type if prometheus endpoint since it will tell us the type
			if e.Type != ENDPOINT_TYPE_PROMETHEUS {
				return fmt.Errorf("Endpoint [%v] metric [%v] is missing its type", e.URL, m.Name)
			}
		} else {
			if m.Type != metrics.Gauge && m.Type != metrics.Counter {
				return fmt.Errorf("Endpoint [%v] metric [%v] has invalid type [%v]", e.URL, m.Name, m.Type)
			}
		}

		// if there is no metric ID given, just use the metric name itself
		if m.ID == "" {
			e.Metrics[i].ID = m.Name
		}
	}

	if e.Collection_Interval != "" {
		if _, err := time.ParseDuration(e.Collection_Interval); err != nil {
			return fmt.Errorf("Endpoint [%v] has an invalid collection interval. err=%v", e.URL, err)
		}
	}

	return nil
}
