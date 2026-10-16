GO      ?= go
PKG     := ./...
FUZZTIME ?= 30s

.PHONY: all test race cover lint vet fuzz bench wasm build clean npm-check snapshot

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

wasm:
	bash wasm/build.sh

# snapshot runs the whole release pipeline locally without a tag: binaries,
# archives, packages, PKGBUILD and cask into dist/, skipping publication,
# container images and SBOMs (which need a registry login and syft).
snapshot:
	goreleaser release --snapshot --clean --skip=publish,docker,sbom

# npm-check builds the WebAssembly bundle and verifies the npm package
# loads through its own loader and ships exactly the published files.
npm-check: wasm
	cp LICENSE wasm/
	node wasm/package.test.mjs
	cd wasm && npm pack --dry-run

clean:
	rm -rf dist
	rm -f gs1 coverage.out wasm/gs1.wasm wasm/wasm_exec.js wasm/LICENSE wasm/example/gs1.wasm wasm/example/wasm_exec.js
