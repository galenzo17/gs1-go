// Package gs1 parses and validates GS1 Application Identifier element
// strings as produced by barcode scanners reading GS1-128, GS1 DataMatrix,
// GS1 QR Code and GS1 DataBar symbols.
//
// The package is dependency-free, allocation-conscious, and designed for the
// data-capture edge: point-of-care dispensing, warehouse receiving, and
// pharmaceutical traceability systems where scanner output is noisy and
// throughput matters.
//
// # Terminology
//
// An Application Identifier (AI) is a two- to four-digit prefix that
// declares the meaning and format of the data that follows it, as defined
// in the GS1 General Specifications. A sequence of AI/value pairs is an
// element string. Variable-length fields are terminated by the FNC1
// character, which scanners transmit as ASCII 29 (Group Separator).
//
// # Parsing
//
// Parse accepts raw scanner output, bracketed human-readable form, and
// AIM symbology identifiers:
//
//	b, err := gs1.Parse("]d20104150000021126172506302112345ABC\x1D10LOT42X")
//	if err != nil {
//		return err
//	}
//	fmt.Println(b.GTIN(), b.Lot(), b.SerialNumber())
//
// ParseInto reuses a caller-owned Barcode for zero-allocation parsing in
// hot loops. Each goroutine must own its Barcode.
// The detected AIM carrier is available as Barcode.Symbology; use
// ParseWithOptions to opt in to ambiguous bare GTIN-8 input.
//
// # Validation
//
// Parsing checks structure only. ValidateGTIN verifies the modulo-10 check
// digit, ParseDate validates YYMMDD dates, and Regulator profiles check that
// the AIs mandated by a national pharmaceutical regulator are present.
//
// # Trademark
//
// GS1 is a registered trademark of GS1 AISBL. This project is an
// independent implementation of publicly available specifications and is
// not affiliated with or endorsed by GS1.
package gs1
