package gs1

import (
	"fmt"
	"sort"
	"strings"
)

// Encode builds a scanner-equivalent GS1 element string from AI elements.
// Elements with a predefined length (GS1 General Specifications table
// 7.8.5-2) are emitted first, in their given order, so no separator follows
// them; the remaining elements keep their given order and are separated by
// FNC1. A leading FNC1 marks the GS1 payload. FNC1 is returned as the ASCII
// GS character (0x1D); use EncodeZPL for Zebra printers.
//
// Encode applies the same semantic rules as strict parsing: every value is
// checked against its AI format and the GS1 AI encodable character set 82,
// duplicate AIs are rejected, check digits are validated for every numeric
// GS1 key (the same set that ParseOptions.ValidateCheckDigits covers), and
// the AI association rules of Barcode.ValidateAssociations are enforced, so
// an element string that Encode accepts also parses with
// ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true}.
func Encode(elements []Element) (string, error) {
	ordered, err := orderForEncoding(elements)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(1 + len(ordered)*4)
	b.WriteByte(byte(fnc1))
	for i, element := range ordered {
		if i > 0 && !isPredefinedLengthAI(ordered[i-1].AI) {
			b.WriteByte(byte(fnc1))
		}
		b.WriteString(element.AI)
		b.WriteString(element.Value)
	}
	return b.String(), nil
}

// EncodeZPL builds the ^FD field data of a GS1-128 symbol for Zebra ZPL II
// printers from the same elements and rules as Encode. Every FNC1,
// including the leading one, is written as the Code 128 invocation code
// ">8", and a literal ">" in the data as "><". The result is meant for a
// ^BC command in automatic mode, for example:
//
//	^BCN,100,Y,N,N,A^FD>8010415000002112610LOT42^FS
//
// GS1 character set 82 contains neither "^" nor "~", so no other ZPL control
// characters can appear in the output.
func EncodeZPL(elements []Element) (string, error) {
	ordered, err := orderForEncoding(elements)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(2 + len(ordered)*5)
	b.WriteString(zplFNC1)
	for i, element := range ordered {
		if i > 0 && !isPredefinedLengthAI(ordered[i-1].AI) {
			b.WriteString(zplFNC1)
		}
		b.WriteString(element.AI)
		b.WriteString(strings.ReplaceAll(element.Value, ">", zplGreaterThan))
	}
	return b.String(), nil
}

// Code 128 invocation codes of the ZPL II ^BC command.
const (
	zplFNC1        = ">8"
	zplGreaterThan = "><"
)

// orderForEncoding validates the elements and returns a copy sorted for
// encoding: predefined-length AIs first, otherwise in the given order.
func orderForEncoding(elements []Element) ([]Element, error) {
	if len(elements) == 0 {
		return nil, ErrEmptyInput
	}
	ordered := append([]Element(nil), elements...)
	seen := make(map[string]struct{}, len(ordered))
	for i, element := range ordered {
		if err := validateEncodeElement(element, i, seen); err != nil {
			return nil, err
		}
	}
	if err := (Barcode{Elements: ordered}).ValidateAssociations(); err != nil {
		return nil, err
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return isPredefinedLengthAI(ordered[i].AI) && !isPredefinedLengthAI(ordered[j].AI)
	})
	return ordered, nil
}

func validateEncodeElement(element Element, index int, seen map[string]struct{}) error {
	spec, ok := aiTable[element.AI]
	if !ok {
		return fmt.Errorf("%w: unknown AI (%s) at element %d", ErrUnknownAI, element.AI, index)
	}
	if strings.ContainsRune(element.Value, fnc1) {
		return fmt.Errorf("%w: AI (%s) contains FNC1", ErrInvalidData, element.AI)
	}
	if _, exists := seen[element.AI]; exists {
		return fmt.Errorf("%w: duplicate AI (%s)", ErrInvalidData, element.AI)
	}
	seen[element.AI] = struct{}{}
	if err := validateEncodableValue(element.Value, element.AI); err != nil {
		return err
	}
	if spec.FixedLen > 0 && len(element.Value) != spec.FixedLen {
		return fmt.Errorf("%w: AI (%s) needs %d chars, got %d", ErrInvalidData, element.AI, spec.FixedLen, len(element.Value))
	}
	if len(element.Value) == 0 {
		return fmt.Errorf("%w: AI (%s) data must not be empty", ErrInvalidData, element.AI)
	}
	if len(element.Value) > spec.MaxLen {
		return fmt.Errorf("%w: AI (%s) data length %d exceeds max %d", ErrInvalidData, element.AI, len(element.Value), spec.MaxLen)
	}
	if err := validateData(element.Value, spec); err != nil {
		return err
	}
	return validateElementCheckDigit(element)
}

// validateEncodableValue restricts values to GS1 AI encodable character set
// 82 minus the parentheses, so that the HRI stays parseable.
func validateEncodableValue(value, ai string) error {
	for i := 0; i < len(value); i++ {
		if !inCSET82(value[i]) || value[i] == '(' || value[i] == ')' {
			return fmt.Errorf("%w: AI (%s) contains unsupported character at position %d", ErrInvalidData, ai, i)
		}
	}
	return nil
}

// isPredefinedLengthAI reports whether GS1 table 7.8.5-2 permits omitting
// FNC1 after this AI when another element follows it, as flagged in the
// Syntax Dictionary.
func isPredefinedLengthAI(ai string) bool {
	spec, ok := aiTable[ai]
	return ok && spec.Predefined
}

// ParseHRI splits a human-readable interpretation such as
// "(01)04150000021126(10)LOT42" into elements. Unlike Parse, it does not
// consult the AI table or apply scanner heuristics: the AI is exactly the
// text between each pair of parentheses and the value is everything up to
// the next "(", so "(999)1" reaches Encode as AI 999 and is rejected as
// ErrUnknownAI instead of being re-split as AI 99. It is the inverse of
// Barcode.HRI for encoder input typed by people or stored in label
// templates. Malformed input wraps ErrInvalidData.
func ParseHRI(hri string) ([]Element, error) {
	if len(hri) == 0 {
		return nil, ErrEmptyInput
	}
	if hri[0] != '(' {
		return nil, fmt.Errorf("%w: HRI must start with \"(\"", ErrInvalidData)
	}
	var elements []Element
	for pos := 0; pos < len(hri); {
		close := strings.IndexByte(hri[pos:], ')')
		if close < 0 {
			return nil, fmt.Errorf("%w: missing \")\" after position %d", ErrInvalidData, pos)
		}
		ai := hri[pos+1 : pos+close]
		if len(ai) < 2 || len(ai) > 4 || !isDigits(ai) {
			return nil, fmt.Errorf("%w: AI %q at position %d must be 2 to 4 digits", ErrInvalidData, ai, pos)
		}
		pos += close + 1
		end := strings.IndexByte(hri[pos:], '(')
		if end < 0 {
			end = len(hri) - pos
		}
		value := hri[pos : pos+end]
		if value == "" {
			return nil, fmt.Errorf("%w: AI (%s) has no value", ErrInvalidData, ai)
		}
		if strings.IndexByte(value, ')') >= 0 {
			return nil, fmt.Errorf("%w: AI (%s) value %q contains \")\"", ErrInvalidData, ai, value)
		}
		elements = append(elements, Element{AI: ai, Value: value})
		pos += end
	}
	return elements, nil
}

// HRI returns the human-readable interpretation of the barcode elements.
func (b Barcode) HRI() string {
	var out strings.Builder
	for _, element := range b.Elements {
		fmt.Fprintf(&out, "(%s)%s", element.AI, element.Value)
	}
	return out.String()
}

// String returns the human-readable interpretation of the barcode elements.
func (b Barcode) String() string {
	return b.HRI()
}
