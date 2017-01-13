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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hawkular/hawkular-openshift-agent/config/security"
	"github.com/hawkular/hawkular-openshift-agent/jolokia"
	"github.com/hawkular/hawkular-openshift-agent/prometheus"
)

const (
	authorizedUsername = "theUser"
	authorizedPassword = "thePassword"
	authorizedToken    = "theToken"
)

const (
	testHostname = "127.0.0.1"
)

var tmpDir = os.TempDir()

func TestSecureComm(t *testing.T) {
	testPort, err := getFreePort(testHostname)
	if err != nil {
		t.Fatalf("Cannot get a free port to run tests on host [%v]", testHostname)
	} else {
		t.Logf("Will use free port [%v] on host [%v] for tests", testPort, testHostname)
	}

	testServerCertFile := tmpDir + "/http-client-test_server.cert"
	testServerKeyFile := tmpDir + "/http-client-test-server.key"
	testServerHostPort := fmt.Sprintf("%v:%v", testHostname, testPort)
	err = generateCertificate(t, testServerCertFile, testServerKeyFile, testServerHostPort)
	if err != nil {
		t.Fatalf("Failed to create server cert/key files: %v", err)
	}
	defer os.Remove(testServerCertFile)
	defer os.Remove(testServerKeyFile)

	testClientCertFile := tmpDir + "/http-client-test_client.cert"
	testClientKeyFile := tmpDir + "/http-client-test-client.key"
	testClientHost := testHostname
	err = generateCertificate(t, testClientCertFile, testClientKeyFile, testClientHost)
	if err != nil {
		t.Fatalf("Failed to create client cert/key files: %v", err)
	}
	defer os.Remove(testClientCertFile)
	defer os.Remove(testClientKeyFile)

	http.HandleFunc("/prometheus-basic", handlerPrometheusBasic)
	http.HandleFunc("/prometheus-bearer", handlerPrometheusBearer)
	http.HandleFunc("/jolokia-basic", handlerJolokiaBasic)
	http.HandleFunc("/jolokia-bearer", handlerJolokiaBearer)
	go http.ListenAndServeTLS(testServerHostPort, testServerCertFile, testServerKeyFile, nil)
	t.Logf("Started test http server: https://%v", testServerHostPort)

	// the client
	httpConfig := HttpClientConfig{
		Identity: &security.Identity{
			Cert_File:        testClientCertFile,
			Private_Key_File: testClientKeyFile,
		},
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	httpClient, err := httpConfig.BuildHttpClient()
	if err != nil {
		t.Fatalf("Failed to create http client")
	}

	// the good basic credentials
	basicCredentials := &security.Credentials{
		Username: authorizedUsername,
		Password: authorizedPassword,
	}

	// the good bearer token credentials
	bearerTokenCredentials := &security.Credentials{
		Token: authorizedToken,
	}

	// bad basic credentials
	badBasicCredentials := &security.Credentials{
		Username: "invalid username",
		Password: "invalid password",
	}

	// bad bearer token credentials
	badBearerTokenCredentials := &security.Credentials{
		Token: "invalid token",
	}

	// no credentials
	noCredentials := &security.Credentials{}

	// a valid jolokia request
	reqs := jolokia.NewJolokiaRequests()
	reqs.AddRequest(jolokia.JolokiaRequest{
		Type:      jolokia.RequestTypeRead,
		MBean:     "java.lang:type=Memory",
		Attribute: "HeapMemoryUsage",
		Path:      "used",
	})

	promBasicUrl := fmt.Sprintf("https://%v/prometheus-basic", testServerHostPort)
	promBearerUrl := fmt.Sprintf("https://%v/prometheus-bearer", testServerHostPort)
	jolokiaBasicUrl := fmt.Sprintf("https://%v/jolokia-basic", testServerHostPort)
	jolokiaBearerUrl := fmt.Sprintf("https://%v/jolokia-bearer", testServerHostPort)

	// wait for our test http server to come up
	checkHttpReady(httpClient, promBasicUrl)

	// TEST WITH AN AUTHORIZED USER

	if _, err = prometheus.Scrape(promBasicUrl, basicCredentials, httpClient); err != nil {
		t.Fatalf("Failed: Prometheus/Basic Auth: %v", err)
	}

	if _, err = prometheus.Scrape(promBearerUrl, bearerTokenCredentials, httpClient); err != nil {
		t.Fatalf("Failed: Prometheus/Bearer Auth: %v", err)
	}

	if _, err = reqs.SendRequests(jolokiaBasicUrl, basicCredentials, httpClient); err != nil {
		t.Fatalf("Failed: Jolokia/Basic Auth: %v", err)
	}

	if _, err = reqs.SendRequests(jolokiaBearerUrl, bearerTokenCredentials, httpClient); err != nil {
		t.Fatalf("Failed: Jolokia/Bearer Auth: %v", err)
	}

	// TEST WITH AN INVALID USER

	if _, err = prometheus.Scrape(promBasicUrl, badBasicCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Prometheus/Basic Auth should have failed with 401")
	}

	if _, err = prometheus.Scrape(promBearerUrl, badBearerTokenCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Prometheus/Bearer Auth should have failed with 401")
	}

	if _, err = reqs.SendRequests(jolokiaBasicUrl, badBasicCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Jolokia/Basic Auth should have failed with 401")
	}

	if _, err = reqs.SendRequests(jolokiaBearerUrl, badBearerTokenCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Jolokia/Bearer Auth should have failed with 401")
	}

	// TEST WITH NO USER

	if _, err = prometheus.Scrape(promBasicUrl, noCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Prometheus/Basic Auth should have failed with 401 with no credentials")
	}

	if _, err = prometheus.Scrape(promBearerUrl, noCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Prometheus/Bearer Auth should have failed with 401 with no credentials")
	}

	if _, err = reqs.SendRequests(jolokiaBasicUrl, noCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Jolokia/Basic Auth should have failed with 401 with no credentials")
	}

	if _, err = reqs.SendRequests(jolokiaBearerUrl, noCredentials, httpClient); err == nil {
		t.Fatalf("Failed: Jolokia/Bearer Auth should have failed with 401 with no credentials")
	}

}

func handlerPrometheusBasic(w http.ResponseWriter, r *http.Request) {
	err := verifyBasicAuth(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		writePrometheusResponse(w)
	}
}

func handlerPrometheusBearer(w http.ResponseWriter, r *http.Request) {
	err := verifyBearerAuth(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		writePrometheusResponse(w)
	}
}

func handlerJolokiaBasic(w http.ResponseWriter, r *http.Request) {
	err := verifyBasicAuth(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		writeJolokiaResponse(w)
	}
}

func handlerJolokiaBearer(w http.ResponseWriter, r *http.Request) {
	err := verifyBearerAuth(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		writeJolokiaResponse(w)
	}
}

func verifyBasicAuth(r *http.Request) error {
	user, pass, ok := r.BasicAuth()
	if !ok || user != authorizedUsername || pass != authorizedPassword {
		return fmt.Errorf("Bad basic auth")
	}
	return nil
}

func verifyBearerAuth(r *http.Request) error {
	header := r.Header.Get("Authorization")
	authHeader := strings.SplitN(header, " ", 2)

	if len(authHeader) < 2 || authHeader[0] != "Bearer" || authHeader[1] != authorizedToken {
		return fmt.Errorf("Bad bearer auth")
	}
	return nil
}

func writePrometheusResponse(w http.ResponseWriter) {
	mockResponse := `#
# HELP http_requests_total Total number of HTTP requests made.
# TYPE http_requests_total counter
http_requests_total{code="200",handler="prometheus",method="get"} 162030`
	fmt.Fprintln(w, mockResponse)
}

func writeJolokiaResponse(w http.ResponseWriter) {
	mockResponse := `[{"request":{"path":"used","mbean":"java.lang:type=Memory","attribute":"HeapMemoryUsage","type":"read"},"value":123,"timestamp":123456,"status":200}]`
	fmt.Fprintln(w, mockResponse)
}

func generateCertificate(t *testing.T, certPath string, keyPath string, host string) error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		Subject: pkix.Name{
			Organization: []string{"ABC Corp."},
		},
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	pemBlockForKey := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	pem.Encode(keyOut, pemBlockForKey)
	keyOut.Close()

	t.Logf("Generated security data: %v|%v|%v", certPath, keyPath, host)
	return nil
}

func getFreePort(host string) (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", host+":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func checkHttpReady(httpClient *http.Client, url string) {
	for i := 0; i < 60; i++ {
		if r, err := httpClient.Get(url); err == nil {
			r.Body.Close()
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}
