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

package manager

import (
	"fmt"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/impl"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
)

func CreateMetricsCollector(id collector.CollectorID, identity security.Identity, endpoint collector.Endpoint, env map[string]string) (theCollector collector.MetricsCollector, err error) {
	switch endpoint.Type {
	case collector.ENDPOINT_TYPE_PROMETHEUS:
		{
			theCollector = impl.NewPrometheusMetricsCollector(id, identity, endpoint, env)
		}
	case collector.ENDPOINT_TYPE_JOLOKIA:
		{
			theCollector = impl.NewJolokiaMetricsCollector(id, identity, endpoint, env)
		}
	default:
		{
			err = fmt.Errorf("Unknown endpoint type [%v]", endpoint.Type)
		}
	}
	return
}
