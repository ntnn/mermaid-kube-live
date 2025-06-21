go ?= go

.PHONY: check
check: lint test

.PHONY: lint
lint:
	$(go) vet ./...

nproc ?= $(shell nproc)
gotest := $(go) test -v -race -parallel $(nproc)
gotarget := ./...

.PHONY: test
test:
	$(gotest) -short $(gotarget)

.PHONY: test-integration
test-integration:
	$(gotest) $(gotarget)
