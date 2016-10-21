package jolokia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hawkular/hawkular-openshift-agent/log"
)

const jsonContentType string = "application/json"

// today we only support read
type RequestType string

const (
	RequestTypeRead RequestType = "read"
)

// USED FOR JSON
type JolokiaRequest struct {
	Type      RequestType `json:"type"`
	MBean     string      `json:"mbean,omitempty"`
	Attribute string      `json:"attribute,omitempty"`
	Path      string      `json:"path,omitempty"`
}

type JolokiaRequests struct {
	Requests []JolokiaRequest
}

func NewJolokiaRequests() *JolokiaRequests {
	return &JolokiaRequests{
		Requests: make([]JolokiaRequest, 0),
	}
}

func (jrs *JolokiaRequests) String() string {
	bytes, err := json.Marshal(jrs.Requests)
	if err != nil {
		log.Debugf("Failed to marshal request JSON in String(). err=%v", err)
		return fmt.Sprintf("%v", jrs) // can't marshal into JSON, just use sprintf
	}
	return string(bytes)
}

func (jrs *JolokiaRequests) AddRequest(jr JolokiaRequest) {
	jrs.Requests = append(jrs.Requests, jr)
}

// SendRequests will send all the requests in a bulk request message to given Jolokia endpoint URL and return the responses in
// a bulk JolokiaResponses object.
func (jrs *JolokiaRequests) SendRequests(url string, httpClient *http.Client) (*JolokiaResponses, error) {
	reqBody, err := json.Marshal(jrs.Requests)
	if err != nil {
		return nil, err
	}

	if log.IsTrace() {
		log.Tracef("Sending bulk Jolokia request to [%v] with [%v] individual requests:\n%v", url, len(jrs.Requests), string(reqBody))
	}

	resp, err := httpClient.Post(url, jsonContentType, bytes.NewBuffer(reqBody))
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	respBuffer := new(bytes.Buffer)
	respBuffer.ReadFrom(resp.Body)

	respObject := NewJolokiaResponses()
	err = json.Unmarshal(respBuffer.Bytes(), &respObject.Responses)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse Jolokia response. err=%v", err)
	}

	log.Tracef("Received Jolokia response:\n%v", respObject)

	return respObject, nil
}
