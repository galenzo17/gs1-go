package symbol

import (
	"image"
	"image/png"
	"io"
	"strconv"
	"strings"
)

// Linear is a one-dimensional barcode symbol: a row of dark and light
// modules and the layout the GS1 General Specifications prescribe around it.
// Column 0 is the first module of the leftmost bar. The quiet zones are not
// part of the module row, but Image, PNG and SVG include them, because GS1
// measures them as part of the symbol and EAN/UPC symbols print part of
// their HRI inside them.
type Linear struct {
	// Symbology is the symbol type: "EAN-13", "EAN-8", "UPC-A", "UPC-E" or
	// "ITF-14".
	Symbology string
	// Modules is the width of the bar pattern in modules, quiet zones
	// excluded.
	Modules int
	// Height is the bar height in modules at the nominal X-dimension. The
	// guard bars of EAN/UPC symbols extend five modules below it.
	Height int
	// QuietLeft and QuietRight are the minimum quiet zones in modules that
	// the GS1 General Specifications require on each side of the bars.
	QuietLeft, QuietRight int

	mods   []bool
	guards []span     // column ranges whose bars extend below Height
	bearer bearerBars // ITF-14 bearer bars; zero for the other symbologies
	hri    string
	text   []textRun
}

// span is a half-open column range [start, end).
type span struct{ start, end int }

// bearerBars holds the thickness in modules of the horizontal and vertical
// bearer bars; zero means none.
type bearerBars struct{ horizontal, vertical int }

// textRun places HRI text with its glyph top-left corner at (x, y) relative
// to column 0 and row 0 of the bars. x may be negative for text in the left
// quiet zone.
type textRun struct {
	x, y int
	text string
}

// Layout constants in modules.
const (
	guardExtension = 5 // EAN/UPC guard bars extend this far below the data bars
	glyphPitch     = 7 // one EAN/UPC character is seven modules wide, like a digit slot
	glyphIndent    = 1 // centers a five-module glyph in a seven-module slot
	hriGap         = 1 // between the bars (or bearer bar) and the HRI
)

func newLinear(symbology string, height, quietLeft, quietRight int) *Linear {
	return &Linear{Symbology: symbology, Height: height, QuietLeft: quietLeft, QuietRight: quietRight}
}

// bar appends n dark modules and space appends n light modules.
func (l *Linear) bar(n int)   { l.run(n, true) }
func (l *Linear) space(n int) { l.run(n, false) }

func (l *Linear) run(n int, dark bool) {
	for i := 0; i < n; i++ {
		l.mods = append(l.mods, dark)
	}
	l.Modules = len(l.mods)
}

// pattern appends the lowest n bits of p, most significant first, as
// modules: 1 is dark and 0 is light.
func (l *Linear) pattern(p uint16, n int) {
	for i := n - 1; i >= 0; i-- {
		l.mods = append(l.mods, p&(1<<i) != 0)
	}
	l.Modules = len(l.mods)
}

// Dark reports whether the module at col is dark. Columns outside the bar
// pattern are light, which matches the quiet zones.
func (l *Linear) Dark(col int) bool {
	if col < 0 || col >= l.Modules {
		return false
	}
	return l.mods[col]
}

// HRI returns the human readable interpretation printed with the symbol:
// the digits the symbol encodes, check digit included.
func (l *Linear) HRI() string { return l.hri }

// extended reports whether the bars in [start, end) are guard bars that
// extend below the data bars.
func (l *Linear) extended(start, end int) bool {
	for _, g := range l.guards {
		if start >= g.start && end <= g.end {
			return true
		}
	}
	return false
}

// rect is a dark rectangle in rendered module coordinates, whose origin is
// the top-left corner of the drawing, bearer bars and quiet zones included.
type rect struct{ x, y, w, h int }

// layout returns the rendered size in modules and every dark rectangle.
func (l *Linear) layout() (width, height int, rects []rect) {
	bh, bv := l.bearer.horizontal, l.bearer.vertical
	x0 := bv + l.QuietLeft // column 0 of the bars
	y0 := bh               // row 0 of the bars
	width = 2*bv + l.QuietLeft + l.Modules + l.QuietRight
	height = y0 + l.Height + bh

	for col := 0; col < l.Modules; {
		if !l.mods[col] {
			col++
			continue
		}
		run := 1
		for col+run < l.Modules && l.mods[col+run] {
			run++
		}
		h := l.Height
		if l.extended(col, col+run) {
			h += guardExtension
		}
		rects = append(rects, rect{x0 + col, y0, run, h})
		height = max(height, y0+h)
		col += run
	}
	if bh > 0 {
		rects = append(rects, rect{0, 0, width, bh}, rect{0, y0 + l.Height, width, bh})
	}
	if bv > 0 {
		rects = append(rects, rect{0, 0, bv, 2*bh + l.Height}, rect{width - bv, 0, bv, 2*bh + l.Height})
	}
	for _, t := range l.text {
		for i := 0; i < len(t.text); i++ {
			rects = glyphRects(rects, t.text[i], x0+t.x+i*glyphPitch, y0+t.y)
		}
		height = max(height, y0+t.y+glyphHeight)
	}
	return width, height, rects
}

// Size returns the width and height in pixels of the rendering at scale
// pixels per module, quiet zones, bearer bars and HRI included.
func (l *Linear) Size(scale int) (width, height int) {
	scale = clampScale(scale)
	w, h, _ := l.layout()
	return w * scale, h * scale
}

// Image renders the symbol as a grayscale image at scale pixels per module.
// The drawing includes the GS1 minimum quiet zones, the bearer bars and the
// HRI, so it is complete as printed; Size gives its dimensions. Dark modules
// are black (0) and everything else is white (255). A scale below 1 is
// treated as 1.
func (l *Linear) Image(scale int) *image.Gray {
	scale = clampScale(scale)
	w, h, rects := l.layout()
	img := image.NewGray(image.Rect(0, 0, w*scale, h*scale))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	for _, r := range rects {
		left, right := r.x*scale, (r.x+r.w)*scale
		for y := r.y * scale; y < (r.y+r.h)*scale; y++ {
			clear(img.Pix[y*img.Stride+left : y*img.Stride+right])
		}
	}
	return img
}

// PNG writes the symbol as a PNG image; see Image for the parameters.
func (l *Linear) PNG(w io.Writer, scale int) error {
	return png.Encode(w, l.Image(scale))
}

// SVG returns the symbol as a standalone SVG document; see Image for what it
// contains. One user unit equals one pixel at the given scale. The document
// is a white background rectangle and a single black path with one subpath
// per bar, bearer bar and glyph stroke.
func (l *Linear) SVG(scale int) string {
	scale = clampScale(scale)
	w, h, rects := l.layout()
	width, height := w*scale, h*scale

	var b strings.Builder
	b.Grow(256 + 32*len(rects))
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" version="1.1" width="`)
	b.WriteString(strconv.Itoa(width))
	b.WriteString(`" height="`)
	b.WriteString(strconv.Itoa(height))
	b.WriteString(`" viewBox="0 0 `)
	b.WriteString(strconv.Itoa(width))
	b.WriteByte(' ')
	b.WriteString(strconv.Itoa(height))
	b.WriteString(`" shape-rendering="crispEdges">`)
	b.WriteByte('\n')
	b.WriteString(`<rect width="100%" height="100%" fill="#fff"/>`)
	b.WriteByte('\n')
	b.WriteString(`<path fill="#000" d="`)
	for _, r := range rects {
		b.WriteByte('M')
		b.WriteString(strconv.Itoa(r.x * scale))
		b.WriteByte(' ')
		b.WriteString(strconv.Itoa(r.y * scale))
		b.WriteByte('h')
		b.WriteString(strconv.Itoa(r.w * scale))
		b.WriteByte('v')
		b.WriteString(strconv.Itoa(r.h * scale))
		b.WriteString("h-")
		b.WriteString(strconv.Itoa(r.w * scale))
		b.WriteByte('z')
	}
	b.WriteString("\"/>\n</svg>\n")
	return b.String()
}

// clampScale maps the scale argument of Image, PNG, SVG and Size to the
// values they accept.
func clampScale(scale int) int {
	if scale < 1 {
		return 1
	}
	return scale
}
