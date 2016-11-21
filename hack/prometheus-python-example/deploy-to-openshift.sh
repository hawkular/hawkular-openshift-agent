#!/bin/sh

# The OpenShift project where the example Prometheus Python endpoint is to be deployed
OS_PROJECT=${1:-openshift-infra}

echo Will deploy example Prometheus Python endpoint to project $OS_PROJECT

# Login and switch to the target project
oc login
oc project $OS_PROJECT

# Deploy the example Prometheus Python endpoint
oc process -f prometheus-python.yaml | oc create -f -
oc create -f prometheus-python-configmap.yaml
