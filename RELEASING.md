# Releasing

gs1-go is a Go module served by `proxy.golang.org` and indexed by
`pkg.go.dev` from its git tags. Some mistakes are irreversible there, so the
rules below exist to avoid them.

## Rules of the module proxy

- **Tags are immutable.** Once `vX.Y.Z` has been fetched by the proxy it is
  cached forever. Never delete, move or re-point a published tag. A bad
  release is withdrawn with a `retract` directive in `go.mod` and a new
  patch release, never by re-tagging:

  ```go
  retract v0.3.0 // check-digit regression, use v0.3.1
  ```

- **Semantic versioning drives compatibility.** `v0.x` may change the
  public API between minor versions, documented under *Changed* in the
  CHANGELOG. `v1.0.0` freezes the API: additions only, until `v2`. A `v2`
  requires the `/v2` module path (`module github.com/galenzo17/gs1-go/v2`)
  and a `v2/` import path for consumers.
- **Zero dependencies.** `go.mod` must have no `require` block and no
  `go.sum`. CI enforces it; a pull request that adds a dependency is
  declined.
- **The `go` directive is the minimum supported Go version.** Raising it is
  a *Changed* entry and follows the Go release policy of the two latest
  major versions; CI tests the minimum, the next version and `stable`.

## Checklist before tagging

Run on a clean checkout of `main`:

```bash
make all            # vet, lint, tests
make race           # race detector
make check-generate # ai_table.go matches the vendored Syntax Dictionary
make tidy-check     # go mod tidy produces no diff, no require block
make conformance    # GS1 vector agreement unchanged
make bench          # BenchmarkParseInto at 0 allocs/op
make wasm           # WebAssembly build and example bundle
node wasm/*.test.mjs
```

Then:

1. In `CHANGELOG.md`, rename the `[Unreleased]` section to `[X.Y.Z] - YYYY-MM-DD`,
   add a fresh empty `[Unreleased]` above it, and update the compare links
   at the bottom (`[Unreleased]: .../compare/vX.Y.Z...HEAD` and
   `[X.Y.Z]: .../compare/vPREV...vX.Y.Z`).
2. Review the public API diff since the previous tag:
   `go doc -all . > /tmp/api.txt` on both commits and compare. Every new
   exported identifier needs a godoc comment and, where it is a user-facing
   entry point, an `Example*` function in `example_test.go` so that
   pkg.go.dev shows it.
3. Commit as `chore(release): vX.Y.Z`, open a pull request, merge it.
4. Tag the merge commit and push the tag:

   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```

## What the tag publishes

| Channel | Mechanism | Notes |
|---|---|---|
| GitHub Release | GoReleaser from `release.yml` (`.goreleaser.yaml`) | CLI binaries for linux/darwin/windows × amd64/arm64 as bare files and tar.gz/zip archives, deb/rpm/apk packages, `checksums.txt`, SPDX SBOMs, `gs1.wasm`, `wasm_exec.js`, loader, types, and a changelog grouped by commit type. |
| `proxy.golang.org` | automatic on first `go get` | `go get github.com/galenzo17/gs1-go@vX.Y.Z`. The proxy caches the tag within minutes of the first request. |
| `pkg.go.dev` | crawler, after the proxy | Shows README, licence (MIT) and the `Example*` functions. Allow up to an hour. |
| npm `@galenzo17/gs1-wasm` | `release.yml`, after GoReleaser | WebAssembly package from the same build, versioned from the tag and published with npm provenance. Needs the `NPM_TOKEN` secret. |
| Homebrew, AUR, ghcr.io | GoReleaser | Cask in `galenzo17/homebrew-tap`, the `gs1-bin` AUR package and the multi-arch `ghcr.io/galenzo17/gs1` image. Need the `HOMEBREW_TAP_TOKEN` and `AUR_KEY` secrets; `skip_upload: auto` keeps prereleases out of the tap and AUR. |

After the release, check `https://pkg.go.dev/github.com/galenzo17/gs1-go@vX.Y.Z`
renders the new examples and that `go install github.com/galenzo17/gs1-go/cmd/gs1@vX.Y.Z`
reports the version with `gs1 version`.

## Criteria for v1.0.0

`v1.0.0` is tagged when all of the following hold:

- The GS1 certification readiness milestone (#22) is closed: every Required
  row of `docs/gs1-certification.md` reads *Covered* or *Application layer*,
  and the `conformance` CI job is green with the documented divergences.
- The public API has been reviewed against the GS1 Barcode Syntax
  Dictionary naming (data titles, AI formats, association vocabulary) and
  no rename is pending.
- No change to the `Barcode` or `Element` shape is pending; the fields and
  the zero-allocation `ParseInto` contract are final.
- The CLI flags and the WebAssembly API have parity with the library (#21)
  and the TypeScript declarations are complete.
- At least one release has shipped from the final `release.yml` and been
  consumed from the proxy without a `retract`.

Until then releases are `v0.x`, and each minor version lists its API
changes under *Changed*.
