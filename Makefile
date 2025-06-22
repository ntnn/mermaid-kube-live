include .bingo/Variables.mk

go ?= go

.PHONY: tools
tools:
	$(go) tool bingo get

.PHONY: check
check: imports lint test

.PHONY: imports
imports: $(GOIMPORTS)
	$(GOIMPORTS) -w -l -local github.com/ntnn/mermaid-kube-live .

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

nproc ?= $(shell nproc)
gotest := $(go) test -v -race -parallel $(nproc)
gotarget := ./...

.PHONY: test
test:
	$(gotest) -short $(gotarget)

.PHONY: test-integration
test-integration:
	$(gotest) $(gotarget)
