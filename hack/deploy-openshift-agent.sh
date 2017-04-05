#!/bin/sh

##############################################################################
# deploy-openshift-agent.sh
#
# Run this script to deploy the Hawkular OpenShift Agent into your
# OpenShift node.
#
# The purpose of this script to enable someone to deploy the agent if
# all they have is this script. There is no need to git clone the
# source repository and no need to build anything. OpenShift will download
# the agent docker image from docker hub.
#
# To customize this script, these environment variables are used:
#
#   DOCKER_VERSION:
#      The version of the agent to deploy. You can find what versions
#      are available on Docker Hub here:
#         https://hub.docker.com/r/hawkular/hawkular-openshift-agent/tags/
#      If not specified, the default value is "latest".
#
#   HAWKULAR_OPENSHIFT_AGENT_NAMESPACE:
#      The namespace (aka OpenShift project) where the agent is to be
#      deployed. If using ovs_multitenant this must be "default".
#      If not specified, the default value is "default".
#
# Examples:
#
# 1. To deploy the latest version of the agent:
#
#   deploy-openshift-agent.sh
#
# 2. To deploy agent version "1.4.0.Final" to the "openshift-infra" project:
#
#   DOCKER_VERSION=1.4.0.Final HAWKULAR_OPENSHIFT_AGENT_NAMESPACE=openshift-infra deploy-openshift-agent.sh
#
##############################################################################

DOCKER_VERSION=${DOCKER_VERSION:-latest}
HAWKULAR_OPENSHIFT_AGENT_NAMESPACE=${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE:-default}

_GIT_REV=${DOCKER_VERSION}
if [ "$_GIT_REV" == "latest" ]; then
  _GIT_REV=master
fi

echo
echo "DOWNLOADING AGENT OPENSHIFT TEMPLATE FILES AND MAKEFILE FROM GIT HUB (${_GIT_REV})..."
echo

mkdir -p /tmp/hawkular-openshift-agent-deploy
cd /tmp/hawkular-openshift-agent-deploy
rm -rf /tmp/hawkular-openshift-agent-deploy/*

wget https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/${_GIT_REV}/Makefile || exit 1
wget -P deploy/openshift https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/${_GIT_REV}/deploy/openshift/hawkular-openshift-agent-configmap.yaml || exit 1
wget -P deploy/openshift https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/${_GIT_REV}/deploy/openshift/hawkular-openshift-agent.yaml || exit 1
wget -P deploy/openshift https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/${_GIT_REV}/deploy/openshift/hawkular-openshift-agent-route.yaml || exit 1

# If the user is not yet logged in, log in now
oc whoami > /dev/null 2>&1
if [ "$?" != "0" ]; then
  echo
  echo "LOGGING INTO OPENSHIFT..."
  echo
  oc login
fi

echo
echo "DEPLOYING AGENT (version=${DOCKER_VERSION}) TO OPENSHIFT (namespace=${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE})..."
echo

DOCKER_VERSION=${DOCKER_VERSION} HAWKULAR_OPENSHIFT_AGENT_NAMESPACE=${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE} make openshift-deploy
