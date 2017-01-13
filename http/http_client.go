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

package http

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
)

type HttpClientConfig struct {
	Identity      *security.Identity
	TLSConfig     *tls.Config
	HTTPTransport *http.Transport
}

func (conf *HttpClientConfig) BuildHttpClient() (*http.Client, error) {

	// make our own copy of TLS config
	tlsConfig := &tls.Config{}
	if conf.TLSConfig != nil {
		*tlsConfig = *conf.TLSConfig
	}

	if conf.Identity != nil && conf.Identity.Cert_File != "" {
		cert, err := tls.LoadX509KeyPair(conf.Identity.Cert_File, conf.Identity.Private_Key_File)
		if err != nil {
			return nil, fmt.Errorf("Error loading the client certificates: %v", err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	// make our own copy of HTTP transport
	transport := &http.Transport{}
	if conf.HTTPTransport != nil {
		*transport = *conf.HTTPTransport
	}

	// make sure the transport has some things we know we need
	transport.TLSClientConfig = tlsConfig

	if transport.IdleConnTimeout == 0 {
		transport.IdleConnTimeout = time.Second * 600
	}
	if transport.ResponseHeaderTimeout == 0 {
		transport.ResponseHeaderTimeout = time.Second * 600
	}

	// build the http client
	httpClient := http.Client{Transport: transport}

	return &httpClient, nil
}
