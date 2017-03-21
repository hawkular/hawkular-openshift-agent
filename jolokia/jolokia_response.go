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

package jolokia

import (
	"encoding/json"
	"fmt"

	"github.com/hawkular/hawkular-openshift-agent/log"
)

// USED FOR JSON
type JolokiaResponse struct {
	Status uint32 `json:"status"`

	// populated on success

	Request   map[string]interface{} `json:"request,omitempty"`
	Timestamp uint32                 `json:"timestamp,omitempty"`
	Value     interface{}            `json:"value,omitempty"`

	// populated when an error occurs

	Error      string `json:"error,omitempty"`
	ErrorType  string `json:"error_type,omitempty"`
	Stacktrace string `json:"stacktrace,omitempty"`
}

type JolokiaResponses struct {
	Responses []JolokiaResponse
}

func NewJolokiaResponses() *JolokiaResponses {
	return &JolokiaResponses{
		Responses: make([]JolokiaResponse, 0),
	}
}

func (jrs *JolokiaResponses) String() string {
	bytes, err := json.Marshal(jrs.Responses)
	if err != nil {
		log.Debugf("Failed to marshal response JSON in String(). err=%v", err)
		return fmt.Sprintf("%v", jrs) // can't marshal into JSON, just use sprintf
	}
	return string(bytes)
}

func (jr *JolokiaResponse) IsSuccess() bool {
	return jr.Status == 200
}
