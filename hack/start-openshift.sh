#!/bin/sh

##############################################################################
# start-openshift.sh
#
# Run this script to start/stop OpenShift Origin with Origin-Metrics
# deployed within it.
#
# This script takes one argument whose value is one of the following:
#       up: starts the OpenShift environment (Origin + Origin-Metrics)
#     down: stops the OpenShift environment (Origin + Origin-Metrics)
#   status: outputs the current status of the OpenShift environment
##############################################################################

source ./env-openshift.sh

WHERE_IS_PYTHON3=`which python3`
if [ "$?" = "0" ]; then
  echo Python3 installed: $WHERE_IS_PYTHON3
else
  echo You must install Python3 as it is required for OpenShift Ansible to work.
  exit 1
fi

WHERE_IS_ANSIBLE=`which ansible-playbook`
if [ "$?" = "0" ]; then
  echo Ansible installed: $WHERE_IS_ANSIBLE
else
  echo You must install Ansible.
  echo You can do this by git cloning the Ansible repo and
  echo then setting up your environment to use it. Something like:
  echo
  echo '$' mkdir `dirname ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}`
  echo '$' cd `dirname ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}`
  echo '$' git clone git://github.com/ansible/ansible.git --recursive
  echo '$' cd ansible
  echo '$' source ./hacking/env-setup
  echo '$' sudo dnf install -y python-paramiko python-jinja2 python-yaml pyOpenSSL python-cryptography python-lxml
  echo '$' sudo mkdir -p /etc/ansible
  echo '$' sudo sh -c "'"'echo -e "[masters]\nlocalhost" >> /etc/ansible/hosts'"'"
  exit 1
fi

echo Will use OpenShift that is located here: ${OPENSHIFT_BINARY_DIR}
cd ${OPENSHIFT_BINARY_DIR}
echo OpenShift git hash: $(git rev-parse HEAD)

_KUBECONFIG="${OPENSHIFT_BINARY_DIR}/openshift.local.config/master/admin.kubeconfig"
#_SKIP_VERIFY_ARG="--insecure-skip-tls-verify=true"
OPENSHIFT_EXE_OC="${OPENSHIFT_EXE_OC} ${_SKIP_VERIFY_ARG} --config=${_KUBECONFIG}"

if [ "$1" = "up" ];then

  # The OpenShift docs say to disable firewalld for now. Just in case it is running, stop it now.
  # If firewalld was running and is shutdown, it changes the iptable rules and screws up docker,
  # so we must restart docker in order for it to rebuild its iptable rules.
  sudo systemctl status firewalld > /dev/null 2>&1
  if [ "$?" == "0" ]; then
    echo Turning off firewalld as per OpenShift recommendation and then restarting docker to rebuild iptable rules
    sudo systemctl stop firewalld
    sudo systemctl restart docker.service
  fi

  echo Will start OpenShift server at ${OPENSHIFT_IP_ADDRESS}
  ${OPENSHIFT_EXE_OPENSHIFT} start --master=${OPENSHIFT_IP_ADDRESS} --hostname=${OPENSHIFT_IP_ADDRESS} --listen=https://${OPENSHIFT_IP_ADDRESS}:8443 > /tmp/openshift-console.log 2>&1 &

  # Wait for it to get to a point where we can connect to it
  echo -n Waiting for OpenShift to start
  tail -f /tmp/openshift-console.log | while read _LOGLINE
  do
    echo -n .
    if [[ "${_LOGLINE}" == *"Started Origin Controllers"* ]]; then
      pkill -P $$ tail
      break
    fi
  done
  echo OpenShift started.

  # The OpenShift docs say to do this
  echo Creating registry
  sudo chmod +r ${_KUBECONFIG}
  ${OPENSHIFT_EXE_OC} adm registry -n default

  echo 'Do you want the admin user to be assigned the cluster-admin role?'
  echo 'NOTE: This could expose your machine to root access!'
  echo 'Select "1" for Yes and "2" for No:'
  select yn in "Yes" "No"; do
    case $yn in
      Yes )
        echo Will assign the cluster-admin role to the admin user.
        ${OPENSHIFT_EXE_OC} login -u admin -p admin
        ${OPENSHIFT_EXE_OC} login -u system:admin
        ${OPENSHIFT_EXE_OC} adm policy add-cluster-role-to-user cluster-admin admin
        break;;
      No )
        echo Admin user will not be assigned the cluster-admin role.
        break;;
    esac
  done

  # Deploy a Router
  echo Creating router
  ${OPENSHIFT_EXE_OC} adm policy add-scc-to-user hostnetwork system:serviceaccount:default:router -n default
  ${OPENSHIFT_EXE_OC} adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:default:router -n default
  ${OPENSHIFT_EXE_OC} adm router router --replicas=1 --service-account=router -n default

  # Deploy Origin-Metrics Using Ansible
  echo Deploying Origin-Metrics
  echo Using OpenShift-Ansible found here: ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}
  cd ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}/playbooks
  echo OpenShift-Ansible git hash: $(git rev-parse HEAD)
  ansible-playbook -vv ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}/playbooks/byo/openshift-cluster/openshift-metrics.yml -e "openshift_deployment_type=origin" -e "openshift_metrics_install_metrics=True" -e "openshift_metrics_hawkular_hostname=hawkular-metrics.example.com" -e "ansible_python_interpreter=${WHERE_IS_PYTHON3}" -e "ansible_become=true" -e "ansible_become_user=root" ${OPENSHIFTMETRICS_ANSIBLE_VERSION_ARG} --ask-become-pass

elif [ "$1" = "down" ];then

  echo Will shutdown the OpenShift server
  sudo pkill -x openshift
  ${DOCKER_SUDO} docker ps | awk 'index($NF,"k8s_")==1 { print $1 }' | xargs -l -r docker stop
  mount | grep "openshift.local.volumes" | awk '{ print $3}' | xargs -l -r sudo umount
  sudo rm -rf openshift.local.*
  echo The OpenShift server is shutdown

elif [ "$1" = "status" ];then

  ${OPENSHIFT_EXE_OC} version
  ps -a | grep "[^-]openshift" | ps -v `awk '{print $1}'`

else
  echo 'Required argument must be either: up, down, or status'
  exit 1
fi
