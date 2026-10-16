GO      ?= go
PKG     := ./...
FUZZTIME ?= 30s

.PHONY: all test race cover lint vet fuzz bench wasm build clean tidy-check

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

# tidy-check verifies the zero-dependency promise: go mod tidy changes
# nothing, go.mod has no require block and no go.sum exists.
tidy-check:
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum
	@! grep -q '^require' go.mod || { echo "go.mod must not declare dependencies"; exit 1; }
	@test ! -e go.sum || { echo "go.sum must not exist in a zero-dependency module"; exit 1; }
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }

wasm:
	bash wasm/build.sh

clean:
	rm -f gs1 coverage.out wasm/gs1.wasm wasm/wasm_exec.js wasm/example/gs1.wasm wasm/example/wasm_exec.js
