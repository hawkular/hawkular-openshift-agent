# Copyright 2016-2017 Red Hat, Inc. and/or its affiliates
# and other contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DOCKER_NAME = hawkular/hawkular-openshift-agent-example-prometheus-python
DOCKER_VERSION ?= latest
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

all: build

build:
	docker build -t ${DOCKER_TAG} .

openshift-deploy: openshift-undeploy
	@echo Deploying the Prometheus Python Example to OpenShift. Using project `oc project --short`
	oc process -f prometheus-python.yaml -v IMAGE_VERSION=${DOCKER_VERSION} | oc create -f -

openshift-undeploy:
	@echo Undeploying the Prometheus Python Example from OpenShift in project `oc project --short`
	oc delete all,secrets,sa,templates,configmaps --selector=hawkular-openshift-agent-example=prometheus-python
