package symbol

import (
	"fmt"

	"github.com/galenzo17/gs1-go"
)

// Bearer selects the bearer bars of an ITF-14 symbol. GS1 General
// Specifications section 5.3 (ITF-14) require them: a full frame when the symbol
// is printed from a plate, otherwise at least horizontal bars touching the
// top and bottom of the bars. They stop the scanner from reading a partial
// symbol and even out print pressure.
type Bearer int

const (
	// BearerHorizontal draws a bar along the top and the bottom of the
	// symbol, quiet zones included. It is the default.
	BearerHorizontal Bearer = iota
	// BearerFrame draws a closed frame; the vertical bars sit outside the
	// quiet zones.
	BearerFrame
	// BearerNone draws no bearer bar.
	BearerNone
)

// ITF14Options configures an ITF-14 symbol. The zero value is the GS1
// nominal symbol with horizontal bearer bars.
type ITF14Options struct {
	// Bearer selects the bearer bar style.
	Bearer Bearer
	// BearerWidth is the bearer bar thickness in modules. Zero selects the
	// GS1 minimum for the style: two modules for horizontal bars and five
	// (4.83 mm at the nominal X-dimension of 1.016 mm) for a frame. A
	// smaller explicit value is drawn as given but does not conform.
	BearerWidth int
	// Height is the bar height in modules. Zero selects 32, the nominal
	// 31.75 mm at the nominal X-dimension.
	Height int
}

// itf14Patterns gives, per digit, the five elements of an Interleaved 2 of 5
// character (ISO/IEC 16390 encodation table): a set bit is a wide element, bit 4
// being the first.
var itf14Patterns = [10]uint8{
	0b00110, 0b10001, 0b01001, 0b11000, 0b00101,
	0b10100, 0b01100, 0b00011, 0b10010, 0b01010,
}

// ITF-14 geometry in modules. The wide-to-narrow ratio is 3:1, the upper
// bound of the 2.25:1 to 3:1 range of GS1 General Specifications section
// 5.3 and the ratio Zebra printers and zint use, so the symbol is 135
// modules wide. Quiet zones are 10X each (same section).
const (
	itf14Narrow         = 1
	itf14Wide           = 3
	itf14Height         = 32
	itf14Quiet          = 10
	itf14BearerMin      = 2
	itf14BearerFrameMin = 5
	itf14Digits         = 14
)

// ITF14 encodes a 14-digit GTIN-14 as an ITF-14 symbol with the digits
// printed below the bottom bearer bar. The check digit is validated.
func ITF14(gtin string, opts ITF14Options) (*Linear, error) {
	if err := checkGTIN("ITF-14", gtin, itf14Digits); err != nil {
		return nil, err
	}
	if opts.BearerWidth < 0 || opts.Height < 0 {
		return nil, fmt.Errorf("%w: ITF-14 bearer width and height must not be negative", gs1.ErrInvalidData)
	}
	height := opts.Height
	if height == 0 {
		height = itf14Height
	}
	var bearer bearerBars
	switch opts.Bearer {
	case BearerHorizontal:
		bearer.horizontal = bearerWidth(opts.BearerWidth, itf14BearerMin)
	case BearerFrame:
		bearer.horizontal = bearerWidth(opts.BearerWidth, itf14BearerFrameMin)
		bearer.vertical = bearer.horizontal
	case BearerNone:
	default:
		return nil, fmt.Errorf("%w: unknown ITF-14 bearer style %d", gs1.ErrInvalidData, opts.Bearer)
	}

	l := newLinear("ITF-14", height, itf14Quiet, itf14Quiet)
	l.hri = gtin
	l.bearer = bearer
	// Start pattern: narrow bar, space, bar, space.
	l.bar(itf14Narrow)
	l.space(itf14Narrow)
	l.bar(itf14Narrow)
	l.space(itf14Narrow)
	for i := 0; i < itf14Digits; i += 2 {
		bars, spaces := itf14Patterns[gtin[i]-'0'], itf14Patterns[gtin[i+1]-'0']
		for e := 4; e >= 0; e-- {
			l.bar(itf14Width(bars, e))
			l.space(itf14Width(spaces, e))
		}
	}
	// Stop pattern: wide bar, narrow space, narrow bar.
	l.bar(itf14Wide)
	l.space(itf14Narrow)
	l.bar(itf14Narrow)

	text := glyphPitch*itf14Digits - (glyphPitch - glyphWidth)
	l.text = []textRun{{(l.Modules - text) / 2, height + bearer.horizontal + hriGap, gtin}}
	return l, nil
}

// bearerWidth returns the requested bearer bar thickness, or the GS1 minimum
// for the style when none was given.
func bearerWidth(requested, minimum int) int {
	if requested == 0 {
		return minimum
	}
	return requested
}

// itf14Width returns the width of element e (bit e of pattern) in modules.
func itf14Width(pattern uint8, e int) int {
	if pattern&(1<<e) != 0 {
		return itf14Wide
	}
	return itf14Narrow
}
