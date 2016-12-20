#!/bin/sh

##############################################################################
# undeploy-agent.sh
#
# This removes the agent from OpenShift.
#
# This assumes the agent is deployed in the "openshift-infra" project
# (which is usually where it is), but if it is found in a different
# project, pass the project name to this script as its first argument.
#
# This script requires the "oc" OpenShift client. If it is not in the
# current PATH, it will be assumed it can be found in the source build.
# If there is no source build (i.e. you did not build OpenShift from
# source via the "build-openshift.sh" script) this script will abort.
##############################################################################

# Where the agent is currently deployed
OPENSHIFT_PROJECT=${1:-openshift-infra}

# Find the oc executable
which oc > /dev/null 2>&1
if [ "$?" = "0" ]; then
  OPENSHIFT_OC="sudo $(which oc)"
  echo oc installed: ${OPENSHIFT_OC}
else
  source ./env-openshift.sh
  OPENSHIFT_OC=${OPENSHIFT_EXE_OC}
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

# Switch to the project where the agent is deployed
echo Will undeploy the agent from project ${OPENSHIFT_PROJECT}
${OPENSHIFT_OC} project ${OPENSHIFT_PROJECT}

# Undeploy the agent
${OPENSHIFT_OC} process -f ../deploy/openshift/hawkular-openshift-agent.yaml | ${OPENSHIFT_OC} delete -f -

# Delete the configmap containing the agent global configuration
${OPENSHIFT_OC} delete -f ../deploy/openshift/hawkular-openshift-agent-configmap.yaml
