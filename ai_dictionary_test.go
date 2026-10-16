package gs1

import (
	"errors"
	"testing"
)

func TestAITableCoversDictionaryRelease(t *testing.T) {
	if got := SyntaxDictionaryRelease(); got != "2026-01-27" {
		t.Errorf("SyntaxDictionaryRelease() = %q", got)
	}
	if len(aiTable) != 541 {
		t.Errorf("aiTable has %d AIs, want 541 from release 2026-01-27", len(aiTable))
	}
	// AIs the certification audit flagged as missing from the hand-written table.
	for _, code := range []string{"7003", "422", "8200", "715", "16", "20", "22", "235", "255", "3900", "4300", "7230", "8030", "8110"} {
		if _, ok := LookupAI(code); !ok {
			t.Errorf("LookupAI(%q) = false", code)
		}
	}
	// Keys are prefix-free, which lookupAIAt relies on.
	for code := range aiTable {
		for n := 2; n < len(code); n++ {
			if _, ok := aiTable[code[:n]]; ok {
				t.Errorf("AI %q has prefix %q in the table", code, code[:n])
			}
		}
	}
}

func TestLookupAIReportsDictionaryFields(t *testing.T) {
	tests := []struct {
		code       string
		name       string
		title      string
		format     string
		predefined bool
	}{
		{code: "01", name: "GTIN", title: "GTIN", format: "N14", predefined: true},
		{code: "10", name: "Batch/Lot", title: "BATCH/LOT", format: "X..20"},
		{code: "3102", name: "Net Weight kg", title: "NET WEIGHT (kg)", format: "N6", predefined: true},
		{code: "402", name: "GSIN", title: "GSIN", format: "N17"},
		{code: "421", name: "SHIP TO POST", title: "SHIP TO POST", format: "N3 X..9"},
		{code: "423", name: "COUNTRY - INITIAL PROCESS", title: "COUNTRY - INITIAL PROCESS", format: "N3 [N3] [N3] [N3] [N3]"},
		{code: "4330", name: "MAX TEMP F.", title: "MAX TEMP F.", format: "N6 [X1]"},
		{code: "7003", name: "EXPIRY TIME", title: "EXPIRY TIME", format: "N6 N4"},
		{code: "7040", name: "GS1 UIC with extension", title: "UIC+EXT", format: "N1 X1 X1 X1"},
		{code: "8003", name: "GRAI", title: "GRAI", format: "N1 N13 [X..16]"},
		{code: "8010", name: "CPID", title: "CPID", format: "Y..30"},
		{code: "8030", name: "DIGSIG", title: "DIGSIG", format: "Z..90"},
		{code: "8043", name: "PSIM", title: "PSIM", format: "N18 [N..2]"},
		{code: "8110", name: "Coupon Code (North America)", title: "", format: "X..70"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got, ok := LookupAI(tt.code)
			if !ok {
				t.Fatalf("LookupAI(%q) = false", tt.code)
			}
			if got.Name != tt.name || got.Title != tt.title || got.Format != tt.format || got.Predefined != tt.predefined {
				t.Errorf("LookupAI(%q) = %+v, want name %q title %q format %q predefined %t", tt.code, got, tt.name, tt.title, tt.format, tt.predefined)
			}
		})
	}
}

func TestParseDictionaryComponents(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Element
		wantErr error
	}{
		{name: "expiry time", input: "0104150000021126" + "70032506301230", want: []Element{{AI: "01", Value: "04150000021126"}, {AI: "7003", Value: "2506301230"}}},
		{name: "expiry time too short", input: "0104150000021126" + "7003250630", wantErr: ErrTruncatedData},
		{name: "origin country", input: "0104150000021126" + "422528", want: []Element{{AI: "01", Value: "04150000021126"}, {AI: "422", Value: "528"}}},
		{name: "ship to post numeric prefix", input: "421528ABC12", want: []Element{{AI: "421", Value: "528ABC12"}}},
		{name: "ship to post non-numeric prefix", input: "42152AABC", wantErr: ErrInvalidData},
		{name: "ship to post too long", input: "421528ABCDEFGHIJK", wantErr: ErrInvalidData},
		{name: "optional repeated countries", input: "0104150000021126" + "423528076040", want: []Element{{AI: "01", Value: "04150000021126"}, {AI: "423", Value: "528076040"}}},
		{name: "optional countries partial", input: "42352807", wantErr: ErrInvalidData},
		{name: "temperature with sign", input: "4330012345-", want: []Element{{AI: "4330", Value: "012345-"}}},
		{name: "temperature without sign", input: "4330012345", want: []Element{{AI: "4330", Value: "012345"}}},
		{name: "temperature trailing data", input: "4330012345-X", wantErr: ErrInvalidData},
		{name: "CPID character set 39", input: "8010ABC-12/3#", want: []Element{{AI: "8010", Value: "ABC-12/3#"}}},
		{name: "CPID rejects lowercase", input: "8010abc", wantErr: ErrInvalidData},
		{name: "digital signature base64url", input: "8030AbC_-123", want: []Element{{AI: "8030", Value: "AbC_-123"}}},
		{name: "digital signature rejects padding", input: "8030AbC=", wantErr: ErrInvalidData},
		{name: "GRAI leading zero", input: "800300614141000012XYZ", want: []Element{{AI: "8003", Value: "00614141000012XYZ"}}},
		{name: "GRAI non-zero first digit", input: "800310614141000012XYZ", wantErr: ErrInvalidData},
		{name: "UIC with extension", input: "70401ABC", want: []Element{{AI: "7040", Value: "1ABC"}}},
		{name: "UIC non-numeric first", input: "7040AABC", wantErr: ErrInvalidData},
		{name: "PSIM with optional suffix", input: "804312345678901234567890", want: []Element{{AI: "8043", Value: "12345678901234567890"}}},
		{name: "PSIM without suffix", input: "8043123456789012345678", want: []Element{{AI: "8043", Value: "123456789012345678"}}},
		{name: "product URL", input: "0104150000021126" + "8200https://example.com/p", want: []Element{{AI: "01", Value: "04150000021126"}, {AI: "8200", Value: "https://example.com/p"}}},
		{name: "US NHRN", input: "0104150000021126" + "7151234567890", want: []Element{{AI: "01", Value: "04150000021126"}, {AI: "715", Value: "1234567890"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := Parse(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Parse(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(b.Elements) != len(tt.want) {
				t.Fatalf("Parse(%q) = %+v, want %+v", tt.input, b.Elements, tt.want)
			}
			for i := range tt.want {
				if b.Elements[i] != tt.want[i] {
					t.Errorf("element[%d] = %+v, want %+v", i, b.Elements[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidateAssociationsFromDictionary(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "expiry time with GTIN", input: "0104150000021126" + "70032506301230"},
		{name: "expiry time without item", input: "70032506301230" + "10LOT", wantErr: "AI (7003) requires AI (01) or AI (02)"},
		{name: "lot alone", input: "10LOT1", wantErr: "AI (10) requires AI (01), AI (02), AI (03), AI (8006), or AI (8026)"},
		{name: "origin excludes full process", input: "0104150000021126" + "422528" + "426528", wantErr: "AI (422) excludes AI (426)"},
		{name: "ship to post variants exclude", input: "00106141411234567897" + "42012345\x1D421528ABC", wantErr: "AI (420) excludes AI (421)"},
		{name: "amount with currency excludes bare amount", input: "0104150000021126" + "3920100\x1D3932978100", wantErr: "AI (3920) excludes AI (3932)"},
		{name: "IMEI needs GTIN with serial", input: "0104150000021126" + "8040123456789012345", wantErr: "AI (8040) requires AI (01) with AI (21)"},
		{name: "IMEI with GTIN and serial", input: "0104150000021126" + "8040123456789012345" + "21SER"},
		{name: "ship to address needs SSCC", input: "4300ACME", wantErr: "AI (4300) requires AI (00)"},
		{name: "logistic label", input: "00106141411234567897" + "0204150000021126" + "3712\x1D10LOT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseWithOptions(tt.input, ParseOptions{ValidateAssociations: true})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidAssociation) || err.Error() != "gs1: invalid AI association: "+tt.wantErr {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestMatchAIPattern(t *testing.T) {
	tests := []struct {
		pattern, ai string
		want        bool
	}{
		{"310n", "3102", true}, {"310n", "3110", false}, {"31nn", "3195", true}, {"01", "01", true}, {"01", "010", false}, {"n1", "01", true},
	}
	for _, tt := range tests {
		if got := matchAIPattern(tt.pattern, tt.ai); got != tt.want {
			t.Errorf("matchAIPattern(%q, %q) = %t, want %t", tt.pattern, tt.ai, got, tt.want)
		}
	}
}
