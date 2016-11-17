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

#-----------------------------------------------------------------------------
# Variables below have values derived from the user-defined variables above.
# These are not meant for users to change.
#-----------------------------------------------------------------------------

# This is where the OpenShift Origin github source code will live when building from source.
OPENSHIFT_GITHUB_SOURCE_DIR=${OPENSHIFT_GOPATH}/src/github.com/openshift/origin

# This is where the OpenShift Origin binaries will be after the source is built
OPENSHIFT_BINARY_DIR=${OPENSHIFT_GITHUB_SOURCE_DIR}/_output/local/bin/`go env GOHOSTOS`/`go env GOARCH`

#
# See if sudo is required. It is required if the user is not in the docker group.
#
if groups ${USER} | grep >/dev/null 2>&1 '\bdocker\b'; then
  DOCKER_SUDO=
else
  DOCKER_SUDO=sudo
fi

# This is the full path to the 'oc' executable
OPENSHIFT_EXE_OC="sudo ${OPENSHIFT_BINARY_DIR}/oc"

# This is the full path to the 'openshift' executable
OPENSHIFT_EXE_OPENSHIFT="sudo ${OPENSHIFT_BINARY_DIR}/openshift"

#
# Make sure the environment is as expected
#

go env > /dev/null 2>&1
if [ "$?" != "0" ]; then
  echo Go is not in your PATH. Aborting.
  exit 1
fi
