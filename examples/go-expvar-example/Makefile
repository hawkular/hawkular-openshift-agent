# Copyright 2017 Red Hat, Inc. and/or its affiliates
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

DOCKER_NAME = hawkular/hawkular-openshift-agent-example-go-expvar
DOCKER_VERSION ?= latest
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

HAWKULAR_OPENSHIFT_AGENT_EXAMPLE_GO_EXPVAR_HOSTNAME ?= "hawkular-openshift-agent-example-go-expvar-$(shell oc project -q).$(shell oc version | grep 'Server ' | awk '{print $$2;}' | egrep -o '([0-9]{1,3}[.]){3}[0-9]{1,3}').xip.io"

GOPATH := ${GOPATH}/src/github.com/hawkular/hawkular-openshift-agent/examples/go-expvar-example

all: clean compile build

clean:
	@echo Cleaning...
	rm -f go-expvar-example
	rm -rf ${GOPATH}/_output

compile:
	@echo Compiling...
	cd ${GOPATH}/src/go-expvar-example && go build -o ${GOPATH}/_output/go-expvar-example

build: compile
	docker build -t ${DOCKER_TAG} .

openshift-deploy: openshift-undeploy
	@echo Deploying the Go Expvar Example to OpenShift. Using project `oc project --short`
	oc process -f go-expvar.yaml -v IMAGE_VERSION=${DOCKER_VERSION} -v HAWKULAR_OPENSHIFT_AGENT_EXAMPLE_GO_EXPVAR_HOSTNAME=${HAWKULAR_OPENSHIFT_AGENT_EXAMPLE_GO_EXPVAR_HOSTNAME} | oc create -f -

openshift-undeploy:
	@echo Undeploying the Go Expvar Example from OpenShift. Using project `oc project --short`
	oc delete all,secrets,sa,templates,configmaps --selector=hawkular-openshift-agent-example=go-expvar
