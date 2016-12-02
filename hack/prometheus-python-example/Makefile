all: build

build:
	docker build -t hawkular/openshift-agent-example-prometheus-python:dev .

openshift-deploy: openshift-undeploy
	@echo Deploying the Prometheus Python Example to OpenShift. Using project `oc project --short`
	oc process -f prometheus-python.yaml | oc create -f -

openshift-undeploy:
	@echo Undeploying the Prometheus Python Example from OpenShift in project `oc project --short`
	oc delete all,secrets,sa,templates,configmaps --selector=hawkular-openshift-agent-example=prometheus-python
