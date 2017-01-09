DOCKER_NAME = hawkular/hawkular-openshift-agent-example-jolokia-wildfly
DOCKER_VERSION ?= dev
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

all: build

build:
	if [ ! -f "jolokia-war-1.3.5.war" ]; then \
		wget http://repo1.maven.org/maven2/org/jolokia/jolokia-war/1.3.5/jolokia-war-1.3.5.war ; \
	fi
	if [ ! -f "wildfly-10.1.0.Final.tar.gz" ]; then \
		wget http://download.jboss.org/wildfly/10.1.0.Final/wildfly-10.1.0.Final.tar.gz ; \
	fi
	docker build -t ${DOCKER_TAG} .

openshift-deploy: openshift-undeploy
	@echo Deploying the Jolokia WildFly Example to OpenShift. Using project `oc project --short`
	oc process -f jolokia-wildfly.yaml | oc create -f -

openshift-undeploy:
	@echo Undeploying the Jolokia WildFly Example from OpenShift. Using project `oc project --short`
	oc delete all,secrets,sa,templates,configmaps --selector=hawkular-openshift-agent-example=jolokia-wildfly
