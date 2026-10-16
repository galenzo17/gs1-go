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

// Warning identifies a non-fatal advisory found while parsing a barcode.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WarnDueDateAsExpiry warns that AI 12 is a due date, not an expiry date.
const WarnDueDateAsExpiry = "due_date_as_expiry"

var dueDateAsExpiryWarning = Warning{
	Code:    WarnDueDateAsExpiry,
	Message: "AI (12) is a due date; use AI (17) for expiration dates",
}

// Warnings returns non-fatal advisories found while parsing the barcode.
func (b Barcode) Warnings() []Warning {
	return append([]Warning(nil), b.warnings...)
}

// Barcode represents a fully parsed GS1 barcode (GS1-128 or DataMatrix).
type Barcode struct {
	Raw       string    // original input string
	Elements  []Element // parsed AI-value pairs in scan order
	Symbology Symbology // detected carrier, when an AIM prefix was present
	warnings  []Warning
}

// ParseOptions controls ambiguous carrier handling and optional validation.
// AssumeBareGTIN8 takes precedence over AI interpretation for unprefixed
// eight-digit numeric input. GTIN check-digit validation is opt-in.
type ParseOptions struct {
	// AssumeBareGTIN8 treats an unprefixed eight-digit numeric input as GTIN-8.
	AssumeBareGTIN8 bool
	// ValidateAssociations enforces AI association rules after parsing.
	ValidateAssociations bool
	// ValidateCheckDigits checks GTIN check digits for AI 01 and AI 02,
	// including bare GTINs and EAN/UPC and ITF-14 carrier payloads.
	// It is independent of ValidateAssociations and is disabled by default.
	ValidateCheckDigits bool
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

// DueDate returns the parsed due date (AI 12).
func (b Barcode) DueDate() (time.Time, error) {
	v, ok := b.Get("12")
	if !ok {
		return time.Time{}, fmt.Errorf("%w: AI (12) not present", ErrInvalidData)
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
	b.warnings = b.warnings[:0]
}

// Parse parses a GS1 barcode string (GS1-128 or DataMatrix scanner output)
// into a Barcode with typed elements. It handles AIM symbology identifiers
// (e.g., ]C1, ]d2), FNC1 separators (ASCII 29), and bracket notation
// (e.g., "(01)04150000021126(17)250630").
func Parse(input string) (Barcode, error) {
	return ParseWithOptions(input, ParseOptions{})
}

// ParseWithOptions parses a GS1 barcode with explicit ambiguity options and
// optional AI association and GTIN check-digit validation.
func ParseWithOptions(input string, opts ParseOptions) (Barcode, error) {
	b := Barcode{Elements: make([]Element, 0, 8)}
	if err := ParseIntoWithOptions(input, &b, opts); err != nil {
		return Barcode{}, err
	}
	return b, nil
}

// ParseInto parses a GS1 barcode string into an existing Barcode, reusing
// its allocated memory. Call b.Reset() before reuse to clear previous data.
// On error, the contents of b are unspecified and must not be used as a
// parsed result. Each goroutine must use its own Barcode.
func ParseInto(input string, b *Barcode) error {
	return ParseIntoWithOptions(input, b, ParseOptions{})
}

// ParseIntoWithOptions parses into an existing Barcode with explicit
// ambiguity options and optional semantic validation, reusing its allocated
// memory. Call b.Reset() before reuse to clear previous data. On error,
// the contents of b are unspecified and must not be used as a parsed result.
// Syntax, length, and data-type errors precede semantic errors;
// association errors precede check-digit errors when both are enabled.
// Check-digit error positions identify AI starts or carrier-payload starts
// in the normalized input, after scanner cleaning and bracket conversion.
func ParseIntoWithOptions(input string, b *Barcode, opts ParseOptions) error {
	if strings.TrimSpace(input) == "" {
		return ErrEmptyInput
	}

	data := cleanScannerInput(input)
	data = stripBracketNotation(data)

	b.Raw = input
	b.warnings = b.warnings[:0]
	// Detect bare GTIN (EAN-13, EAN-8, UPC-A, GTIN-14 without AI prefix).
	if isBareGTIN(data, opts.AssumeBareGTIN8) {
		padded := padGTIN(data)
		element := Element{AI: "01", Value: padded}
		b.Elements = append(b.Elements, element)
		if opts.ValidateCheckDigits {
			if err := validateElementCheckDigit(element); err != nil {
				return fmt.Errorf("at position 0: %w", err)
			}
		}
		return nil
	}

	pos, sym, aimCode := skipPrefix(data)
	b.Symbology = sym
	if done, err := parseCarrierElement(data, pos, sym, aimCode, b, opts.ValidateCheckDigits); done || err != nil {
		return err
	}

	checkDigitErr, err := parseAIElements(data, pos, b, opts.ValidateCheckDigits)
	if err != nil {
		return err
	}
	if len(b.Elements) == 0 {
		return ErrEmptyInput
	}
	if _, hasDueDate := b.Get("12"); hasDueDate {
		if _, hasExpirationDate := b.Get("17"); !hasExpirationDate {
			b.warnings = append(b.warnings, dueDateAsExpiryWarning)
		}
	}
	if opts.ValidateAssociations {
		if err := b.ValidateAssociations(); err != nil {
			return err
		}
	}
	return checkDigitErr
}

// parseAIElements defers the first check-digit error so all structure and
// association checks can take precedence without storing element offsets.
func parseAIElements(data string, pos int, b *Barcode, checkDigits bool) (checkDigitErr, parseErr error) {
	for pos < len(data) {
		if data[pos] == byte(fnc1) {
			pos++
			continue
		}

		aiStart := pos
		spec, aiLen, ok := lookupAIAt(data, pos)
		if !ok {
			return nil, fmt.Errorf("%w: at position %d", ErrUnknownAI, pos)
		}
		pos += aiLen

		value, newPos, err := extractData(data, pos, spec)
		if err != nil {
			return nil, err
		}
		pos = newPos

		if err := validateData(value, spec); err != nil {
			return nil, err
		}

		element := Element{AI: spec.AI, Value: value}
		b.Elements = append(b.Elements, element)
		if checkDigits && checkDigitErr == nil {
			if err := validateElementCheckDigit(element); err != nil {
				checkDigitErr = fmt.Errorf("at position %d: %w", aiStart, err)
			}
		}
	}
	return checkDigitErr, nil
}

// Validate is the umbrella for all semantic Application Identifier checks,
// including association rules and GTIN check digits for AI 01 and AI 02.
// It checks the current Elements without reparsing Raw. Association errors
// take precedence over check-digit errors.
func (b Barcode) Validate() error {
	if err := b.ValidateAssociations(); err != nil {
		return err
	}
	for _, element := range b.Elements {
		if err := validateElementCheckDigit(element); err != nil {
			return err
		}
	}
	return nil
}

func validateElementCheckDigit(element Element) error {
	if element.AI == "01" || element.AI == "02" {
		if err := ValidateGTIN(element.Value); err != nil {
			return fmt.Errorf("AI (%s): %w", element.AI, err)
		}
	}
	return nil
}

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

func parseCarrierElement(data string, pos int, sym Symbology, code byte, b *Barcode, checkDigits bool) (bool, error) {
	if sym != SymEANUPC && sym != SymITF14 {
		return false, nil
	}
	value, ok, err := parseCarrierPayload(data[pos:], sym, code)
	if err != nil || !ok {
		return true, err
	}
	if checkDigits {
		if err := validateCarrierCheckDigit(data[pos:], value, sym, code); err != nil {
			return true, fmt.Errorf("at position %d: AI (01): %w", pos, err)
		}
	}
	b.Elements = append(b.Elements, Element{AI: "01", Value: value})
	return true, nil
}

func validateCarrierCheckDigit(payload, gtin string, sym Symbology, code byte) error {
	// UPC-E expansion recomputes the digit, so check the received digit before
	// the original payload is discarded. A stripped ITF-14 digit is synthesized.
	if sym == SymEANUPC && code == '0' && len(payload) == 8 && payload[0] == '0' {
		expected := gtin[len(gtin)-1]
		if payload[7] != expected {
			return fmt.Errorf("%w: expected '%c', got '%c'", ErrInvalidCheckDigit, expected, payload[7])
		}
	}
	return ValidateGTIN(gtin)
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
func extractData(data string, pos int, spec *aiSpec) (string, int, error) {
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

// validateData applies the AI's components to the value in order, each
// consuming a prefix of the prescribed length, and checks the character set
// of every component. Optional components are skipped once the data is
// exhausted; data left after the last component is an error.
func validateData(value string, spec *aiSpec) error {
	pos := 0
	for _, c := range spec.Components {
		remaining := len(value) - pos
		if remaining == 0 && c.Optional {
			break
		}
		n := c.Len
		if c.Variable {
			n = min(remaining, c.Len)
			if n == 0 {
				return fmt.Errorf("%w: AI (%s) needs at least %d characters, got %d",
					ErrInvalidData, spec.AI, spec.MinLen, len(value))
			}
		} else if remaining < n {
			return fmt.Errorf("%w: AI (%s) needs at least %d characters, got %d",
				ErrInvalidData, spec.AI, spec.MinLen, len(value))
		}
		segment := value[pos : pos+n]
		if err := validateCharset(segment, c.Type, spec); err != nil {
			return err
		}
		for _, linter := range c.Linters {
			if linter == "zero" && !isZeros(segment) {
				return fmt.Errorf("%w: AI (%s) must start with %q, got %q",
					ErrInvalidData, spec.AI, "0", segment)
			}
		}
		pos += n
	}
	if pos < len(value) {
		return fmt.Errorf("%w: AI (%s) data length %d exceeds format %s",
			ErrInvalidData, spec.AI, len(value), spec.format())
	}
	return nil
}

func validateCharset(segment string, t dataType, spec *aiSpec) error {
	for i := 0; i < len(segment); i++ {
		if !inCharset(segment[i], t) {
			return fmt.Errorf("%w: AI (%s) expects %s data, got %q",
				ErrInvalidData, spec.AI, t.description(), segment)
		}
	}
	return nil
}

// inCharset reports whether c belongs to the dictionary character set t.
// Set 82 is checked by Encode; the parser accepts any byte for X so that
// scanner output with stray characters still yields structured elements.
func inCharset(c byte, t dataType) bool {
	switch t {
	case dataNumeric:
		return isDigit(c)
	case dataCSET39:
		return isDigit(c) || isUpper(c) || c == '#' || c == '-' || c == '/'
	case dataBase64URL:
		return isDigit(c) || isUpper(c) || c >= 'a' && c <= 'z' || c == '-' || c == '_'
	}
	return true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isUpper(c byte) bool { return c >= 'A' && c <= 'Z' }

func (t dataType) description() string {
	switch t {
	case dataNumeric:
		return "numeric"
	case dataCSET39:
		return "character set 39"
	case dataBase64URL:
		return "base64url"
	}
	return "character set 82"
}

func isZeros(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return false
		}
	}
	return true
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
	// A leading pair that is an AI whose data could fit in the remaining
	// digits makes the input ambiguous; the check digit then decides.
	if n >= 2 {
		if spec, ok := aiTable[data[0:2]]; ok && (spec.FixedLen == 0 || spec.FixedLen <= n-2) {
			return ValidateGTIN(data) == nil
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
// can be fully parsed, it prefers the split that yields the fewest
// elements, since a single dropped separator leaves one trailing element,
// and then the candidate closest to the midpoint of the search range. This
// "balanced split" avoids false positives when AI codes like "21" appear
// inside lot numbers (e.g., "HC23L25212800" contains "21" at multiple
// positions), and the element count keeps digit runs from being read as
// chains of short numeric AIs.
func findAIBoundary(data string, from, to int) (int, bool) {
	mid := (from + to) / 2
	best := boundaryCandidate{pos: -1, dist: len(data)}
	for i := from; i < to; i++ {
		spec, aiLen, ok := lookupAIAt(data, i)
		if !ok || !plausibleAIData(data, i+aiLen, spec) {
			continue
		}
		count, ok := canParseFrom(data, i)
		if !ok {
			continue
		}
		candidate := boundaryCandidate{pos: i, dist: max(i-mid, mid-i), aiLen: aiLen, count: count}
		if best.pos < 0 || candidate.betterThan(best) {
			best = candidate
		}
	}
	if best.pos >= 0 {
		return best.pos, true
	}
	return 0, false
}

// boundaryCandidate is one possible AI boundary during missing-FNC1
// recovery.
type boundaryCandidate struct {
	pos, dist, aiLen, count int
}

// betterThan ranks candidates: fewer resulting elements first, then a
// 2-digit AI over a longer one, then the position nearest the midpoint of
// the search range, and on a tie the later position.
func (c boundaryCandidate) betterThan(best boundaryCandidate) bool {
	if c.count != best.count {
		return c.count < best.count
	}
	if c.aiLen == 2 && best.aiLen > 2 {
		return true
	}
	if c.aiLen != best.aiLen {
		return false
	}
	return c.dist < best.dist || (c.dist == best.dist && c.pos > best.pos)
}

// plausibleAIData checks whether the data after a candidate AI looks valid.
func plausibleAIData(data string, dataStart int, spec *aiSpec) bool {
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
// that the remaining data contains valid AI-value pairs, and counts them.
// No FNC1 recovery is attempted in the dry run — only exact matches.
func canParseFrom(data string, pos int) (int, bool) {
	count := 0
	for pos < len(data) {
		if data[pos] == byte(fnc1) {
			pos++
			continue
		}
		spec, aiLen, ok := lookupAIAt(data, pos)
		if !ok {
			return 0, false
		}
		pos += aiLen
		if !validAIDataPrefix(data, pos, spec) {
			return 0, false
		}
		if spec.FixedLen > 0 {
			if !validFixedField(data, pos, spec) {
				return 0, false
			}
			pos += spec.FixedLen
		} else {
			end := pos
			for end < len(data) && data[end] != byte(fnc1) {
				end++
			}
			if end == pos || end-pos > spec.MaxLen {
				return 0, false
			}
			pos = end
			if pos < len(data) && data[pos] == byte(fnc1) {
				pos++
			}
		}
		count++
	}
	return count, true
}

// validAIDataPrefix reports whether the data after a candidate AI fits the
// AI's leading fixed components, so that missing-FNC1 recovery does not
// split on digits that merely look like an AI code.
func validAIDataPrefix(data string, dataStart int, spec *aiSpec) bool {
	pos := dataStart
	for _, c := range spec.Components {
		if c.Variable || c.Optional {
			break
		}
		if pos+c.Len > len(data) {
			return false
		}
		segment := data[pos : pos+c.Len]
		for i := 0; i < len(segment); i++ {
			if !inCharset(segment[i], c.Type) {
				return false
			}
		}
		for _, linter := range c.Linters {
			if linter == "zero" && !isZeros(segment) {
				return false
			}
		}
		pos += c.Len
	}
	return true
}

func validFixedField(data string, pos int, spec *aiSpec) bool {
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
