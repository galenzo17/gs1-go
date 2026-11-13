# ADR 0009: Symbol generation in a `symbol` sub-package

## Status
Proposed. Supersedes the "barcode generation" clause of ADR 0002.

## Date
2026-10-30

## Context
[ADR 0002](0002-parser-scope.md) puts barcode generation out of scope. That
decision was taken while the parser lived inside a larger application that
already had a generator. As a standalone module aimed at the GS1 LATAM
solution-provider certification, section 3.5 of the checklist asks the
software to print GS1 symbols and logistic labels (#24), and it is the one
Required gap left in the [certification matrix](../gs1-certification.md).
Integrators currently combine a parser, an encoder and an unrelated
renderer, and FNC1 placement is where those combinations break.

`Encode` and `EncodeZPL` (#16) already build validated element strings with
correct AI order and FNC1 placement. What is missing is the symbol itself:
the modules or bars of a GS1 DataMatrix, GS1-128, EAN/UPC or ITF-14, drawn
for a printer or a screen.

## Decision

### Supersede the generation clause of ADR 0002
Symbol generation is in scope. Print-quality verification (ISO/IEC 15415,
15416) and GS1 Digital Link stay out of scope. The rest of ADR 0002 stands.

### Sub-package `github.com/galenzo17/gs1-go/symbol`
| Option | Cost for parser-only users | Discoverability |
|---|---|---|
| **Sub-package in this module** | **none: not imported, not linked** | **same repository, same release** |
| Separate module | none | second repository, versions to keep in sync |
| Root package | `image` and `image/png` linked into every binary | high |

The sub-package keeps the root package, the CLI's parse path and the
WebAssembly build free of rendering code. It uses only the standard
library, like the rest of the module.

### Validated keys and elements in, never a scanned string
Symbols that carry an element string (GS1 DataMatrix, GS1-128) take
`[]gs1.Element` and build the string with `Encode`, so a symbol can only
carry what `Encode` accepts: character set 82, valid check digits, no
repeated AIs, predefined-length AIs first, association rules. They do not
take a scanned string, because `Parse` is deliberately lenient and recovers
a missing FNC1 heuristically (ADR 0006); a printer must not print a guess.
Callers holding bracket notation parse it first.

Symbols that carry a single key (EAN/UPC, ITF-14) take the key and validate
its length and check digit.

### Module model out, standard library renderers
Each encoder returns the symbol as modules: a `Matrix` for 2D symbols, the
bar pattern for linear ones. The model renders to `image.Image`, PNG and
SVG at a caller-chosen number of pixels per module.

- Linear symbols include their quiet zones and human-readable digits, since
  GS1 fixes those zones per symbology and EAN/UPC print part of the digits
  inside them.
- 2D symbols take the quiet zone in modules from the caller. GS1 requires
  at least 1X on every side (General Specifications 5.12.3) and printers
  often use more.

Pixels per module come first. Sizing by X-dimension in millimetres and
printer resolution, following the General Specifications 5.12.3 tables,
follows as a later change. ZPL output for symbols can be added without a
library, as `EncodeZPL` already does for the GS1-128 field data.

### Specification sources
Encoders are written from the ISO/IEC symbology standards (16022 Data
Matrix, 15417 Code 128, 15420 EAN/UPC, 16390 ITF) and the GS1 General
Specifications, not ported from other encoders. Where deployed readers
disagree with a literal reading of a standard, the encoder follows the
readers and says so in a doc comment. For 144x144 Data Matrix, the ECC
codewords of the two shorter Reed-Solomon blocks come first, the layout
zxing, libdmtx, BWIPP and zint use (zxing-cpp issue 259).

### Verification without test dependencies
- Golden data produced by an external encoder (zint) is committed, and
  `go test` compares every symbol module for module. zint is needed only to
  regenerate the file.
- A test-only reader decodes symbols back, so round trips run in `go test`
  and under fuzzing.
- `make verify-symbol` decodes rendered PNGs with an independent reader
  (zxing-cpp) from scripts outside `go.mod`.

## Consequences
- ADR 0002 remains accepted for everything except the generation clause.
- Each symbology lands as its own files in `symbol`, with its own golden
  data and round trip: GS1 DataMatrix, EAN/UPC and ITF-14, GS1-128, and
  later GS1 QR Code and the logistic label layout.
- Data Matrix starts with ASCII encodation only. ASCII can express every
  GS1 element string, but long alphanumeric data gets a larger symbol than
  an optimizing encoder would pick, and may not fit where one would.
- The CLI and the WebAssembly bindings gain rendering commands once the
  first symbologies are in, so their surface is designed once (#21).
