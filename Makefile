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
#   - evmscc - build evmscc shared library for native OS
#   - evmscc-linux - build evmscc shared library for Linux, so it could be used in Docker
#   - gotools - installs go tools like golint
#   - license - checks go source files for Apache license header
#   - linter - runs all code checks
#   - unit-test - runs the go-test based unit tests

ARCH=$(shell uname -m)
BASEIMAGE_RELEASE=0.4.7

BASE_DOCKER_NS ?= hyperledger
BASE_DOCKER_TAG=$(ARCH)-$(BASEIMAGE_RELEASE)
EVMSCC=github.com/hyperledger/fabric-chaincode-evm
FABRIC=github.com/hyperledger/fabric
LIB_DIR=/opt/gopath/lib

BUILD_DIR ?= .build

PACKAGES = ./statemanager/... ./plugin/...

# We need this flag due to https://github.com/golang/go/issues/23739
CGO_LDFLAGS_ALLOW = CGO_LDFLAGS_ALLOW="-I/usr/local/share/libtool"

EXECUTABLES ?= go docker git curl
K := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH: Check dependencies")))

all: checks evmscc-linux

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
	echo "Running unit-tests"
	go test -tags \"$(GO_TAGS)\" $(PACKAGES)

unit-tests: unit-test

linter: check-deps
	@echo "LINT: Running code checks.."
	./scripts/golinter.sh

check-deps: 
	@echo "DEP: Checking for dependency issues.."
	./scripts/check_deps.sh

changelog:
	./scripts/changelog.sh v$(PREV_VERSION) v$(BASE_VERSION)

DRUN = docker run -i --rm $(DOCKER_RUN_FLAGS) -w /opt/gopath/src/$(EVMSCC)

evmscc-linux: $(BUILD_DIR)/linux/lib/evmscc.so
$(BUILD_DIR)/linux/lib/evmscc.so:
	@mkdir -p $(@D)
	$(eval TMPDIR := $(shell mktemp -d /tmp/evmscc-build.XXXXX))
	@echo $(TMPDIR)
	@rsync -az --exclude=".*/" --exclude=".*" --exclude="build/" $(GOPATH)/src/$(FABRIC) $(TMPDIR)
	@rsync -az --exclude=".*/" --exclude=".*" --exclude="build/" $(GOPATH)/src/$(EVMSCC) $(TMPDIR)
	@echo "Building $@"
	@$(DRUN) \
		-v $(TMPDIR)/fabric-chaincode-evm:/opt/gopath/src/$(EVMSCC) \
		-v $(TMPDIR)/fabric:/opt/gopath/src/$(FABRIC) \
		-v $(abspath $(@D)):$(LIB_DIR) \
		-v $(PWD)/scripts/build.sh:/opt/build.sh \
		-e LIB_DIR=$(LIB_DIR) \
		$(BASE_DOCKER_NS)/fabric-baseimage:$(BASE_DOCKER_TAG) \
		bash -c '/opt/build.sh /opt/gopath/src/$(FABRIC) /opt/gopath/src/$(EVMSCC)'
	@rm -rf $(TMPDIR)

.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)
