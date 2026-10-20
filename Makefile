GO      ?= go
PKG     := ./...
FUZZTIME ?= 30s

.PHONY: all test race cover lint vet fuzz bench wasm build clean npm-check generate check-generate conformance

all: vet lint test

test:
	$(GO) test -count=1 $(PKG)

race:
	$(GO) test -count=1 -race $(PKG)

cover:
	$(GO) test -count=1 -coverprofile=coverage.out -covermode=atomic $(PKG)
	$(GO) tool cover -func=coverage.out | tail -1

vet:
	$(GO) vet $(PKG)

lint:
	golangci-lint run

fuzz:
	$(GO) test -run=^$$ -fuzz=FuzzParse -fuzztime=$(FUZZTIME) .

bench:
	$(GO) test -run=^$$ -bench=. -benchmem .

build:
	$(GO) build -o gs1 ./cmd/gs1

generate:
	$(GO) generate ./...

# bash with pipefail so the target fails on go test, not on grep.
conformance: SHELL := /bin/bash
conformance: .SHELLFLAGS := -o pipefail -c
conformance:
	$(GO) test -count=1 -run '^TestConformance$$|^TestLogisticLabelFixtures$$' -v . | grep -E 'conformance|^(--- |ok|FAIL)'

check-generate: generate
	git diff --exit-code -- ai_table.go

wasm:
	bash wasm/build.sh

# npm-check builds the WebAssembly bundle and verifies the npm package
# loads through its own loader and ships exactly the published files.
npm-check: wasm
	cp LICENSE wasm/
	node wasm/package.test.mjs
	cd wasm && npm pack --dry-run

clean:
	rm -f gs1 coverage.out wasm/gs1.wasm wasm/wasm_exec.js wasm/LICENSE wasm/example/gs1.wasm wasm/example/wasm_exec.js
