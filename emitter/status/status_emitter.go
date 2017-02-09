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

package status

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/tracker"
)

// Name is the name of the agent.
// Version is the x.y.z version string of the agent.
// Commit_Hash is the git commit hash of the source used to build the agent.
// Metrics will be filled in from data within the metrics tracker - it will contain
// all pods, their endpoints, and how many metrics from them are being collected.
// Endpoints is keyed on an endpoint ID whose value is the last message related to the endpoint.
// (endpoint IDs are always unique even across pods)
// Log is a small rolling log of important messages.
// Do not directly access StatusReport data fields; for thread safety, use the funcs.
// USED FOR YAML
type StatusReportType struct {
	Name           string
	Version        string
	Commit_Hash    string
	Metrics        map[string]map[string]int
	Endpoints      map[string]string
	Log            []string
	lock           sync.RWMutex
	metricsTracker *tracker.MetricsTracker
}

var StatusReport StatusReportType

func InitStatusReport(name string, version string, commitHash string, logSize int) {
	StatusReport = StatusReportType{
		Name:        name,
		Version:     version,
		Commit_Hash: commitHash,
		Endpoints:   make(map[string]string, 0),
		Log:         make([]string, logSize),
		lock:        sync.RWMutex{},
	}
}

// SetMetricsTracker is called when the collector manager is initialized. This
// will let the status report know how many metrics are being collected and from where.
func (s *StatusReportType) SetMetricsTracker(metricsTracker *tracker.MetricsTracker) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.metricsTracker = metricsTracker
}

// GetEndpoint will get the status message assigned to the given endpoint ID.
func (s *StatusReportType) GetEndpointMessage(id collector.CollectorID) (messages string, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	messages, ok = s.Endpoints[id.EndpointID]
	return
}

// SetEndpoint will assign the given status message to the given endpoint ID.
func (s *StatusReportType) SetEndpointMessage(id collector.CollectorID, msg string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if msg == "" {
		delete(s.Endpoints, id.EndpointID)
	} else {
		s.Endpoints[id.EndpointID] = msg
	}
}

func (s *StatusReportType) DeleteAllEndpointMessages() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for id := range s.Endpoints {
		delete(s.Endpoints, id)
	}
}

// AddLogMessage pushes the given message to the rolling log.
func (s *StatusReportType) AddLogMessage(m string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.Log) > 1 {
		copy(s.Log, s.Log[1:])
		s.Log[len(s.Log)-1] = fmt.Sprintf("%v: %v", time.Now().Format(time.RFC1123Z), m)
	}
}

func (s *StatusReportType) Marshal() (str string) {
	s.populate()

	s.lock.RLock()
	defer s.lock.RUnlock()
	if statusBytes, err := yaml.Marshal(&StatusReport); err != nil {
		str = fmt.Sprintf("Error: %v", err)
	} else {
		str = string(statusBytes)
	}
	return
}

// populate will fill in the report.
// It will take data from the metrics tracker and fill in the status report
// with details on how many metrics are being collected from where.
func (s *StatusReportType) populate() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.cleanup()

	if s.metricsTracker != nil {
		allPods := s.metricsTracker.GetAllPods()
		s.Metrics = make(map[string]map[string]int, len(allPods))
		for pid, _ := range allPods {
			s.Metrics[pid] = s.metricsTracker.GetAllEndpoints(pid)
		}
	} else {
		s.Metrics = make(map[string]map[string]int, 0)
	}
}

// cleanup removes obsolete endpoints.
// Caller MUST hold a write lock.
func (s *StatusReportType) cleanup() {
	if s.metricsTracker == nil {
		return // we aren't ready to do cleanup
	}

	// sometimes endpoints get leftover, even though they are not being monitored anymore - remove them
	validEndpoints := make(map[string]bool, 0)
	for pid, _ := range s.metricsTracker.GetAllPods() {
		for eid, _ := range s.metricsTracker.GetAllEndpoints(pid) {
			validEndpoints[eid] = true
		}
	}

	for endpoint, _ := range s.Endpoints {
		if !validEndpoints[endpoint] {
			delete(s.Endpoints, endpoint)
		}
	}
}
