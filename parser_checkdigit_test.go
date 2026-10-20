package gs1

import (
	"errors"
	"strings"
	"testing"
)

func TestBarcodeValidateRejectsInvalidGTINCheckDigit(t *testing.T) {
	b, err := Parse("0104150000021127")
	if err != nil {
		t.Fatalf("lenient Parse() error = %v, want nil", err)
	}
	if err := ValidateGTIN(b.GTIN()); !errors.Is(err, ErrInvalidCheckDigit) {
		t.Fatalf("fixture ValidateGTIN() error = %v, want ErrInvalidCheckDigit", err)
	}
	if err := b.ValidateAssociations(); err != nil {
		t.Fatalf("fixture ValidateAssociations() error = %v, want nil", err)
	}
	if err := b.Validate(); !errors.Is(err, ErrInvalidCheckDigit) {
		t.Errorf("Validate() error = %v, want ErrInvalidCheckDigit", err)
	}
}

func TestParseIndependentValidationOptions(t *testing.T) {
	options := []struct {
		name string
		opts ParseOptions
	}{
		{name: "lenient"},
		{name: "associations", opts: ParseOptions{ValidateAssociations: true}},
		{name: "check digits", opts: ParseOptions{ValidateCheckDigits: true}},
		{name: "both", opts: ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true}},
	}
	inputs := []struct {
		name           string
		input          string
		badAssociation bool
		badCheckDigit  bool
	}{
		{name: "valid GTIN", input: "0104150000021126"},
		{name: "valid content GTIN", input: "0001234567890123456002041500000211263720"},
		{name: "checksum only", input: "0104150000021127", badCheckDigit: true},
		{name: "content GTIN checksum only", input: "0001234567890123456002041500000211273720", badCheckDigit: true},
		{name: "association only", input: "0204150000021126", badAssociation: true},
		{name: "both errors", input: "0204150000021127", badAssociation: true, badCheckDigit: true},
	}
	for _, option := range options {
		for _, input := range inputs {
			t.Run(option.name+"/"+input.name, func(t *testing.T) {
				var wantErr error
				if option.opts.ValidateAssociations && input.badAssociation {
					wantErr = ErrInvalidAssociation
				} else if option.opts.ValidateCheckDigits && input.badCheckDigit {
					wantErr = ErrInvalidCheckDigit
				}
				_, err := ParseWithOptions(input.input, option.opts)
				if !errors.Is(err, wantErr) {
					t.Errorf("ParseWithOptions() error = %v, want %v", err, wantErr)
				}
			})
		}
	}
}

func TestParseCheckDigitCarriers(t *testing.T) {
	tests := []struct {
		name        string
		valid       string
		invalid     string
		wantGTIN    string
		assumeGTIN8 bool
	}{
		{name: "bare GTIN-12", valid: "036000291452", invalid: "036000291453", wantGTIN: "00036000291452"},
		{name: "bare GTIN-13", valid: "5901234123457", invalid: "5901234123458", wantGTIN: "05901234123457"},
		{name: "bare GTIN-14", valid: "54150000021121", invalid: "54150000021122", wantGTIN: "54150000021121"},
		{name: "bare GTIN-8 opt-in", valid: "96385074", invalid: "96385075", wantGTIN: "00000096385074", assumeGTIN8: true},
		{name: "EAN-8", valid: "]E496385074", invalid: "]E496385075", wantGTIN: "00000096385074"},
		{name: "UPC-A", valid: "]E0036000291452", invalid: "]E0036000291453", wantGTIN: "00036000291452"},
		{name: "EAN-13", valid: "]E15901234123457", invalid: "]E15901234123458", wantGTIN: "05901234123457"},
		{name: "EAN/UPC GTIN-14", valid: "]E204150000021126", invalid: "]E204150000021127", wantGTIN: "04150000021126"},
		{name: "UPC-E", valid: "]E000123457", invalid: "]E000123458", wantGTIN: "00001234000057"},
		{name: "ITF-14", valid: "]I012345678901231", invalid: "]I012345678901232", wantGTIN: "12345678901231"},
		{name: "ITF-14 variant 1", valid: "]I112345678901231", invalid: "]I112345678901232", wantGTIN: "12345678901231"},
		{name: "ITF-14 variant 2", valid: "]I212345678901231", invalid: "]I212345678901232", wantGTIN: "12345678901231"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, checkDigits := range []bool{false, true} {
				opts := ParseOptions{AssumeBareGTIN8: tt.assumeGTIN8, ValidateAssociations: true, ValidateCheckDigits: checkDigits}
				b, err := ParseWithOptions(tt.valid, opts)
				if err != nil || b.GTIN() != tt.wantGTIN {
					t.Fatalf("valid carrier, check digits %t: GTIN = %q, error = %v", checkDigits, b.GTIN(), err)
				}
				b, err = ParseWithOptions(tt.invalid, opts)
				var wantErr error
				if checkDigits {
					wantErr = ErrInvalidCheckDigit
				}
				if !errors.Is(err, wantErr) {
					t.Errorf("invalid carrier, check digits %t: error = %v, want %v", checkDigits, err, wantErr)
				}
				if !checkDigits && tt.name == "UPC-E" && b.GTIN() != tt.wantGTIN {
					t.Errorf("lenient UPC-E GTIN = %q, want unchanged expansion %q", b.GTIN(), tt.wantGTIN)
				}
			}
		})
	}
}

func TestParseCheckDigitsPreservesCarrierInterpretation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Element
	}{
		{name: "valid bare GTIN with AI prefix", input: "123456789012", want: []Element{{AI: "01", Value: "00123456789012"}}},
		{name: "invalid bare candidate uses AIs", input: "123456789013", want: []Element{{AI: "12", Value: "345678"}, {AI: "90", Value: "13"}}},
		{name: "eight digits use AI by default", input: "24012345", want: []Element{{AI: "240", Value: "12345"}}},
		{name: "stripped ITF-14 digit is synthesized", input: "]I20123456789012", want: []Element{{AI: "01", Value: "01234567890128"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, checkDigits := range []bool{false, true} {
				b, err := ParseWithOptions(tt.input, ParseOptions{ValidateCheckDigits: checkDigits})
				if err != nil {
					t.Fatalf("check digits %t: error = %v, want nil", checkDigits, err)
				}
				if len(b.Elements) != len(tt.want) {
					t.Fatalf("Elements = %+v, want %+v", b.Elements, tt.want)
				}
				for i := range tt.want {
					if b.Elements[i] != tt.want[i] {
						t.Errorf("Elements[%d] = %+v, want %+v", i, b.Elements[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestParseCheckDigitErrorContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantAI   string
		position string
	}{
		{name: "AI 01", input: "0104150000021127", wantAI: "AI (01)", position: "position 0"},
		{name: "AI 02", input: "0204150000021127", wantAI: "AI (02)", position: "position 0"},
		{name: "later AI", input: "172506300104150000021127", wantAI: "AI (01)", position: "position 8"},
		{name: "brackets", input: "(17)250630(01)04150000021127", wantAI: "AI (01)", position: "position 9"},
		{name: "AIM and scanner noise", input: "\xEF\xBB\xBF ]d2(17)250630(01)04150000021127\r\n", wantAI: "AI (01)", position: "position 12"},
		{name: "carrier payload", input: "]E15901234123458", wantAI: "AI (01)", position: "position 3"},
		{name: "UPC-E payload after FNC1", input: "\x1D]E000123458", wantAI: "AI (01)", position: "position 4"},
		{name: "bare GTIN after scanner noise", input: "\xEF\xBB\xBF 5901234123458\r\n", wantAI: "AI (01)", position: "position 0"},
		{name: "first bad digit wins", input: "01041500000211270204150000021128", wantAI: "AI (01)", position: "position 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseWithOptions(tt.input, ParseOptions{ValidateCheckDigits: true})
			if !errors.Is(err, ErrInvalidCheckDigit) {
				t.Fatalf("error = %v, want ErrInvalidCheckDigit", err)
			}
			if !strings.Contains(err.Error(), tt.wantAI) || !strings.Contains(err.Error(), tt.position) {
				t.Errorf("error = %q, want %q and normalized %q", err, tt.wantAI, tt.position)
			}
		})
	}
}

func TestParseStructurePrecedesSemanticValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "unknown AI after bad checksum", input: "0104150000021127XX", wantErr: ErrUnknownAI},
		{name: "truncated field after both errors", input: "020415000002112717", wantErr: ErrTruncatedData},
		{name: "non-numeric field after bad checksum", input: "01041500000211271725AB30", wantErr: ErrInvalidData},
		{name: "short GTIN field", input: "01041500", wantErr: ErrTruncatedData},
		{name: "non-numeric GTIN", input: "010415000002112A", wantErr: ErrInvalidData},
		{name: "short carrier", input: "]E49638507", wantErr: ErrInvalidData},
		{name: "non-numeric carrier", input: "]E49638507A", wantErr: ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseWithOptions(tt.input, ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestBarcodeValidateUsesCurrentElements(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		elements []Element
		wantErr  error
	}{
		{name: "valid elements override bad Raw", raw: "0104150000021127", elements: []Element{{AI: "01", Value: "04150000021126"}}},
		{name: "bad elements override valid Raw", raw: "0104150000021126", elements: []Element{{AI: "01", Value: "04150000021127"}}, wantErr: ErrInvalidCheckDigit},
		{name: "content GTIN", elements: []Element{{AI: "00", Value: "012345678901234560"}, {AI: "02", Value: "04150000021127"}, {AI: "37"}}, wantErr: ErrInvalidCheckDigit},
		{name: "association priority", elements: []Element{{AI: "02", Value: "04150000021127"}}, wantErr: ErrInvalidAssociation},
		{name: "repeated GTIN with same bad value", elements: []Element{{AI: "01", Value: "04150000021127"}, {AI: "01", Value: "04150000021127"}}, wantErr: ErrInvalidCheckDigit},
		{name: "repeated GTIN with different values", elements: []Element{{AI: "01", Value: "04150000021126"}, {AI: "01", Value: "04150000021127"}}, wantErr: ErrInvalidAssociation},
		{name: "malformed element", elements: []Element{{AI: "01", Value: "123"}}, wantErr: ErrInvalidData},
		{name: "valid SSCC", raw: "malformed", elements: []Element{{AI: "00", Value: "012345678901234560"}}},
		{name: "invalid SSCC", elements: []Element{{AI: "00", Value: "012345678901234567"}}, wantErr: ErrInvalidCheckDigit},
		{name: "invalid GLN after valid SSCC", elements: []Element{{AI: "00", Value: "012345678901234560"}, {AI: "414", Value: "0614141000013"}}, wantErr: ErrInvalidCheckDigit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (Barcode{Raw: tt.raw, Elements: tt.elements}).Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrInvalidCheckDigit) && (!strings.Contains(err.Error(), "AI (") || strings.Contains(err.Error(), "position")) {
				t.Errorf("post-parse error = %q, want AI context without input position", err)
			}
		})
	}
	// Lenient conversion has already replaced the received UPC-E check digit.
	b, err := Parse("]E000123458")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Validate(); err != nil {
		t.Errorf("Validate() of normalized UPC-E error = %v, want nil", err)
	}
}

func TestParseIntoCheckDigitsReuse(t *testing.T) {
	steps := []struct {
		input    string
		wantErr  error
		wantSym  Symbology
		warnings int
	}{
		{input: "]d2010415000002112617250630", wantSym: SymDataMatrix},
		{input: "0104150000021127", wantErr: ErrInvalidCheckDigit, wantSym: SymUnknown},
		{input: "]I012345678901231", wantSym: SymITF14},
	}
	b := Barcode{Elements: make([]Element, 0, 8)}
	for _, step := range steps {
		b.Reset()
		err := ParseIntoWithOptions(step.input, &b, ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true})
		if !errors.Is(err, step.wantErr) {
			t.Fatalf("ParseIntoWithOptions(%q) error = %v, want %v", step.input, err, step.wantErr)
		}
		if b.Raw != step.input || b.Symbology != step.wantSym || len(b.Warnings()) != step.warnings {
			t.Errorf("reused Barcode = %+v, warnings = %+v", b, b.Warnings())
		}
		if err == nil {
			if err := b.Validate(); err != nil {
				t.Errorf("reused Barcode.Validate() error = %v", err)
			}
		}
	}
}
