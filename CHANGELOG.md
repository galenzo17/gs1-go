# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Strict association validation applies the complete dictionary rules, so
  inputs such as (12) without (8020) or (10) without an item key now fail
  under `ValidateAssociations`. Lenient parsing is unchanged.
- Twelve-digit input whose leading AI cannot fit its fixed data, such as
  `010415000002`, parses as a bare UPC-A instead of a truncated AI (01).
- Association error messages name the AI and alternatives from the
  dictionary, e.g. `AI (3102) excludes AI (3103)`.

### Added

- The AI table is generated from the GS1 Barcode Syntax Dictionary
  (release 2026-01-27, Apache 2.0) by `go generate`, growing from 63 to 541
  Application Identifiers with typed components, predefined-length flags,
  Digital Link attributes and the dictionary's `req`/`ex` association rules.
  `ValidateAssociations` evaluates those rules for every AI; `LookupAI`
  reports `Title` and `Predefined` and renders `Format` in dictionary
  notation; `SyntaxDictionaryRelease()` exposes the release. CI fails when
  the committed table differs from the dictionary.
- Opt-in GTIN check-digit validation for AI 01 and AI 02 through
  `ParseOptions.ValidateCheckDigits`, CLI `-check-digits`, and WASM
  `validateCheckDigits`, independently of association validation.
- Typed `Barcode` accessors and matching CLI JSON, WebAssembly and TypeScript fields.
- SSCC/GTIN identifier builders and check-digit validation helpers.
- Identification AIs 253, 401, 8003, 8004, 8017 and 8018, with `GDTI()` and
  `GRAI()` accessors.
- Missing-FNC1 recovery coverage for mixed-format identification AIs.
- Add dependency-free element string encoding and HRI formatting.
- Element encoding validates 01/02 GTIN check digits; SSCC and GLN check
  digit validation remains deferred.
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
