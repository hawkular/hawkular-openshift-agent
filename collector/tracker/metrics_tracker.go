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
package tracker

import (
	"bytes"
	"fmt"
	"sync"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
)

type MetricsTracker struct {
	// AllMetrics is tracks all metrics currently known.
	// Outer map key is pod ID.
	// Middle map key is endpoint ID.
	// Inner key is fully expanded metric ID.
	// If the pod ID is empty string, that represents external endpoints.
	allMetrics       map[string]map[string]map[string]bool
	maxMetricsPerPod int
	lock             sync.RWMutex
}

func NewMetricsTracker(max int) MetricsTracker {
	return MetricsTracker{
		allMetrics:       make(map[string]map[string]map[string]bool, 0),
		maxMetricsPerPod: max,
		lock:             sync.RWMutex{},
	}
}

// GetCounts returns three integers:
//   1. The total number of pods
//   2. The total number of endpoints
//   3. The total number of metrics
func (mt *MetricsTracker) GetCounts() (pCount int, eCount int, mCount int) {
	mt.lock.RLock()
	defer mt.lock.RUnlock()
	for _, endpoints := range mt.allMetrics {
		pCount++
		for _, metrics := range endpoints {
			eCount++
			mCount += len(metrics)
		}
	}
	return
}

// GetAllPods returns pod identifiers for all known pods as the keys to the
// returned map. The values are the number of endpoints in each pod.
func (mt *MetricsTracker) GetAllPods() map[string]int {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	podIds := make(map[string]int, len(mt.allMetrics))
	for pid, endpoints := range mt.allMetrics {
		podIds[pid] = len(endpoints)
	}
	return podIds
}

// GetAllEndpoints returns endpoint IDs for all known endpoints in the given pod as
// the keys to the returned map. The values are the number of metrics being collected
// for that endpoint.
func (mt *MetricsTracker) GetAllEndpoints(podIdentifier string) map[string]int {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	if endpoints, ok := mt.allMetrics[podIdentifier]; ok {
		endpointIds := make(map[string]int, len(endpoints))
		for eid, metrics := range endpoints {
			endpointIds[eid] = len(metrics)
		}
		return endpointIds
	} else {
		return map[string]int{}
	}
}

func (mt *MetricsTracker) PurgeMetricsForCollectorEndpoint(cid collector.CollectorID) {
	mt.lock.Lock()
	defer mt.lock.Unlock()
	delete(mt.allMetrics[cid.PodID], cid.EndpointID)
	if len(mt.allMetrics[cid.PodID]) == 0 {
		delete(mt.allMetrics, cid.PodID)
	}
}

func (mt *MetricsTracker) PurgeAllMetrics() {
	mt.lock.Lock()
	defer mt.lock.Unlock()
	mt.allMetrics = make(map[string]map[string]map[string]bool, 0)
}

// AddMetricsFromCollector will associate the given metrics with the given collector's pod.
// If there end up to be more metrics associated with the given collector's pod than the maximum allowed,
// the overflow metrics are not added and returned. If the max limit has not been reached, nil is returned.
func (mt *MetricsTracker) AddMetricsFromCollector(cid collector.CollectorID, metrics []hmetrics.MetricHeader) []hmetrics.MetricHeader {
	mt.lock.Lock()
	defer mt.lock.Unlock()

	// count how many metrics we already have for this pod and at the same time look for our endpoint's metrics
	var podEndpoints map[string]map[string]bool
	var endpointMetrics map[string]bool
	count := 0
	if endpoints, ok := mt.allMetrics[cid.PodID]; ok {
		podEndpoints = endpoints
		for eid, metrics := range endpoints {
			count += len(metrics)
			if eid == cid.EndpointID {
				endpointMetrics = metrics
			}
		}
	}

	if podEndpoints == nil {
		podEndpoints = make(map[string]map[string]bool, 1)
		mt.allMetrics[cid.PodID] = podEndpoints
	}

	if endpointMetrics == nil {
		endpointMetrics = make(map[string]bool, 0)
		podEndpoints[cid.EndpointID] = endpointMetrics
	}

	var overflow []hmetrics.MetricHeader

	for _, metric := range metrics {
		if _, ok := endpointMetrics[metric.ID]; !ok {
			if count < mt.maxMetricsPerPod {
				endpointMetrics[metric.ID] = true
				count++
			} else {
				overflow = append(overflow, metric)
			}
		}
	}

	return overflow
}

func (mt *MetricsTracker) String() string {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	var buffer bytes.Buffer
	for pid, endpoints := range mt.allMetrics {
		buffer.WriteString(fmt.Sprintf("%v\n", pid))
		for eid, metrics := range endpoints {
			buffer.WriteString(fmt.Sprintf("-- %v\n", eid))
			for mid, _ := range metrics {
				buffer.WriteString(fmt.Sprintf("---- %v\n", mid))
			}
		}
	}
	return buffer.String()
}
