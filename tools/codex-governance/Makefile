.PHONY: build install uninstall test vet fmt

GO_BIN_DIR := $(shell go env GOBIN)
ifeq ($(GO_BIN_DIR),)
GO_BIN_DIR := $(shell go env GOPATH | cut -d: -f1)/bin
endif

build:
	go build ./cmd/codex-governance

install:
	go install ./cmd/codex-governance

uninstall:
	rm -f "$(GO_BIN_DIR)/codex-governance"

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go')
