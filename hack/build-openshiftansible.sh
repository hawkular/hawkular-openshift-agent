#!/bin/sh

##############################################################################
# build-openshiftansible.sh
#
# This will download the OpenShift Ansible source code.
# There is really nothing to build but is used by start-openshift.sh.
##############################################################################

source ./env-openshift.sh

echo Will git clone OpenShift Ansible here: ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}

if [ ! -d "${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}" ]; then
  echo The OpenShift Ansible source code repository has not been cloned yet - doing that now.

  # Create the location where the source code will live.

  PARENT_DIR=`dirname ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}`
  mkdir -p ${PARENT_DIR}

  if [ ! -d "${PARENT_DIR}" ]; then
    echo Aborting. Cannot create the parent source directory: ${PARENT_DIR}
    exit 1
  fi

  # Clone the OpenShift Ansible source code repository via git.

  cd ${PARENT_DIR}
  git clone git@github.com:openshift/openshift-ansible.git
else
  echo The OpenShift Ansible source code repository exists - it will be updated now.
fi

# Update the OpenShift Ansible git repo.

cd ${OPENSHIFTANSIBLE_GITHUB_SOURCE_DIR}
git pull

if [ "$?" = "0" ]; then
  echo OpenShift Ansible build is complete!
fi
