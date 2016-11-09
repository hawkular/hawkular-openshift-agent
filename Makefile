VERSION = 0.0.1
COMMIT_HASH = $(shell git rev-parse HEAD)

VERBOSE_MODE ?= 4

GO_BUILD_ENVVARS = \
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=0 \

DOCKER_NAME = hawkular/hawkular-openshift-agent
DOCKER_VERSION = latest
DOCKER_TAG = ${DOCKER_NAME}:${DOCKER_VERSION}

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

docker: build
	@echo Building Docker Image...
	@mkdir -p _output/docker
	@cp -r deploy/docker/* _output/docker
	@cp hawkular-openshift-agent _output/docker	
	sudo docker build -t ${DOCKER_TAG} _output/docker
 
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
#   install_glide - Installs the Glide executable itself. Just need to do this once.
#   glide_create  - Examines all imports and creates Glide YAML file.
#   install_deps  - Installs the dependencies declared in the Glide Lock file in the
#                   vendor directory. Does an update and creates the Glide Lock file if necessary.
#                   Use this to install the dependencies after cloning the git repo.
#   update_deps   - Updates the dependencies found in the Glide YAML file and
#                   installs them in the vendor directory. Creates/Updates the Glide Lock file.
#                   Use this if you've updated or added dependencies.

install_glide:
	@echo Installing Glide itself
	@mkdir -p ${GOPATH}/bin
	@curl https://glide.sh/get | sh

glide_create:
	@echo Creating Glide YAML file
	@glide create

install_deps:
	@echo Installing dependencies in vendor directory
	@glide install --strip-vendor

update_deps:
	@echo Updating dependencies and storing in vendor directory
	@glide update --strip-vendor
