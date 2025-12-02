GO ?= go

GOIMPORTS ?= $(GO) tool goimports
GOLANGCI_LINT ?= $(GO) tool golangci-lint
DEEPCOPY_GEN := $(GO) tool deepcopy-gen
VALIDATION_GEN := $(GO) tool validation-gen

.PHONY: check
check: codegen fmt imports lint test

bin:
	mkdir -p bin

.PHONY: codegen
codegen:
	$(DEEPCOPY_GEN) --output-file zz_generated.deepcopy.go ./apis/v1alpha1
	$(VALIDATION_GEN) \
		--output-file zz_generated.validation.go \
		--readonly-pkg k8s.io/apimachinery/pkg/apis/meta/v1 \
		--readonly-pkg k8s.io/apimachinery/pkg/runtime/schema \
		./apis/v1alpha1

.PHONY: build
build: bin codegen
	$(GO) build -o bin/mermaid-kube-live ./cmd/mermaid-kube-live

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
