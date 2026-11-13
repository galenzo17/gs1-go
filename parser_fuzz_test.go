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
	f.Add("10LOT(90)X")
	f.Add("10LOT(X")
	f.Add("10LOT)X")
	f.Add("10LOT(90)X\x1D21S(1)")
	f.Add("]d2\x1D(10)LOT(21)SERIAL")
	f.Add("\xEF\xBB\xBF \x00]d2\x1D(10)LOT(21)SERIAL\r\n")
	f.Add("21SERIAL1\x1D10BATCH2")
	f.Add("3102001500")
	f.Add("713BR12345678")
	f.Add("0104150000021126700325063012301020ABC")
	f.Add("00106141411234567897421528ABC12\x1D4300ACME")
	f.Add("8010ABC-123\x1D8011123")
	f.Add("0104150000021126" + "8030AbC_-123")
	f.Add("4330012345-")
	f.Add("423528076040")
	f.Add("00106141411234567897")
	f.Add("4140614141000012")
	f.Add("40206141410000000012")
	f.Add("8017061414112345678902")
	f.Add("2530614141000012ABC")
	f.Add("800300614141000012XYZ")
	f.Add("0312345678901231")
	f.Add("2551234567890128123456789012")
	f.Add("8006123456789012310202")
	f.Add("001061414112345678978026123456789012310102\x1D372")
	// Invalid inputs
	f.Add("")
	f.Add("abc")
	f.Add("\x1D")
	f.Add("9999999")
	f.Add("01")
	f.Add("0104150000021127")
	f.Add("0204150000021127")
	f.Add("]E000123458")
	f.Add("]I20123456789012")
	f.Add("00106141411234567891")
	f.Add("4140614141000013")
	f.Add("0312345678901232")
	f.Add("2551234567890129")
	f.Add("8006123456789012320102")
	f.Add("8026123456789012320102")

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

		assertCheckDigitParseReuse(t, input, &reused)
	})
}

func assertCheckDigitParseReuse(t *testing.T, input string, reused *Barcode) {
	t.Helper()
	// Optional check-digit validation must agree across allocating and
	// reusable parsing, including carrier normalization.
	opts := ParseOptions{ValidateCheckDigits: true}
	checked, checkErr := ParseWithOptions(input, opts)
	reused.Reset()
	intoErr := ParseIntoWithOptions(input, reused, opts)
	if (checkErr == nil) != (intoErr == nil) {
		t.Fatalf("check-digit parsing errors differ: ParseWithOptions %v, ParseIntoWithOptions %v", checkErr, intoErr)
	}
	if checkErr != nil {
		if checkErr.Error() != intoErr.Error() {
			t.Errorf("check-digit error context differs: %v != %v", checkErr, intoErr)
		}
		return
	}
	if len(checked.Elements) != len(reused.Elements) {
		t.Fatalf("check-digit parsing element counts differ: %d != %d", len(checked.Elements), len(reused.Elements))
	}
	for i := range checked.Elements {
		if checked.Elements[i] != reused.Elements[i] {
			t.Errorf("check-digit element[%d] differs: %+v != %+v", i, checked.Elements[i], reused.Elements[i])
		}
	}
}
