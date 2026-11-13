package gs1

import "fmt"

// KeyType identifies a family of numeric GS1 identification keys that end
// in a modulo-10 check digit. Families that share a length, such as GTIN-13
// and GLN, cannot be told apart from the digits alone; KeyTypes reports
// every family a value can belong to.
type KeyType int

const (
	// KeyUnknown is returned for lengths that no supported key family uses.
	KeyUnknown KeyType = iota
	// KeyGTIN is a Global Trade Item Number: GTIN-8, GTIN-12, GTIN-13 or GTIN-14.
	KeyGTIN
	// KeySSCC is an 18-digit Serial Shipping Container Code, AI (00).
	KeySSCC
	// KeyGLN is a 13-digit Global Location Number, AIs (410) to (417).
	KeyGLN
	// KeyGSIN is a 17-digit Global Shipment Identification Number, AI (402).
	KeyGSIN
	// KeyGSRN is an 18-digit Global Service Relation Number, AIs (8017) and (8018).
	KeyGSRN
)

// String returns the GS1 abbreviation of the key family.
func (k KeyType) String() string {
	switch k {
	case KeyGTIN:
		return "GTIN"
	case KeySSCC:
		return "SSCC"
	case KeyGLN:
		return "GLN"
	case KeyGSIN:
		return "GSIN"
	case KeyGSRN:
		return "GSRN"
	}
	return "unknown"
}

// ComputeCheckDigit computes the GS1 modulo-10 check digit for the data
// digits of any numeric GS1 key (GS1 General Specifications section 7.9).
// partial holds every digit except the check digit and must be between 1
// and 30 digits long. The result is returned as a byte ('0'-'9').
func ComputeCheckDigit(partial string) (byte, error) {
	if len(partial) == 0 {
		return 0, fmt.Errorf("%w: partial key must not be empty", ErrInvalidData)
	}
	return computeCheckDigit(partial)
}

// ValidateSSCC validates an 18-digit Serial Shipping Container Code,
// including its modulo-10 check digit.
func ValidateSSCC(sscc string) error {
	return validateFixedKey("SSCC", sscc, 18)
}

// ValidateGLN validates a 13-digit Global Location Number, including its
// modulo-10 check digit.
func ValidateGLN(gln string) error {
	return validateFixedKey("GLN", gln, 13)
}

// ValidateGSIN validates a 17-digit Global Shipment Identification Number,
// including its modulo-10 check digit.
func ValidateGSIN(gsin string) error {
	return validateFixedKey("GSIN", gsin, 17)
}

// ValidateGSRN validates an 18-digit Global Service Relation Number,
// including its modulo-10 check digit.
func ValidateGSRN(gsrn string) error {
	return validateFixedKey("GSRN", gsrn, 18)
}

// validateFixedKey enforces the length and digit constraints of a numeric
// key before checking its modulo-10 check digit. Length and data errors wrap
// ErrInvalidData; a check-digit mismatch wraps ErrInvalidCheckDigit.
func validateFixedKey(label, key string, length int) error {
	if len(key) != length {
		return fmt.Errorf("%w: %s must be %d digits, got %d", ErrInvalidData, label, length, len(key))
	}
	if err := validateDigits(label, key); err != nil {
		return err
	}
	return ValidateCheckDigit(key)
}

// KeyTypes reports the key families that a numeric string of this length
// can represent, in the order GTIN, SSCC, GLN, GSIN, GSRN. GTIN-13 and GLN
// share 13 digits, and SSCC and GSRN share 18, so those lengths yield two
// candidates. Lengths used by no family return nil. The digits themselves
// are not validated; use ValidateKey for that.
func KeyTypes(key string) []KeyType {
	switch len(key) {
	case 8, 12, 14:
		return []KeyType{KeyGTIN}
	case 13:
		return []KeyType{KeyGTIN, KeyGLN}
	case 17:
		return []KeyType{KeyGSIN}
	case 18:
		return []KeyType{KeySSCC, KeyGSRN}
	}
	return nil
}

// ValidateKey validates the modulo-10 check digit of a numeric GS1 key whose
// family is not known in advance, such as a value typed by an operator. It
// returns the families the length admits (see KeyTypes). The error wraps
// ErrInvalidData when the length matches no family or the value is not all
// digits, and ErrInvalidCheckDigit when the check digit does not match.
func ValidateKey(key string) ([]KeyType, error) {
	types := KeyTypes(key)
	if types == nil {
		return nil, fmt.Errorf("%w: length %d matches no GS1 key (expected 8, 12, 13, 14, 17, or 18 digits)", ErrInvalidData, len(key))
	}
	if err := validateDigits("key", key); err != nil {
		return nil, err
	}
	if err := ValidateCheckDigit(key); err != nil {
		return nil, err
	}
	return types, nil
}

// validateElementCheckDigit validates the modulo-10 check digit carried by
// an element whose AI identifies a numeric GS1 key. It covers every AI in
// the table whose GS1 Syntax Dictionary entry carries the csum constraint:
// GTIN (01, 02, 03), SSCC (00), GSIN (402), the GLN family (410-417), GSRN
// (8017, 8018), the 13-digit key of GDTI (253), GCN (255) and GRAI (8003),
// and the 14-digit GTIN prefix of ITIP (8006, 8026). Other AIs pass unchanged.
func validateElementCheckDigit(element Element) error {
	var err error
	switch element.AI {
	case "01", "02", "03":
		err = ValidateGTIN(element.Value)
	case "00":
		err = ValidateSSCC(element.Value)
	case "402":
		err = ValidateGSIN(element.Value)
	case "410", "411", "412", "413", "414", "415", "416", "417":
		err = ValidateGLN(element.Value)
	case "8017", "8018":
		err = ValidateGSRN(element.Value)
	case "253", "255":
		err = validateKeyPrefix(element.Value, 0, 13)
	case "8003":
		err = validateKeyPrefix(element.Value, 1, 14)
	case "8006", "8026":
		err = validateKeyPrefix(element.Value, 0, 14)
	default:
		return nil
	}
	if err != nil {
		return fmt.Errorf("AI (%s): %w", element.AI, err)
	}
	return nil
}

// validateKeyPrefix validates the check digit of the numeric key embedded in
// value[start:end]. The parser and encoder check lengths before check
// digits, so a short value here can only come from a hand-built Barcode
// passed to Validate; it is rejected rather than silently skipped.
func validateKeyPrefix(value string, start, end int) error {
	if len(value) < end {
		return fmt.Errorf("%w: key needs at least %d digits, got %d", ErrInvalidData, end, len(value))
	}
	return ValidateCheckDigit(value[start:end])
}
