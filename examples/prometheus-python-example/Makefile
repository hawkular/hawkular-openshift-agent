DOCKER_NAME = hawkular/hawkular-openshift-agent-example-prometheus-python
DOCKER_VERSION ?= latest
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

all: build

build:
	docker build -t ${DOCKER_TAG} .

openshift-deploy: openshift-undeploy
	@echo Deploying the Prometheus Python Example to OpenShift. Using project `oc project --short`
	oc process -f prometheus-python.yaml -p IMAGE_VERSION=${DOCKER_VERSION} | oc create -f -

openshift-undeploy:
	@echo Undeploying the Prometheus Python Example from OpenShift in project `oc project --short`
	oc delete all,secrets,sa,templates,configmaps --selector=hawkular-openshift-agent-example=prometheus-python
