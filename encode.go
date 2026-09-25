package gs1

import (
	"fmt"
	"sort"
	"strings"
)

// Encode builds a scanner-equivalent GS1 element string from AI elements.
// Fixed-length elements are emitted first; variable-length elements are
// separated by FNC1, with a leading FNC1 marking the GS1 payload.
// Check digits are currently validated for AIs 01 and 02; SSCC and GLN check
// digit validation is deferred until those identifiers have dedicated support.
func Encode(elements []Element) (string, error) {
	if len(elements) == 0 {
		return "", ErrEmptyInput
	}
	ordered := append([]Element(nil), elements...)
	seen := make(map[string]struct{}, len(ordered))
	for i, element := range ordered {
		if err := validateEncodeElement(element, i, seen); err != nil {
			return "", err
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return isPredefinedLengthAI(ordered[i].AI) && !isPredefinedLengthAI(ordered[j].AI)
	})

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
	if spec.FixedLen == 0 && len(element.Value) == 0 {
		return fmt.Errorf("%w: AI (%s) data must not be empty", ErrInvalidData, element.AI)
	}
	if spec.FixedLen == 0 && len(element.Value) > spec.MaxLen {
		return fmt.Errorf("%w: AI (%s) data length %d exceeds max %d", ErrInvalidData, element.AI, len(element.Value), spec.MaxLen)
	}
	if err := validateData(element.Value, spec); err != nil {
		return err
	}
	if element.AI == "01" || element.AI == "02" {
		return ValidateGTIN(element.Value)
	}
	return nil
}

func validateEncodableValue(value, ai string) error {
	for i := 0; i < len(value); i++ {
		if value[i] <= 0x20 || value[i] > 0x7e || value[i] == '(' || value[i] == ')' {
			return fmt.Errorf("%w: AI (%s) contains unsupported character at position %d", ErrInvalidData, ai, i)
		}
	}
	return nil
}

// isPredefinedLengthAI reports whether GS1 table 7.8.5-2 permits omitting
// FNC1 after this AI when another element follows it.
func isPredefinedLengthAI(ai string) bool {
	if len(ai) < 2 {
		return false
	}
	prefix := int(ai[0]-'0')*10 + int(ai[1]-'0')
	return prefix <= 4 || prefix == 41 || prefix >= 11 && prefix <= 20 || prefix >= 31 && prefix <= 36
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
