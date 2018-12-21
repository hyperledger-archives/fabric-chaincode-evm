# Copyright IBM Corp All Rights Reserved.
# Copyright London Stock Exchange Group All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

GOTOOLS = counterfeiter dep golint goimports ginkgo misspell
BUILD_DIR ?= .build
GOTOOLS_GOPATH ?= $(BUILD_DIR)/gotools
GOTOOLS_BINDIR ?= $(firstword $(subst :, ,$(GOPATH)))/bin
GOROOT ?= $(firstword $(subst :, ,$(GOPATH))

# go tool->path mapping
go.fqp.counterfeiter := github.com/maxbrunsfeld/counterfeiter
go.fqp.goimports     := golang.org/x/tools/cmd/goimports
go.fqp.golint        := github.com/golang/lint/golint
go.fqp.misspell      := github.com/client9/misspell/cmd/misspell

.PHONY: gotools-install
gotools-install: $(addprefix gotool., $(GOTOOLS))

.PHONY: gotools-clean
gotools-clean:
	-@rm -rf $(BUILD_DIR)/gotools

# Special override for ginkgo since we want to use the version vendored with the project
gotool.ginkgo:
	@echo "Building github.com/onsi/ginkgo/ginkgo -> ginkgo"
	@go install ./vendor/github.com/onsi/ginkgo/ginkgo

# Lock to a versioned dep
gotool.dep: DEP_VERSION ?= "v0.5.0"
gotool.dep:
	@GOPATH=$(abspath $(GOTOOLS_GOPATH)) go get -d -u github.com/golang/dep
	@git -C $(abspath $(GOTOOLS_GOPATH))/src/github.com/golang/dep checkout -q $(DEP_VERSION)
	@echo "Building github.com/golang/dep $(DEP_VERSION) -> dep"
	@GOPATH=$(abspath $(GOTOOLS_GOPATH)) GOBIN=$(abspath $(GOTOOLS_BINDIR)) go install -ldflags="-X main.version=$(DEP_VERSION) -X main.buildDate=$$(date '+%Y-%m-%d')" github.com/golang/dep/cmd/dep
# reset to a branch, so that the next time this target is run, go get starts on a branch, as it must
	@git -C $(abspath $(GOTOOLS_GOPATH))/src/github.com/golang/dep checkout -q master

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
