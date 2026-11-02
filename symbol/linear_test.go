package symbol

import (
	"bytes"
	"image"
	"image/png"
	"strconv"
	"strings"
	"testing"
)

func mustEAN13(t *testing.T) *Linear {
	t.Helper()
	l, err := EAN13("4006381333931")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestDark(t *testing.T) {
	l := mustEAN13(t)
	if l.Dark(-1) || l.Dark(l.Modules) || l.Dark(1<<20) {
		t.Error("columns outside the symbol must be light")
	}
	if !l.Dark(0) || l.Dark(1) || !l.Dark(2) {
		t.Error("left guard bars must read 101")
	}
}

func TestSizeAndClamp(t *testing.T) {
	l := mustEAN13(t)
	// 11 + 95 + 7 modules wide; 69 bars, a 1-module gap and 7 glyph rows high.
	if w, h := l.Size(1); w != 113 || h != 77 {
		t.Errorf("Size(1) = %dx%d, want 113x77", w, h)
	}
	if w, h := l.Size(3); w != 339 || h != 231 {
		t.Errorf("Size(3) = %dx%d, want 339x231", w, h)
	}
	w1, h1 := l.Size(1)
	if w, h := l.Size(0); w != w1 || h != h1 {
		t.Errorf("Size(0) = %dx%d, want Size(1) %dx%d", w, h, w1, h1)
	}
	if b := l.Image(-5).Bounds(); b.Dx() != w1 || b.Dy() != h1 {
		t.Errorf("Image(-5) bounds %v, want %dx%d", b, w1, h1)
	}
}

// pixelAt returns a probe for the module at (x, y) of an image rendered at
// scale pixels per module.
func pixelAt(img *image.Gray, scale int) func(x, y int) bool {
	return func(x, y int) bool { return img.GrayAt(x*scale, y*scale).Y == 0 }
}

func TestImageBarsAndGuards(t *testing.T) {
	l := mustEAN13(t)
	const scale = 2
	img := l.Image(scale)
	if b := img.Bounds(); b.Dx() != 113*scale || b.Dy() != 77*scale {
		t.Fatalf("bounds %v", b)
	}
	dark := pixelAt(img, scale)
	x0 := l.QuietLeft
	// Quiet zone, then the 101 guard, on the first row.
	if dark(0, 0) || dark(x0-1, 0) || !dark(x0, 0) || dark(x0+1, 0) || !dark(x0+2, 0) || dark(x0+95, 0) {
		t.Error("first row does not start with a quiet zone and the 101 guard")
	}
	// Guard bars extend below the data bars, data bars do not. Column 9 is
	// the last module of the first character, always a bar in sets A and B.
	if !dark(x0, l.Height+2) || dark(x0+9, l.Height+2) || !dark(x0+9, l.Height-1) {
		t.Error("guard extension wrong")
	}
	for _, p := range img.Pix {
		if p != 0 && p != 0xFF {
			t.Fatalf("gray level %d", p)
		}
	}
}

func TestImageQuietZoneAndHRI(t *testing.T) {
	l := mustEAN13(t)
	img := l.Image(1)
	dark := pixelAt(img, 1)
	x0 := l.QuietLeft
	// Nothing is drawn in the quiet zone beside the bars.
	for y := 0; y < l.Height; y++ {
		for x := 0; x < x0; x++ {
			if dark(x, y) {
				t.Fatalf("dark pixel in the quiet zone at %d,%d", x, y)
			}
		}
	}
	// The leading digit is drawn in the left quiet zone, below the bars.
	found := false
	for y := l.Height + 1; y < l.Height+8; y++ {
		for x := 0; x < x0; x++ {
			found = found || dark(x, y)
		}
	}
	if !found {
		t.Error("leading digit missing from the left quiet zone")
	}
}

func TestPNG(t *testing.T) {
	l := mustEAN13(t)
	var buf bytes.Buffer
	if err := l.PNG(&buf, 2); err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if want := l.Image(2).Bounds(); decoded.Bounds() != want {
		t.Errorf("decoded bounds %v, want %v", decoded.Bounds(), want)
	}
}

// TestSVGMatchesImage parses the SVG path back into a bitmap and compares it
// with Image, so the two renderers cannot drift apart.
func TestSVGMatchesImage(t *testing.T) {
	symbols := map[string]func() (*Linear, error){
		"EAN-13": func() (*Linear, error) { return EAN13("4006381333931") },
		"EAN-8":  func() (*Linear, error) { return EAN8("96385074") },
		"UPC-A":  func() (*Linear, error) { return UPCA("036000291452") },
		"UPC-E":  func() (*Linear, error) { return UPCE("01234565") },
		"ITF-14": func() (*Linear, error) { return ITF14("10614141000019", ITF14Options{Bearer: BearerFrame}) },
	}
	for name, build := range symbols {
		t.Run(name, func(t *testing.T) {
			l, err := build()
			if err != nil {
				t.Fatal(err)
			}
			const scale = 2
			svg := l.SVG(scale)
			w, h := l.Size(scale)
			header, rest, _ := strings.Cut(svg, "\n")
			want := `<svg xmlns="http://www.w3.org/2000/svg" version="1.1" width="` + strconv.Itoa(w) +
				`" height="` + strconv.Itoa(h) + `" viewBox="0 0 ` + strconv.Itoa(w) + " " + strconv.Itoa(h) +
				`" shape-rendering="crispEdges">`
			if header != want {
				t.Errorf("header\n%s\nwant\n%s", header, want)
			}
			if !strings.HasPrefix(rest, `<rect width="100%" height="100%" fill="#fff"/>`+"\n"+`<path fill="#000" d="`) ||
				!strings.HasSuffix(svg, "\"/>\n</svg>\n") {
				t.Fatalf("unexpected document structure:\n%s", svg)
			}
			_, d, _ := strings.Cut(rest, `d="`)
			d, _, _ = strings.Cut(d, `"`)

			got := image.NewGray(image.Rect(0, 0, w, h))
			for i := range got.Pix {
				got.Pix[i] = 0xFF
			}
			for _, sub := range strings.Split(strings.TrimSuffix(d, "z"), "z") {
				// Mx yhwvhh-wz
				var x, y, rw, rh int
				if _, err := parseSubpath(sub, &x, &y, &rw, &rh); err != nil {
					t.Fatalf("subpath %q: %v", sub, err)
				}
				for yy := y; yy < y+rh; yy++ {
					for xx := x; xx < x+rw; xx++ {
						got.SetGray(xx, yy, got.GrayAt(xx, yy)) // bounds check
						got.Pix[yy*got.Stride+xx] = 0
					}
				}
			}
			if !bytes.Equal(got.Pix, l.Image(scale).Pix) {
				t.Error("SVG path and Image differ")
			}
		})
	}
}

// parseSubpath reads "Mx yhwvhh-w" into its four numbers.
func parseSubpath(s string, x, y, w, h *int) (int, error) {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == 'M' || r == ' ' || r == 'h' || r == 'v' })
	if len(fields) != 5 {
		return 0, strconv.ErrSyntax
	}
	var err error
	for i, p := range []*int{x, y, w, h} {
		if *p, err = strconv.Atoi(fields[i]); err != nil {
			return 0, err
		}
	}
	if fields[4] != "-"+fields[2] {
		return 0, strconv.ErrSyntax
	}
	return 4, nil
}

func TestGlyphRects(t *testing.T) {
	if r := glyphRects(nil, 'x', 0, 0); r != nil {
		t.Errorf("non-digit drew %v", r)
	}
	for d := byte('0'); d <= '9'; d++ {
		rects := glyphRects(nil, d, 10, 20)
		if len(rects) == 0 {
			t.Errorf("digit %c draws nothing", d)
		}
		for _, r := range rects {
			if r.x < 10 || r.x+r.w > 10+glyphWidth || r.y < 20 || r.y >= 20+glyphHeight || r.h != 1 || r.w < 1 {
				t.Errorf("digit %c: rect %v outside the %dx%d cell", d, r, glyphWidth, glyphHeight)
			}
		}
	}
	// Distinct glyphs: no two digits render the same.
	seen := map[string]byte{}
	for d := byte('0'); d <= '9'; d++ {
		var b strings.Builder
		for _, r := range glyphRects(nil, d, 0, 0) {
			b.WriteString(strconv.Itoa(r.x) + "," + strconv.Itoa(r.y) + "," + strconv.Itoa(r.w) + ";")
		}
		if prev, ok := seen[b.String()]; ok {
			t.Errorf("digits %c and %c share a glyph", prev, d)
		}
		seen[b.String()] = d
	}
}
