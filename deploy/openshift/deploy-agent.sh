#!/bin/sh

##############################################################################
# deploy-agent.sh
#
# This deploys the agent into OpenShift.
#
# This assumes the agent is to be deployed in the "openshift-infra" project
# (which is usually what you want), but if you want it in a different
# project, pass the project name to this script as its first argument.
#
# This script requires the "oc" OpenShift client. If it is not in the
# current PATH, it will be assumed it can be found in the source build.
# If there is no source build (i.e. you did not build OpenShift from
# source via the "build-openshift.sh" script) this script will abort.
##############################################################################

# Origin Metrics must already be installed in this project. The agent must go in this same project.
OPENSHIFT_PROJECT=${1:-openshift-infra}

# Find the oc executable
which oc > /dev/null 2>&1
if [ "$?" = "0" ]; then
  OPENSHIFT_OC=`which oc`
  echo oc installed: ${OPENSHIFT_OC}
else
  source ./env-openshift.sh
  OPENSHIFT_OC=${OPENSHIFT_BINARY_DIR}/oc
  echo Using oc from the source build: ${OPENSHIFT_OC}
  ${OPENSHIFT_OC} version > /dev/null 2>&1
  if [ "$?" != "0" ]; then
    echo There is no available oc executable. Aborting.
    exit 1
  fi
fi

# Log into OpenShift
echo Log into OpenShift
${OPENSHIFT_OC} login

# Switch to the project where the agent is to be deployed
echo Will deploy the agent in project ${OPENSHIFT_PROJECT}
${OPENSHIFT_OC} project ${OPENSHIFT_PROJECT}

# Create the agent service account
${OPENSHIFT_OC} create -f - <<API
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hawkular-agent
API

# Grant security permisions to the agent
${OPENSHIFT_OC} adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:openshift-infra:hawkular-agent

# Create the configmap containing the agent global configuration
${OPENSHIFT_OC} create -f hawkular-openshift-agent-configmap.yaml

# Deploy the agent
${OPENSHIFT_OC} process -f hawkular-openshift-agent.yaml | ${OPENSHIFT_OC} create -f -
