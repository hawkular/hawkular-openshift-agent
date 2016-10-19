package http

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

func GetHttpClient(certFile string, privateKeyFile string) (*http.Client, error) {

	// Enable accessing insecure endpoints. We should be able to access metrics from any endpoint
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	if certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, privateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("Error loading the client certificates: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	httpClient := http.Client{Transport: transport}

	return &httpClient, nil
}
