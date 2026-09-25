package gs1

import (
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	// Valid barcodes
	f.Add("0104150000021126172506301012345")
	f.Add("0104150000021126172506302112345ABC\x1D10LOT42X")
	f.Add("]d20104150000021126172506301012345")
	f.Add("\x1D0104150000021126")
	f.Add("10ABC123")
	f.Add("21SERIAL1\x1D10BATCH2")
	f.Add("3102001500")
	f.Add("713BR12345678")
	// Invalid inputs
	f.Add("")
	f.Add("abc")
	f.Add("\x1D")
	f.Add("9999999")
	f.Add("01")

	var reused Barcode

	f.Fuzz(func(t *testing.T, input string) {
		b, err := Parse(input)
		if err != nil {
			// ParseInto must return the same error class.
			reused.Reset()
			err2 := ParseInto(input, &reused)
			if err2 == nil {
				t.Errorf("Parse returned error but ParseInto did not: %v", err)
			}
			return
		}
		if strings.TrimSpace(input) != "" && len(b.Elements) == 0 {
			t.Fatalf("successful parse of non-empty input %q returned no elements", input)
		}

		// Invariants: all elements have non-empty AI and Value.
		for i, e := range b.Elements {
			if e.AI == "" {
				t.Errorf("element[%d] has empty AI", i)
			}
			if e.Value == "" {
				t.Errorf("element[%d] (AI %s) has empty Value", i, e.AI)
			}
		}

		// Convenience methods must not panic.
		_ = b.GTIN()
		_ = b.Lot()
		_ = b.SerialNumber()
		_ = b.SSCC()
		_ = b.Count()
		_, _ = b.ExpirationDate()
		_, _ = b.ProductionDate()
		_, _ = b.BestBeforeDate()

		// Get for each parsed AI must return the value.
		for _, e := range b.Elements {
			v, ok := b.Get(e.AI)
			if !ok {
				t.Errorf("Get(%q) returned false for parsed AI", e.AI)
			}
			if v == "" {
				t.Errorf("Get(%q) returned empty string for parsed AI", e.AI)
			}
		}

		// Cross-validate ParseInto produces identical results.
		reused.Reset()
		if err := ParseInto(input, &reused); err != nil {
			t.Fatalf("ParseInto failed where Parse succeeded: %v", err)
		}
		if len(reused.Elements) != len(b.Elements) {
			t.Fatalf("ParseInto elements count %d != Parse %d", len(reused.Elements), len(b.Elements))
		}
		for i := range b.Elements {
			if reused.Elements[i] != b.Elements[i] {
				t.Errorf("ParseInto element[%d] = %+v, Parse = %+v", i, reused.Elements[i], b.Elements[i])
			}
		}
	})
}
