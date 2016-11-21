#!/bin/sh

# The OpenShift project where the example Jolokia endpoint is to be deployed
OS_PROJECT=${1:-openshift-infra}

echo Will deploy example Jolokia WildFly endpoint to project $OS_PROJECT

# Login and switch to the target project
oc login
oc project $OS_PROJECT

# Deploy the example Jolokia WildFly endpoint
oc process -f jolokia-wildfly.yaml | oc create -f -
oc create -f jolokia-wildfly-configmap.yaml
