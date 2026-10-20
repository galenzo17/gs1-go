# AI table generator

`go generate ./...` (or `make generate`) runs this program to produce
`ai_table.go` at the repository root from the vendored copy of the
[GS1 Barcode Syntax Dictionary](https://github.com/gs1/gs1-syntax-dictionary).

- `gs1-syntax-dictionary.txt` is the unmodified release file. Its release
  date is in the file header and is emitted as `syntaxDictionaryRelease`.
- `LICENSE` is the Apache License 2.0 under which GS1 AISBL, the BWIPP
  project and the Zint project publish the dictionary. The generated table
  is a derived work of that file.

To update the dictionary, replace `gs1-syntax-dictionary.txt` with the new
release, run `make generate`, review the diff of `ai_table.go` and the
`aiNames` overrides in `ai.go`, and run `make all`. CI regenerates the table
and fails on any difference, so the committed table cannot drift from the
dictionary.
