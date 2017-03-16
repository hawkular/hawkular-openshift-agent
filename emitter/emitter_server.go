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

package emitter

import (
	"io"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/emitter/health"
	"github.com/hawkular/hawkular-openshift-agent/emitter/metrics"
	"github.com/hawkular/hawkular-openshift-agent/emitter/status"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

func StartEmitter(conf *config.Config) {
	enabled := false

	if conf.Emitter.Metrics_Enabled == "true" {
		enabled = true
		metrics.RegisterMetrics()
		metricsHandler := MetricsHandler{
			credentials: security.Credentials{
				Username: conf.Emitter.Metrics_Credentials.Username,
				Password: conf.Emitter.Metrics_Credentials.Password,
			},
			prometheusHandler: promhttp.Handler(),
		}
		http.HandleFunc("/metrics", metricsHandler.handler)
		log.Info("Agent emitter will emit metrics")
	} else {
		log.Info("Agent emitter will NOT emit metrics")
	}

	if conf.Emitter.Status_Enabled == "true" {
		enabled = true
		statusHandler := StatusHandler{
			credentials: security.Credentials{
				Username: conf.Emitter.Status_Credentials.Username,
				Password: conf.Emitter.Status_Credentials.Password,
			},
		}
		http.HandleFunc("/status", statusHandler.handler)
		log.Info("Agent emitter will emit status")
	} else {
		log.Info("Agent emitter will NOT emit status")
	}

	if conf.Emitter.Health_Enabled == "true" {
		enabled = true
		http.HandleFunc("/health", HealthProbeHandler)
		log.Info("Agent emitter will provide a health probe")
	} else {
		log.Info("Agent emitter will NOT provide a health probe")
	}

	if !enabled {
		log.Info("Agent emitter endpoint has been disabled")
		return
	}

	secure := conf.Identity.Cert_File != "" && conf.Identity.Private_Key_File != ""

	// TODO: For now, never use https. If turn off the emitter endpoints if you don't want them over http.
	//       Delete the line below setting secure=false once we are ok with using the identity with https.
	secure = false

	addr := conf.Emitter.Address

	if addr == "" {
		if secure {
			addr = ":8443"
		} else {
			addr = ":8080"
		}
	}

	server := &http.Server{
		Addr: addr,
	}

	log.Infof("Agent will start the emitter endpoint at [%v]", server.Addr)
	go func() {
		var err error
		if secure {
			err = server.ListenAndServeTLS(conf.Identity.Cert_File, conf.Identity.Private_Key_File)
		} else {
			err = server.ListenAndServe()
		}
		log.Warning(err)
	}()
}

type MetricsHandler struct {
	credentials       security.Credentials
	prometheusHandler http.Handler
}

func (h *MetricsHandler) handler(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusOK

	if h.credentials.Username != "" || h.credentials.Password != "" {
		u, p, ok := r.BasicAuth()
		if !ok {
			statusCode = http.StatusUnauthorized
		} else if h.credentials.Username != u || h.credentials.Password != p {
			statusCode = http.StatusForbidden
		}
	}

	switch statusCode {
	case http.StatusOK:
		{
			h.prometheusHandler.ServeHTTP(w, r)
		}
	case http.StatusUnauthorized:
		{
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Hawkular OpenShift Agent\"")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	default:
		{
			http.Error(w, http.StatusText(statusCode), statusCode)
			log.Errorf("Cannot send metrics response to unauthorized user. %v", statusCode)
		}
	}
}

type StatusHandler struct {
	credentials security.Credentials
}

func (h *StatusHandler) handler(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusOK

	if h.credentials.Username != "" || h.credentials.Password != "" {
		u, p, ok := r.BasicAuth()
		if !ok {
			statusCode = http.StatusUnauthorized
		} else if h.credentials.Username != u || h.credentials.Password != p {
			statusCode = http.StatusForbidden
		}
	} else {
		log.Warning("Access to the status endpoint is not secure. It is recommended you define credentials for the status emitter.")
	}

	switch statusCode {
	case http.StatusOK:
		{
			str := status.StatusReport.Marshal()
			if _, err := io.WriteString(w, str); err != nil {
				log.Errorf("Cannot send status response. err=%v", err)
			}
		}
	case http.StatusUnauthorized:
		{
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Hawkular OpenShift Agent\"")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	default:
		{
			http.Error(w, http.StatusText(statusCode), statusCode)
			log.Errorf("Cannot send status response to unauthorized user. %v", statusCode)
		}
	}
}

func HealthProbeHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(health.HealthStatusCode)
	w.Write([]byte(health.HealthContent))
}
