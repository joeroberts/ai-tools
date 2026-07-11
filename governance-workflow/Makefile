.PHONY: build install test vet fmt

build:
	go build ./cmd/codex-governance

install:
	go install ./cmd/codex-governance

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go')
