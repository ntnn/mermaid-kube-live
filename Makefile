GO ?= go

GOLANGCI_LINT ?= $(GO) tool golangci-lint
DEEPCOPY_GEN := $(GO) tool deepcopy-gen
VALIDATION_GEN := $(GO) tool validation-gen

WHAT ?= ./...

.PHONY: check
check: codegen fmt lint test

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

.PHONY: fmt
fmt:
	$(GO) fmt $(WHAT)

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run $(WHAT)

.PHONY: lint-fix
lint-fix:
	$(GOLANGCI_LINT) run --fix $(WHAT)

NPROC ?= $(shell nproc)
GOTEST := $(GO) test -v -race -parallel $(NPROC)

.PHONY: test
test:
	$(GOTEST) -short $(WHAT)
