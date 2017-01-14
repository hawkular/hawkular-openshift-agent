DOCKER_NAME = hawkular/hawkular-openshift-agent-example-jolokia-wildfly
DOCKER_VERSION ?= latest
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

all: build

build:
	docker build -t ${DOCKER_TAG} .

openshift-deploy: openshift-undeploy
	@echo Deploying the Jolokia WildFly Example to OpenShift. Using project `oc project --short`
	oc process -f jolokia-wildfly.yaml -v IMAGE_VERSION=${DOCKER_VERSION} | oc create -f -

openshift-undeploy:
	@echo Undeploying the Jolokia WildFly Example from OpenShift. Using project `oc project --short`
	oc delete all,secrets,sa,templates,configmaps --selector=hawkular-openshift-agent-example=jolokia-wildfly
