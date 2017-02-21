#!/bin/sh

##############################################################################
# build-openshiftmetrics.sh
#
# This will download the OpenShift Origin-Metrics source code and build it.
##############################################################################

# Before we do anything, make sure all software prerequisites are available.

WHERE_IS_MAKE=`which make`
if [ "$?" = "0" ]; then
  echo Make installed: $WHERE_IS_MAKE
else
  echo You must install Make
  exit 1
fi

WHERE_IS_GO=`which go`
if [ "$?" = "0" ]; then
  echo Go installed: $WHERE_IS_GO
else
  echo You must install the Go Programming Language.
  exit 1
fi

WHERE_IS_GIT=`which git`
if [ "$?" = "0" ]; then
  echo Git installed: $WHERE_IS_GIT
else
  echo You must install Git.
  exit 1
fi

WHERE_IS_DOCKER=`which docker`
if [ "$?" = "0" ]; then
  echo Docker installed: $WHERE_IS_DOCKER
else
  echo You must install Docker.
  exit 1
fi

# Software prerequisites have been met so we can continue.

source ./env-openshift.sh

echo Will build OpenShift Origin-Metrics here: ${OPENSHIFTMETRICS_GITHUB_SOURCE_DIR}

if [ ! -d "${OPENSHIFTMETRICS_GITHUB_SOURCE_DIR}" ]; then
  echo The OpenShift Origin-Metrics source code repository has not been cloned yet - doing that now.

  # Create the location where the source code will live.

  PARENT_DIR=`dirname ${OPENSHIFTMETRICS_GITHUB_SOURCE_DIR}`
  mkdir -p ${PARENT_DIR}

  if [ ! -d "${PARENT_DIR}" ]; then
    echo Aborting. Cannot create the parent source directory: ${PARENT_DIR}
    exit 1
  fi

  # Clone the OpenShift Origin-Metrics source code repository via git.

  cd ${PARENT_DIR}
  git clone git@github.com:openshift/origin-metrics.git
else
  echo The OpenShift Origin-Metrics source code repository exists - it will be updated now.
fi

# Build OpenShift Origin-Metrics.

cd ${OPENSHIFTMETRICS_GITHUB_SOURCE_DIR}
git pull

export GOPATH=${OPENSHIFTMETRICS_GOPATH}
make

if [ "$?" = "0" ]; then
  echo OpenShift Origin-Metrics build is complete!
fi
