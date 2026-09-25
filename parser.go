package gs1

import (
	"fmt"
	"strings"
	"time"
)

const fnc1 = '\x1D' // GS (Group Separator), used as FNC1 in scanner output

// Element represents a single AI-value pair extracted from a GS1 barcode.
type Element struct {
	AI    string // application identifier code, e.g., "01"
	Value string // raw data value
}

// Symbology identifies the carrier reported by an AIM symbology identifier.
type Symbology uint8

const (
	// SymUnknown indicates that no recognized AIM prefix was present.
	SymUnknown Symbology = iota
	// SymGS1128 identifies a GS1-128 carrier.
	SymGS1128
	// SymDataMatrix identifies a GS1 DataMatrix carrier.
	SymDataMatrix
	// SymQR identifies a GS1 QR carrier.
	SymQR
	// SymEANUPC identifies an EAN/UPC carrier.
	SymEANUPC
	// SymITF14 identifies an ITF-14 carrier.
	SymITF14
	// SymDataBar is reserved for a future DataBar distinction. AIM ]e0 is
	// shared by DataBar and composite carriers, so this parser cannot emit it.
	SymDataBar
	// SymComposite identifies a GS1 composite carrier.
	SymComposite
	// SymDotCode identifies a GS1 DotCode carrier.
	SymDotCode
)

func (s Symbology) String() string {
	switch s {
	case SymGS1128:
		return "GS1-128"
	case SymDataMatrix:
		return "Data Matrix"
	case SymQR:
		return "QR Code"
	case SymEANUPC:
		return "EAN/UPC"
	case SymITF14:
		return "ITF-14"
	case SymDataBar:
		return "DataBar"
	case SymComposite:
		return "Composite"
	case SymDotCode:
		return "DotCode"
	default:
		return "unknown"
	}
}

// Barcode represents a fully parsed GS1 barcode (GS1-128 or DataMatrix).
type Barcode struct {
	Raw       string    // original input string
	Elements  []Element // parsed AI-value pairs in scan order
	Symbology Symbology // detected carrier, when an AIM prefix was present
}

// ParseOptions controls ambiguous carrier handling and optional validation.
// AssumeBareGTIN8 takes precedence over AI interpretation for unprefixed
// eight-digit numeric input; parsing does not validate the GTIN check digit.
type ParseOptions struct {
	// AssumeBareGTIN8 treats an unprefixed eight-digit numeric input as GTIN-8.
	AssumeBareGTIN8 bool
	// ValidateAssociations enforces AI association rules after parsing.
	ValidateAssociations bool
}

// GTIN returns the GTIN value (AI 01), or "" if not present.
func (b Barcode) GTIN() string {
	v, _ := b.Get("01")
	return v
}

// Lot returns the batch/lot number (AI 10), or "" if not present.
func (b Barcode) Lot() string {
	v, _ := b.Get("10")
	return v
}

// SerialNumber returns the serial number (AI 21), or "" if not present.
func (b Barcode) SerialNumber() string {
	v, _ := b.Get("21")
	return v
}

// SSCC returns the SSCC value (AI 00), or "" if not present.
func (b Barcode) SSCC() string {
	v, _ := b.Get("00")
	return v
}

// GDTI returns the document type identifier (AI 253), or "" if absent.
func (b Barcode) GDTI() string {
	v, _ := b.Get("253")
	return v
}

// GRAI returns the returnable asset identifier (AI 8003), or "" if absent.
func (b Barcode) GRAI() string {
	v, _ := b.Get("8003")
	return v
}

// Count returns the item count (AI 30), or "" if not present.
func (b Barcode) Count() string {
	v, _ := b.Get("30")
	return v
}

// ContentGTIN returns the contained-item GTIN value (AI 02), or "" if not present.
func (b Barcode) ContentGTIN() string {
	v, _ := b.Get("02")
	return v
}

// CountOfTradeItems returns the count of trade items (AI 37), or "" if not present.
func (b Barcode) CountOfTradeItems() string {
	v, _ := b.Get("37")
	return v
}

// GLN returns the Global Location Number (AI 414), or "" if not present.
func (b Barcode) GLN() string {
	v, _ := b.Get("414")
	return v
}

// GSIN returns the Global Shipment Identification Number (AI 402), or "" if not present.
func (b Barcode) GSIN() string {
	v, _ := b.Get("402")
	return v
}

// ExpirationDate returns the parsed expiration date (AI 17).
func (b Barcode) ExpirationDate() (time.Time, error) {
	v, ok := b.Get("17")
	if !ok {
		return time.Time{}, fmt.Errorf("%w: AI (17) not present", ErrInvalidData)
	}
	return ParseDate(v)
}

// ProductionDate returns the parsed production date (AI 11).
func (b Barcode) ProductionDate() (time.Time, error) {
	v, ok := b.Get("11")
	if !ok {
		return time.Time{}, fmt.Errorf("%w: AI (11) not present", ErrInvalidData)
	}
	return ParseDate(v)
}

// PackagingDate returns the parsed packaging date (AI 13).
func (b Barcode) PackagingDate() (time.Time, error) {
	v, ok := b.Get("13")
	if !ok {
		return time.Time{}, fmt.Errorf("%w: AI (13) not present", ErrInvalidData)
	}
	return ParseDate(v)
}

// BestBeforeDate returns the parsed best-before date (AI 15).
func (b Barcode) BestBeforeDate() (time.Time, error) {
	v, ok := b.Get("15")
	if !ok {
		return time.Time{}, fmt.Errorf("%w: AI (15) not present", ErrInvalidData)
	}
	return ParseDate(v)
}

// Get returns the value for the given AI code and whether it was found.
// If the AI appears multiple times, the first occurrence is returned.
func (b Barcode) Get(ai string) (string, bool) {
	for i := range b.Elements {
		if b.Elements[i].AI == ai {
			return b.Elements[i].Value, true
		}
	}
	return "", false
}

// Reset clears a Barcode for reuse, retaining allocated memory.
func (b *Barcode) Reset() {
	b.Raw = ""
	b.Elements = b.Elements[:0]
	b.Symbology = SymUnknown
}

// Parse parses a GS1 barcode string (GS1-128 or DataMatrix scanner output)
// into a Barcode with typed elements. It handles AIM symbology identifiers
// (e.g., ]C1, ]d2), FNC1 separators (ASCII 29), and bracket notation
// (e.g., "(01)04150000021126(17)250630").
func Parse(input string) (Barcode, error) {
	return ParseWithOptions(input, ParseOptions{})
}

// ParseWithOptions parses a GS1 barcode with explicit ambiguity options and
// optional AI association rules.
func ParseWithOptions(input string, opts ParseOptions) (Barcode, error) {
	b := Barcode{Elements: make([]Element, 0, 8)}
	if err := ParseIntoWithOptions(input, &b, opts); err != nil {
		return Barcode{}, err
	}
	return b, nil
}

// ParseInto parses a GS1 barcode string into an existing Barcode, reusing
// its allocated memory. Call b.Reset() before reuse to clear previous data.
// Each goroutine must use its own Barcode.
func ParseInto(input string, b *Barcode) error {
	return ParseIntoWithOptions(input, b, ParseOptions{})
}

// ParseIntoWithOptions parses into an existing Barcode with explicit
// ambiguity options and optional AI association validation, reusing its
// allocated memory.
func ParseIntoWithOptions(input string, b *Barcode, opts ParseOptions) error {
	if strings.TrimSpace(input) == "" {
		return ErrEmptyInput
	}

	data := cleanScannerInput(input)
	data = stripBracketNotation(data)

	b.Raw = input
	// Detect bare GTIN (EAN-13, EAN-8, UPC-A, GTIN-14 without AI prefix).
	if isBareGTIN(data, opts.AssumeBareGTIN8) {
		padded := padGTIN(data)
		b.Elements = append(b.Elements, Element{AI: "01", Value: padded})
		return nil
	}

	pos, sym, aimCode := skipPrefix(data)
	b.Symbology = sym
	if done, err := parseCarrierElement(data, pos, sym, aimCode, b); done || err != nil {
		return err
	}

	for pos < len(data) {
		if data[pos] == byte(fnc1) {
			pos++
			continue
		}

		spec, aiLen, ok := lookupAIAt(data, pos)
		if !ok {
			return fmt.Errorf("%w: at position %d", ErrUnknownAI, pos)
		}
		pos += aiLen

		value, newPos, err := extractData(data, pos, spec)
		if err != nil {
			return err
		}
		pos = newPos

		if err := validateData(value, spec); err != nil {
			return err
		}

		b.Elements = append(b.Elements, Element{AI: spec.AI, Value: value})
	}

	if len(b.Elements) == 0 {
		return ErrEmptyInput
	}
	if opts.ValidateAssociations {
		return b.ValidateAssociations()
	}

	return nil
}

// Validate is the umbrella for all semantic Application Identifier checks,
// including association rules.
func (b Barcode) Validate() error { return b.ValidateAssociations() }

// skipPrefix skips leading FNC1 and AIM symbology identifiers.
func skipPrefix(data string) (int, Symbology, byte) {
	pos := 0
	if pos < len(data) && data[pos] == byte(fnc1) {
		pos++
	}
	var sym Symbology
	var code byte
	if pos < len(data) && data[pos] == ']' && pos+3 <= len(data) {
		aimSym := data[pos+1]
		code = data[pos+2]
		sym = symbologyForAIM(aimSym)
		if isAIMPrefix(aimSym) {
			pos += 3
		}
	}
	return pos, sym, code
}

func symbologyForAIM(code byte) Symbology {
	switch code {
	case 'C':
		return SymGS1128
	case 'd':
		return SymDataMatrix
	case 'Q':
		return SymQR
	case 'e':
		return SymComposite
	case 'J':
		return SymDotCode
	case 'E':
		return SymEANUPC
	case 'I':
		return SymITF14
	default:
		return SymUnknown
	}
}

func isAIMPrefix(code byte) bool {
	return strings.ContainsRune("CdQeJEIAX", rune(code))
}

func parseCarrierElement(data string, pos int, sym Symbology, code byte, b *Barcode) (bool, error) {
	if sym != SymEANUPC && sym != SymITF14 {
		return false, nil
	}
	value, ok, err := parseCarrierPayload(data[pos:], sym, code)
	if err != nil || !ok {
		return true, err
	}
	b.Elements = append(b.Elements, Element{AI: "01", Value: value})
	return true, nil
}

func parseCarrierPayload(data string, sym Symbology, code byte) (string, bool, error) {
	if !isDigits(data) {
		return "", false, fmt.Errorf("%w: symbology payload must be numeric", ErrInvalidData)
	}
	if sym == SymITF14 {
		if code == '2' && len(data) == 13 {
			check, err := ComputeGTINCheckDigit(data)
			if err != nil {
				return "", false, err
			}
			data += string(check)
		}
		return fixedCarrierPayload(data, 14, "ITF-14")
	}
	switch code {
	case '4':
		return fixedCarrierPayload(data, 8, "EAN-8")
	case '0':
		if len(data) == 8 && data[0] == '0' {
			upca, err := ExpandUPCE(data)
			if err != nil {
				return "", false, err
			}
			return padGTIN(upca), true, nil
		}
		return variableCarrierPayload(data)
	case '1', '2':
		return variableCarrierPayload(data)
	default:
		return "", false, fmt.Errorf("%w: unsupported EAN/UPC symbology ]E%c", ErrInvalidData, code)
	}
}

func fixedCarrierPayload(data string, length int, name string) (string, bool, error) {
	if len(data) != length {
		return "", false, fmt.Errorf("%w: %s payload must be %d digits, got %d", ErrInvalidData, name, length, len(data))
	}
	return padGTIN(data), true, nil
}

func variableCarrierPayload(data string) (string, bool, error) {
	if len(data) != 12 && len(data) != 13 && len(data) != 14 {
		return "", false, fmt.Errorf("%w: EAN/UPC payload length %d not supported", ErrInvalidData, len(data))
	}
	return padGTIN(data), true, nil
}

// extractData reads the data field for an AI starting at pos.
func extractData(data string, pos int, spec aiSpec) (string, int, error) {
	if spec.FixedLen > 0 {
		if pos+spec.FixedLen > len(data) {
			return "", pos, fmt.Errorf("%w: AI (%s) needs %d chars, got %d",
				ErrTruncatedData, spec.AI, spec.FixedLen, len(data)-pos)
		}
		return data[pos : pos+spec.FixedLen], pos + spec.FixedLen, nil
	}

	end := pos
	for end < len(data) && data[end] != byte(fnc1) {
		end++
	}
	value := data[pos:end]
	if len(value) > spec.MaxLen {
		// Missing FNC1 recovery: try to find a known AI embedded in the
		// data. Scanners sometimes omit FNC1 between variable-length
		// fields, causing the next AI code to be read as part of the value.
		searchEnd := end
		if pos+spec.MaxLen < searchEnd {
			searchEnd = pos + spec.MaxLen
		}
		if split, ok := findAIBoundary(data, pos+1, searchEnd); ok {
			value = data[pos:split]
			end = split
		} else {
			return "", pos, fmt.Errorf("%w: AI (%s) data length %d exceeds max %d",
				ErrInvalidData, spec.AI, len(value), spec.MaxLen)
		}
	}
	if len(value) == 0 {
		return "", pos, fmt.Errorf("%w: AI (%s) has empty data", ErrInvalidData, spec.AI)
	}
	newPos := end
	if newPos < len(data) && data[newPos] == byte(fnc1) {
		newPos++
	}
	return value, newPos, nil
}

// validateData checks that the value conforms to the AI's data type.
func validateData(value string, spec aiSpec) error {
	if spec.DataType == dataUIC {
		if len(value) != 4 || value[0] < '0' || value[0] > '9' {
			return fmt.Errorf("%w: AI (%s) expects N1 + X3 data, got %q",
				ErrInvalidData, spec.AI, value)
		}
		return nil
	}
	if err := validateNumericPrefix(value, spec); err != nil {
		return err
	}
	if spec.DataType != dataNumeric {
		return nil
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return fmt.Errorf("%w: AI (%s) expects numeric data, got %q",
				ErrInvalidData, spec.AI, value)
		}
	}
	return nil
}

func validateNumericPrefix(value string, spec aiSpec) error {
	if spec.NumericPrefixLen == 0 {
		return nil
	}
	if len(value) < spec.NumericPrefixLen {
		return fmt.Errorf("%w: AI (%s) needs at least %d characters, got %d",
			ErrInvalidData, spec.AI, spec.NumericPrefixLen, len(value))
	}
	if spec.FirstChar != 0 && value[0] != spec.FirstChar {
		return fmt.Errorf("%w: AI (%s) must start with %q, got %q",
			ErrInvalidData, spec.AI, spec.FirstChar, value[0])
	}
	for i := 0; i < spec.NumericPrefixLen; i++ {
		if value[i] < '0' || value[i] > '9' {
			return fmt.Errorf("%w: AI (%s) expects a numeric prefix, got %q",
				ErrInvalidData, spec.AI, value)
		}
	}
	return nil
}

// stripBracketNotation converts bracket notation "(01)0415...(17)250630"
// to raw AI string with FNC1 separators between fields. If no brackets are
// found, returns the input unchanged.
func stripBracketNotation(input string) string {
	if !strings.Contains(input, "(") {
		return input
	}
	var b strings.Builder
	b.Grow(len(input))
	first := true
	i := 0
	for i < len(input) {
		if input[i] == '(' {
			// Insert FNC1 before each AI except the first, so the parser
			// can detect field boundaries for variable-length AIs.
			if !first {
				b.WriteByte(byte(fnc1))
			}
			first = false
			i++ // skip '('
			for i < len(input) && input[i] != ')' {
				b.WriteByte(input[i])
				i++
			}
			if i < len(input) {
				i++ // skip ')'
			}
		} else {
			b.WriteByte(input[i])
			i++
		}
	}
	return b.String()
}

// isBareGTIN reports whether data looks like a standalone GTIN without AI
// prefix (e.g., EAN-13 "7800038041425", EAN-8 "96385074", UPC-A, GTIN-14).
// It returns false if the data could be parsed as AI-prefixed data.
func isBareGTIN(data string, assumeGTIN8 bool) bool {
	n := len(data)
	// Only detect 12, 13, 14 digit bare GTINs. EAN-8 (8 digits) is too
	// ambiguous with 2-digit AI codes and is rare in healthcare.
	if n != 8 && n != 12 && n != 13 && n != 14 {
		return false
	}
	if !isDigits(data) {
		return false
	}
	if n == 8 && !assumeGTIN8 {
		return false
	}
	// If the first 2 digits match a known AI, treat as AI-prefixed data unless
	// the caller explicitly selected the ambiguous bare GTIN-8 interpretation.
	if n == 8 && assumeGTIN8 {
		return true
	}
	if n >= 2 {
		if _, ok := aiTable[data[0:2]]; ok {
			return false
		}
	}
	return true
}

func isDigits(data string) bool {
	if len(data) == 0 {
		return false
	}
	for i := 0; i < len(data); i++ {
		if data[i] < '0' || data[i] > '9' {
			return false
		}
	}
	return true
}

// findAIBoundary scans data[from:to] for the best AI boundary when FNC1
// separators are missing. Among all candidates where the remaining data
// can be fully parsed, it picks the one closest to the midpoint of the
// search range. This "balanced split" heuristic avoids false positives
// when AI codes like "21" appear inside lot numbers (e.g., "HC23L25212800"
// contains "21" at multiple positions).
func findAIBoundary(data string, from, to int) (int, bool) {
	mid := (from + to) / 2
	bestPos := -1
	bestDist := len(data)
	bestAILen := 0
	for i := from; i < to; i++ {
		spec, aiLen, ok := lookupAIAt(data, i)
		if !ok {
			continue
		}
		if !plausibleAIData(data, i+aiLen, spec) {
			continue
		}
		if !canParseFrom(data, i) {
			continue
		}
		dist := i - mid
		if dist < 0 {
			dist = -dist
		}
		preferShortAI := aiLen == 2 && bestAILen > 2
		preferCandidate := bestAILen == 0 || preferShortAI ||
			(aiLen == bestAILen && (dist < bestDist || (dist == bestDist && i > bestPos)))
		if preferCandidate {
			bestPos = i
			bestDist = dist
			bestAILen = aiLen
		}
	}
	if bestPos >= 0 {
		return bestPos, true
	}
	return 0, false
}

// plausibleAIData checks whether the data after a candidate AI looks valid.
func plausibleAIData(data string, dataStart int, spec aiSpec) bool {
	if !validAIDataPrefix(data, dataStart, spec) {
		return false
	}
	if spec.FixedLen > 0 {
		if dataStart+spec.FixedLen > len(data) {
			return false
		}
		if spec.DataType == dataNumeric {
			for j := dataStart; j < dataStart+spec.FixedLen; j++ {
				if data[j] < '0' || data[j] > '9' {
					return false
				}
			}
		}
		return true
	}
	// Variable-length: at least 1 char of data must follow.
	return dataStart < len(data)
}

// canParseFrom does a dry-run parse from pos to end-of-string to verify
// that the remaining data contains valid AI-value pairs. No FNC1 recovery
// is attempted in the dry run — only exact matches.
func canParseFrom(data string, pos int) bool {
	for pos < len(data) {
		if data[pos] == byte(fnc1) {
			pos++
			continue
		}
		spec, aiLen, ok := lookupAIAt(data, pos)
		if !ok {
			return false
		}
		pos += aiLen
		if !validAIDataPrefix(data, pos, spec) {
			return false
		}
		if spec.FixedLen > 0 {
			if !validFixedField(data, pos, spec) {
				return false
			}
			pos += spec.FixedLen
		} else {
			end := pos
			for end < len(data) && data[end] != byte(fnc1) {
				end++
			}
			if end == pos || end-pos > spec.MaxLen {
				return false
			}
			pos = end
			if pos < len(data) && data[pos] == byte(fnc1) {
				pos++
			}
		}
	}
	return true
}

func validAIDataPrefix(data string, dataStart int, spec aiSpec) bool {
	if spec.NumericPrefixLen > 0 {
		if dataStart+spec.NumericPrefixLen > len(data) {
			return false
		}
		for i := dataStart; i < dataStart+spec.NumericPrefixLen; i++ {
			if data[i] < '0' || data[i] > '9' {
				return false
			}
		}
	}
	return spec.FirstChar == 0 || (dataStart < len(data) && data[dataStart] == spec.FirstChar)
}

func validFixedField(data string, pos int, spec aiSpec) bool {
	if pos+spec.FixedLen > len(data) {
		return false
	}
	if spec.DataType == dataNumeric {
		for j := pos; j < pos+spec.FixedLen; j++ {
			if data[j] < '0' || data[j] > '9' {
				return false
			}
		}
	}
	return true
}

// padGTIN left-pads a GTIN to 14 digits with zeros (GTIN-14 canonical form).
func padGTIN(gtin string) string {
	for len(gtin) < 14 {
		gtin = "0" + gtin
	}
	return gtin
}
