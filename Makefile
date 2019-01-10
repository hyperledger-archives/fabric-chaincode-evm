# Copyright IBM Corp All Rights Reserved.
# Copyright London Stock Exchange Group All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# -------------------------------------------------------------
# This makefile defines the following targets
#
#   - all (default) - builds all targets and runs all tests/checks
#   - basic-checks - performs basic checks like license, spelling and linter
#   - check-deps - check for vendored dependencies that are no longer used
#   - checks - runs all non-integration tests/checks
#   - gotools - installs go tools like golint
#   - license - checks go source files for Apache license header
#   - linter - runs all code checks
#   - unit-test - runs the go-test based unit tests
#   - integration-test - runs the e2e_cli based test
#   - docker-images - pulls the latest docker ccenv and couchdb images needed for integration tests
#   - update-mocks - update the counterfeiter test doubles
#
ARCH=$(shell go env GOARCH)
BASEIMAGE_RELEASE=0.4.13
BASE_DOCKER_NS ?= hyperledger
BASE_DOCKER_TAG=$(ARCH)-$(BASEIMAGE_RELEASE)
FABRIC_RELEASE=1.4
PREV_VERSION=6111630c6cf12d3ca31559e93e33e9dad1e6f402
BASE_VERSION=0.1.0

PACKAGES = ./statemanager/... ./evmcc/... ./fabproxy/

EXECUTABLES ?= go git curl docker
K := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH: Check dependencies")))

all: checks integration-test

checks: basic-checks unit-test

basic-checks: license spelling linter build

.PHONY: spelling
spelling:
	@scripts/check_spelling.sh

.PHONY: license
license:
	@scripts/check_license.sh

.PHONY: build
build: bin/fab3 bin/evmcc

.PHONY: clean
clean:
	rm -rf bin/ node_modules/

include gotools.mk

.PHONY: gotools
gotools: gotools-install

unit-test: $(PROJECT_FILES) gotool.ginkgo
	@echo "Running unit-tests"
	@ginkgo -r -randomizeAllSpecs -noColor -keepGoing -race -tags "$(GO_TAGS)" $(PACKAGES)

unit-tests: unit-test

linter: gotools check-deps
	@echo "LINT: Running code checks.."
	@scripts/golinter.sh

check-deps:
	@echo "DEP: Checking for dependency issues.."
	@scripts/check_deps.sh

changelog:
	@scripts/changelog.sh v$(PREV_VERSION) v$(BASE_VERSION)

docker-images:
	docker pull $(BASE_DOCKER_NS)/fabric-javaenv:$(FABRIC_RELEASE)
	docker tag $(BASE_DOCKER_NS)/fabric-javaenv:$(FABRIC_RELEASE) $(BASE_DOCKER_NS)/fabric-javaenv:$(ARCH)-latest
	docker pull $(BASE_DOCKER_NS)/fabric-couchdb:$(BASE_DOCKER_TAG)
	docker tag $(BASE_DOCKER_NS)/fabric-couchdb:$(BASE_DOCKER_TAG) $(BASE_DOCKER_NS)/fabric-couchdb
	docker pull $(BASE_DOCKER_NS)/fabric-zookeeper:$(BASE_DOCKER_TAG)
	docker tag $(BASE_DOCKER_NS)/fabric-zookeeper:$(BASE_DOCKER_TAG) $(BASE_DOCKER_NS)/fabric-zookeeper
	docker pull $(BASE_DOCKER_NS)/fabric-kafka:$(BASE_DOCKER_TAG)
	docker tag $(BASE_DOCKER_NS)/fabric-kafka:$(BASE_DOCKER_TAG) $(BASE_DOCKER_NS)/fabric-kafka

.PHONY: integration-test
integration-test: docker-images gotool.ginkgo
	@echo "Running integration-test"
	@scripts/run-integration-tests.sh

.PHONY: bin/fab3 # let 'go build' handle caching and whether to rebuild
bin/fab3:
	mkdir -p bin/
	go build -o bin/fab3 github.com/hyperledger/fabric-chaincode-evm/fabproxy/cmd

.PHONY: bin/evmcc # let 'go build' handle caching and whether to rebuild
bin/evmcc:
	mkdir -p bin/
	go build -o bin/evmcc github.com/hyperledger/fabric-chaincode-evm/evmcc
	rm bin/evmcc # checking that it compiled, evmcc not meant to be run directly

.PHONY:
update-mocks:
	go generate ./fabproxy/
	counterfeiter -o mocks/evmcc/mockstub.go --fake-name MockStub vendor/github.com/hyperledger/fabric/core/chaincode/shim/interfaces.go ChaincodeStubInterface
