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
FABRIC_RELEASE=1.4
PREV_VERSION=0.2.0
BASE_VERSION=0.3.0

EXECUTABLES ?= go git curl docker
K := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH: Check dependencies")))

all: checks integration-test

checks: basic-checks unit-test

basic-checks: license spelling linter build

.PHONY: spelling
spelling: gotool.misspell
	@scripts/check_spelling.sh

.PHONY: license
license:
	@scripts/check_license.sh

.PHONY: build
build: bin/fab3 bin/evmcc

include gotools.mk

.PHONY: clean
clean: gotools-clean
	rm -rf bin/ node_modules/

.PHONY: gotools
gotools: gotools-install

unit-test: gotool.ginkgo
	@echo "Running unit-tests"
	ginkgo -p -randomizeAllSpecs -randomizeSuites -requireSuite -noColor -keepGoing -race -r evmcc
	cd fab3 && GO111MODULE=on ginkgo -p -randomizeAllSpecs -randomizeSuites -requireSuite -noColor -keepGoing -race -r

unit-tests: unit-test

dev-test:
	ginkgo watch -notify -randomizeAllSpecs -requireSuite -race -cover -skipPackage integration -r

linter: gotool.goimports
	@echo "LINT: Running code checks.."
	@scripts/golinter.sh

changelog:
	@scripts/changelog.sh v$(PREV_VERSION) v$(BASE_VERSION)

# we don't use any of these images, they just need to exist for the integration
# tests, so pull busybox and tag it as needed
.PHONY: docker-images
docker-images:
	docker pull busybox
	@# check if the image exists, we only want the exit code, so give an empty format string
	for IMAGE in couchdb kafka zookeeper; do \
		docker inspect hyperledger/fabric-$$IMAGE --format ' ' || \
			echo "tag $$IMAGE" && docker tag busybox hyperledger/fabric-$$IMAGE ; \
	done
	@docker inspect hyperledger/fabric-javaenv:amd64-latest --format ' ' || \
		echo "tag javaenv" && docker tag busybox hyperledger/fabric-javaenv:amd64-latest

.PHONY: integration-test
integration-test: docker-images gotool.ginkgo
	@echo "Running integration-test"
	@scripts/run-integration-tests.sh

fab3: bin/fab3

.PHONY: bin/fab3 # let 'go build' handle caching and whether to rebuild
bin/fab3:
	mkdir -p bin/
	cd fab3 && GO111MODULE=on go build -o ./../bin/fab3 ./cmd

.PHONY: bin/evmcc # let 'go build' handle caching and whether to rebuild
bin/evmcc:
	mkdir -p bin/
	go build -o bin/evmcc ./evmcc
	rm bin/evmcc # checking that it compiled, evmcc not meant to be run directly

# Requires go v1.11+
.PHONY:
update-mocks: gotool.counterfeiter
	go generate ./fab3/
	counterfeiter -o evmcc/mocks/mockstub.go --fake-name MockStub evmcc/vendor/github.com/hyperledger/fabric/core/chaincode/shim/interfaces.go ChaincodeStubInterface
