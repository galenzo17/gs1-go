package symbol

import (
	"errors"
	"strings"
	"testing"

	"github.com/galenzo17/gs1-go"
)

// This file is a small reader for the linear symbols, used to check that
// every symbol decodes back to the digits it was built from. The tables are
// typed independently of the encoder as the ISO/IEC 15420 and 16390 figures
// show them, so a slip in either copy fails the round trip.

var (
	readSetA        = [10]string{"0001101", "0011001", "0010011", "0111101", "0100011", "0110001", "0101111", "0111011", "0110111", "0001011"}
	readEAN13Parity = [10]string{"AAAAAA", "AABABB", "AABBAB", "AABBBA", "ABAABB", "ABBAAB", "ABBBAA", "ABABAB", "ABABBA", "ABBABA"}
	readUPCEParity  = [10]string{"BBBAAA", "BBABAA", "BBAABA", "BBAAAB", "BABBAA", "BAABBA", "BAAABB", "BABABA", "BABAAB", "BAABAB"}
	readITFPatterns = [10]string{"NNWWN", "WNNNW", "NWNNW", "WWNNN", "NNWNW", "WNWNN", "NWWNN", "NNNWW", "WNNWN", "NWNWN"}
)

func modules(l *Linear) string {
	var b strings.Builder
	for i := 0; i < l.Modules; i++ {
		if l.Dark(i) {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}

func complement(s string) string {
	b := []byte(s)
	for i := range b {
		b[i] ^= '0' ^ '1'
	}
	return string(b)
}

func reversed(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// readChar decodes a seven-module EAN/UPC character into its digit and
// number set, or returns ok false.
func readChar(s string) (digit byte, set byte, ok bool) {
	for d, a := range readSetA {
		c := complement(a)
		switch s {
		case a:
			return byte('0' + d), 'A', true
		case c:
			return byte('0' + d), 'C', true
		case reversed(c):
			return byte('0' + d), 'B', true
		}
	}
	return 0, 0, false
}

// readChars decodes n consecutive characters starting at col and returns
// the digits and the number set of each.
func readChars(t *testing.T, m string, col, n int) (digits, sets string) {
	t.Helper()
	for i := 0; i < n; i++ {
		s := m[col+7*i : col+7*i+7]
		d, set, ok := readChar(s)
		if !ok {
			t.Fatalf("unreadable character %q at column %d", s, col+7*i)
		}
		digits += string(d)
		sets += string(set)
	}
	return digits, sets
}

func expectGuard(t *testing.T, m string, col int, want string) {
	t.Helper()
	if got := m[col : col+len(want)]; got != want {
		t.Fatalf("guard at column %d = %q, want %q", col, got, want)
	}
}

func indexOf(table [10]string, s string) int {
	for i, p := range table {
		if p == s {
			return i
		}
	}
	return -1
}

// readEANUPC decodes an EAN-13, UPC-A, EAN-8 or UPC-E module row into the
// digits the symbol stands for: thirteen for the 95-module symbols (UPC-A
// as its EAN-13 equivalent with a leading zero), eight for EAN-8 and the
// eight-digit UPC-E form, number system and check digit included, for
// UPC-E. The check digit is verified.
func readEANUPC(t *testing.T, l *Linear) string {
	t.Helper()
	m := modules(l)
	switch len(m) {
	case 95:
		expectGuard(t, m, 0, "101")
		expectGuard(t, m, 45, "01010")
		expectGuard(t, m, 92, "101")
		left, parity := readChars(t, m, 3, 6)
		right, sets := readChars(t, m, 50, 6)
		if sets != "CCCCCC" {
			t.Fatalf("right-hand number sets %s, want CCCCCC", sets)
		}
		first := indexOf(readEAN13Parity, parity)
		if first < 0 {
			t.Fatalf("left-hand parity %s matches no leading digit", parity)
		}
		gtin := string(byte('0'+first)) + left + right
		if err := gs1.ValidateGTIN(gtin); err != nil {
			t.Fatalf("decoded %s: %v", gtin, err)
		}
		return gtin
	case 67:
		expectGuard(t, m, 0, "101")
		expectGuard(t, m, 31, "01010")
		expectGuard(t, m, 64, "101")
		left, sets := readChars(t, m, 3, 4)
		if sets != "AAAA" {
			t.Fatalf("left-hand number sets %s, want AAAA", sets)
		}
		right, sets := readChars(t, m, 36, 4)
		if sets != "CCCC" {
			t.Fatalf("right-hand number sets %s, want CCCC", sets)
		}
		gtin := left + right
		if err := gs1.ValidateGTIN(gtin); err != nil {
			t.Fatalf("decoded %s: %v", gtin, err)
		}
		return gtin
	case 51:
		expectGuard(t, m, 0, "101")
		expectGuard(t, m, 45, "010101")
		core, parity := readChars(t, m, 3, 6)
		ns, check := byte('0'), indexOf(readUPCEParity, parity)
		if check < 0 {
			ns, check = '1', indexOf(readUPCEParity, strings.NewReplacer("A", "B", "B", "A").Replace(parity))
			if check < 0 {
				t.Fatalf("parity %s matches no number system and check digit", parity)
			}
		}
		// gs1.ExpandUPCE assumes number system 0; swap the number system in
		// and recompute the check digit for number system 1.
		upca, err := gs1.ExpandUPCE(core)
		if err != nil {
			t.Fatalf("ExpandUPCE(%s): %v", core, err)
		}
		partial := string(ns) + upca[1:11]
		cd, _ := gs1.ComputeGTINCheckDigit(partial)
		if cd != byte('0'+check) {
			t.Fatalf("UPC-E %c%s: parity says check digit %d, UPC-A %s%c says %c", ns, core, check, partial, cd, cd)
		}
		return string(ns) + core + string(cd)
	}
	t.Fatalf("%d modules is not an EAN/UPC symbol", len(m))
	return ""
}

// elementRuns returns the width in modules of every bar and space of the
// symbol, in order; the symbol must start with a bar.
func elementRuns(t *testing.T, l *Linear) []int {
	t.Helper()
	if !l.Dark(0) {
		t.Fatal("symbol must start with a bar")
	}
	var runs []int
	for col := 0; col < l.Modules; {
		run := 1
		for col+run < l.Modules && l.Dark(col+run) == l.Dark(col) {
			run++
		}
		runs = append(runs, run)
		col += run
	}
	return runs
}

// itfElement classifies an element width as narrow or wide.
func itfElement(t *testing.T, w int) byte {
	t.Helper()
	switch w {
	case 1:
		return 'N'
	case 3:
		return 'W'
	}
	t.Fatalf("element %d modules wide is neither narrow nor wide", w)
	return 0
}

// readITF decodes an ITF-14 module row into its fourteen digits and
// verifies the check digit.
func readITF(t *testing.T, l *Linear) string {
	t.Helper()
	runs := elementRuns(t, l)
	if len(runs) != 4+10*7+3 {
		t.Fatalf("%d elements, want %d", len(runs), 4+10*7+3)
	}
	if start := runs[:4]; start[0] != 1 || start[1] != 1 || start[2] != 1 || start[3] != 1 {
		t.Fatalf("start pattern widths %v, want [1 1 1 1]", start)
	}
	if stop := runs[len(runs)-3:]; stop[0] != 3 || stop[1] != 1 || stop[2] != 1 {
		t.Fatalf("stop pattern widths %v, want [3 1 1]", stop)
	}
	var digits string
	for pair := 0; pair < 7; pair++ {
		var bars, spaces string
		for e := 0; e < 5; e++ {
			bars += string(itfElement(t, runs[4+10*pair+2*e]))
			spaces += string(itfElement(t, runs[4+10*pair+2*e+1]))
		}
		for _, p := range []string{bars, spaces} {
			d := indexOf(readITFPatterns, p)
			if d < 0 {
				t.Fatalf("pattern %s is not an Interleaved 2 of 5 character", p)
			}
			digits += string(byte('0' + d))
		}
	}
	if err := gs1.ValidateGTIN(digits); err != nil {
		t.Fatalf("decoded %s: %v", digits, err)
	}
	return digits
}

// compressUPCA applies the UPC-E zero-suppression rules of GS1 General
// Specifications section 5.2 to a 12-digit UPC-A and returns the six
// UPC-E data digits, or "" when the UPC-A cannot be compressed.
func compressUPCA(upca string) string {
	m, p := upca[1:6], upca[6:11]
	switch {
	case (m[2:] == "000" || m[2:] == "100" || m[2:] == "200") && p[:2] == "00":
		return m[:2] + p[2:] + m[2:3]
	case m[3:] == "00" && p[:3] == "000":
		return m[:3] + p[3:] + "3"
	case m[4] == '0' && p[:4] == "0000":
		return m[:4] + p[4:] + "4"
	case p[:4] == "0000" && p[4] >= '5':
		return m + p[4:]
	}
	return ""
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// withCheck appends the GS1 check digit to partial.
func withCheck(t *testing.T, partial string) string {
	t.Helper()
	cd, err := gs1.ComputeGTINCheckDigit(partial)
	if err != nil {
		t.Fatalf("ComputeGTINCheckDigit(%s): %v", partial, err)
	}
	return partial + string(cd)
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		build func(string) (*Linear, error)
		read  func(*testing.T, *Linear) string
		in    string
		want  string
	}{
		{"EAN-13", EAN13, readEANUPC, "4006381333931", "4006381333931"},
		{"EAN-13 leading 0", EAN13, readEANUPC, "0036000291452", "0036000291452"},
		{"EAN-13 leading 9", EAN13, readEANUPC, "9780201379624", "9780201379624"},
		{"EAN-8", EAN8, readEANUPC, "96385074", "96385074"},
		{"UPC-A", UPCA, readEANUPC, "036000291452", "0036000291452"},
		{"UPC-E 8 digits", UPCE, readEANUPC, "01234565", "01234565"},
		{"UPC-E 7 digits", UPCE, readEANUPC, "0123456", "01234565"},
		{"UPC-E 6 digits", UPCE, readEANUPC, "123456", "01234565"},
		{"UPC-E number system 1", UPCE, readEANUPC, "1123456", "11234562"},
		{"ITF-14", func(s string) (*Linear, error) { return ITF14(s, ITF14Options{}) }, readITF, "10614141000019", "10614141000019"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := tc.build(tc.in)
			if err != nil {
				t.Fatalf("%s(%s): %v", tc.name, tc.in, err)
			}
			if got := tc.read(t, l); got != tc.want {
				t.Errorf("read back %s, want %s", got, tc.want)
			}
		})
	}
}

func FuzzEAN13(f *testing.F) {
	f.Add("400638133393")
	f.Add("978020137962")
	f.Add("000000000000")
	f.Fuzz(func(t *testing.T, partial string) {
		if len(partial) != 12 || !allDigits(partial) {
			t.Skip()
		}
		gtin := withCheck(t, partial)
		l, err := EAN13(gtin)
		if err != nil {
			t.Fatalf("EAN13(%s): %v", gtin, err)
		}
		if got := readEANUPC(t, l); got != gtin {
			t.Fatalf("EAN13(%s) read back as %s", gtin, got)
		}
		if l.HRI() != gtin {
			t.Fatalf("HRI %s, want %s", l.HRI(), gtin)
		}
	})
}

func FuzzEAN8(f *testing.F) {
	f.Add("9638507")
	f.Add("0000000")
	f.Fuzz(func(t *testing.T, partial string) {
		if len(partial) != 7 || !allDigits(partial) {
			t.Skip()
		}
		gtin := withCheck(t, partial)
		l, err := EAN8(gtin)
		if err != nil {
			t.Fatalf("EAN8(%s): %v", gtin, err)
		}
		if got := readEANUPC(t, l); got != gtin {
			t.Fatalf("EAN8(%s) read back as %s", gtin, got)
		}
	})
}

func FuzzUPCA(f *testing.F) {
	f.Add("03600029145")
	f.Add("99999999999")
	f.Fuzz(func(t *testing.T, partial string) {
		if len(partial) != 11 || !allDigits(partial) {
			t.Skip()
		}
		gtin := withCheck(t, partial)
		l, err := UPCA(gtin)
		if err != nil {
			t.Fatalf("UPCA(%s): %v", gtin, err)
		}
		if got := readEANUPC(t, l); got != "0"+gtin {
			t.Fatalf("UPCA(%s) read back as %s", gtin, got)
		}
		// A UPC-A symbol is the EAN-13 symbol of the same GTIN with a leading zero.
		ean, err := EAN13("0" + gtin)
		if err != nil {
			t.Fatalf("EAN13(0%s): %v", gtin, err)
		}
		if modules(ean) != modules(l) {
			t.Fatalf("UPC-A and EAN-13 module rows differ for %s", gtin)
		}
	})
}

func FuzzUPCE(f *testing.F) {
	f.Add("0123456")
	f.Add("1123456")
	f.Add("0000000")
	f.Add("1999999")
	f.Fuzz(func(t *testing.T, in string) {
		if len(in) != 7 || !allDigits(in) || in[0] > '1' {
			t.Skip()
		}
		l, err := UPCE(in)
		// A six-digit code is a valid UPC-E only when compressing the UPC-A it
		// expands to gives the same six digits again.
		if upca := expandUPCE(in[0], in[1:]); compressUPCA(upca) != in[1:] {
			if !errors.Is(err, gs1.ErrInvalidData) {
				t.Fatalf("UPCE(%s): %v, want ErrInvalidData for a code that is not the zero suppression of %s", in, err, upca)
			}
			return
		}
		if err != nil {
			t.Fatalf("UPCE(%s): %v", in, err)
		}
		hri := l.HRI()
		if got := readEANUPC(t, l); got != hri {
			t.Fatalf("UPCE(%s) read back as %s, HRI %s", in, got, hri)
		}
		if len(hri) != 8 || hri[:7] != in {
			t.Fatalf("HRI %s does not start with %s", hri, in)
		}
		// The eight-digit form must be accepted back unchanged.
		again, err := UPCE(hri)
		if err != nil {
			t.Fatalf("UPCE(%s): %v", hri, err)
		}
		if modules(again) != modules(l) {
			t.Fatalf("UPCE(%s) and UPCE(%s) differ", in, hri)
		}
		if in[0] == '0' {
			// gs1.ExpandUPCE must agree on the UPC-A, check digit included.
			upca, err := gs1.ExpandUPCE(hri)
			if err != nil {
				t.Fatalf("ExpandUPCE(%s): %v", hri, err)
			}
			if upca[11] != hri[7] {
				t.Fatalf("check digit %c, gs1.ExpandUPCE says %c", hri[7], upca[11])
			}
		}
	})
}

func FuzzITF14(f *testing.F) {
	f.Add("1061414100001", 0, 0, 0)
	f.Add("0000000000000", 1, 7, 40)
	f.Add("9999999999999", 2, 0, 12)
	f.Fuzz(func(t *testing.T, partial string, bearer, width, height int) {
		if len(partial) != 13 || !allDigits(partial) || bearer < 0 || bearer > 2 || width < 0 || width > 50 || height < 0 || height > 500 {
			t.Skip()
		}
		gtin := withCheck(t, partial)
		l, err := ITF14(gtin, ITF14Options{Bearer: Bearer(bearer), BearerWidth: width, Height: height})
		if err != nil {
			t.Fatalf("ITF14(%s): %v", gtin, err)
		}
		if got := readITF(t, l); got != gtin {
			t.Fatalf("ITF14(%s) read back as %s", gtin, got)
		}
		w, h, _ := l.layout()
		if sw, sh := l.Size(1); sw != w || sh != h || w <= 0 || h <= 0 {
			t.Fatalf("Size(1) = %dx%d, layout %dx%d", sw, sh, w, h)
		}
	})
}
