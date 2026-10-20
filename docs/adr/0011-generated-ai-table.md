# ADR 0011: Generate the AI table and association rules from the GS1 Syntax Dictionary

## Status
Accepted

## Date
2026-10-16

## Context
The hand-written table in `ai.go` covered 63 Application Identifiers. The
GS1 General Specifications define around 500, and any scan carrying another
AI failed with `ErrUnknownAI`. A certifier hits that on the first logistic
label with (7003), (422), (8200) or (715). The association rules in
`association.go` were likewise a hand-picked subset.

GS1 publishes the [Barcode Syntax Dictionary](https://github.com/gs1/gs1-syntax-dictionary)
under the Apache License 2.0: one line per AI with its typed components
(`N14,csum,gcppos2`, `N3,iso3166 X..9`, `N13,csum [X..17]`), flags for
predefined length and Digital Link use, and the `req=`, `ex=` and `dlpkey`
association attributes. It is the reference the GS1 Syntax Engine, BWIPP
and Zint build from.

## Decision
- `internal/gen/aitable` parses a vendored copy of the dictionary and writes
  `ai_table.go` through `go generate`. The dictionary file, its licence and
  the release date (`SyntaxDictionaryRelease()`) are committed; CI
  regenerates the table and fails on any diff, so it cannot drift.
- `aiSpec` carries the dictionary model: ordered components with type,
  length, variability, optionality and linter names; the predefined-length
  and Digital Link flags; `req`, `ex` and `dlpkey` attributes; plus derived
  `FixedLen`, `MinLen`, `MaxLen` and `DataType` for the parser's hot path.
  The table is a map of pointers so lookups copy nothing.
- The parser validates data by walking the components, which replaces the
  `NumericPrefixLen`, `FirstChar` and UIC special cases, and adds the CSET 39
  and base64url character sets. Only the structural `zero` linter is
  applied; value linters such as `yymmdd`, `iso3166` or `csum` are recorded
  on the components for the parser's existing options and for future work.
- `ValidateAssociations` evaluates the dictionary's `req` and `ex`
  attributes generically for every AI present, with `n` as a digit wildcard
  and `+` for conjunctions, instead of the hand-written rule set. Errors
  name the AI and the alternatives, e.g. `AI (37) requires AI (02) or AI
  (8026)`, `AI (3102) excludes AI (3103)`.
- Established data titles are kept in `aiNames` so CLI and JSON output do
  not change for the AIs documented before; every other AI reports the
  dictionary title. `LookupAI` exposes `Title` and `Predefined`, and
  `Format` now uses dictionary notation (`N13 [X..17]`, `N1 X1 X1 X1`).
- Missing-FNC1 recovery prefers the boundary that yields the fewest
  elements before the balanced-split rule, because the full table turns many
  digit runs into chains of plausible short numeric AIs.
- Bare GTIN detection treats a leading two-digit AI as ambiguous only when
  that AI's fixed data could fit in the remaining digits. With (03) in the
  table, every UPC-A starting with 03 would otherwise have been read as a
  truncated made-to-order GTIN.

Alternatives rejected: a switch on the first digits instead of a map (the
map of pointers benchmarks faster than the previous map of values and keeps
`ParseInto` at zero allocations); downloading the dictionary at build time
(builds must stay offline and reproducible); keeping the hand-written
association rules next to the generated attributes (two sources of truth).

## Consequences
- Every AI in the dictionary release parses. Strict association validation
  now applies the complete dictionary rules, so some inputs that passed
  before fail, notably (12) without (8020) and (10) without an item key.
  Lenient parsing is unchanged.
- Twelve-digit input starting with an AI whose data cannot fit, such as
  `010415000002`, is a UPC-A rather than a truncated AI (01).
- Updating to a new dictionary release is a file replacement, `make
  generate` and a review of the diff and of `aiNames`.
- The module still has zero dependencies: the generator uses only the
  standard library and runs only through `go generate`.
