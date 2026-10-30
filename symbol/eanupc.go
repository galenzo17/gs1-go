package symbol

import (
	"fmt"

	"github.com/galenzo17/gs1-go"
)

// EAN/UPC symbol characters (ISO/IEC 15420, symbol character encodation)
// are seven modules wide. Number set A has odd parity, number set C is its complement
// and number set B is number set C reversed. eanSetA lists the ten
// characters of set A, most significant bit first.
var eanSetA = [10]uint16{
	0b0001101, 0b0011001, 0b0010011, 0b0111101, 0b0100011,
	0b0110001, 0b0101111, 0b0111011, 0b0110111, 0b0001011,
}

// ean13Parity gives, per leading digit, the number set of the six left-hand
// characters of an EAN-13 symbol (ISO/IEC 15420 parity table): bit 5 is the
// leftmost character and a set bit selects number set B.
var ean13Parity = [10]uint8{
	0b000000, 0b001011, 0b001101, 0b001110, 0b010011,
	0b011001, 0b011100, 0b010101, 0b010110, 0b011010,
}

// upceParity gives, per check digit, the number sets of the six characters
// of a UPC-E symbol with number system 0 (ISO/IEC 15420 parity table); number
// system 1 uses the complement. Bit 5 is the leftmost character and a set
// bit selects number set B.
var upceParity = [10]uint8{
	0b111000, 0b110100, 0b110010, 0b110001, 0b101100,
	0b100110, 0b100011, 0b101010, 0b101001, 0b100101,
}

// Guard bar patterns, most significant bit first.
const (
	eanNormalGuard  = 0b101    // left and right guard bars
	eanCenterGuard  = 0b01010  // center guard bars
	upceRightGuard  = 0b010101 // UPC-E has no center pattern and a six-module right guard
	eanNormalGuardW = 3
	eanCenterGuardW = 5
	upceRightGuardW = 6
)

// Nominal bar heights in modules (GS1 General Specifications section 5.2,
// EAN/UPC symbol dimensions): EAN-13 and UPC-A bars are 22.85 mm and EAN-8
// bars 18.23 mm at X = 0.330 mm. Quiet zones follow the same section.
const (
	ean13Height = 69
	ean8Height  = 55
)

// setA, setB and setC append the character for digit d in the given number set.
func (l *Linear) setA(d byte) { l.pattern(eanSetA[d-'0'], glyphPitch) }
func (l *Linear) setC(d byte) { l.pattern(^eanSetA[d-'0']&0b1111111, glyphPitch) }
func (l *Linear) setB(d byte) { l.pattern(reverse7(^eanSetA[d-'0']&0b1111111), glyphPitch) }

// setAB appends d in number set B when b is set, in set A otherwise.
func (l *Linear) setAB(d byte, b bool) {
	if b {
		l.setB(d)
	} else {
		l.setA(d)
	}
}

func reverse7(p uint16) uint16 {
	var r uint16
	for i := 0; i < 7; i++ {
		r = r<<1 | p&1
		p >>= 1
	}
	return r
}

// EAN13 encodes a 13-digit GTIN-13 as an EAN-13 symbol: 95 modules, the
// leading digit printed to the left of the symbol in the quiet zone, the
// twelve others below the bars, and the three guard bar groups extended.
// The check digit is validated.
func EAN13(gtin string) (*Linear, error) {
	if err := checkGTIN("EAN-13", gtin, 13); err != nil {
		return nil, err
	}
	l := newLinear("EAN-13", ean13Height, 11, 7)
	l.hri = gtin
	l.pattern(eanNormalGuard, eanNormalGuardW)
	parity := ean13Parity[gtin[0]-'0']
	for i := 1; i <= 6; i++ {
		l.setAB(gtin[i], parity&(1<<(6-i)) != 0)
	}
	l.pattern(eanCenterGuard, eanCenterGuardW)
	for i := 7; i <= 12; i++ {
		l.setC(gtin[i])
	}
	l.pattern(eanNormalGuard, eanNormalGuardW)
	l.guards = []span{{0, 3}, {45, 50}, {92, 95}}
	y := ean13Height + hriGap
	l.text = []textRun{
		{-glyphPitch, y, gtin[:1]},
		{3 + glyphIndent, y, gtin[1:7]},
		{50 + glyphIndent, y, gtin[7:]},
	}
	return l, nil
}

// EAN8 encodes an 8-digit GTIN-8 as an EAN-8 symbol: 67 modules with four
// digits printed below each half. The check digit is validated.
func EAN8(gtin string) (*Linear, error) {
	if err := checkGTIN("EAN-8", gtin, 8); err != nil {
		return nil, err
	}
	l := newLinear("EAN-8", ean8Height, 7, 7)
	l.hri = gtin
	l.pattern(eanNormalGuard, eanNormalGuardW)
	for i := 0; i < 4; i++ {
		l.setA(gtin[i])
	}
	l.pattern(eanCenterGuard, eanCenterGuardW)
	for i := 4; i < 8; i++ {
		l.setC(gtin[i])
	}
	l.pattern(eanNormalGuard, eanNormalGuardW)
	l.guards = []span{{0, 3}, {31, 36}, {64, 67}}
	y := ean8Height + hriGap
	l.text = []textRun{
		{3 + glyphIndent, y, gtin[:4]},
		{36 + glyphIndent, y, gtin[4:]},
	}
	return l, nil
}

// UPCA encodes a 12-digit GTIN-12 as a UPC-A symbol: 95 modules, the number
// system digit printed to the left and the check digit to the right of the
// symbol, the first and last symbol characters extended like the guard bars.
// The check digit is validated.
func UPCA(gtin string) (*Linear, error) {
	if err := checkGTIN("UPC-A", gtin, 12); err != nil {
		return nil, err
	}
	l := newLinear("UPC-A", ean13Height, 9, 9)
	l.hri = gtin
	l.pattern(eanNormalGuard, eanNormalGuardW)
	for i := 0; i < 6; i++ {
		l.setA(gtin[i])
	}
	l.pattern(eanCenterGuard, eanCenterGuardW)
	for i := 6; i < 12; i++ {
		l.setC(gtin[i])
	}
	l.pattern(eanNormalGuard, eanNormalGuardW)
	l.guards = []span{{0, 10}, {45, 50}, {85, 95}}
	y := ean13Height + hriGap
	l.text = []textRun{
		{-glyphPitch, y, gtin[:1]},
		{10 + glyphIndent, y, gtin[1:6]},
		{50 + glyphIndent, y, gtin[6:11]},
		{95 + 2, y, gtin[11:]},
	}
	return l, nil
}

// UPCE encodes a zero-suppressed UPC-E code as a UPC-E symbol: 51 modules,
// the number system digit printed to the left and the check digit to the
// right. The input is the six data digits (number system 0 assumed), seven
// digits with the number system first, or eight digits with the number
// system first and the check digit last. The number system must be 0 or 1
// and a given check digit must match the one of the equivalent UPC-A, which
// is computed with the UPC-E expansion rules of GS1 General Specifications
// section 5.2 (the same rules as gs1.ExpandUPCE).
//
// The six digits must be a zero suppression that those rules can
// produce: when the last digit is 3 the third cannot be 0, 1 or 2, when it is
// 4 the fourth cannot be 0, and when it is 5 to 9 the fifth cannot be 0,
// because those UPC-A numbers compress to a different UPC-E. Codes that
// break the rule still expand to a UPC-A, but no GS1 symbol carries them.
func UPCE(upce string) (*Linear, error) {
	if err := checkDigits("UPC-E", upce); err != nil {
		return nil, err
	}
	ns, core, check := byte('0'), "", byte(0)
	switch len(upce) {
	case 6:
		core = upce
	case 7:
		ns, core = upce[0], upce[1:]
	case 8:
		ns, core, check = upce[0], upce[1:7], upce[7]
	default:
		return nil, fmt.Errorf("%w: UPC-E requires 6, 7 or 8 digits, got %d", gs1.ErrInvalidData, len(upce))
	}
	if ns != '0' && ns != '1' {
		return nil, fmt.Errorf("%w: UPC-E number system must be 0 or 1, got %c", gs1.ErrInvalidData, ns)
	}
	if err := checkUPCECore(core); err != nil {
		return nil, err
	}
	upca := expandUPCE(ns, core)
	if check != 0 && check != upca[11] {
		return nil, fmt.Errorf("UPC-E: %w: expected '%c', got '%c'", gs1.ErrInvalidCheckDigit, upca[11], check)
	}
	check = upca[11]

	l := newLinear("UPC-E", ean13Height, 9, 7)
	l.hri = string(ns) + core + string(check)
	l.pattern(eanNormalGuard, eanNormalGuardW)
	parity := upceParity[check-'0']
	if ns == '1' {
		parity = ^parity & 0b111111
	}
	for i := 0; i < 6; i++ {
		l.setAB(core[i], parity&(1<<(5-i)) != 0)
	}
	l.pattern(upceRightGuard, upceRightGuardW)
	l.guards = []span{{0, 3}, {45, 51}}
	y := ean13Height + hriGap
	l.text = []textRun{
		{-glyphPitch, y, l.hri[:1]},
		{3 + glyphIndent, y, core},
		{51 + 2, y, l.hri[7:]},
	}
	return l, nil
}

// checkUPCECore rejects six data digits that no UPC-A compresses to.
func checkUPCECore(core string) error {
	var bad bool
	switch core[5] {
	case '3':
		bad = core[2] <= '2'
	case '4':
		bad = core[3] == '0'
	case '5', '6', '7', '8', '9':
		bad = core[4] == '0'
	}
	if bad {
		return fmt.Errorf("%w: UPC-E %s is not a valid zero suppression, the UPC-A it expands to compresses differently", gs1.ErrInvalidData, core)
	}
	return nil
}

// expandUPCE returns the 12-digit UPC-A, check digit included, that the six
// UPC-E data digits stand for under number system ns.
func expandUPCE(ns byte, core string) string {
	var body string
	switch core[5] {
	case '0', '1', '2':
		body = core[0:2] + core[5:6] + "0000" + core[2:5]
	case '3':
		body = core[0:3] + "00000" + core[3:5]
	case '4':
		body = core[0:4] + "00000" + core[4:5]
	default:
		body = core[0:5] + "0000" + core[5:6]
	}
	partial := string(ns) + body
	check, _ := gs1.ComputeGTINCheckDigit(partial)
	return partial + string(check)
}

// checkGTIN verifies that gtin has exactly n digits and a valid check digit.
func checkGTIN(symbology, gtin string, n int) error {
	if len(gtin) != n {
		return fmt.Errorf("%w: %s requires %d digits, got %d", gs1.ErrInvalidData, symbology, n, len(gtin))
	}
	if err := gs1.ValidateGTIN(gtin); err != nil {
		return fmt.Errorf("%s: %w", symbology, err)
	}
	return nil
}

func checkDigits(symbology, s string) error {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return fmt.Errorf("%w: %s contains a non-digit character at position %d", gs1.ErrInvalidData, symbology, i)
		}
	}
	return nil
}
