package symbol_test

import (
	"fmt"
	"strings"

	"github.com/galenzo17/gs1-go/symbol"
)

func ExampleEAN13() {
	l, err := symbol.EAN13("4006381333931")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	w, h := l.Size(1)
	fmt.Println(l.Symbology, l.Modules, "modules wide,", l.HRI())
	fmt.Println("renders", w, "x", h, "modules with quiet zones and HRI")
	// Output:
	// EAN-13 95 modules wide, 4006381333931
	// renders 113 x 77 modules with quiet zones and HRI
}

// ExampleUPCE shows that the check digit is derived from the UPC-A the
// zero-suppressed code expands to.
func ExampleUPCE() {
	l, err := symbol.UPCE("0123456")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(l.HRI())
	// Output: 01234565
}

// ExampleITF14 draws the full bearer frame used when the symbol is printed
// directly on corrugated cases.
func ExampleITF14() {
	l, err := symbol.ITF14("10614141000019", symbol.ITF14Options{Bearer: symbol.BearerFrame})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	w, h := l.Size(4)
	fmt.Println(l.Modules, "modules;", w, "x", h, "pixels at 4 pixels per module")
	// Output: 135 modules; 660 x 200 pixels at 4 pixels per module
}

func ExampleLinear_SVG() {
	l, err := symbol.EAN8("96385074")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	header, _, _ := strings.Cut(l.SVG(3), "\n")
	fmt.Println(header)
	// Output: <svg xmlns="http://www.w3.org/2000/svg" version="1.1" width="243" height="189" viewBox="0 0 243 189" shape-rendering="crispEdges">
}
