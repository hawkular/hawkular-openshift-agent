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

func (jr *JolokiaResponse) GetValueAsFloat() float64 {
	return jr.Value.(float64)
}
