# ADR 0012: Detect bracket notation at the start of the element string

## Status
Proposed

## Context
The GS1 Syntax Dictionary's `Syntax` section defines `X` as CSET 82, whose
characters include parentheses; AI 10 has format `X..20`. The vendored
[CSET 82 reference](../../testdata/gs1/syntax-dictionary/lint_cset82.c)
includes these characters in its `CSET 82 characters` list and tests.
Treating every `(` as bracket notation silently changes a raw lot such as
`10LOT(90)X` into the value `LOT90X`.

## Decision
After scanner cleaning, locate the input start with the existing AIM and
FNC1 prefix rules. Only `(` at that position selects bracket notation for
the entire input. Otherwise preserve all parentheses and parse the input
as a raw element string.

Keep the existing bracket conversion, AI lookup, scanner tolerance and
validation order. `ParseHRI` remains the separate literal HRI parser for
encoder input.

Alternatives rejected: detecting a bracket anywhere corrupts valid raw
data; switching formats at each field makes AI-like text inside values
ambiguous; an explicit format option adds API surface to a data-loss fix
that can use the existing input distinction.

## Consequences
- Raw values retain paired, unpaired and AI-like parentheses without a
  conversion allocation.
- Existing bracket notation, including recognized scanner prefixes,
  follows the same conversion path as before.
- A raw input does not switch to bracket notation at a later `(AI)`.
  Variable-length raw fields still use FNC1 for separation.
