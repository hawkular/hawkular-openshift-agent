/*
   Copyright 2017 Red Hat, Inc. and/or its affiliates
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

package json

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

const userAgent string = "Hawkular/Hawkular-OpenShift-Agent"

func Scrape(url string, credentials *security.Credentials, client *http.Client) (map[string]interface{}, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Cannot create HTTP GET request for JSON URL [%v]: err= %v", url, err)
	}

	req.Header.Add("User-Agent", userAgent)

	// Add the auth header if we need one
	headerName, headerValue, err := credentials.GetHttpAuthHeader()
	if err != nil {
		return nil, fmt.Errorf("Cannot create HTTP GET request auth header for JSON URL [%v]: err= %v", url, err)
	}
	if headerName != "" {
		req.Header.Add(headerName, headerValue)
	}

	// Submit the request to read the URL
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cannot scrape JSON URL [%v]: err=%v", url, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JSON URL [%v] returned error status [%v/%v]", url, resp.StatusCode, resp.Status)
	}

	log.Tracef("JSON URL [%v] returned data [%v/%v]", url, resp.StatusCode, resp.Status)

	// Perform ad-hoc JSON unmarshalling
	decoder := json.NewDecoder(resp.Body)
	var unmarshalledData interface{}
	if err = decoder.Decode(&unmarshalledData); err != nil {
		return nil, fmt.Errorf("Cannot unmarshal JSON data from URL [%v]: err=%v", url, err)
	}

	if d, ok := unmarshalledData.(map[string]interface{}); ok {
		return d, nil
	} else {
		return nil, fmt.Errorf("Cannot unmarshal JSON data from URL [%v]", url)
	}
}
