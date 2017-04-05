#!/bin/sh

##############################################################################
# deploy-example.sh
#
# Run this script to deploy example pods into your OpenShift node that can
# then be monitored by the Hawkular OpenShift Agent.
#
# The purpose of this script to enable someone to deploy the examples if
# all they have is this script. There is no need to git clone the
# source repository and no need to build anything. OpenShift will download
# the example docker images from docker hub.
#
# If the user has not yet logged into OpenShift via "oc login" then this
# script will log in for you (you will have to enter your credentials
# at the prompts). By default, the example will be deployed in the
# OpenShift user's current project. You can change the project that the
# example is deployed to via the "oc project" command or by setting
# the EXAMPLE_NAMESPACE environment variable when running this script.
#
# To customize this script, these environment variables are used:
#
#   DOCKER_VERSION:
#      The version of the example to deploy.
#      If not specified, the default value is "latest".
#
#   EXAMPLE_NAMESPACE:
#      The namespace (aka OpenShift project) where the example
#      is to be deployed.
#      If not specified, the default is the OpenShift user default.
#
# Examples:
#
# 1. To deploy the latest version of the jolokia-wildfly-example:
#
#   deploy-example.sh jolokia-wildfly-example
#
# 2. To deploy version "1.4.0.Final" of the multiple-endpoints-example
#    into the myproject project:
#
#   DOCKER_VERSION=1.4.0.Final EXAMPLE_NAMESPACE=myproject deploy-example.sh multiple-endpoints-example
#
##############################################################################

if [ "$1" == "" ]; then
  echo Please specify what example you want to install.
  echo For the list, see https://github.com/hawkular/hawkular-openshift-agent/tree/master/examples
  exit 1
fi

# If the example name argument doesn't end with "-example", add it since by convention
# all examples have that suffix. This allows one to run this script by passing the example name
# without having to explicitly type "-example" at the end.
# Note also that the example template yaml file, by convention, is the name minus the "-example" suffix.

if [ "${1: -8}" == "-example" ]; then
  EXAMPLE_NAME="${1}"
  EXAMPLE_YAML="${1:0:-8}.yaml"
else
  EXAMPLE_NAME="${1}-example"
  EXAMPLE_YAML="${1}.yaml"
fi

DOCKER_VERSION=${DOCKER_VERSION:-latest}
EXAMPLE_NAMESPACE=${EXAMPLE_NAMESPACE}

_GIT_REV=${DOCKER_VERSION}
if [ "$_GIT_REV" == "latest" ]; then
  _GIT_REV=master
fi

echo
echo "DOWNLOADING EXAMPLE TEMPLATE (${EXAMPLE_YAML}) AND MAKEFILE FROM GIT HUB (${_GIT_REV})..."
echo

mkdir -p /tmp/hawkular-openshift-agent-examples
cd /tmp/hawkular-openshift-agent-examples
rm -f /tmp/hawkular-openshift-agent-examples/*

wget https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/${_GIT_REV}/examples/${EXAMPLE_NAME}/Makefile || exit 1
wget https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/${_GIT_REV}/examples/${EXAMPLE_NAME}/${EXAMPLE_YAML} || exit 1

# If the user is not logged in yet, log in now

oc whoami > /dev/null 2>&1
if [ "$?" != "0" ]; then
  echo
  echo "LOGGING INTO OPENSHIFT..."
  echo
  oc login
fi

if [ "${EXAMPLE_NAMESPACE}" != "" ]; then
  oc project ${EXAMPLE_NAMESPACE}
fi

echo
echo "DEPLOYING EXAMPLE ${EXAMPLE_NAME} (version=${DOCKER_VERSION}) TO OPENSHIFT (namespace=$(oc project -q))..."
echo

DOCKER_VERSION=${DOCKER_VERSION} make openshift-deploy
