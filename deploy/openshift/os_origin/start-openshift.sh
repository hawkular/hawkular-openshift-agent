#!/bin/sh

source ./env-openshift.sh

echo Will start OpenShift that is located here: ${OPENSHIFT_BINARY_DIR}

# There must be an IP assigned to the Hawkular Metrics URL - make sure of this.
echo Finding the IP for hawkular-metrics.example.com
gethostip -d hawkular-metrics.example.com
if [ $? != 0 ]; then
   echo Put a 127.0.0.1 entry for hawkular-metrics.example.com in /etc/hosts 
   exit 1
fi

# The OpenShift docs say to disable firewalld for now. Just in case it is running, stop it now
sudo systemctl stop firewalld
echo Turned off firewalld

# Create some convienence links
ln -s ${OPENSHIFT_BINARY_DIR}/ openshift-dir
ln -s ${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/ca.crt

# Go to where OpenShift build is
cd ${OPENSHIFT_BINARY_DIR}

# Tell OpenShift to bind to an IP
sudo ${OPENSHIFT_BINARY_DIR}/openshift start --write-config=${OPENSHIFT_BINARY_DIR}/openshift.local.config --hostname=${OPENSHIFT_IP_ADDRESS} --public-master=${OPENSHIFT_IP_ADDRESS} --master=${OPENSHIFT_IP_ADDRESS}
echo Binding OpenShift to: ${OPENSHIFT_IP_ADDRESS}

# Tell OpenShift what the Hawkular Metrics URL should be
sudo sed -i 's/metricsPublicURL: ""/metricsPublicURL: https:\/\/hawkular-metrics.example.com\/hawkular\/metrics/g' ${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/master-config.yaml
echo OpenShift will use hawkular-metrics.example.com for Hawkular Metrics URL

# Start OpenShift
sudo ${OPENSHIFT_BINARY_DIR}/openshift start --node-config=${OPENSHIFT_BINARY_DIR}/openshift.local.config/node-${OPENSHIFT_IP_ADDRESS}/node-config.yaml --master-config=${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/master-config.yaml
