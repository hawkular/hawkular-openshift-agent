#!/bin/sh

# The OpenShift project where the example Jolokia endpoint is to be deployed
OS_PROJECT=${1:-openshift-infra}

echo Will undeploy example Jolokia Wildfly endpoint from project $OS_PROJECT

# Login and switch to the target project
oc login
oc project $OS_PROJECT

# Deploy the example Prometheus endpoint
oc process -f jolokia-wildfly.yaml | oc delete -f -
oc delete -f jolokia-wildfly-configmap.yaml
