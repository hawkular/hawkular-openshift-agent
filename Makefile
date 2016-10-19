VERSION = 0.0.1
COMMIT_HASH = $(shell git rev-parse HEAD)

ROOT_PKG = "src/github.com/hawkular/hawkular-openshift-agent"
VERBOSE_MODE ?= 1

GO_BUILD_ENVVARS = \
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=0 \

all: build

clean:
	@echo Cleaning...
	@rm -f bin/hawkular-openshift-agent
	@rm -rf pkg/*

build:
	@echo Building...
	cd ${ROOT_PKG} && \
	${GO_BUILD_ENVVARS} go install \
	   -ldflags "-X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH}"

build-test:
	@echo Building and installing test dependencies to help speed up test runs.
	cd ${ROOT_PKG} && go test -i $(shell go list ./... | grep -v -e /vendor/)

test:
	@echo Running tests, excluding third party tests under vendor
	cd ${ROOT_PKG} && go test $(shell go list ./... | grep -v -e /vendor/)

test-debug:
	@echo Running tests in debug mode, excluding third party tests under vendor
	cd ${ROOT_PKG} && go test -v $(shell go list ./... | grep -v -e /vendor/)

run:
	@echo Running...
	@bin/hawkular-openshift-agent -v ${VERBOSE_MODE} -config config.yaml

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
	@curl https://glide.sh/get | sh

glide_create:
	@echo Creating Glide YAML file
	@cd ${ROOT_PKG} && glide create

install_deps:
	@echo Installing dependencies in vendor directory
	@cd ${ROOT_PKG} && glide install --strip-vendor

update_deps:
	@echo Updating dependencies and storing in vendor directory
	@cd ${ROOT_PKG} && glide update --strip-vendor
