package gs1

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestKeyValidators(t *testing.T) {
	tests := []struct {
		name     string
		validate func(string) error
		input    string
		wantErr  error
	}{
		// Vectors computed with the GS1 check digit calculator
		// (https://www.gs1.org/services/check-digit-calculator).
		{name: "SSCC", validate: ValidateSSCC, input: "106141411234567897"},
		{name: "SSCC extension 3", validate: ValidateSSCC, input: "378012340000000095"},
		{name: "SSCC all zeros", validate: ValidateSSCC, input: "000000000000000000"},
		{name: "SSCC bad check digit", validate: ValidateSSCC, input: "106141411234567891", wantErr: ErrInvalidCheckDigit},
		{name: "SSCC too short", validate: ValidateSSCC, input: "10614141123456789", wantErr: ErrInvalidData},
		{name: "SSCC too long", validate: ValidateSSCC, input: "1061414112345678970", wantErr: ErrInvalidData},
		{name: "SSCC letters", validate: ValidateSSCC, input: "10614141123456789A", wantErr: ErrInvalidData},
		{name: "SSCC empty", validate: ValidateSSCC, input: "", wantErr: ErrInvalidData},

		{name: "GLN", validate: ValidateGLN, input: "0614141000012"},
		{name: "GLN example 2", validate: ValidateGLN, input: "7801234000001"},
		{name: "GLN bad check digit", validate: ValidateGLN, input: "0614141000013", wantErr: ErrInvalidCheckDigit},
		{name: "GLN GTIN-14 length", validate: ValidateGLN, input: "04150000021126", wantErr: ErrInvalidData},
		{name: "GLN letters", validate: ValidateGLN, input: "061414100001A", wantErr: ErrInvalidData},

		{name: "GSIN", validate: ValidateGSIN, input: "06141410000000012"},
		{name: "GSIN example 2", validate: ValidateGSIN, input: "78012340000000001"},
		{name: "GSIN bad check digit", validate: ValidateGSIN, input: "06141410000000013", wantErr: ErrInvalidCheckDigit},
		{name: "GSIN SSCC length", validate: ValidateGSIN, input: "106141411234567897", wantErr: ErrInvalidData},

		{name: "GSRN", validate: ValidateGSRN, input: "061414112345678902"},
		{name: "GSRN bad check digit", validate: ValidateGSRN, input: "061414112345678903", wantErr: ErrInvalidCheckDigit},
		{name: "GSRN too short", validate: ValidateGSRN, input: "06141411234567890", wantErr: ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validate(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validate(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			if errors.Is(err, ErrInvalidCheckDigit) && errors.Is(err, ErrInvalidData) {
				t.Errorf("validate(%q) error = %v, wraps both sentinels", tt.input, err)
			}
		})
	}
}

func TestComputeCheckDigit(t *testing.T) {
	tests := []struct {
		name    string
		partial string
		want    byte
		wantErr error
	}{
		{name: "SSCC data", partial: "10614141123456789", want: '7'},
		{name: "GLN data", partial: "061414100001", want: '2'},
		{name: "GSIN data", partial: "0614141000000001", want: '2'},
		{name: "GSRN data", partial: "06141411234567890", want: '2'},
		{name: "GTIN-13 data", partial: "590123412345", want: '7'},
		{name: "single digit", partial: "7", want: '9'},
		{name: "thirty digits", partial: "123456789012345678901234567890", want: '5'},
		{name: "empty", partial: "", wantErr: ErrInvalidData},
		{name: "thirty-one digits", partial: "1234567890123456789012345678901", wantErr: ErrInvalidData},
		{name: "letters", partial: "1061414112345678A", wantErr: ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeCheckDigit(tt.partial)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ComputeCheckDigit(%q) error = %v, want %v", tt.partial, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("ComputeCheckDigit(%q) = %q, want %q", tt.partial, got, tt.want)
			}
			if err == nil {
				if verr := ValidateCheckDigit(tt.partial + string(got)); verr != nil {
					t.Errorf("ValidateCheckDigit(partial+check) error = %v", verr)
				}
			}
		})
	}
}

func TestKeyTypes(t *testing.T) {
	tests := []struct {
		input string
		want  []KeyType
	}{
		{input: "96385074", want: []KeyType{KeyGTIN}},
		{input: "036000291452", want: []KeyType{KeyGTIN}},
		{input: "0614141000012", want: []KeyType{KeyGTIN, KeyGLN}},
		{input: "04150000021126", want: []KeyType{KeyGTIN}},
		{input: "06141410000000012", want: []KeyType{KeyGSIN}},
		{input: "106141411234567897", want: []KeyType{KeySSCC, KeyGSRN}},
		{input: "", want: nil},
		{input: "1234567", want: nil},
		{input: "123456789", want: nil},
		{input: "1234567890123456789", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := KeyTypes(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("KeyTypes(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("KeyTypes(%q)[%d] = %v, want %v", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestKeyTypeString(t *testing.T) {
	tests := map[KeyType]string{
		KeyUnknown: "unknown", KeyGTIN: "GTIN", KeySSCC: "SSCC", KeyGLN: "GLN", KeyGSIN: "GSIN", KeyGSRN: "GSRN",
		KeyType(200): "unknown",
	}
	for k, want := range tests {
		if got := k.String(); got != want {
			t.Errorf("KeyType(%d).String() = %q, want %q", k, got, want)
		}
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTypes []KeyType
		wantErr   error
	}{
		{name: "GTIN-8", input: "96385074", wantTypes: []KeyType{KeyGTIN}},
		{name: "GTIN-12", input: "036000291452", wantTypes: []KeyType{KeyGTIN}},
		{name: "GTIN-13 or GLN", input: "0614141000012", wantTypes: []KeyType{KeyGTIN, KeyGLN}},
		{name: "GTIN-14", input: "04150000021126", wantTypes: []KeyType{KeyGTIN}},
		{name: "GSIN", input: "06141410000000012", wantTypes: []KeyType{KeyGSIN}},
		{name: "SSCC or GSRN", input: "106141411234567897", wantTypes: []KeyType{KeySSCC, KeyGSRN}},
		{name: "bad GTIN check digit", input: "04150000021127", wantErr: ErrInvalidCheckDigit},
		{name: "bad SSCC check digit", input: "106141411234567891", wantErr: ErrInvalidCheckDigit},
		{name: "bad GSIN check digit", input: "06141410000000013", wantErr: ErrInvalidCheckDigit},
		{name: "unsupported length", input: "123456789", wantErr: ErrInvalidData},
		{name: "empty", input: "", wantErr: ErrInvalidData},
		{name: "letters", input: "0614141A00012", wantErr: ErrInvalidData},
		{name: "whitespace", input: " 0614141000012", wantErr: ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			types, err := ValidateKey(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateKey(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			if err != nil && types != nil {
				t.Errorf("ValidateKey(%q) types = %v on error, want nil", tt.input, types)
			}
			if len(types) != len(tt.wantTypes) {
				t.Fatalf("ValidateKey(%q) types = %v, want %v", tt.input, types, tt.wantTypes)
			}
			for i := range types {
				if types[i] != tt.wantTypes[i] {
					t.Errorf("ValidateKey(%q) types[%d] = %v, want %v", tt.input, i, types[i], tt.wantTypes[i])
				}
			}
		})
	}
}

// TestParseCheckDigitsForKeyAIs covers every AI whose GS1 Syntax Dictionary
// entry carries the csum constraint, under lenient and check-digit parsing.
func TestParseCheckDigitsForKeyAIs(t *testing.T) {
	tests := []struct {
		name    string
		valid   string
		invalid string
	}{
		{name: "SSCC (00)", valid: "00106141411234567897", invalid: "00106141411234567891"},
		{name: "MTO GTIN (03)", valid: "0312345678901231", invalid: "0312345678901232"},
		{name: "GCN (255)", valid: "2551234567890128", invalid: "2551234567890129"},
		{name: "GCN (255) one-digit serial", valid: "25512345678901280", invalid: "25512345678901290"},
		{name: "GCN (255) non-zero one-digit serial", valid: "25512345678901281", invalid: "25512345678901291"},
		{name: "GCN (255) twelve-digit serial", valid: "2551234567890128123456789012", invalid: "2551234567890129123456789012"},
		{name: "ITIP (8006)", valid: "8006123456789012310102", invalid: "8006123456789012320102"},
		{name: "ITIP (8006) second piece", valid: "8006123456789012310202", invalid: "8006123456789012320202"},
		{name: "ITIP CONTENT (8026)", valid: "8026123456789012310102", invalid: "8026123456789012320102"},
		{name: "ITIP CONTENT (8026) second piece", valid: "8026123456789012310202", invalid: "8026123456789012320202"},
		{name: "GSIN (402)", valid: "40206141410000000012", invalid: "40206141410000000013"},
		{name: "GLN (410)", valid: "4100614141000012", invalid: "4100614141000013"},
		{name: "GLN (411)", valid: "4110614141000012", invalid: "4110614141000013"},
		{name: "GLN (412)", valid: "4120614141000012", invalid: "4120614141000013"},
		{name: "GLN (413)", valid: "4130614141000012", invalid: "4130614141000013"},
		{name: "GLN (414)", valid: "4140614141000012", invalid: "4140614141000013"},
		{name: "GLN (415)", valid: "4150614141000012", invalid: "4150614141000013"},
		{name: "GLN (416)", valid: "4160614141000012", invalid: "4160614141000013"},
		{name: "GLN (417)", valid: "4170614141000012", invalid: "4170614141000013"},
		{name: "GSRN provider (8017)", valid: "8017061414112345678902", invalid: "8017061414112345678903"},
		{name: "GSRN recipient (8018)", valid: "8018061414112345678902", invalid: "8018061414112345678903"},
		{name: "GDTI (253) key only", valid: "2530614141000012", invalid: "2530614141000013"},
		{name: "GDTI (253) with serial", valid: "2530614141000012ABC", invalid: "2530614141000013ABC"},
		{name: "GRAI (8003) with serial", valid: "800300614141000012XYZ", invalid: "800300614141000013XYZ"},
		{name: "key after fixed-length AI", valid: "0104150000021126" + "00106141411234567897", invalid: "0104150000021126" + "00106141411234567891"},
		{name: "key after variable-length AI", valid: "10LOT42\x1D4140614141000012", invalid: "10LOT42\x1D4140614141000013"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, input := range []string{tt.valid, tt.invalid} {
				if _, err := Parse(input); err != nil {
					t.Fatalf("lenient Parse(%q) error = %v", input, err)
				}
			}
			if _, err := ParseWithOptions(tt.valid, ParseOptions{ValidateCheckDigits: true}); err != nil {
				t.Errorf("ParseWithOptions(%q) error = %v, want nil", tt.valid, err)
			}
			_, err := ParseWithOptions(tt.invalid, ParseOptions{ValidateCheckDigits: true})
			if !errors.Is(err, ErrInvalidCheckDigit) {
				t.Fatalf("ParseWithOptions(%q) error = %v, want ErrInvalidCheckDigit", tt.invalid, err)
			}
		})
	}
}

func TestParseCheckDigitsReportsKeyAIAndPosition(t *testing.T) {
	_, err := ParseWithOptions("0104150000021126"+"00106141411234567891", ParseOptions{ValidateCheckDigits: true})
	if !errors.Is(err, ErrInvalidCheckDigit) {
		t.Fatalf("error = %v, want ErrInvalidCheckDigit", err)
	}
	for _, want := range []string{"position 16", "AI (00)", "expected '7', got '1'"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	// The first check-digit error wins, and association errors still take
	// precedence over it.
	_, err = ParseWithOptions("00106141411234567891"+"0204150000021127"+"3720", ParseOptions{ValidateCheckDigits: true})
	if !errors.Is(err, ErrInvalidCheckDigit) || !strings.Contains(err.Error(), "AI (00)") {
		t.Errorf("first error = %v, want AI (00) check digit", err)
	}
	_, err = ParseWithOptions("00106141411234567891"+"0204150000021126", ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true})
	if !errors.Is(err, ErrInvalidAssociation) {
		t.Errorf("error = %v, want ErrInvalidAssociation first", err)
	}
}

func TestValidateRejectsShortKeyPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		elements []Element
		wantErr  error
	}{
		{name: "short GDTI", elements: []Element{{AI: "253", Value: "123"}}, wantErr: ErrInvalidData},
		{name: "short GRAI", elements: []Element{{AI: "8003", Value: "0123"}}, wantErr: ErrInvalidData},
		{name: "short MTO GTIN", elements: []Element{{AI: "03", Value: "123"}}, wantErr: ErrInvalidData},
		{name: "short GCN key", elements: []Element{{AI: "255", Value: "123456789012"}}, wantErr: ErrInvalidData},
		{name: "short ITIP key", elements: []Element{{AI: "8006", Value: "1234567890128"}}, wantErr: ErrInvalidData},
		{
			name: "short ITIP CONTENT key",
			elements: []Element{
				{AI: "00", Value: "106141411234567897"},
				{AI: "8026", Value: "1234567890128"},
				{AI: "37", Value: "2"},
			},
			wantErr: ErrInvalidData,
		},
		{name: "short SSCC", elements: []Element{{AI: "00", Value: "123"}}, wantErr: ErrInvalidData},
		{name: "GDTI key only", elements: []Element{{AI: "253", Value: "0614141000012"}}},
		{name: "GRAI key only", elements: []Element{{AI: "8003", Value: "00614141000012"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (Barcode{Elements: tt.elements}).Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrInvalidCheckDigit) {
				t.Errorf("Validate() error = %v, want a length error, not a check-digit error", err)
			}
		})
	}
}

func TestEncodeValidatesKeyCheckDigits(t *testing.T) {
	tests := []struct {
		name    string
		element Element
		wantErr error
	}{
		{name: "valid SSCC", element: Element{AI: "00", Value: "106141411234567897"}},
		{name: "invalid SSCC", element: Element{AI: "00", Value: "106141411234567891"}, wantErr: ErrInvalidCheckDigit},
		{name: "valid GLN", element: Element{AI: "414", Value: "0614141000012"}},
		{name: "invalid GLN", element: Element{AI: "410", Value: "0614141000013"}, wantErr: ErrInvalidCheckDigit},
		{name: "valid GSIN", element: Element{AI: "402", Value: "06141410000000012"}},
		{name: "invalid GSIN", element: Element{AI: "402", Value: "06141410000000013"}, wantErr: ErrInvalidCheckDigit},
		{name: "valid GSRN", element: Element{AI: "8017", Value: "061414112345678902"}},
		{name: "invalid GSRN", element: Element{AI: "8018", Value: "061414112345678903"}, wantErr: ErrInvalidCheckDigit},
		{name: "valid GDTI", element: Element{AI: "253", Value: "0614141000012ABC"}},
		{name: "invalid GDTI", element: Element{AI: "253", Value: "0614141000013ABC"}, wantErr: ErrInvalidCheckDigit},
		{name: "valid GRAI", element: Element{AI: "8003", Value: "00614141000012XYZ"}},
		{name: "invalid GRAI", element: Element{AI: "8003", Value: "00614141000013XYZ"}, wantErr: ErrInvalidCheckDigit},
		{name: "GIAI has no check digit", element: Element{AI: "8004", Value: "0614141999"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Encode([]Element{tt.element})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Encode() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			parsed, err := ParseWithOptions(encoded, ParseOptions{ValidateCheckDigits: true})
			if err != nil {
				t.Fatalf("Parse(Encode()) with check digits error = %v", err)
			}
			if len(parsed.Elements) != 1 || parsed.Elements[0] != tt.element {
				t.Errorf("round trip = %+v, want %+v", parsed.Elements, tt.element)
			}
		})
	}
}

func TestAdditionalKeyValidationEntryPoints(t *testing.T) {
	tests := []struct {
		ai         string
		valid      string
		invalid    string
		companions []Element
	}{
		{ai: "03", valid: "12345678901231", invalid: "12345678901232"},
		{ai: "255", valid: "1234567890128123456789012", invalid: "1234567890129123456789012"},
		{ai: "8006", valid: "123456789012310202", invalid: "123456789012320202"},
		{
			ai: "8026", valid: "123456789012310202", invalid: "123456789012320202",
			companions: []Element{{AI: "00", Value: "106141411234567897"}, {AI: "37", Value: "2"}},
		},
	}
	options := []struct {
		name string
		opts ParseOptions
	}{
		{name: "default"},
		{name: "associations", opts: ParseOptions{ValidateAssociations: true}},
		{name: "check digits", opts: ParseOptions{ValidateCheckDigits: true}},
		{name: "both", opts: ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true}},
	}
	for _, tt := range tests {
		t.Run(tt.ai, func(t *testing.T) {
			var reused Barcode
			for _, step := range []struct {
				value   string
				wantErr error
			}{
				{value: tt.valid},
				{value: tt.invalid, wantErr: ErrInvalidCheckDigit},
				{value: tt.valid},
			} {
				elements := append([]Element{{AI: tt.ai, Value: step.value}}, tt.companions...)
				barcode := Barcode{Elements: elements}
				if err := barcode.ValidateAssociations(); err != nil {
					t.Fatalf("fixture associations error = %v", err)
				}
				if err := barcode.Validate(); !errors.Is(err, step.wantErr) {
					t.Errorf("Validate(%q) error = %v, want %v", step.value, err, step.wantErr)
				}
				if _, err := Encode(elements); !errors.Is(err, step.wantErr) {
					t.Errorf("Encode(%q) error = %v, want %v", step.value, err, step.wantErr)
				}
				input := barcode.HRI()
				for _, option := range options {
					var wantErr error
					if option.opts.ValidateCheckDigits {
						wantErr = step.wantErr
					}
					got, err := ParseWithOptions(input, option.opts)
					if !errors.Is(err, wantErr) {
						t.Fatalf("ParseWithOptions(%q, %s) error = %v, want %v", input, option.name, err, wantErr)
					}
					reused.Reset()
					intoErr := ParseIntoWithOptions(input, &reused, option.opts)
					if !errors.Is(intoErr, wantErr) {
						t.Fatalf("ParseIntoWithOptions(%q, %s) error = %v, want %v", input, option.name, intoErr, wantErr)
					}
					if wantErr == nil && (got.Raw != input || reused.Raw != input ||
						!slices.Equal(got.Elements, elements) || !slices.Equal(reused.Elements, elements)) {
						t.Errorf("%s parsing did not preserve elements %+v", option.name, elements)
					}
				}
			}
		})
	}
}

func TestAdditionalKeyCheckDigitErrorPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		opts    ParseOptions
		wantErr error
	}{
		{name: "syntax before check digit", input: "0312345678901232XX", wantErr: ErrUnknownAI},
		{name: "length before check digit", input: "03123456789012321712", wantErr: ErrTruncatedData},
		{name: "data type before check digit", input: "031234567890123230X", wantErr: ErrInvalidData},
		{
			name: "associations before check digit", input: "8026123456789012320102",
			opts: ParseOptions{ValidateAssociations: true}, wantErr: ErrInvalidAssociation,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.ValidateCheckDigits = true
			_, err := ParseWithOptions(tt.input, tt.opts)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseWithOptions() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
	_, err := ParseWithOptions("10LOT\x1D031234567890123200106141411234567891", ParseOptions{ValidateCheckDigits: true})
	if !errors.Is(err, ErrInvalidCheckDigit) {
		t.Fatalf("error = %v, want ErrInvalidCheckDigit", err)
	}
	for _, want := range []string{"position 6", "AI (03)", "expected '1', got '2'"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}
