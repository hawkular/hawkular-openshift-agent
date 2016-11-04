/*
   Copyright 2016 Red Hat, Inc. and/or its affiliates
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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
	hawkhttp "github.com/hawkular/hawkular-openshift-agent/http"
)

func TestParseMetricName(t *testing.T) {
	name1 := "java.lang:type=Threading#ThreadCount"
	name2 := "java.lang:type=Memory#HeapMemoryUsage#init"
	name3 := "java.lang:type=Memory" // BAD - not enough parts

	req := &JolokiaRequest{}
	if err := ParseMetricName(name1, req); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if req.MBean != "java.lang:type=Threading" || req.Attribute != "ThreadCount" || req.Path != "" {
		t.Fatalf("Failed to parse [%v]=%v", name1, req)
	}

	req = &JolokiaRequest{}
	if err := ParseMetricName(name2, req); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if req.MBean != "java.lang:type=Memory" || req.Attribute != "HeapMemoryUsage" || req.Path != "init" {
		t.Fatalf("Failed to parse [%v]=%v", name2, req)
	}

	req = &JolokiaRequest{}
	if err := ParseMetricName(name3, req); err == nil {
		t.Fatalf("Parse failed: %v", err)
	}
}

func TestJsonRequests(t *testing.T) {
	reqs := NewJolokiaRequests()

	reqs.AddRequest(JolokiaRequest{
		Type:      RequestTypeRead,
		MBean:     "java.lang:type=Memory",
		Attribute: "HeapMemoryUsage",
		Path:      "used",
	})

	t.Logf("REQUESTS JSON 1==>%v", reqs)

	if reqs.String() != `[{"type":"read","mbean":"java.lang:type=Memory","attribute":"HeapMemoryUsage","path":"used"}]` {
		t.Fatal("Failed to marshal JSON requests")
	}

	reqs.AddRequest(JolokiaRequest{
		Type:      RequestTypeRead,
		MBean:     "a:b=c",
		Attribute: "d",
	})

	t.Logf("REQUESTS JSON 2==>%v", reqs)

	if reqs.String() != `[{"type":"read","mbean":"java.lang:type=Memory","attribute":"HeapMemoryUsage","path":"used"},{"type":"read","mbean":"a:b=c","attribute":"d"}]` {
		t.Fatal("Failed to marshal JSON requests")
	}
}

func TestJsonResponses(t *testing.T) {
	resps := NewJolokiaResponses()

	resp := JolokiaResponse{
		Status:    200,
		Timestamp: 123456,
		Value:     987,
	}
	resps.Responses = append(resps.Responses, resp)

	t.Logf("RESPONSES JSON 1==>%v", resps)
	if resps.String() != `[{"status":200,"timestamp":123456,"value":987}]` {
		t.Fatal("Failed to marshal JSON responses")
	}

	resp = JolokiaResponse{
		Status:    404,
		Error:     "The error string",
		ErrorType: "The error type string",
	}
	resps.Responses = append(resps.Responses, resp)

	t.Logf("RESPONSES JSON 2==>%v", resps)
	if resps.String() != `[{"status":200,"timestamp":123456,"value":987},{"status":404,"error":"The error string","error_type":"The error type string"}]` {
		t.Fatal("Failed to marshal JSON responses")
	}
}

func TestJolokia(t *testing.T) {

	// setup our mock jolokia server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponseJson := `[{"request":{"path":"used","mbean":"java.lang:type=Memory","attribute":"HeapMemoryUsage","type":"read"},"value":123,"timestamp":123456,"status":200},{"request":{"mbean":"A:B=C","attribute":"D","type":"read"},"stacktrace":"stack trace would be here","error_type":"javax.management.InstanceNotFoundException","error":"javax.management.InstanceNotFoundException : A:B=C","status":404}]`
		fmt.Fprintln(w, mockResponseJson)
	}))
	defer ts.Close()

	httpClient, err := hawkhttp.GetHttpClient(nil)

	reqs := NewJolokiaRequests()
	reqs.AddRequest(JolokiaRequest{
		Type:      RequestTypeRead,
		MBean:     "java.lang:type=Memory",
		Attribute: "HeapMemoryUsage",
		Path:      "used",
	})
	reqs.AddRequest(JolokiaRequest{
		Type:      RequestTypeRead,
		MBean:     "A:B=C",
		Attribute: "D",
	})

	resp, err := reqs.SendRequests(ts.URL, &security.Credentials{}, httpClient)
	if err != nil {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	t.Logf("MOCK RESPONSES====>%v", resp)

	if len(resp.Responses) != 2 {
		t.Fatalf("Failed to send Jolokia requests to mock server: err=%v", err)
	}

	if !resp.Responses[0].IsSuccess() {
		t.Fatalf("First response should have been success (200). %v", resp)
	}

	if resp.Responses[1].IsSuccess() {
		t.Fatalf("First response should have been failure (404). %v", resp)
	}

	if resp.Responses[0].Status != 200 || resp.Responses[0].Error != "" || resp.Responses[0].GetValueAsFloat() != 123 || resp.Responses[0].Timestamp != 123456 {
		t.Fatalf("First response had unexpected data. %v", resp)
	}

	if resp.Responses[1].Status != 404 || resp.Responses[1].Error == "" || resp.Responses[1].Value != nil || resp.Responses[1].Timestamp != 0 {
		t.Fatalf("Second response had unexpected data. %v", resp)
	}
}
