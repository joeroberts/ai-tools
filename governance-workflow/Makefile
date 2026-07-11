.PHONY: build test vet fmt

build:
	go build ./cmd/codex-governance

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go')
