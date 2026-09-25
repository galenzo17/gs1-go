package gs1

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   error
		wantCount int
		wantAIs   []string
	}{
		// Single AI — fixed length
		{
			name:      "GTIN only",
			input:     "0104150000021126",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		// Single AI — variable length
		{
			name:      "lot only",
			input:     "10ABC123",
			wantCount: 1,
			wantAIs:   []string{"10"},
		},
		// Multiple fixed-length AIs (no FNC1 needed)
		{
			name:      "GTIN + expiry",
			input:     "0104150000021126" + "17250630",
			wantCount: 2,
			wantAIs:   []string{"01", "17"},
		},
		// Fixed + variable with FNC1
		{
			name:      "GTIN + expiry + lot",
			input:     "0104150000021126" + "17250630" + "10BATCH42",
			wantCount: 3,
			wantAIs:   []string{"01", "17", "10"},
		},
		// Variable + variable with FNC1 separator
		{
			name:      "serial + lot with FNC1",
			input:     "21SN12345\x1D10LOT999",
			wantCount: 2,
			wantAIs:   []string{"21", "10"},
		},
		// Full healthcare barcode (GTIN + expiry + serial + lot)
		{
			name:      "full healthcare barcode",
			input:     "0104150000021126172506302112345ABC\x1D10LOT42X",
			wantCount: 4,
			wantAIs:   []string{"01", "17", "21", "10"},
		},
		// AIM prefix DataMatrix
		{
			name:      "AIM prefix ]d2",
			input:     "]d20104150000021126172506301012345",
			wantCount: 3,
			wantAIs:   []string{"01", "17", "10"},
		},
		// AIM prefix GS1-128
		{
			name:      "AIM prefix ]C1",
			input:     "]C10104150000021126172506301012345",
			wantCount: 3,
			wantAIs:   []string{"01", "17", "10"},
		},
		// Bare GTIN (EAN-13, EAN-8, UPC-A — no AI prefix)
		{
			name:      "bare EAN-13",
			input:     "7800038041425",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		{
			name:      "bare UPC-A",
			input:     "036000291452",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		// Leading FNC1
		{
			name:      "leading FNC1",
			input:     "\x1D0104150000021126",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		// 3-digit AI
		{
			name:      "AI 240 additional product",
			input:     "240PRODUCTCODE001",
			wantCount: 1,
			wantAIs:   []string{"240"},
		},
		{
			name:      "GDTI",
			input:     "2531234567890123DOC",
			wantCount: 1,
			wantAIs:   []string{"253"},
		},
		{
			name:      "GINC",
			input:     "401GINC123",
			wantCount: 1,
			wantAIs:   []string{"401"},
		},
		{
			name:      "GRAI with serial",
			input:     "800301234567890123SERIAL",
			wantCount: 1,
			wantAIs:   []string{"8003"},
		},
		{
			name:      "GRAI without serial",
			input:     "800301234567890123",
			wantCount: 1,
			wantAIs:   []string{"8003"},
		},
		{
			name:      "GIAI",
			input:     "8004ASSET123",
			wantCount: 1,
			wantAIs:   []string{"8004"},
		},
		{
			name:      "GSRN provider",
			input:     "8017123456789012345678",
			wantCount: 1,
			wantAIs:   []string{"8017"},
		},
		{
			name:      "GSRN recipient",
			input:     "8018123456789012345678",
			wantCount: 1,
			wantAIs:   []string{"8018"},
		},
		// 4-digit AI (weight)
		{
			name:      "AI 3102 net weight kg",
			input:     "3102001500",
			wantCount: 1,
			wantAIs:   []string{"3102"},
		},
		// 3-digit AI NHRN Brazil
		{
			name:      "AI 713 NHRN Brazil",
			input:     "713BR12345678",
			wantCount: 1,
			wantAIs:   []string{"713"},
		},
		// SSCC
		{
			name:      "SSCC",
			input:     "00123456789012345675",
			wantCount: 1,
			wantAIs:   []string{"00"},
		},
		// GLN
		{
			name:      "GLN",
			input:     "4141234567890123",
			wantCount: 1,
			wantAIs:   []string{"414"},
		},
		{
			name:      "ship-to GLN",
			input:     "4101234567890123",
			wantCount: 1,
			wantAIs:   []string{"410"},
		},
		{
			name:      "party GLN",
			input:     "4171234567890123",
			wantCount: 1,
			wantAIs:   []string{"417"},
		},
		{
			name:      "GLN extension component",
			input:     "254EXTENSION",
			wantCount: 1,
			wantAIs:   []string{"254"},
		},
		{
			name:      "GS1 UIC with extension",
			input:     "70401ABC",
			wantCount: 1,
			wantAIs:   []string{"7040"},
		},
		// GSIN
		{
			name:      "GSIN",
			input:     "40212345678901234567",
			wantCount: 1,
			wantAIs:   []string{"402"},
		},
		// Gross weight
		{
			name:      "gross weight kg",
			input:     "3302001500",
			wantCount: 1,
			wantAIs:   []string{"3302"},
		},
		// Count AI
		{
			name:      "count",
			input:     "3025",
			wantCount: 1,
			wantAIs:   []string{"30"},
		},
		// Production date
		{
			name:      "production date",
			input:     "11250101",
			wantCount: 1,
			wantAIs:   []string{"11"},
		},
		// Trailing FNC1 (some scanners emit it)
		{
			name:      "trailing FNC1",
			input:     "10LOT1\x1D",
			wantCount: 1,
			wantAIs:   []string{"10"},
		},

		// Bracket notation
		{
			name:      "bracket notation simple",
			input:     "(01)04150000021126(17)250630",
			wantCount: 2,
			wantAIs:   []string{"01", "17"},
		},
		{
			name:      "bracket notation full",
			input:     "(02)17795678901213(10)AAB123(17)201231(37)20",
			wantCount: 4,
			wantAIs:   []string{"02", "10", "17", "37"},
		},

		// Scanner resilience
		{
			name:      "trailing CRLF",
			input:     "0104150000021126172506301012345\r\n",
			wantCount: 3,
			wantAIs:   []string{"01", "17", "10"},
		},
		{
			name:      "trailing LF",
			input:     "0104150000021126\n",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		{
			name:      "BOM prefix",
			input:     "\xEF\xBB\xBF0104150000021126",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		{
			name:      "CR as FNC1",
			input:     "2112345\r10LOT1",
			wantCount: 2,
			wantAIs:   []string{"21", "10"},
		},
		{
			name:      "null bytes stripped",
			input:     "01041500000211261725063010\x0012345",
			wantCount: 3,
			wantAIs:   []string{"01", "17", "10"},
		},
		{
			name:      "AIM prefix ]d1",
			input:     "]d10104150000021126",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		{
			name:      "AIM prefix ]Q3",
			input:     "]Q30104150000021126",
			wantCount: 1,
			wantAIs:   []string{"01"},
		},
		{
			name:      "BOM + AIM + data + CRLF",
			input:     "\xEF\xBB\xBF]d20104150000021126172506301012345\r\n",
			wantCount: 3,
			wantAIs:   []string{"01", "17", "10"},
		},
		{
			name:      "double FNC1 between fields",
			input:     "2112345\x1D\x1D10LOT1",
			wantCount: 2,
			wantAIs:   []string{"21", "10"},
		},

		// Error cases
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrEmptyInput,
		},
		{
			name:    "only spaces",
			input:   "   ",
			wantErr: ErrEmptyInput,
		},
		{
			name:    "unknown AI",
			input:   "8812345",
			wantErr: ErrUnknownAI,
		},
		// Custom AI 90-99
		{
			name:      "custom AI 90",
			input:     "90CUSTOMDATA\x1D",
			wantCount: 1,
			wantAIs:   []string{"90"},
		},
		{
			name:    "truncated GTIN",
			input:   "010415000002",
			wantErr: ErrTruncatedData,
		},
		{
			name:    "non-numeric in numeric field",
			input:   "01ABCDEFGHIJKLMN",
			wantErr: ErrInvalidData,
		},
		{
			name:    "UIC extension must start with a digit",
			input:   "7040A123",
			wantErr: ErrInvalidData,
		},
		{
			name:    "GRAI must start with zero",
			input:   "80031234567890123X",
			wantErr: ErrInvalidData,
		},
		{
			name:    "GDTI numeric prefix is required",
			input:   "253123456789012X",
			wantErr: ErrInvalidData,
		},
		{
			name:    "variable data exceeds max",
			input:   "10AAAAABBBBBCCCCCDDDDDE",
			wantErr: ErrInvalidData,
		},
		// Missing FNC1 recovery — scanner omits separator between variable-length fields.
		{
			name:      "missing FNC1 between lot and serial",
			input:     "0108906025521297112310001727090010HC23I1605245021121746021",
			wantCount: 5,
			wantAIs:   []string{"01", "11", "17", "10", "21"},
		},
		{
			name:      "missing FNC1 with AI code inside lot value",
			input:     "0108906025521365112401001727120010HC23L2521280021122025725",
			wantCount: 5,
			wantAIs:   []string{"01", "11", "17", "10", "21"},
		},
		{
			name:    "only FNC1",
			input:   "\x1D",
			wantErr: ErrEmptyInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Parse(tt.input)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Parse(%q) = %v, want error %v", tt.input, b.Elements, tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Parse(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if len(b.Elements) != tt.wantCount {
				t.Errorf("Parse(%q) got %d elements, want %d", tt.input, len(b.Elements), tt.wantCount)
			}
			for i, wantAI := range tt.wantAIs {
				if i >= len(b.Elements) {
					break
				}
				if b.Elements[i].AI != wantAI {
					t.Errorf("Parse(%q) element[%d].AI = %q, want %q", tt.input, i, b.Elements[i].AI, wantAI)
				}
			}
		})
	}
}

func TestMissingFNC1RecoveryKeepsTwoDigitBoundary(t *testing.T) {
	for _, input := range []string{
		"10PO401ABCDEFGHIJKLMN21SERIAL",
		"10L2401ABCDEFGHIJKLMN21SERIAL",
	} {
		b, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", input, err)
		}
		if got := b.Elements; len(got) != 2 || got[0].AI != "10" || got[1].AI != "21" {
			t.Errorf("Parse(%q) elements = %+v, want AI 10 followed by AI 21", input, got)
		}
	}
}

func TestParseConvenienceMethods(t *testing.T) {
	input := "0104150000021126" + "17250630" + "2112345ABC\x1D" + "10LOT42X"
	b, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := b.GTIN(); got != "04150000021126" {
		t.Errorf("GTIN() = %q, want %q", got, "04150000021126")
	}
	if got := b.Lot(); got != "LOT42X" {
		t.Errorf("Lot() = %q, want %q", got, "LOT42X")
	}
	if got := b.SerialNumber(); got != "12345ABC" {
		t.Errorf("SerialNumber() = %q, want %q", got, "12345ABC")
	}
	expiry, err := b.ExpirationDate()
	if err != nil {
		t.Fatalf("ExpirationDate() error = %v", err)
	}
	if !expiry.Equal(time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("ExpirationDate() = %v, want 2025-06-30", expiry)
	}

	v, ok := b.Get("01")
	if !ok || v != "04150000021126" {
		t.Errorf("Get(01) = (%q, %v), want (%q, true)", v, ok, "04150000021126")
	}

	_, ok = b.Get("00")
	if ok {
		t.Error("Get(00) should return false for missing AI")
	}
}

func TestParseAdditionalConvenienceMethods(t *testing.T) {
	input := "0104150000021126" + "0204150000021126" + "13250601" + "3742\x1D" + "40212345678901234567" + "4141234567890123"
	b, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	for _, tt := range []struct {
		name, got, want string
	}{
		{"ContentGTIN", b.ContentGTIN(), "04150000021126"},
		{"CountOfTradeItems", b.CountOfTradeItems(), "42"},
		{"GLN", b.GLN(), "1234567890123"},
		{"GSIN", b.GSIN(), "12345678901234567"},
	} {
		if tt.got != tt.want {
			t.Errorf("%s() = %q, want %q", tt.name, tt.got, tt.want)
		}
	}

	packaging, err := b.PackagingDate()
	if err != nil {
		t.Fatalf("PackagingDate() error = %v", err)
	}
	if !packaging.Equal(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("PackagingDate() = %v, want 2025-06-01", packaging)
	}
}

func TestParseConvenienceMethodsMissing(t *testing.T) {
	b, err := Parse("0104150000021126")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := b.Lot(); got != "" {
		t.Errorf("Lot() = %q, want empty", got)
	}
	if got := b.SerialNumber(); got != "" {
		t.Errorf("SerialNumber() = %q, want empty", got)
	}

	_, err = b.ExpirationDate()
	if err == nil {
		t.Error("ExpirationDate() should error when AI 17 is missing")
	}

	_, err = b.PackagingDate()
	if !errors.Is(err, ErrInvalidData) {
		t.Errorf("PackagingDate() error = %v, want ErrInvalidData", err)
	}
}

func TestDueDateWarning(t *testing.T) {
	b, err := Parse("12250630")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	due, err := b.DueDate()
	if err != nil {
		t.Fatalf("DueDate() error = %v", err)
	}
	if !due.Equal(time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("DueDate() = %v, want 2025-06-30", due)
	}
	if got := b.Warnings(); len(got) != 1 || got[0].Code != WarnDueDateAsExpiry {
		t.Errorf("Warnings() = %+v, want one %q warning", got, WarnDueDateAsExpiry)
	}
	if _, err := b.ExpirationDate(); err == nil {
		t.Error("ExpirationDate() should still require AI (17)")
	}
}

func TestBareGTINWithKnownAIPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"UPC-A", "123456789012", "00123456789012"},
		{"EAN-13", "1234567890128", "01234567890128"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(b.Elements) != 1 || b.Elements[0].AI != "01" || b.Elements[0].Value != tt.want {
				t.Fatalf("Parse() elements = %+v, want AI 01 value %q", b.Elements, tt.want)
			}
			if got := b.Warnings(); len(got) != 0 {
				t.Fatalf("Parse() warnings = %+v, want none", got)
			}

			var reused Barcode
			if err := ParseInto(tt.input, &reused); err != nil {
				t.Fatalf("ParseInto() error = %v", err)
			}
			if len(reused.Elements) != 1 || reused.Elements[0].AI != "01" || reused.Elements[0].Value != tt.want {
				t.Fatalf("ParseInto() elements = %+v, want AI 01 value %q", reused.Elements, tt.want)
			}
			if got := reused.Warnings(); len(got) != 0 {
				t.Fatalf("ParseInto() warnings = %+v, want none", got)
			}
		})
	}
}

func TestDueDateWithExpirationHasNoWarning(t *testing.T) {
	b, err := Parse("1225063017250630")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := b.Warnings(); len(got) != 0 {
		t.Errorf("Warnings() = %+v, want none", got)
	}
}

func TestExpirationDateWithoutDueDateHasNoWarning(t *testing.T) {
	b, err := Parse("17250630")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := b.Warnings(); len(got) != 0 {
		t.Errorf("Warnings() = %+v, want none", got)
	}
}

func TestValidateAssociations(t *testing.T) {
	tests := []struct {
		name   string
		events []Element
		want   string
	}{
		{name: "01 excludes 02", events: []Element{{AI: "01"}, {AI: "02"}}, want: "AI (01) excludes AI (02)"},
		{name: "02 requires 37", events: []Element{{AI: "02"}}, want: "AI (02) requires AI (37)"},
		{name: "37 requires logistic pair", events: []Element{{AI: "37"}}, want: "AI (37) requires AI (00) with AI (02) or AI (8026)"},
		{name: "21 requires identifier", events: []Element{{AI: "21"}}, want: "AI (21) requires AI (01), AI (03), or AI (8006)"},
		{name: "weight requires GTIN", events: []Element{{AI: "3102"}}, want: "AI (310) requires AI (01) or AI (02)"},
		{name: "one weight decimal variant", events: []Element{{AI: "01"}, {AI: "3102"}, {AI: "3103"}}, want: "AI (310) allows only one decimal variant"},
		{name: "8017 excludes 8018", events: []Element{{AI: "8017"}, {AI: "8018"}}, want: "AI (8017) excludes AI (8018)"},
		{name: "01 excludes 37", events: []Element{{AI: "01"}, {AI: "37"}}, want: "AI (01) excludes AI (37)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (Barcode{Elements: tt.events}).Validate()
			if !errors.Is(err, ErrInvalidAssociation) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate() error = %v, want %q", err, tt.want)
			}
		})
	}

	for _, valid := range []Barcode{
		{Elements: []Element{{AI: "00"}}},
		{Elements: []Element{{AI: "00"}, {AI: "02"}, {AI: "37"}}},
		{Elements: []Element{{AI: "00"}, {AI: "8026"}, {AI: "37"}}},
		{Elements: []Element{{AI: "01"}, {AI: "21"}}},
		{Elements: []Element{{AI: "01"}, {AI: "3102"}}},
		{Elements: []Element{{AI: "00"}, {AI: "3302"}}},
	} {
		if err := valid.Validate(); err != nil {
			t.Errorf("valid association error = %v", err)
		}
	}
}

func TestParseWithAssociationValidation(t *testing.T) {
	input := "00012345678901234567" + "0204150000021126" + "3720"
	if _, err := ParseWithOptions(input, ParseOptions{ValidateAssociations: true}); err != nil {
		t.Fatalf("valid associations error = %v", err)
	}
	if _, err := ParseWithOptions("0204150000021126", ParseOptions{ValidateAssociations: true}); !errors.Is(err, ErrInvalidAssociation) {
		t.Errorf("missing association error = %v, want ErrInvalidAssociation", err)
	}
}

func TestParseRemainsLenientForAssociationMismatches(t *testing.T) {
	for _, input := range []string{
		"3720",
		"0204150000021126",
		"01041500000211260204150000021126",
		"10LOT42",
	} {
		if _, err := Parse(input); err != nil {
			t.Errorf("Parse(%q) error = %v, want lenient success", input, err)
		}
	}
}

func TestIdentifierConvenienceMethods(t *testing.T) {
	b, err := Parse("2531234567890123DOC\x1D800301234567890123SERIAL")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := b.GDTI(); got != "1234567890123DOC" {
		t.Errorf("GDTI() = %q, want %q", got, "1234567890123DOC")
	}
	if got := b.GRAI(); got != "01234567890123SERIAL" {
		t.Errorf("GRAI() = %q, want %q", got, "01234567890123SERIAL")
	}
}

func TestMissingFNC1DoesNotTreatEmbeddedTextAsMixedIdentifier(t *testing.T) {
	b, err := Parse("10LOT253ABCDEFGHIJKLM21SERIAL")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := b.Lot(); got != "LOT253ABCDEFGHIJKLM" {
		t.Errorf("Lot() = %q, want LOT253ABCDEFGHIJKLM", got)
	}
	if got := b.SerialNumber(); got != "SERIAL" {
		t.Errorf("SerialNumber() = %q, want SERIAL", got)
	}
}

func TestBarcodeReset(t *testing.T) {
	b, err := Parse("12250630")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Elements) != 1 {
		t.Fatalf("got %d elements, want 1", len(b.Elements))
	}
	if got := b.Warnings(); len(got) != 1 || got[0].Code != WarnDueDateAsExpiry {
		t.Fatalf("Warnings() before Reset() = %+v, want one %q warning", got, WarnDueDateAsExpiry)
	}
	origCap := cap(b.Elements)

	b.Reset()
	if got := b.Warnings(); len(got) != 0 {
		t.Errorf("Warnings() after Reset() = %+v, want empty", got)
	}

	if b.Raw != "" {
		t.Errorf("Raw = %q after Reset, want empty", b.Raw)
	}
	if len(b.Elements) != 0 {
		t.Errorf("len(Elements) = %d after Reset, want 0", len(b.Elements))
	}
	if cap(b.Elements) != origCap {
		t.Errorf("cap(Elements) = %d after Reset, want %d (retained)", cap(b.Elements), origCap)
	}
}

func TestParseInto(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"full barcode", "0104150000021126172506302112345ABC\x1D10LOT42X"},
		{"GTIN only", "0104150000021126"},
		{"lot only", "10ABC123"},
		{"bracket notation", "(01)04150000021126(17)250630(10)LOT42"},
		{"bare EAN-13", "7800038041425"},
		{"AIM prefix", "]C10104150000021126"},
		{"FNC1 prefix", "\x1D0104150000021126"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var got Barcode
			if err := ParseInto(tt.input, &got); err != nil {
				t.Fatalf("ParseInto() error = %v", err)
			}

			if got.Raw != want.Raw {
				t.Errorf("Raw = %q, want %q", got.Raw, want.Raw)
			}
			if len(got.Elements) != len(want.Elements) {
				t.Fatalf("len(Elements) = %d, want %d", len(got.Elements), len(want.Elements))
			}
			for i := range want.Elements {
				if got.Elements[i] != want.Elements[i] {
					t.Errorf("Elements[%d] = %+v, want %+v", i, got.Elements[i], want.Elements[i])
				}
			}
		})
	}
}

func TestParseIntoReuse(t *testing.T) {
	inputs := []string{
		"0104150000021126172506302112345ABC\x1D10LOT42X",
		"0104150000021126",
		"10BATCH42",
		"(01)04150000021126(17)250630",
		"7800038041425",
	}

	var b Barcode
	for _, input := range inputs {
		b.Reset()
		if err := ParseInto(input, &b); err != nil {
			t.Fatalf("ParseInto(%q) error: %v", input, err)
		}

		want, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", input, err)
		}

		if len(b.Elements) != len(want.Elements) {
			t.Fatalf("input %q: len(Elements) = %d, want %d", input, len(b.Elements), len(want.Elements))
		}
		for i := range want.Elements {
			if b.Elements[i] != want.Elements[i] {
				t.Errorf("input %q: Elements[%d] = %+v, want %+v", input, i, b.Elements[i], want.Elements[i])
			}
		}
	}
}

func TestParseIntoErrors(t *testing.T) {
	var b Barcode
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"empty", "", ErrEmptyInput},
		{"whitespace", "   ", ErrEmptyInput},
		{"unknown AI", "XX12345", ErrUnknownAI},
		{"truncated", "01041500", ErrTruncatedData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.Reset()
			err := ParseInto(tt.input, &b)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseInto() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	input := "0104150000021126172506302112345ABC\x1D10LOT42X"
	for i := 0; i < b.N; i++ {
		_, _ = Parse(input)
	}
}

func BenchmarkParseMinimal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Parse("0104150000021126")
	}
}

func BenchmarkParseInto(b *testing.B) {
	input := "0104150000021126172506302112345ABC\x1D10LOT42X"
	var bc Barcode
	for i := 0; i < b.N; i++ {
		bc.Reset()
		_ = ParseInto(input, &bc)
	}
}

func BenchmarkParseSustained(b *testing.B) {
	input := "0104150000021126172506302112345ABC\x1D10LOT42X"
	var bc Barcode
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			bc.Reset()
			_ = ParseInto(input, &bc)
		}
	}
}

func BenchmarkParseConcurrent(b *testing.B) {
	input := "0104150000021126172506302112345ABC\x1D10LOT42X"
	b.RunParallel(func(pb *testing.PB) {
		var bc Barcode
		for pb.Next() {
			bc.Reset()
			_ = ParseInto(input, &bc)
		}
	})
}
