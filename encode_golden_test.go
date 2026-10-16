package gs1

import (
	"errors"
	"testing"
)

// Golden element strings from GS1 publications. Check digits of every key
// were verified against the GS1 check digit calculator.
var encodeGoldens = []struct {
	name     string
	source   string
	elements []Element
	raw      string
	hri      string
	zpl      string
}{
	{
		name:   "SSCC-only label",
		source: "GS1 Logistics Label Guideline release 1.1, section 10.1 and figure 8-4",
		elements: []Element{
			{AI: "00", Value: "790123451234500116"},
		},
		raw: "\x1D00790123451234500116",
		hri: "(00)790123451234500116",
		zpl: ">800790123451234500116",
	},
	{
		name:   "logistic unit that is a single trade item",
		source: "GS1 Logistics Label Guideline release 1.1, section 10.5 and figure 8-4",
		elements: []Element{
			{AI: "01", Value: "49012345008794"},
			{AI: "00", Value: "790123451234500116"},
		},
		raw: "\x1D014901234500879400790123451234500116",
		hri: "(01)49012345008794(00)790123451234500116",
		zpl: ">8014901234500879400790123451234500116",
	},
	{
		name:   "homogeneous logistic unit",
		source: "GS1 Logistics Label Guideline release 1.1, section 10.2 (SSCC, CONTENT, COUNT, BATCH/LOT, BEST BEFORE)",
		elements: []Element{
			{AI: "10", Value: "ABC123"},
			{AI: "37", Value: "12"},
			{AI: "15", Value: "261231"},
			{AI: "02", Value: "49012345008794"},
			{AI: "00", Value: "790123451234500116"},
		},
		raw: "\x1D15261231024901234500879400790123451234500116" + "10ABC123\x1D3712",
		hri: "(15)261231(02)49012345008794(00)790123451234500116(10)ABC123(37)12",
		zpl: ">815261231024901234500879400790123451234500116" + "10ABC123>83712",
	},
	{
		name:   "homogeneous logistic unit with variable measure items",
		source: "GS1 Logistics Label Guideline release 1.1, section 10.3 (GTIN leading with 9, net weight, count)",
		elements: []Element{
			{AI: "00", Value: "790123451234500116"},
			{AI: "02", Value: "99012345008799"},
			{AI: "3102", Value: "012500"},
			{AI: "37", Value: "20"},
			{AI: "10", Value: "LOT-7"},
		},
		raw: "\x1D0079012345123450011602990123450087993102012500" + "3720\x1D10LOT-7",
		hri: "(00)790123451234500116(02)99012345008799(3102)012500(37)20(10)LOT-7",
		zpl: ">80079012345123450011602990123450087993102012500" + "3720>810LOT-7",
	},
	{
		name:   "trade item with expiry and batch",
		source: "GS1 General Specifications and GS1 DataMatrix Guideline example data",
		elements: []Element{
			{AI: "01", Value: "09501101530003"},
			{AI: "17", Value: "140704"},
			{AI: "10", Value: "AB-123"},
		},
		raw: "\x1D010950110153000317140704" + "10AB-123",
		hri: "(01)09501101530003(17)140704(10)AB-123",
		zpl: ">8010950110153000317140704" + "10AB-123",
	},
	{
		name:   "syntax engine example",
		source: "GS1 Barcode Syntax Engine, src/c-lib/example.c",
		elements: []Element{
			{AI: "01", Value: "09521234543213"},
			{AI: "10", Value: "ABC123"},
			{AI: "99", Value: "TEST"},
		},
		raw: "\x1D0109521234543213" + "10ABC123\x1D99TEST",
		hri: "(01)09521234543213(10)ABC123(99)TEST",
		zpl: ">80109521234543213" + "10ABC123>899TEST",
	},
	{
		name:   "greater-than in batch",
		source: "ZPL II Programming Guide, ^BC Code 128 invocation codes (>< for a literal >)",
		elements: []Element{
			{AI: "01", Value: "09521234543213"},
			{AI: "10", Value: "A>B"},
		},
		raw: "\x1D0109521234543213" + "10A>B",
		hri: "(01)09521234543213(10)A>B",
		zpl: ">80109521234543213" + "10A><B",
	},
}

func TestEncodeGoldens(t *testing.T) {
	for _, tt := range encodeGoldens {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := Encode(tt.elements)
			if err != nil {
				t.Fatalf("Encode() error = %v (%s)", err, tt.source)
			}
			if raw != tt.raw {
				t.Errorf("Encode() = %q, want %q", raw, tt.raw)
			}
			zpl, err := EncodeZPL(tt.elements)
			if err != nil {
				t.Fatalf("EncodeZPL() error = %v", err)
			}
			if zpl != tt.zpl {
				t.Errorf("EncodeZPL() = %q, want %q", zpl, tt.zpl)
			}

			parsed, err := ParseWithOptions(raw, ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true})
			if err != nil {
				t.Fatalf("strict Parse(Encode()) error = %v", err)
			}
			if got := parsed.HRI(); got != tt.hri {
				t.Errorf("HRI() = %q, want %q", got, tt.hri)
			}
			if len(parsed.Elements) != len(tt.elements) {
				t.Fatalf("Parse(Encode()) got %d elements, want %d", len(parsed.Elements), len(tt.elements))
			}
			for _, want := range tt.elements {
				if got, ok := parsed.Get(want.AI); !ok || got != want.Value {
					t.Errorf("Parse(Encode()) AI (%s) = %q, %t, want %q", want.AI, got, ok, want.Value)
				}
			}

			// The HRI is itself valid encoder input (bracket notation).
			reparsed, err := Parse(tt.hri)
			if err != nil {
				t.Fatalf("Parse(HRI) error = %v", err)
			}
			again, err := Encode(reparsed.Elements)
			if err != nil || again != raw {
				t.Errorf("Encode(Parse(HRI)) = %q, %v, want %q", again, err, raw)
			}
		})
	}
}

func TestEncodeZPLErrors(t *testing.T) {
	tests := []struct {
		name     string
		elements []Element
		wantErr  error
	}{
		{name: "empty", wantErr: ErrEmptyInput},
		{name: "unknown AI", elements: []Element{{AI: "999", Value: "x"}}, wantErr: ErrUnknownAI},
		{name: "association", elements: []Element{{AI: "21", Value: "S"}}, wantErr: ErrInvalidAssociation},
		{name: "check digit", elements: []Element{{AI: "01", Value: "04150000021127"}}, wantErr: ErrInvalidCheckDigit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := EncodeZPL(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("EncodeZPL() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodeDoesNotReorderInput(t *testing.T) {
	elements := []Element{{AI: "10", Value: "LOT"}, {AI: "01", Value: "04150000021126"}}
	if _, err := Encode(elements); err != nil {
		t.Fatal(err)
	}
	if elements[0].AI != "10" || elements[1].AI != "01" {
		t.Errorf("Encode() reordered the caller's slice: %+v", elements)
	}
}

func TestEncodeAcceptsStrictParseOutput(t *testing.T) {
	inputs := []string{
		"00790123451234500116024901234500879415261231371210ABC123",
		"]d2010950110153000317140704" + "10AB-123",
		"(01)09521234543213(10)ABC123(99)TEST",
		"(00)106141411234567897(3302)012500",
	}
	for _, input := range inputs {
		parsed, err := ParseWithOptions(input, ParseOptions{ValidateAssociations: true})
		if err != nil {
			t.Fatalf("strict Parse(%q) error = %v", input, err)
		}
		if _, err := Encode(parsed.Elements); err != nil {
			t.Errorf("Encode(strict Parse(%q)) error = %v", input, err)
		}
	}
}
