#!/bin/sh

# Origin Metrics must already be installed in this project. The agent must go in this same project.
OS_PROJECT=${1:-openshift-infra}

# Log into OpenShift
echo Log into OpenShift
oc login

# Switch to the project where the agent is to be deployed
echo Will deploy the agent in project $OS_PROJECT
oc project $OS_PROJECT

# Create the agent service account
oc create -f - <<API
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hawkular-agent
API

# Grant security permisions to the agent
oc adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:openshift-infra:hawkular-agent

# Create the configmap containing the agent global configuration
oc create -f hawkular-openshift-agent-configmap.yaml

# Deploy the agent
oc process -f hawkular-openshift-agent.yaml | oc create -f -
