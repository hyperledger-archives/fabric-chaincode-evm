# Copyright IBM Corp All Rights Reserved.
# Copyright London Stock Exchange Group All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# -------------------------------------------------------------
# This makefile defines the following targets
#
#   - all (default) - builds all targets and runs all tests/checks
#   - evmscc - build evmscc shared library for native OS
#	- evmscc-linux - build evmscc shared library for Linux, so it could be used in Docker

ARCH=$(shell uname -m)
BASEIMAGE_RELEASE=0.4.7

BASE_DOCKER_NS ?= hyperledger
BASE_DOCKER_TAG=$(ARCH)-$(BASEIMAGE_RELEASE)
EVMSCC=github.com/hyperledger/fabric-chaincode-evm
FABRIC=github.com/hyperledger/fabric
LIB_DIR=/opt/gopath/lib

BUILD_DIR ?= .build

# We need this flag due to https://github.com/golang/go/issues/23739
CGO_LDFLAGS_ALLOW = CGO_LDFLAGS_ALLOW="-I/usr/local/share/libtool"

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