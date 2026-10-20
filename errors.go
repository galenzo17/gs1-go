package gs1

import "errors"

// ErrEmptyInput indicates that the input string is empty or contains only whitespace.
var ErrEmptyInput = errors.New("gs1: empty input")

// ErrUnknownAI indicates that the parser encountered an unrecognized Application Identifier.
var ErrUnknownAI = errors.New("gs1: unknown application identifier")

// ErrTruncatedData indicates that the input ended before a fixed-length field was fully consumed.
var ErrTruncatedData = errors.New("gs1: truncated data")

// ErrInvalidData indicates that field data violates the AI's type or length constraints.
var ErrInvalidData = errors.New("gs1: invalid data")

// ErrInvalidCheckDigit indicates that the modulo-10 check digit of a numeric
// GS1 key (GTIN, SSCC, GLN, GSIN, GSRN, GDTI, GRAI) does not match.
var ErrInvalidCheckDigit = errors.New("gs1: invalid check digit")

// ErrInvalidDate indicates that a YYMMDD date field contains an impossible date.
var ErrInvalidDate = errors.New("gs1: invalid date")

// ErrMissingRequiredAI indicates that a barcode is missing an AI required by a regulator.
var ErrMissingRequiredAI = errors.New("gs1: missing required AI")

// ErrUnsupportedEPCScheme indicates that an EPC header or URI names a coding
// scheme this package does not decode.
var ErrUnsupportedEPCScheme = errors.New("gs1: unsupported EPC scheme")

// ErrInvalidAssociation indicates that Application Identifiers violate a
// required pairing, exclusion, or standalone rule.
var ErrInvalidAssociation = errors.New("gs1: invalid AI association")
