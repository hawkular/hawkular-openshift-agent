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
)

type LogMessages []string

// Name is the name of the agent.
// Version is the x.y.z version string of the agent.
// Commit_Hash is the git commit hash of the source used to build the agent.
// Pods is keyed on the pod identifier whose value is the endpoint IDs for the pod.
// Endpoints is keyed on an endpoint ID whose value is the last message related to the endpoint.
// Log is a small rolling log of important messages.
// USED FOR YAML
type StatusReportType struct {
	Name        string
	Version     string
	Commit_Hash string
	Pods        map[string][]string
	Endpoints   map[string]string
	Log         LogMessages
	lock        sync.RWMutex
}

var StatusReport = StatusReportType{
	lock: sync.RWMutex{},
}

func (s *StatusReportType) AddLogMessage(m string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.Log) > 1 {
		copy(s.Log, s.Log[1:])
		s.Log[len(s.Log)-1] = fmt.Sprintf("%v: %v", time.Now().Format(time.RFC1123Z), m)
	}
}

func (s *StatusReportType) Marshal() (str string) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if statusBytes, err := yaml.Marshal(&StatusReport); err != nil {
		str = fmt.Sprintf("Error: %v", err)
	} else {
		str = string(statusBytes)
	}
	return
}
