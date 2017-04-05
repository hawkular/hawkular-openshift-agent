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

echo
echo "DOWNLOADING AGENT OPENSHIFT TEMPLATE FILES..."
echo

mkdir -p /tmp/hawkular-openshift-agent-deploy
cd /tmp/hawkular-openshift-agent-deploy
rm /tmp/hawkular-openshift-agent-deploy/*.yaml

wget https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/master/deploy/openshift/hawkular-openshift-agent-configmap.yaml
wget https://raw.githubusercontent.com/hawkular/hawkular-openshift-agent/master/deploy/openshift/hawkular-openshift-agent.yaml

echo
echo "LOGGING INTO OPENSHIFT..."
echo

oc login

echo
echo "UNDEPLOYING ANY PREVIOUSLY DEPLOYED AGENT FROM OPENSHIFT..."
echo

oc delete all,secrets,sa,templates,configmaps,daemonsets,clusterroles --selector=metrics-infra=agent -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}
oc delete clusterroles hawkular-openshift-agent

echo
echo "DEPLOYING AGENT (version=${DOCKER_VERSION}) TO OPENSHIFT (namespace=${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE})..."
echo

oc create -f hawkular-openshift-agent-configmap.yaml -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}
oc process -f hawkular-openshift-agent.yaml -v IMAGE_VERSION=${DOCKER_VERSION} | oc create -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE} -f -
oc adm policy add-cluster-role-to-user hawkular-openshift-agent system:serviceaccount:${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}:hawkular-openshift-agent
