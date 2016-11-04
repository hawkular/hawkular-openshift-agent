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

package http

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
)

func GetHttpClient(identity *security.Identity) (*http.Client, error) {

	// Enable accessing insecure endpoints. We should be able to access metrics from any endpoint
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	if identity != nil && identity.Cert_File != "" {
		cert, err := tls.LoadX509KeyPair(identity.Cert_File, identity.Private_Key_File)
		if err != nil {
			return nil, fmt.Errorf("Error loading the client certificates: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		IdleConnTimeout:       time.Second * 600,
		ResponseHeaderTimeout: time.Second * 600,
	}

	httpClient := http.Client{Transport: transport}

	return &httpClient, nil
}
