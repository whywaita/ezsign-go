GO ?= go
LISTEN ?= 127.0.0.1:8080

.DEFAULT_GOAL := build
.PHONY: build test vet fmt check run

build:
	$(GO) build -o bin/ezsign ./cmd/ezsign
	$(GO) build -o bin/ezsign-go-api ./cmd/api
	$(GO) build -o bin/ezsign-go-slideshow ./cmd/slideshow

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test vet

run: build
	./bin/ezsign-go-api -listen "$(LISTEN)"
