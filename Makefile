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
build: bin
	$(GO) build -o bin/mermaid-kube-live .

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
