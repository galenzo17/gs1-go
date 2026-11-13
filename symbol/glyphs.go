package symbol

// HRI digits are drawn with a built-in 5x7 glyph set so that rendering needs
// no font. The GS1 General Specifications ask for OCR-B at a size tied to the
// X-dimension; a seven-module glyph in a seven-module slot approximates that
// geometry (EAN-13 HRI is nominally 2.75 mm high at X = 0.33 mm, about eight
// modules) and keeps the digits legible at any scale.
const (
	glyphWidth  = 5
	glyphHeight = 7
)

// digitGlyphs holds one row per byte, the five least significant bits being
// the columns from left (bit 4) to right (bit 0).
var digitGlyphs = [10][glyphHeight]uint8{
	{0b01110, 0b10001, 0b10011, 0b10101, 0b11001, 0b10001, 0b01110}, // 0
	{0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110}, // 1
	{0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b01000, 0b11111}, // 2
	{0b11111, 0b00010, 0b00100, 0b00010, 0b00001, 0b10001, 0b01110}, // 3
	{0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010}, // 4
	{0b11111, 0b10000, 0b11110, 0b00001, 0b00001, 0b10001, 0b01110}, // 5
	{0b00110, 0b01000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110}, // 6
	{0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000}, // 7
	{0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110}, // 8
	{0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00010, 0b01100}, // 9
}

// glyphRects appends the dark rectangles of the glyph for c, one per
// horizontal run of set bits in each row, with the glyph's top-left corner
// at (x, y). Characters other than digits draw nothing.
func glyphRects(rects []rect, c byte, x, y int) []rect {
	if c < '0' || c > '9' {
		return rects
	}
	for row, bits := range digitGlyphs[c-'0'] {
		for col := 0; col < glyphWidth; {
			if bits&(1<<(glyphWidth-1-col)) == 0 {
				col++
				continue
			}
			run := 1
			for col+run < glyphWidth && bits&(1<<(glyphWidth-1-col-run)) != 0 {
				run++
			}
			rects = append(rects, rect{x + col, y + row, run, 1})
			col += run
		}
	}
	return rects
}
