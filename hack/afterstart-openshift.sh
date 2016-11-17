#!/bin/sh

##############################################################################
# afterstart-openshift.sh
#
# This script is automatically run after OpenShift has been started via
# start-openshift.sh.
#
# This script will complete the setup of OpenShift to make it ready
# for Hawkular OpenShift Agent to be deployed in it.
##############################################################################

source ./env-openshift.sh

echo Will complete the setup of OpenShift that is located here: ${OPENSHIFT_BINARY_DIR}

cd ${OPENSHIFT_BINARY_DIR}

# Log in to allow the oc commands to pass
${OPENSHIFT_EXE_OC} login https://${OPENSHIFT_IP_ADDRESS}:8443 --username=admin --password=admin --insecure-skip-tls-verify=true

# Add the docker registry
sudo chmod +r ${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/admin.kubeconfig
${OPENSHIFT_EXE_OC} adm registry -n default --config=openshift.local.config/master/admin.kubeconfig

# Allow the admin user to see all the internal projects
${OPENSHIFT_EXE_OC} adm --config=${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/admin.kubeconfig policy add-cluster-role-to-user cluster-admin admin
echo Admin user has been given permissions to see all internal projects

# Create service account and the router
${OPENSHIFT_EXE_OC} project default
${OPENSHIFT_EXE_OC} create serviceaccount router -n default
${OPENSHIFT_EXE_OC} adm policy add-scc-to-user privileged system:serviceaccount:default:router
${OPENSHIFT_EXE_OC} adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:default:router
sudo chmod +r ${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/openshift-router.kubeconfig
${OPENSHIFT_EXE_OC} adm router example.com --credentials="${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/openshift-router.kubeconfig" --service-account=router
${OPENSHIFT_EXE_OC} project openshift-infra
echo Router has been created

# Install Hawkular-Metrics

# Grab the metrics.yaml from github and put it in /tmp
wget -O /tmp/metrics.yaml https://raw.githubusercontent.com/openshift/origin-metrics/master/metrics.yaml
echo Downloaded metrics.yaml and placed it in /tmp

# Create the service account
${OPENSHIFT_EXE_OC} create -f - <<API
apiVersion: v1
kind: ServiceAccount
metadata:
  name: metrics-deployer
secrets:
- name: metrics-deployer
API
echo Created metrics deployer service account

# Set up everything and create Hawkular-Metrics deployment
${OPENSHIFT_EXE_OC} adm policy add-role-to-user edit system:serviceaccount:openshift-infra:metrics-deployer
${OPENSHIFT_EXE_OC} adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:openshift-infra:heapster
${OPENSHIFT_EXE_OC} secrets new metrics-deployer nothing=/dev/null
${OPENSHIFT_EXE_OC} policy add-role-to-user view system:serviceaccount:openshift-infra:hawkular -n openshift-infra
${OPENSHIFT_EXE_OC} process -f /tmp/metrics.yaml -v HAWKULAR_METRICS_HOSTNAME=hawkular-metrics.example.com -v USE_PERSISTENT_STORAGE=false -v MODE=redeploy | ${OPENSHIFT_EXE_OC} create -f -
echo Hawkular Metrics should be spinning up now. Please be patient, it could take several minutes for it to be ready.
