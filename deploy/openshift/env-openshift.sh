#!/bin/sh

##############################################################################
# env-openshift.sh
#
# Defines variables used by the scripts to build, start, and stop OpenShift.
##############################################################################

# This is the GOPATH where you want the OpenShift Origin project to go
OPENSHIFT_GOPATH=${HOME}/source/go/openshift

# This is the IP address where OpenShift will bind its master.
# This should be a valid IP address for the machine where OpenShift is installed.
# NOTE: Do not use any IP address within the loopback range of 127.0.0.x.
OPENSHIFT_IP_ADDRESS=192.168.1.2

#
# Below are variables whose values are derived from the user-defined variables above.
# These are not meant for users to change.
#

# This is where the OpenShift Origin github source code will live when building from source.
OPENSHIFT_GITHUB_SOURCE_DIR=${OPENSHIFT_GOPATH}/src/github.com/openshift/origin

# This is where the OpenShift Origin binaries will be after the source is built
OPENSHIFT_BINARY_DIR=${OPENSHIFT_GITHUB_SOURCE_DIR}/_output/local/bin/`go env GOHOSTOS`/`go env GOARCH`
