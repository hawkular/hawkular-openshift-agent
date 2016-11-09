#!/bin/sh

# This is the GOPATH where you want the OpenShift Origin project to go
OPENSHIFT_GOPATH=${HOME}/source/go/openshift

# This is where the OpenShift Origin github source code will live when building from source.
OPENSHIFT_GITHUB_SOURCE_DIR=${OPENSHIFT_GOPATH}/src/github.com/openshift/origin

# This is where the OpenShift Origin binaries will be after the source is built
OPENSHIFT_BINARY_DIR=${OPENSHIFT_GITHUB_SOURCE_DIR}/_output/local/bin/`go env GOHOSTOS`/`go env GOARCH`

# This is the IP address where OpenShift will bind its master
OPENSHIFT_IP_ADDRESS=192.168.1.2
