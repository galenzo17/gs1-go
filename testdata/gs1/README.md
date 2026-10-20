# GS1 conformance vectors

`conformance_test.go` runs the test vectors that GS1 publishes with the
Barcode Syntax Engine and the Barcode Syntax Dictionary through this parser
and compares the accept/reject outcome, and where the vector states it, the
element split.

GS1 ships the vectors inside the C unit tests of those projects rather than
as data files, so the relevant source files are vendored here unmodified and
the test extracts the vectors from the `test_*` calls:

| Directory | Project | Licence | Vectors |
|---|---|---|---|
| `syntax-engine/` | GS1 Barcode Syntax Engine (see `VERSION`) | Apache 2.0 | `ai.c`: `test_parseAIdata`, `test_processAIdata`, `test_ai_linters`, `test_validateAIs`; `scandata.c`: `test_testProcessScanData` |
| `syntax-dictionary/` | GS1 Barcode Syntax Dictionary (see `VERSION`) | Apache 2.0 | `lint_*.c`: `UNIT_TEST_PASS` / `UNIT_TEST_FAIL` of the linters this package implements |

`known-divergences.txt` lists every vector whose outcome differs from the
reference, one per line as `<suite>\t<quoted input>\t# reason`. The test
fails when a vector not listed there diverges (a regression) and when a
listed vector passes again (a stale entry that must be removed), so the
conformance rate printed by `make conformance` only moves deliberately.
Divergences fall into three groups: scanner tolerance this parser keeps on
purpose (ADR 0003), features not implemented (unknown-AI passthrough,
Digital Link URIs, composite symbols, escaped brackets), and linters that
run only through the explicit options (`ValidateCheckDigits`, `ParseDate`).

To update: replace the vendored files with the new release, update
`VERSION`, run `make conformance`, and resolve every new or stale divergence.
