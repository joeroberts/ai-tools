.PHONY: build install uninstall test vet fmt smoke-ticket-plan smoke-implementation-agent

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

smoke-ticket-plan:
	go run ./cmd/codex-governance jira plan validate --plan testdata/ticket-plans/valid/plan.json --repo-root .
	go test ./internal/cli -run TestRunJiraPlanCreateDryRunUsesApprovedWorkflowWithoutWritingRecord -count=1

smoke-implementation-agent:
	go test ./internal/implementation ./internal/cli -count=1
