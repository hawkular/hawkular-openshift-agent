#!/bin/sh

# The OpenShift project where the example Prometheus Python endpoint is deployed
OS_PROJECT=${1:-openshift-infra}

echo Will undeploy example Prometheus Python endpoint from project $OS_PROJECT

# Login and switch to the target project
oc login
oc project $OS_PROJECT

# Undeploy the example Prometheus Python endpoint
oc process -f prometheus-python.yaml | oc delete -f -
oc delete -f prometheus-python-configmap.yaml
