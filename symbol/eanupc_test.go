package symbol

import (
	"errors"
	"math/bits"
	"testing"

	"github.com/galenzo17/gs1-go"
)

// TestEANCharacterSets checks the structural properties ISO/IEC 15420
// gives for the three number sets: seven modules, two bars and two spaces
// per character, odd parity for set A, even parity for sets B and C, set A
// and B starting with a space and ending with a bar, set C the opposite.
func TestEANCharacterSets(t *testing.T) {
	for d := byte('0'); d <= '9'; d++ {
		for _, set := range []struct {
			name  string
			write func(*Linear, byte)
			odd   bool
			first bool // first module dark
		}{
			{"A", (*Linear).setA, true, false},
			{"B", (*Linear).setB, false, false},
			{"C", (*Linear).setC, false, true},
		} {
			var l Linear
			set.write(&l, d)
			if l.Modules != 7 {
				t.Fatalf("set %s digit %c: %d modules", set.name, d, l.Modules)
			}
			var p uint16
			runs := 0
			for i := 0; i < 7; i++ {
				if l.mods[i] {
					p |= 1 << (6 - i)
				}
				if i > 0 && l.mods[i] != l.mods[i-1] {
					runs++
				}
			}
			if runs != 3 {
				t.Errorf("set %s digit %c: %07b has %d elements, want 4", set.name, d, p, runs+1)
			}
			if odd := bits.OnesCount16(p)%2 == 1; odd != set.odd {
				t.Errorf("set %s digit %c: %07b parity odd=%v, want %v", set.name, d, p, odd, set.odd)
			}
			if l.mods[0] != set.first || l.mods[6] == set.first {
				t.Errorf("set %s digit %c: %07b edges wrong", set.name, d, p)
			}
		}
	}
}

// TestParityTables checks the shape of the number set selection tables:
// EAN-13 leading digit 0 uses set A throughout and every other leading
// digit uses set B for exactly three of the six characters, never the
// first; UPC-E uses set B for exactly three characters for every check
// digit, and every pattern is distinct so decoding is unambiguous.
func TestParityTables(t *testing.T) {
	seen := map[uint8]bool{}
	for d, p := range ean13Parity {
		if seen[p] {
			t.Errorf("EAN-13 parity pattern %06b repeats at digit %d", p, d)
		}
		seen[p] = true
		if p&0b100000 != 0 {
			t.Errorf("EAN-13 digit %d: first character must be set A", d)
		}
		if want := 3; d == 0 {
			want = 0
			if bits.OnesCount8(p) != want {
				t.Errorf("EAN-13 digit 0 must use set A throughout, got %06b", p)
			}
		} else if bits.OnesCount8(p) != want {
			t.Errorf("EAN-13 digit %d: %06b uses set B %d times, want 3", d, p, bits.OnesCount8(p))
		}
	}
	seen = map[uint8]bool{}
	for c, p := range upceParity {
		if seen[p] || seen[^p&0b111111] {
			t.Errorf("UPC-E parity pattern %06b (check digit %d) collides with another pattern or its complement", p, c)
		}
		seen[p] = true
		if bits.OnesCount8(p) != 3 {
			t.Errorf("UPC-E check digit %d: %06b uses set B %d times, want 3", c, p, bits.OnesCount8(p))
		}
	}
}

type layoutCase struct {
	name                  string
	build                 func() (*Linear, error)
	modules, height       int
	quietLeft, quietRight int
	guards                []span
	hri                   string
	text                  []textRun
}

func TestEANUPCLayout(t *testing.T) {
	tests := []layoutCase{
		{
			"EAN-13", func() (*Linear, error) { return EAN13("4006381333931") },
			95, 69, 11, 7, []span{{0, 3}, {45, 50}, {92, 95}}, "4006381333931",
			[]textRun{{-7, 70, "4"}, {4, 70, "006381"}, {51, 70, "333931"}},
		},
		{
			"EAN-8", func() (*Linear, error) { return EAN8("96385074") },
			67, 55, 7, 7, []span{{0, 3}, {31, 36}, {64, 67}}, "96385074",
			[]textRun{{4, 56, "9638"}, {37, 56, "5074"}},
		},
		{
			"UPC-A", func() (*Linear, error) { return UPCA("036000291452") },
			95, 69, 9, 9, []span{{0, 10}, {45, 50}, {85, 95}}, "036000291452",
			[]textRun{{-7, 70, "0"}, {11, 70, "36000"}, {51, 70, "29145"}, {97, 70, "2"}},
		},
		{
			"UPC-E", func() (*Linear, error) { return UPCE("01234565") },
			51, 69, 9, 7, []span{{0, 3}, {45, 51}}, "01234565",
			[]textRun{{-7, 70, "0"}, {4, 70, "123456"}, {53, 70, "5"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := tc.build()
			if err != nil {
				t.Fatal(err)
			}
			if l.Symbology != tc.name {
				t.Errorf("Symbology %q, want %q", l.Symbology, tc.name)
			}
			if l.Modules != tc.modules || len(l.mods) != tc.modules {
				t.Errorf("Modules %d, want %d", l.Modules, tc.modules)
			}
			if l.Height != tc.height {
				t.Errorf("Height %d, want %d", l.Height, tc.height)
			}
			if l.QuietLeft != tc.quietLeft || l.QuietRight != tc.quietRight {
				t.Errorf("quiet zones %d/%d, want %d/%d", l.QuietLeft, l.QuietRight, tc.quietLeft, tc.quietRight)
			}
			if l.HRI() != tc.hri {
				t.Errorf("HRI %q, want %q", l.HRI(), tc.hri)
			}
			checkGuards(t, l, tc.guards)
			checkText(t, l, tc.text)
		})
	}
}

// checkGuards compares the guard spans and verifies that no bar crosses a
// span boundary, otherwise the extension would split a bar.
func checkGuards(t *testing.T, l *Linear, want []span) {
	t.Helper()
	if len(l.guards) != len(want) {
		t.Fatalf("guards %v, want %v", l.guards, want)
	}
	for i, g := range want {
		if l.guards[i] != g {
			t.Errorf("guard %d = %v, want %v", i, l.guards[i], g)
		}
		if l.Dark(g.start) && l.Dark(g.start-1) || l.Dark(g.end-1) && l.Dark(g.end) {
			t.Errorf("guard %v shares a bar with the adjacent character", g)
		}
	}
}

// checkText compares the HRI runs and verifies that each stays inside the
// quiet zone it uses.
func checkText(t *testing.T, l *Linear, want []textRun) {
	t.Helper()
	if len(l.text) != len(want) {
		t.Fatalf("text runs %v, want %v", l.text, want)
	}
	for i, r := range want {
		if l.text[i] != r {
			t.Errorf("text run %d = %v, want %v", i, l.text[i], r)
		}
		end := r.x + glyphPitch*len(r.text) - (glyphPitch - glyphWidth)
		if r.x < -l.QuietLeft || end > l.Modules+l.QuietRight {
			t.Errorf("text run %v leaves the quiet zone", r)
		}
	}
}

func TestEANUPCErrors(t *testing.T) {
	tests := []struct {
		name  string
		build func(string) (*Linear, error)
		in    string
		want  error
	}{
		{"EAN-13 short", EAN13, "400638133393", gs1.ErrInvalidData},
		{"EAN-13 long", EAN13, "04006381333931", gs1.ErrInvalidData},
		{"EAN-13 letters", EAN13, "40063813339X1", gs1.ErrInvalidData},
		{"EAN-13 check digit", EAN13, "4006381333930", gs1.ErrInvalidCheckDigit},
		{"EAN-8 length", EAN8, "9638507", gs1.ErrInvalidData},
		{"EAN-8 check digit", EAN8, "96385075", gs1.ErrInvalidCheckDigit},
		{"UPC-A length", UPCA, "0036000291452", gs1.ErrInvalidData},
		{"UPC-A check digit", UPCA, "036000291453", gs1.ErrInvalidCheckDigit},
		{"UPC-E empty", UPCE, "", gs1.ErrInvalidData},
		{"UPC-E length", UPCE, "012345678", gs1.ErrInvalidData},
		{"UPC-E letters", UPCE, "01234S65", gs1.ErrInvalidData},
		{"UPC-E number system", UPCE, "2123456", gs1.ErrInvalidData},
		{"UPC-E check digit", UPCE, "01234566", gs1.ErrInvalidCheckDigit},
		{"UPC-E suppression 3", UPCE, "0122453", gs1.ErrInvalidData},
		{"UPC-E suppression 4", UPCE, "0123054", gs1.ErrInvalidData},
		{"UPC-E suppression 5-9", UPCE, "0556809", gs1.ErrInvalidData},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := tc.build(tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error %v, want %v", err, tc.want)
			}
			if l != nil {
				t.Error("symbol returned with an error")
			}
		})
	}
}

func TestExpandUPCE(t *testing.T) {
	// want is the UPC-A without its check digit; the check digit is verified
	// separately and against gs1.ExpandUPCE for number system 0.
	tests := []struct{ ns, core, want string }{
		{"0", "123450", "01200000345"}, // last digit 0-2: manufacturer 12X00, product 00345
		{"0", "123451", "01210000345"},
		{"0", "123452", "01220000345"},
		{"0", "123453", "01230000045"}, // 3: manufacturer 12300, product 00045
		{"0", "123454", "01234000005"}, // 4: manufacturer 12340, product 00005
		{"0", "123456", "01234500006"}, // 5-9: manufacturer 12345, product 0000X
		{"0", "123459", "01234500009"},
		{"1", "123456", "11234500006"},
	}
	for _, tc := range tests {
		got := expandUPCE(tc.ns[0], tc.core)
		if len(got) != 12 || got[:11] != tc.want {
			t.Errorf("expandUPCE(%s, %s) = %s, want %s + check digit", tc.ns, tc.core, got, tc.want)
			continue
		}
		if err := gs1.ValidateGTIN(got); err != nil {
			t.Errorf("expandUPCE(%s, %s) = %s: %v", tc.ns, tc.core, got, err)
		}
		if tc.ns == "0" {
			ref, err := gs1.ExpandUPCE(tc.core)
			if err != nil || ref != got {
				t.Errorf("gs1.ExpandUPCE(%s) = %s, %v; expandUPCE gives %s", tc.core, ref, err, got)
			}
		}
	}
}
