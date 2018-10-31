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

PACKAGES = ./statemanager/... ./evmcc/... ./fabproxy/

EXECUTABLES ?= go git curl docker
K := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH: Check dependencies")))

all: checks integration-test

checks: basic-checks unit-test

basic-checks: license spelling linter

.PHONY: spelling
spelling:
	@scripts/check_spelling.sh

.PHONY: license
license:
	@scripts/check_license.sh

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

.PHONY:
update-mocks:
	@go generate ./fabproxy/
