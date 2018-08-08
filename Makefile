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
#   - clean - cleans the build area
#   - docker - builds the hyperledger/fabric-peer-evm image
#   - docker-base - builds the base peer image for the docker multi-stage build
#   - evmscc - build evmscc shared library for native OS
#   - evmscc-linux build evmscc shared library for linux
#   - gotools - installs go tools like golint
#   - license - checks go source files for Apache license header
#   - linter - runs all code checks
#   - unit-test - runs the go-test based unit tests
#   - integration-test - runs the e2e_cli based test

ARCH=$(shell go env GOARCH)
BASEIMAGE_RELEASE=0.4.8

BASE_DOCKER_NS ?= hyperledger
BASE_DOCKER_TAG=$(ARCH)-$(BASEIMAGE_RELEASE)
EVMSCC=github.com/hyperledger/fabric-chaincode-evm
FABRIC=github.com/hyperledger/fabric
LIB_DIR=/opt/gopath/lib
GO_TAGS=nopkcs11
BUILD_DIR ?= .build
GOOS ?= $(shell go env GOOS)

PACKAGES = ./statemanager/... ./evmcc/...
SRCFILES = ./evmcc/evmcc.go ./statemanager/statemanager.go

# We need this flag due to https://github.com/golang/go/issues/23739
CGO_LDFLAGS_ALLOW = CGO_LDFLAGS_ALLOW="-I/usr/local/share/libtool"

EXECUTABLES ?= go docker git curl
K := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH: Check dependencies")))

all: checks evmscc docker integration-test

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

unit-test: $(PROJECT_FILES)
	@echo "Running unit-tests"
	@go test -tags "$(GO_TAGS)" $(PACKAGES)

unit-tests: unit-test

linter: check-deps
	@echo "LINT: Running code checks.."
	@scripts/golinter.sh

check-deps:
	@echo "DEP: Checking for dependency issues.."
	@scripts/check_deps.sh

changelog:
	@scripts/changelog.sh v$(PREV_VERSION) v$(BASE_VERSION)

.PHONY: docker-base
docker-base: dependencies $(BUILD_DIR)/base-image/$(DUMMY)

$(BUILD_DIR)/base-image/$(DUMMY): Gopkg.toml Gopkg.lock
	@mkdir -p $(@D)
	@docker build --no-cache --build-arg GO_TAGS --build-arg CGO_LDFLAGS_ALLOW . -f Dockerfile.base -t fabric-peer-evm-base:latest
	@touch $@

docker: docker-base $(BUILD_DIR)/peer-image/$(DUMMY)

$(BUILD_DIR)/peer-image/$(DUMMY): $(SRCFILES) ./config/core.yaml
	@mkdir -p $(@D)
	@docker build --no-cache --build-arg GO_TAGS --build-arg CGO_LDFLAGS_ALLOW . -t hyperledger/fabric-peer-evm:latest
	@scripts/docker-cleanup.sh
	@touch $@

.PHONY: dependencies
dependencies:
	@scripts/check_docker_deps.sh

.PHONY: integration-test
integration-test:
	@echo "Running integration-test"
	@cd e2e_cli && ./network_setup.sh down && ./network_setup.sh up mychannel 1

evmscc: $(BUILD_DIR)/$(GOOS)/lib/evmscc.so

evmscc-linux:
	@GOOS=linux make evmscc

$(BUILD_DIR)/$(GOOS)/lib/evmscc.so: docker
	@echo "Extracting $(GOOS) evmscc.so from image"
	@mkdir -p $(@D)
	@docker run --rm -v $(PWD)/$(@D):/tmp/build/ hyperledger/fabric-peer-evm:latest /bin/bash -c 'cp /opt/lib/evmscc.so /tmp/build/'

.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)
