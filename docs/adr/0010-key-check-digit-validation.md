# ADR 0010: Check-digit validation for every numeric GS1 key

## Status
Accepted. Extends ADR 0008.

## Date
2026-10-16

## Context
ADR 0008 introduced `ParseOptions.ValidateCheckDigits`, CLI `-check-digits`
and WASM `validateCheckDigits`, scoped to the GTIN in AI 01 and AI 02. The
GS1 solution-provider checklist also requires check-digit validation for the
SSCC (section 1.10, AI 00), the GLN (1.16, AIs 410 to 417) and the GSIN
(1.19, AI 402). The
[GS1 Syntax Dictionary](https://github.com/gs1/gs1-syntax-dictionary/blob/2026-01-27/gs1-syntax-dictionary.txt)
marks every one of those keys, plus GSRN (8017, 8018), the 13-digit prefix
of GDTI (253) and the 13 digits after the leading zero of GRAI (8003), with
the same `csum` constraint. All of them use the modulo-10 algorithm of GS1
General Specifications section 7.9 that `ValidateCheckDigit` already
implements.

Keys of the same length cannot be told apart from their digits: GTIN-13 and
GLN are both 13 digits, SSCC and GSRN both 18. Operators who type a key into
a tool often do not know which family it belongs to.

## Decision
- `ValidateCheckDigits`, `-check-digits`, `validateCheckDigits` and
  `Barcode.Validate()` validate the check digit of every AI in the table
  whose Syntax Dictionary entry carries `csum`: 00, 01, 02, 253, 402, 410
  to 417, 8003, 8017 and 8018. The option keeps its name and remains
  independent of association validation; the error priority from ADR 0008
  is unchanged (syntax, length and data type first; associations next;
  check digits last, first error only).
- `Encode` validates the same set, so `Parse(Encode(x))` round-trips with
  check-digit validation enabled. The deferral noted in its documentation
  is removed.
- Key-specific validators with length enforcement: `ValidateSSCC`,
  `ValidateGLN`, `ValidateGSIN`, `ValidateGSRN`. Length and digit errors
  wrap `ErrInvalidData`; a mismatch wraps `ErrInvalidCheckDigit`, as
  `ValidateGTIN` does. `ComputeCheckDigit` exposes the generic algorithm for
  1 to 30 data digits alongside the GTIN-only `ComputeGTINCheckDigit`.
- `ValidateKey` validates a key of unknown family and returns the families
  the length admits (`KeyType`, reported by `KeyTypes`). The CLI exposes it
  as `gs1 key KEY` and WASM as `validateKey(key)`, both naming every
  candidate family instead of guessing one.
- GDTI and GRAI are validated inside the parser and encoder hook through the
  generic validator on their numeric prefix. They get no dedicated exported
  validator because their serial component is not part of the check.

Alternatives rejected: a separate option per key family would multiply the
option matrix of ADR 0008 for no benefit, since a caller who wants check
digits wants all of them; detecting a single key family from the length
would silently mislabel GLNs as GTINs and GSRNs as SSCCs.

## Consequences
- Element strings that parsed under `ValidateCheckDigits` with a wrong SSCC,
  GLN, GSIN, GSRN, GDTI or GRAI check digit now fail with
  `ErrInvalidCheckDigit`. Lenient parsing is unchanged. `Barcode.Validate()`
  on hand-built barcodes now requires well-formed values for those AIs, as
  it already did for AI 01 and 02.
- The opt-off `ParseInto` hot path keeps zero allocations. The AI switch
  adds no table lookups.
- Certification rows 1.10, 1.16 and 1.19 are covered. The test vectors come
  from the GS1 check digit calculator.
