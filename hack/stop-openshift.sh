#!/bin/sh

##############################################################################
# stop-openshift.sh
#
# Run this script when you want to fully and completely stop OpenShift.
##############################################################################

source ./env-openshift.sh

echo Will stop OpenShift that is located here: ${OPENSHIFT_BINARY_DIR}

# Remove the convienence soft links that were created by the start script
rm openshift-dir
rm ca.crt

# Go to where the OpenShift build is
cd ${OPENSHIFT_BINARY_DIR}

# Shut things down
sudo pkill -x openshift
${DOCKER_SUDO} docker ps | awk 'index($NF,"k8s_")==1 { print $1 }' | xargs -l -r ${DOCKER_SUDO} docker stop
mount | grep "openshift.local.volumes" | awk '{ print $3}' | xargs -l -r sudo umount
sudo rm -rf ${OPENSHIFT_BINARY_DIR}/openshift.local.*
