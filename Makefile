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

VERSION ?= 1.2.4.Final-SNAPSHOT
COMMIT_HASH ?= $(shell git rev-parse HEAD)

DOCKER_NAME = hawkular/hawkular-openshift-agent
DOCKER_VERSION ?= dev
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

VERBOSE_MODE ?= 4
HAWKULAR_OPENSHIFT_AGENT_NAMESPACE ?= default
HAWKULAR_OPENSHIFT_AGENT_HOSTNAME ?= "hawkular-openshift-agent-${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}.$(shell oc version | grep 'Server ' | awk '{print $$2;}' | egrep -o '([0-9]{1,3}[.]){3}[0-9]{1,3}').xip.io"

GO_BUILD_ENVVARS = \
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=0 \

all: build

clean:
	@echo Cleaning...
	@rm -f hawkular-openshift-agent
	@rm -rf ${GOPATH}/bin/hawkular-openshift-agent
	@rm -rf ${GOPATH}/pkg/*
	@rm -rf _output/*

build:  clean
	@echo Building...
	${GO_BUILD_ENVVARS} go build \
	   -ldflags "-X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH}"

docker:
	@echo Building Docker Image...
	@mkdir -p _output/docker
	@cp -r deploy/docker/* _output/docker
	@cp hawkular-openshift-agent _output/docker	
	docker build -t ${DOCKER_TAG} _output/docker

docker-examples:
	@echo Building Docker Image of Example: Prometheus-Python
	@DOCKER_VERSION=${DOCKER_VERSION} cd examples/prometheus-python-example && make build
	@echo Building Docker Image of Example: Jolokia-WildFly
	@DOCKER_VERSION=${DOCKER_VERSION} cd examples/jolokia-wildfly-example && make build
	@echo Building Docker Image of Example: Multiple-Endpoints
	@DOCKER_VERSION=${DOCKER_VERSION} cd examples/multiple-endpoints-example && make build
	@echo Building Docker Image of Example: Go-Expvar
	@DOCKER_VERSION=${DOCKER_VERSION} cd examples/go-expvar-example && make build

openshift-deploy: openshift-undeploy
	@echo Deploying Components to OpenShift
	oc create -f deploy/openshift/hawkular-openshift-agent-configmap.yaml -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}
	oc process -f deploy/openshift/hawkular-openshift-agent.yaml -v IMAGE_VERSION=${DOCKER_VERSION} | oc create -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE} -f -
	oc process -f deploy/openshift/hawkular-openshift-agent-route.yaml -v HAWKULAR_OPENSHIFT_AGENT_HOSTNAME=${HAWKULAR_OPENSHIFT_AGENT_HOSTNAME} | oc create -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE} -f -
	oc adm policy add-cluster-role-to-user hawkular-openshift-agent system:serviceaccount:${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}:hawkular-openshift-agent

openshift-undeploy:
	@echo Undeploying the Agent from OpenShift
	oc delete all,secrets,sa,templates,configmaps,daemonsets,clusterroles --selector=metrics-infra=agent -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}
	oc delete clusterroles hawkular-openshift-agent

openshift-status:
	@echo Obtaining Status from the Agent
	@curl -k -H "Authorization: Basic $(shell echo -n `oc get secret hawkular-openshift-agent-status -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE} --template='{{.data.username}}' | base64 --decode`:`oc get secret hawkular-openshift-agent-status -n ${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE} --template='{{.data.password}}' | base64 --decode` | base64)" http://hawkular-openshift-agent-${HAWKULAR_OPENSHIFT_AGENT_NAMESPACE}.$(shell oc version | grep 'Server ' | awk '{print $$2;}' | egrep -o '([0-9]{1,3}[.]){3}[0-9]{1,3}').xip.io/status

install:
	@echo Installing...
	${GO_BUILD_ENVVARS} go install \
           -ldflags "-X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH}"

build-test:
	@echo Building and installing test dependencies to help speed up test runs.
	go test -i $(shell go list ./... | grep -v -e /vendor/)

test:
	@echo Running tests, excluding third party tests under vendor
	go test $(shell go list ./... | grep -v -e /vendor/)

test-debug:
	@echo Running tests in debug mode, excluding third party tests under vendor
	go test -v $(shell go list ./... | grep -v -e /vendor/)

run:
	@echo Running...
	@hawkular-openshift-agent -v ${VERBOSE_MODE} -config config.yaml

# Glide Targets
#   install-glide - Installs the Glide executable itself. Just need to do this once.
#   glide-create  - Examines all imports and creates Glide YAML file.
#   install-deps  - Installs the dependencies declared in the Glide Lock file in the
#                   vendor directory. Does an update and creates the Glide Lock file if necessary.
#                   Use this to install the dependencies after cloning the git repo.
#   update-deps   - Updates the dependencies found in the Glide YAML file and
#                   installs them in the vendor directory. Creates/Updates the Glide Lock file.
#                   Use this if you've updated or added dependencies.

install-glide:
	@echo Installing Glide itself
	@mkdir -p ${GOPATH}/bin
	@curl https://glide.sh/get | sh

glide-create:
	@echo Creating Glide YAML file
	@glide create

install-deps:
	@echo Installing dependencies in vendor directory
	@glide install --strip-vendor

update-deps:
	@echo Updating dependencies and storing in vendor directory
	@glide update --strip-vendor
