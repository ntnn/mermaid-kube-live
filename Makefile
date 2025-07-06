go ?= go

goimports := $(go) tool goimports
golangci_lint := $(go) tool golangci-lint

.PHONY: check
check: imports lint test

.PHONY: fmt
fmt:
	$(go) fmt ./...

.PHONY: imports
imports:
	$(goimports) -w -l -local github.com/ntnn/mermaid-kube-live .

.PHONY: lint
lint:
	$(golangci_lint) run ./...

nproc ?= $(shell nproc)
gotest := $(go) test -v -race -parallel $(nproc)
gotarget := ./...

.PHONY: test
test:
	$(gotest) -short $(gotarget)

.PHONY: test-integration
test-integration:
	$(gotest) $(gotarget)
