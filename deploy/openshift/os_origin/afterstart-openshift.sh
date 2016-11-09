#!/bin/sh

source ./env-openshift.sh

echo Will complete the setup of OpenShift that is located here: ${OPENSHIFT_BINARY_DIR}

cd ${OPENSHIFT_BINARY_DIR}

# Log in to allow the oc commands to pass
${OPENSHIFT_BINARY_DIR}/oc login https://${OPENSHIFT_IP_ADDRESS}:8443 --username=admin --password=admin

# Add the docker registry
sudo chmod +r ${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/admin.kubeconfig
${OPENSHIFT_BINARY_DIR}/oadm registry -n default --config=openshift.local.config/master/admin.kubeconfig

# Allow the admin user to see all the internal projects
${OPENSHIFT_BINARY_DIR}/oadm --config=${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/admin.kubeconfig policy add-cluster-role-to-user cluster-admin admin
echo Admin user has been given permissions to see all internal projects

# Create service account and the router
${OPENSHIFT_BINARY_DIR}/oc project default
${OPENSHIFT_BINARY_DIR}/oc create serviceaccount router -n default
${OPENSHIFT_BINARY_DIR}/oadm policy add-scc-to-user privileged system:serviceaccount:default:router
${OPENSHIFT_BINARY_DIR}/oadm policy add-cluster-role-to-user cluster-reader system:serviceaccount:default:router
sudo chmod +r ${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/openshift-router.kubeconfig
${OPENSHIFT_BINARY_DIR}/oadm router example.com --credentials="${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/openshift-router.kubeconfig" --service-account=router
${OPENSHIFT_BINARY_DIR}/oc project openshift-infra
echo Router has been created

# Install Hawkular-Metrics

# Grab the metrics.yaml from github and put it in /tmp
wget -O /tmp/metrics.yaml https://raw.githubusercontent.com/openshift/origin-metrics/master/metrics.yaml
echo Downloaded metrics.yaml and placed it in /tmp

# Create the service account
${OPENSHIFT_BINARY_DIR}/oc create -f - <<API
apiVersion: v1
kind: ServiceAccount
metadata:
  name: metrics-deployer
secrets:
- name: metrics-deployer
API
echo Created metrics deployer service account

# Set up everything and create Hawkular-Metrics deployment
${OPENSHIFT_BINARY_DIR}/oadm policy add-role-to-user edit system:serviceaccount:openshift-infra:metrics-deployer
${OPENSHIFT_BINARY_DIR}/oadm policy add-cluster-role-to-user cluster-reader system:serviceaccount:openshift-infra:heapster
${OPENSHIFT_BINARY_DIR}/oc secrets new metrics-deployer nothing=/dev/null
${OPENSHIFT_BINARY_DIR}/oc policy add-role-to-user view system:serviceaccount:openshift-infra:hawkular -n openshift-infra
${OPENSHIFT_BINARY_DIR}/oc process -f /tmp/metrics.yaml -v HAWKULAR_METRICS_HOSTNAME=hawkular-metrics.example.com -v USE_PERSISTENT_STORAGE=false -v MODE=redeploy | ${OPENSHIFT_BINARY_DIR}/oc create -f -
echo Hawkular Metrics should be spinning up now.
