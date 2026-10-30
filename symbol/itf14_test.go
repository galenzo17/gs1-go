package symbol

import (
	"errors"
	"testing"

	"github.com/galenzo17/gs1-go"
)

func TestITF14Patterns(t *testing.T) {
	// Every Interleaved 2 of 5 character has exactly two wide elements out of
	// five (ISO/IEC 16390) and the ten patterns are distinct.
	seen := map[uint8]bool{}
	for d, p := range itf14Patterns {
		if p > 0b11111 || seen[p] {
			t.Errorf("digit %d: pattern %05b invalid or repeated", d, p)
		}
		seen[p] = true
		wide := 0
		for e := 0; e < 5; e++ {
			if itf14Width(p, e) == itf14Wide {
				wide++
			} else if itf14Width(p, e) != itf14Narrow {
				t.Errorf("digit %d element %d: width %d", d, e, itf14Width(p, e))
			}
		}
		if wide != 2 {
			t.Errorf("digit %d: %05b has %d wide elements, want 2", d, p, wide)
		}
	}
}

func TestITF14Layout(t *testing.T) {
	const gtin = "10614141000019"
	tests := []struct {
		name        string
		opts        ITF14Options
		height      int
		bearer      bearerBars
		width, high int // rendered size in modules at scale 1
	}{
		{"default", ITF14Options{}, 32, bearerBars{2, 0}, 155, 2 + 32 + 2 + 8},
		{"horizontal thick", ITF14Options{BearerWidth: 4}, 32, bearerBars{4, 0}, 155, 4 + 32 + 4 + 8},
		{"frame", ITF14Options{Bearer: BearerFrame}, 32, bearerBars{5, 5}, 165, 5 + 32 + 5 + 8},
		{"frame thin", ITF14Options{Bearer: BearerFrame, BearerWidth: 3}, 32, bearerBars{3, 3}, 161, 3 + 32 + 3 + 8},
		{"none", ITF14Options{Bearer: BearerNone, BearerWidth: 9}, 32, bearerBars{}, 155, 32 + 8},
		{"height", ITF14Options{Height: 50}, 50, bearerBars{2, 0}, 155, 2 + 50 + 2 + 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := ITF14(gtin, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if l.Symbology != "ITF-14" || l.Modules != 135 || l.QuietLeft != 10 || l.QuietRight != 10 {
				t.Errorf("Symbology %q, %d modules, quiet zones %d/%d", l.Symbology, l.Modules, l.QuietLeft, l.QuietRight)
			}
			if l.Height != tc.height {
				t.Errorf("Height %d, want %d", l.Height, tc.height)
			}
			if l.bearer != tc.bearer {
				t.Errorf("bearer %+v, want %+v", l.bearer, tc.bearer)
			}
			if l.guards != nil {
				t.Errorf("ITF-14 has no guard bars, got %v", l.guards)
			}
			checkITF14HRI(t, l, gtin, tc.height+tc.bearer.horizontal+hriGap)
			if w, h := l.Size(1); w != tc.width || h != tc.high {
				t.Errorf("Size(1) = %dx%d, want %dx%d", w, h, tc.width, tc.high)
			}
		})
	}
}

// checkITF14HRI verifies that the digits are printed once, centered under
// the bars and at row y, below the bottom bearer bar.
func checkITF14HRI(t *testing.T, l *Linear, gtin string, y int) {
	t.Helper()
	if l.HRI() != gtin {
		t.Errorf("HRI %q, want %q", l.HRI(), gtin)
	}
	if len(l.text) != 1 || l.text[0].text != gtin {
		t.Fatalf("text runs %v", l.text)
	}
	textWidth := 14*glyphPitch - (glyphPitch - glyphWidth)
	if want := (135 - textWidth) / 2; l.text[0].x != want {
		t.Errorf("HRI x %d, want %d", l.text[0].x, want)
	}
	if l.text[0].y != y {
		t.Errorf("HRI y %d, want %d", l.text[0].y, y)
	}
}

func TestITF14Errors(t *testing.T) {
	tests := []struct {
		name string
		gtin string
		opts ITF14Options
		want error
	}{
		{"short", "1061414100001", ITF14Options{}, gs1.ErrInvalidData},
		{"long", "010614141000019", ITF14Options{}, gs1.ErrInvalidData},
		{"letters", "1061414100001A", ITF14Options{}, gs1.ErrInvalidData},
		{"check digit", "10614141000015", ITF14Options{}, gs1.ErrInvalidCheckDigit},
		{"negative bearer", "10614141000019", ITF14Options{BearerWidth: -1}, gs1.ErrInvalidData},
		{"negative height", "10614141000019", ITF14Options{Height: -1}, gs1.ErrInvalidData},
		{"unknown bearer", "10614141000019", ITF14Options{Bearer: Bearer(7)}, gs1.ErrInvalidData},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := ITF14(tc.gtin, tc.opts)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error %v, want %v", err, tc.want)
			}
			if l != nil {
				t.Error("symbol returned with an error")
			}
		})
	}
}
