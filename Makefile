GO ?= go

GOIMPORTS ?= $(GO) tool goimports
GOLANGCI_LINT ?= $(GO) tool golangci-lint

.PHONY: check
check: fmt imports lint test

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: imports
imports:
	$(GOIMPORTS) -w -l -local github.com/ntnn/mermaid-kube-live .

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

NPROC ?= $(shell nproc)
GOTEST := $(GO) test -v -race -parallel $(NPROC)
WHAT := ./...

.PHONY: test
test:
	$(GOTEST) -short $(WHAT)

.PHONY: test-integration
test-integration:
	$(GOTEST) $(WHAT)
