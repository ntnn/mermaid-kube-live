go ?= go
nproc ?= $(shell nproc)
kind ?= kind
kubectl ?= kubectl

.PHONY: check
check: lint test-integration

.PHONY: lint
lint:
	$(go) vet ./...

.PHONY: test
test:
	$(go) test -v -race -short -parallel $(nproc) ./...

test_clusters := mkl-one mkl-two
integration_setup := $(patsubst %,setup-integration-%,$(test_clusters))

.PHONY: setup-integration $(integration_setup)
setup-integration: $(integration_setup)
$(integration_setup): setup-integration-%:
	$(kind) get clusters | grep -q $* || $(kind) create cluster --kubeconfig integration.kubeconfig.yaml --name $*
	$(kubectl) --kubeconfig integration.kubeconfig.yaml --context kind-$* apply --filename $(wildcard pkg/*/*-$*.yaml)

.PHONY: test-integration
test-integration: setup-integration
	$(go) test -v -race -parallel $(nproc) ./...
