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
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/hawkular/hawkular-openshift-agent/emitter/metrics"
)

// Name is the name of the agent.
// Version is the x.y.z version string of the agent.
// Commit_Hash is the git commit hash of the source used to build the agent.
// Pods is keyed on the pod identifier whose value is the endpoint IDs for the pod.
// Endpoints is keyed on an endpoint ID whose value is the last message related to the endpoint.
// Log is a small rolling log of important messages.
// Do not directly access StatusReport data fields; for thread safety, use the funcs.
// USED FOR YAML
type StatusReportType struct {
	Name        string
	Version     string
	Commit_Hash string
	Pods        map[string][]string
	Endpoints   map[string]string
	Log         []string
	lock        sync.RWMutex
}

var StatusReport StatusReportType

func InitStatusReport(name string, version string, commitHash string, logSize int) {
	StatusReport = StatusReportType{
		Name:        name,
		Version:     version,
		Commit_Hash: commitHash,
		Pods:        make(map[string][]string, 0),
		Endpoints:   make(map[string]string, 0),
		Log:         make([]string, logSize),
		lock:        sync.RWMutex{},
	}
}

// GetPod will get the set of endpoints assigned to the given pod ID.
func (s *StatusReportType) GetPod(podId string) (endpointIds []string, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	endpointIds, ok = s.Pods[podId]
	return
}

// SetPod will assign the given set of endpoints to the given pod ID.
func (s *StatusReportType) SetPod(podId string, endpointIds []string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if endpointIds == nil {
		delete(s.Pods, podId)
		s.cleanup(false)
	} else {
		s.Pods[podId] = endpointIds
	}

	// keep our metric up to date to track how many pods are being monitored
	metrics.Metrics.MonitoredPods.Set(float64(len(s.Pods)))
}

// GetEndpoint will get the status message assigned to the given endpoint ID.
func (s *StatusReportType) GetEndpoint(endpointId string) (messages string, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	messages, ok = s.Endpoints[endpointId]
	return
}

// SetEndpoint will assign the given status message to the given endpoint ID.
func (s *StatusReportType) SetEndpoint(endpointId string, msg string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if msg == "" {
		delete(s.Endpoints, endpointId)
	} else {
		s.Endpoints[endpointId] = msg
	}

	// keep our metric up to date to track how many endpoints are being monitored
	metrics.Metrics.MonitoredEndpoints.Set(float64(len(s.Endpoints)))
}

func (s *StatusReportType) DeleteAllEndpoints() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for id := range s.Endpoints {
		delete(s.Endpoints, id)
	}

	// set our metric to show we are not monitoring any more endpoints
	metrics.Metrics.MonitoredEndpoints.Set(float64(0))
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
	s.cleanup(true)

	s.lock.RLock()
	defer s.lock.RUnlock()
	if statusBytes, err := yaml.Marshal(&StatusReport); err != nil {
		str = fmt.Sprintf("Error: %v", err)
	} else {
		str = string(statusBytes)
	}
	return
}

// cleanup removes obsolete endpoints.
// If "needsLock" is false, caller MUST have obtained the write lock already.
func (s *StatusReportType) cleanup(needsLock bool) {
	if needsLock {
		s.lock.Lock()
		defer s.lock.Unlock()
	}

	// sometimes endpoints get leftover, even though they are not being monitored anymore - remove them
	validEndpoints := make(map[string]bool, 0)
	for _, podEndpoints := range s.Pods {
		for _, podEndpoint := range podEndpoints {
			validEndpoints[podEndpoint] = true
		}
	}

	for endpoint, _ := range s.Endpoints {
		// if endpoint is prefixed with X|X| it is not a pod endpoint and should never be cleaned up
		if !validEndpoints[endpoint] && !strings.HasPrefix(endpoint, "X|X|") {
			delete(s.Endpoints, endpoint)
		}
	}

	return
}
