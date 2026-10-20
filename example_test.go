package gs1_test

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/galenzo17/gs1-go"
)

func ExampleParse() {
	b, err := gs1.Parse("0104150000021126172506302112345ABC\x1D10LOT42X")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("GTIN:  ", b.GTIN())
	fmt.Println("Lot:   ", b.Lot())
	fmt.Println("Serial:", b.SerialNumber())

	exp, _ := b.ExpirationDate()
	fmt.Println("Expiry:", exp.Format("2006-01-02"))
	// Output:
	// GTIN:   04150000021126
	// Lot:    LOT42X
	// Serial: 12345ABC
	// Expiry: 2025-06-30
}

func ExampleParse_bracketNotation() {
	b, _ := gs1.Parse("(01)04150000021126(17)250630(10)ABC123")
	for _, e := range b.Elements {
		ai, _ := gs1.LookupAI(e.AI)
		fmt.Printf("(%s) %-16s %s\n", e.AI, ai.Name, e.Value)
	}
	// Output:
	// (01) GTIN             04150000021126
	// (17) Expiration Date  250630
	// (10) Batch/Lot        ABC123
}

func ExampleParseInto() {
	scans := []string{
		"0104150000021126172506301012345",
		"010415000002112617251231\x1D10LOT2",
	}

	var b gs1.Barcode
	for _, s := range scans {
		b.Reset()
		if err := gs1.ParseInto(s, &b); err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println(b.GTIN(), b.Lot())
	}
	// Output:
	// 04150000021126 12345
	// 04150000021126 LOT2
}

func ExampleParseInto_scannerLoop() {
	scanner := bufio.NewScanner(strings.NewReader("01041500000211261725063010LOT1\n"))
	var barcode gs1.Barcode
	for scanner.Scan() {
		barcode.Reset()
		if err := gs1.ParseInto(scanner.Text(), &barcode); err != nil {
			continue
		}
		fmt.Println(barcode.GTIN(), barcode.Lot())
	}
	// Output:
	// 04150000021126 LOT1
}

func ExampleValidateGTIN() {
	fmt.Println(gs1.ValidateGTIN("04150000021126"))

	err := gs1.ValidateGTIN("04150000021127")
	fmt.Println(errors.Is(err, gs1.ErrInvalidCheckDigit))
	// Output:
	// <nil>
	// true
}

func ExampleEncode() {
	elements := []gs1.Element{
		{AI: "10", Value: "ABC123"},
		{AI: "00", Value: "790123451234500116"},
		{AI: "02", Value: "49012345008794"},
		{AI: "37", Value: "12"},
	}
	raw, _ := gs1.Encode(elements)
	zpl, _ := gs1.EncodeZPL(elements)
	b, _ := gs1.Parse(raw)
	fmt.Printf("%q\n%s\n%s\n", raw, zpl, b.HRI())

	_, err := gs1.Encode([]gs1.Element{{AI: "02", Value: "49012345008794"}})
	fmt.Println(errors.Is(err, gs1.ErrInvalidAssociation), err)
	// Output:
	// "\x1d00790123451234500116024901234500879410ABC123\x1d3712"
	// >800790123451234500116024901234500879410ABC123>83712
	// (00)790123451234500116(02)49012345008794(10)ABC123(37)12
	// true gs1: invalid AI association: AI (02) requires AI (37)
}

func ExampleParseHRI() {
	elements, _ := gs1.ParseHRI("(01)04150000021126(10)LOT42")
	fmt.Println(elements)
	// Output:
	// [{01 04150000021126} {10 LOT42}]
}

func ExampleValidateSSCC() {
	fmt.Println(gs1.ValidateSSCC("106141411234567897"))

	err := gs1.ValidateSSCC("106141411234567891")
	fmt.Println(errors.Is(err, gs1.ErrInvalidCheckDigit))
	// Output:
	// <nil>
	// true
}

func ExampleValidateKey() {
	types, err := gs1.ValidateKey("0614141000012")
	fmt.Println(types, err)

	_, err = gs1.ValidateKey("106141411234567891")
	fmt.Println(errors.Is(err, gs1.ErrInvalidCheckDigit))
	// Output:
	// [GTIN GLN] <nil>
	// true
}

func ExampleParseWithOptions_checkDigits() {
	_, err := gs1.ParseWithOptions("00106141411234567891", gs1.ParseOptions{ValidateCheckDigits: true})
	fmt.Println(errors.Is(err, gs1.ErrInvalidCheckDigit), err)
	// Output:
	// true at position 0: AI (00): gs1: invalid check digit: expected '7', got '1'
}

func ExampleParseDateWithOptions() {
	last, _ := gs1.ParseDate("250200")
	first, _ := gs1.ParseDateWithOptions("250200", gs1.DateOptions{DayZero: gs1.DayZeroFirstDay})
	fmt.Println(last.Format("2006-01-02"), first.Format("2006-01-02"))
	// Output:
	// 2025-02-28 2025-02-01
}

func ExampleRegulator_Validate() {
	b, _ := gs1.Parse("01041500000211261725063010LOT42X")

	fmt.Println(gs1.COFEPRIS.Validate(b))

	err := gs1.ANVISA.Validate(b)
	fmt.Println(errors.Is(err, gs1.ErrMissingRequiredAI), err)
	// Output:
	// <nil>
	// true gs1: missing required AI: ANVISA requires AI (21)
}

func ExampleLookupAI() {
	ai, ok := gs1.LookupAI("3102")
	fmt.Println(ok, ai.Name, ai.Format)
	// Output:
	// true Net Weight kg N6
}

func ExampleParseWithOptions() {
	// Strict parsing enforces AI association rules and GTIN check digits.
	_, err := gs1.ParseWithOptions("0204150000021126", gs1.ParseOptions{ValidateAssociations: true})
	fmt.Println(err)

	_, err = gs1.ParseWithOptions("0104150000021127", gs1.ParseOptions{ValidateCheckDigits: true})
	fmt.Println(errors.Is(err, gs1.ErrInvalidCheckDigit))
	// Output:
	// gs1: invalid AI association: AI (02) requires AI (37)
	// true
}

func ExampleBarcode_ValidateAssociations() {
	b, _ := gs1.Parse("0104150000021126" + "3720")
	fmt.Println(b.ValidateAssociations())

	b, _ = gs1.Parse("00106141411234567897" + "0204150000021126" + "3720")
	fmt.Println(b.ValidateAssociations())
	// Output:
	// gs1: invalid AI association: AI (01) excludes AI (37)
	// <nil>
}

func ExampleBarcode_Validate() {
	b, _ := gs1.Parse("0104150000021127")
	fmt.Println(errors.Is(b.Validate(), gs1.ErrInvalidCheckDigit))
	// Output:
	// true
}

func ExampleBarcode_HRI() {
	b, _ := gs1.Parse("0104150000021126172506301012345")
	fmt.Println(b.HRI())
	// Output:
	// (01)04150000021126(17)250630(10)12345
}

func ExampleNewSSCC() {
	sscc, err := gs1.NewSSCC('3', "7801234", "000000009")
	fmt.Println(sscc, err)

	extension, companyPrefix, serialReference, _ := gs1.SplitSSCC(sscc, 7)
	fmt.Printf("%c %s %s\n", extension, companyPrefix, serialReference)
	// Output:
	// 378012340000000095 <nil>
	// 3 7801234 000000009
}

func ExampleNewGTIN14() {
	gtin13, _ := gs1.NewGTIN13("7801234", "00005")
	caseGTIN, _ := gs1.NewGTIN14('1', gtin13)
	fmt.Println(gtin13, caseGTIN)
	// Output:
	// 7801234000056 17801234000053
}

func ExampleExpandUPCE() {
	upca, err := gs1.ExpandUPCE("012345")
	fmt.Println(upca, err)
	// Output:
	// 001234000057 <nil>
}

func ExampleComputeGTINCheckDigit() {
	check, _ := gs1.ComputeGTINCheckDigit("0415000002112")
	fmt.Printf("%c\n", check)
	// Output:
	// 6
}

func ExampleBarcode_NetWeightKg() {
	b, _ := gs1.Parse("0104150000021126" + "3102001500")
	weight, ok := b.NetWeightKg()
	fmt.Println(ok, weight.Value, weight.Unit, weight.Raw, weight.Decimals)
	// Output:
	// true 15 kg 001500 2
}

func ExampleBarcode_Warnings() {
	b, _ := gs1.Parse("0104150000021126" + "12250630")
	for _, w := range b.Warnings() {
		fmt.Println(w.Code, w.Message)
	}
	// Output:
	// due_date_as_expiry AI (12) is a due date; use AI (17) for expiration dates
}

func ExampleParseDateRaw() {
	raw, err := gs1.ParseDateRaw("250200")
	fmt.Println(raw, err)

	_, err = gs1.ParseDateRaw("251340")
	fmt.Println(errors.Is(err, gs1.ErrInvalidDate))
	// Output:
	// 250200 <nil>
	// true
}
