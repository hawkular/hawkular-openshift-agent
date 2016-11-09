#!/bin/sh

# Where the agent is currently deployed
OS_PROJECT=${1:-openshift-infra}

# Log into OpenShift
echo Log into OpenShift
oc login

# Switch to the project where the agent is deployed
echo Will undeploy the agent from project $OS_PROJECT
oc project $OS_PROJECT

# Undeploy the agent
oc process -f hawkular-openshift-agent.yaml | oc delete -f -

# Delete the configmap containing the agent global configuration
oc delete -f hawkular-openshift-agent-configmap.yaml

# Delete the agent service account
oc delete serviceaccount hawkular-agent
