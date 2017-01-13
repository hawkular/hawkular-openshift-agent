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

package prometheus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

const userAgent string = "Hawkular/Hawkular-OpenShift-Agent"
const acceptContentType string = "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.7,text/plain;version=0.0.4;q=0.3"

func Scrape(url string, credentials *security.Credentials, client *http.Client) (mf map[string]*dto.MetricFamily, err error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Cannot create HTTP GET request for Prometheus URL [%v]: err= %v", url, err)
	}

	req.Header.Add("Accept", acceptContentType)
	req.Header.Add("User-Agent", userAgent)

	// Add the auth header if we need one
	headerName, headerValue, err := credentials.GetHttpAuthHeader()
	if err != nil {
		return nil, fmt.Errorf("Cannot create HTTP GET request auth header for Prometheus URL [%v]: err= %v", url, err)
	}
	if headerName != "" {
		req.Header.Add(headerName, headerValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cannot scrape Prometheus URL [%v]: err=%v", url, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Prometheus URL [%v] returned error status [%v/%v]", url, resp.StatusCode, resp.Status)
	}

	mediatype, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err == nil &&
		mediatype == "application/vnd.google.protobuf" &&
		params["encoding"] == "delimited" &&
		params["proto"] == "io.prometheus.client.MetricFamily" {
		return ScrapeBinary(resp.Body)
	} else {
		// note: even if ParseMediaType returned an error, always fall back to text parsing
		return ScrapeText(resp.Body)
	}
}

func ScrapeText(r io.Reader) (map[string]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(r)
}

func ScrapeBinary(r io.Reader) (mf map[string]*dto.MetricFamily, err error) {
	mf = make(map[string]*dto.MetricFamily)
	for {
		fam := &dto.MetricFamily{}
		if err = scrapeBinaryOneFamily(r, fam); err != nil {
			if err == io.EOF {
				break
			}
			return
		}
		mf[*fam.Name] = fam
	}
	return mf, nil
}

func scrapeBinaryOneFamily(r io.Reader, m *dto.MetricFamily) error {
	var headerArray [binary.MaxVarintLen32]byte
	var varintBytes int
	var messageLength uint64
	var totalBytesRead int

	for varintBytes == 0 {
		if totalBytesRead >= len(headerArray) {
			log.Warning("Prometheus endpoint appears to be exporting invalid data")
			return errors.New("invalid number of bytes read")
		}

		curBytesRead, err := r.Read(headerArray[totalBytesRead : totalBytesRead+1])
		if curBytesRead == 0 {
			if err != nil {
				return err
			}
			continue
		}

		totalBytesRead += curBytesRead
		messageLength, varintBytes = proto.DecodeVarint(headerArray[:totalBytesRead])
	}

	messageByteArray := make([]byte, messageLength)
	curBytesRead, err := io.ReadFull(r, messageByteArray)
	totalBytesRead += curBytesRead
	if err != nil {
		return err
	}

	return proto.Unmarshal(messageByteArray, m)
}
