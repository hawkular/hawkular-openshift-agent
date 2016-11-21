#!/bin/sh

# The OpenShift project where the example Jolokia WildFly endpoint is deployed
OS_PROJECT=${1:-openshift-infra}

echo Will undeploy example Jolokia WildFly endpoint from project $OS_PROJECT

# Login and switch to the target project
oc login
oc project $OS_PROJECT

# Undeploy the example Jolokia WildFly endpoint
oc process -f jolokia-wildfly.yaml | oc delete -f -
oc delete -f jolokia-wildfly-configmap.yaml
