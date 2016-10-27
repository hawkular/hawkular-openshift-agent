package jolokia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

const userAgent string = "Hawkular/Hawkular-OpenShift-Agent"
const acceptContentType string = "application/json"

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
func (jrs *JolokiaRequests) SendRequests(url string, credentials *security.Credentials, httpClient *http.Client) (*JolokiaResponses, error) {
	reqBody, err := json.Marshal(jrs.Requests)
	if err != nil {
		return nil, err
	}

	if log.IsTrace() {
		log.Tracef("Sending bulk Jolokia request to [%v] with [%v] individual requests:\n%v", url, len(jrs.Requests), string(reqBody))
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("Cannot create HTTP POST request for Jolokia URL [%v]: err= %v", url, err)
	}

	req.Header.Add("Accept", acceptContentType)
	req.Header.Add("User-Agent", userAgent)

	// Add the auth header if we need one
	headerName, headerValue, err := credentials.GetHttpAuthHeader()
	if err != nil {
		return nil, fmt.Errorf("Cannot create HTTP request auth header for Jolokia URL [%v]: err= %v", url, err)
	}
	if headerName != "" {
		req.Header.Add(headerName, headerValue)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Jolokia request failed. %v/%v", resp.StatusCode, resp.Status)
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
