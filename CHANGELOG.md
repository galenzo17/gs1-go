# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- CLI and WASM parity: WASM elements carry `name`, `measure` and the ISO
  `date` like the CLI JSON output, `assumeBareGTIN8` in WASM and `-gtin8` in the CLI expose
  `ParseOptions.AssumeBareGTIN8`, and the identifier builders reach both
  entry points as `gs1 gtin13`, `gs1 gtin14`, `newSSCC`, `newGTIN13` and
  `newGTIN14`.
- `Encode` enforces AI association rules, so an element string it accepts
  also parses with `ValidateAssociations`. `EncodeZPL` emits ZPL `^FD` field
  data with `>8` FNC1 escapes and `ParseHRI` splits bracket notation
  literally for encoder input. CLI `gs1 encode [-hri] [-zpl]` and WASM
  `encode()` expose both, and golden tests cover the GS1 Logistics Label
  Guideline and Syntax Engine example strings.
- Check-digit validation for SSCC, GLN, GSIN and GSRN through
  `ValidateSSCC`, `ValidateGLN`, `ValidateGSIN`, `ValidateGSRN`, the generic
  `ComputeCheckDigit`, and `ValidateKey` with `KeyType` detection by length.
  `ParseOptions.ValidateCheckDigits`, `Barcode.Validate()` and `Encode` now
  cover AIs 00, 402, 410–417, 8017, 8018 and the numeric prefix of 253 and
  8003 in addition to 01 and 02. CLI `gs1 key KEY` and WASM `validateKey`.
- Opt-in GTIN check-digit validation for AI 01 and AI 02 through
  `ParseOptions.ValidateCheckDigits`, CLI `-check-digits`, and WASM
  `validateCheckDigits`, independently of association validation.
- Typed `Barcode` accessors and matching CLI JSON, WebAssembly and TypeScript fields.
- SSCC/GTIN identifier builders and check-digit validation helpers.
- Identification AIs 253, 401, 8003, 8004, 8017 and 8018, with `GDTI()` and
  `GRAI()` accessors.
- Missing-FNC1 recovery coverage for mixed-format identification AIs.
- Add dependency-free element string encoding and HRI formatting.
- Element encoding validates 01/02 GTIN check digits.
- Made unsupported carrier variants fail explicitly and exposed detected
  symbology names in CLI and WASM results.
- Added AIM symbology carrier detection and support for EAN/UPC, ITF-14, and
  opt-in bare GTIN-8 parsing.
- `Measure` and typed barcode measurement accessors for net and gross weight.
- Measurement data in CLI JSON output.
- `DueDate()` and `Warnings()` support, including due-date advisories in CLI
  and WASM JSON output.

### Changed

- `Barcode.Validate()` now checks GTIN check digits for AI 01 and AI 02
  after AI associations, using the current elements without reparsing raw
  input. Association errors retain priority; existing strict parsing options
  remain association-only and default parsing remains lenient.
- Corrected strict AI association rules to match the GS1 Syntax Dictionary.

## [0.1.0] - 2026-09-22

First release as a standalone module. The code was extracted from the `gs1`
sub-module of [health-interop](https://github.com/galenzo17/health-interop)
with its full history.

### Added

- `Parse` and `ParseInto` for GS1-128 / GS1 DataMatrix element strings with
  FNC1 (ASCII 29) separators, bracket notation, AIM symbology identifiers and
  bare EAN-13 / UPC-A / GTIN-14.
- Scanner resilience by default: BOM, CR/LF, NUL and duplicated FNC1 are
  normalized before parsing.
- Recovery from missing FNC1 between variable-length fields using a
  balanced-split heuristic.
- `Barcode` accessors: `GTIN`, `Lot`, `SerialNumber`, `SSCC`, `Count`,
  `ExpirationDate`, `ProductionDate`, `BestBeforeDate`, `Get`.
- `LookupAI` exposing AI name and GS1 Syntax Dictionary format (`N14`, `X..20`).
- `ValidateGTIN`, `ComputeGTINCheckDigit` and `ExpandUPCE`.
- `ParseDate`, `ParseDateWithOptions` (configurable day-zero policy) and
  `ParseDateRaw`.
- Regulatory profiles for ANVISA (BR), ANMAT (AR), SNFA (CL) and COFEPRIS (MX).
- `cmd/gs1`: dependency-free CLI with text/JSON output, stdin streaming for
  scanner loops, GTIN and AI lookup subcommands.
- `cmd/gs1-wasm`: WebAssembly bindings with a JavaScript loader and
  TypeScript declarations.
- Fuzz target `FuzzParse` run on every CI build and nightly.

[Unreleased]: https://github.com/galenzo17/gs1-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/galenzo17/gs1-go/releases/tag/v0.1.0
