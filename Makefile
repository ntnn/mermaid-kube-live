GO ?= go

TOOLS_DIR := hack/tools
GOLANGCI_LINT_VER := 2.12.2
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint-$(GOLANGCI_LINT_VER)

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
build: bin
	$(GO) build -o bin/mermaid-kube-live .

.PHONY: fmt
fmt:
	$(GO) fmt $(WHAT)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) $(WHAT)

.PHONY: lint-fix
lint-fix: override GOLANGCI_LINT_FLAGS := $(GOLANGCI_LINT_FLAGS) --fix
lint-fix: lint

$(GOLANGCI_LINT):
	mkdir -p $(TOOLS_DIR)
	$(GO) tool github.com/ntnn/mindl download -tool golangci-lint -common -out $@ -version $(GOLANGCI_LINT_VER)

NPROC ?= $(shell nproc)
GOTEST ?= $(GO) test -v -race -parallel $(NPROC)

.PHONY: test
test:
	$(GOTEST) -short $(WHAT)

.PHONY: run-setup
run-setup:
	kind get clusters | grep mkl || kind create cluster --name mkl
	kind export kubeconfig --name mkl --kubeconfig mkl.kubeconfig
	kubectl create configmap test-configmap --from-literal=hello=world -n default --dry-run=client -o yaml \
		| kubectl --kubeconfig mkl.kubeconfig apply -f-

.PHONY: run
run: | run-setup
	$(GO) run . -config run.yaml -diagram run.mermaid -kubeconfig mkl.kubeconfig -debug
