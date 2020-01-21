# Copyright IBM Corp All Rights Reserved.
# Copyright London Stock Exchange Group All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

GOTOOLS = counterfeiter goimports ginkgo misspell
BUILD_DIR ?= .build
GOTOOLS_GOPATH ?= $(BUILD_DIR)/gotools
GOTOOLS_BINDIR ?= $(firstword $(subst :, ,$(GOPATH)))/bin
GOROOT ?= $(firstword $(subst :, ,$(GOPATH))
GO111MODULE=off

# go tool->path mapping
go.fqp.counterfeiter := github.com/maxbrunsfeld/counterfeiter
go.fqp.goimports     := golang.org/x/tools/cmd/goimports
go.fqp.misspell      := github.com/client9/misspell/cmd/misspell

.PHONY: gotools-install
gotools-install: $(addprefix gotool., $(GOTOOLS))

.PHONY: gotools-clean
gotools-clean:
	-@rm -rf $(BUILD_DIR)/gotools

# Special override for ginkgo since we want to use the version vendored with the project
gotool.ginkgo: GINKGO_VERSION ?= "v1.10.2"
gotool.ginkgo:
	@GOPATH=$(abspath $(GOTOOLS_GOPATH)) go get -d -u github.com/onsi/ginkgo
	@git -C $(abspath $(GOTOOLS_GOPATH))/src/github.com/onsi/ginkgo checkout -q $(GINKGO_VERSION)
	@echo "Building github.com/onsi/ginkgo/ginkgo $(GINKGO_VERSION)-> ginkgo"
	@GOPATH=$(abspath $(GOTOOLS_GOPATH)) GOBIN=$(abspath $(GOTOOLS_BINDIR)) go install -ldflags="-X main.version=$(GINKGO_VERSION) -X main.buildDate=$$(date '+%Y-%m-%d')" github.com/onsi/ginkgo/ginkgo
# reset to a branch, so that the next time this target is run, go get starts on a branch, as it must
	@git -C $(abspath $(GOTOOLS_GOPATH))/src/github.com/onsi/ginkgo/ checkout -q master

gotool.counterfeiter: COUNTERFEITER_VERSION ?= "v6.0.1"
gotool.counterfeiter:
	@GOPATH=$(abspath $(GOTOOLS_GOPATH)) go get -d -u ${go.fqp.counterfeiter}
	@git -C $(abspath $(GOTOOLS_GOPATH))/src/${go.fqp.counterfeiter} checkout -q $(COUNTERFEITER_VERSION)
	@echo "Building counterfeiter"
	@GOPATH=$(abspath $(GOTOOLS_GOPATH)) GOBIN=$(abspath $(GOTOOLS_BINDIR)) go install ${go.fqp.counterfeiter}
# reset to a branch, so that the next time this target is run, go get starts on a branch, as it must
	@git -C $(abspath $(GOTOOLS_GOPATH))/src/${go.fqp.counterfeiter} checkout -q master

# Default rule for gotools uses the name->path map for a generic 'go get' style build
gotool.%:
	$(eval TOOL = ${subst gotool.,,${@}})
	@echo "Building ${go.fqp.${TOOL}} -> $(TOOL)"
	@GOPATH=$(abspath $(GOTOOLS_GOPATH)) GOBIN=$(abspath $(GOTOOLS_BINDIR)) go get -u ${go.fqp.${TOOL}}
