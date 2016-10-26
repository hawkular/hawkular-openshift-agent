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
