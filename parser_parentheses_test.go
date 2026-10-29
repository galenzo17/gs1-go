package gs1

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseRawParentheses(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Element
		wantErr error
	}{
		{
			name:  "AI-like parentheses in lot",
			input: "10LOT(90)X",
			want:  []Element{{AI: "10", Value: "LOT(90)X"}},
		},
		{name: "opening parenthesis", input: "10LOT(X", want: []Element{{AI: "10", Value: "LOT(X"}}},
		{name: "closing parenthesis", input: "10LOT)X", want: []Element{{AI: "10", Value: "LOT)X"}}},
		{name: "value starts with parenthesis", input: "10(90)X", want: []Element{{AI: "10", Value: "(90)X"}}},
		{name: "multiple pairs", input: "10A(B)C(D)", want: []Element{{AI: "10", Value: "A(B)C(D)"}}},
		{name: "nested parentheses", input: "21SER((X))", want: []Element{{AI: "21", Value: "SER((X))"}}},
		{
			name:  "after fixed-length field",
			input: "010415000002112610LOT(90)X",
			want:  []Element{{AI: "01", Value: "04150000021126"}, {AI: "10", Value: "LOT(90)X"}},
		},
		{
			name:  "FNC1 between fields",
			input: "10LOT(90)X\x1D21S(1)",
			want:  []Element{{AI: "10", Value: "LOT(90)X"}, {AI: "21", Value: "S(1)"}},
		},
		{
			name:  "parenthesis only in later field",
			input: "10LOT\x1D21S(1)",
			want:  []Element{{AI: "10", Value: "LOT"}, {AI: "21", Value: "S(1)"}},
		},
		{name: "maximum length", input: "10" + strings.Repeat("A", 18) + "()", want: []Element{{AI: "10", Value: strings.Repeat("A", 18) + "()"}}},
		{name: "scanner tolerance for X data", input: "10LOT(\x01)", want: []Element{{AI: "10", Value: "LOT(\x01)"}}},
		{name: "numeric data rejects parentheses", input: "30(1)", wantErr: ErrInvalidData},
		{name: "over maximum length", input: "10" + strings.Repeat("A", 19) + "()", wantErr: ErrInvalidData},
		{name: "raw does not switch to bracket notation", input: "0104150000021126(10)LOT", wantErr: ErrUnknownAI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Parse(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Raw != tt.input {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.input)
			}
			if !slices.Equal(got.Elements, tt.want) {
				t.Fatalf("Elements = %+v, want %+v", got.Elements, tt.want)
			}
		})
	}
}

func TestParseParenthesesPrefixes(t *testing.T) {
	prefixes := []struct {
		name   string
		prefix string
		suffix string
		sym    Symbology
	}{
		{name: "none"},
		{name: "FNC1", prefix: "\x1D"},
		{name: "CRLF", prefix: "\r\n"},
		{name: "AIM", prefix: "]d2", sym: SymDataMatrix},
		{name: "AIM then FNC1", prefix: "]d2\x1D", sym: SymDataMatrix},
		{name: "FNC1 then AIM", prefix: "\x1D]d2", sym: SymDataMatrix},
		{name: "FNC1 then AIM then FNC1", prefix: "\x1D]d2\x1D", sym: SymDataMatrix},
		{name: "scanner artifacts", prefix: "\xEF\xBB\xBF \t\x00]d2\x1D\x1D", suffix: "\r\n\x00", sym: SymDataMatrix},
	}
	formats := []struct {
		name string
		data string
		want []Element
	}{
		{name: "raw", data: "10LOT(90)X", want: []Element{{AI: "10", Value: "LOT(90)X"}}},
		{name: "bracket notation", data: "(10)LOT(90)X", want: []Element{{AI: "10", Value: "LOT"}, {AI: "90", Value: "X"}}},
	}
	for _, prefix := range prefixes {
		for _, format := range formats {
			t.Run(prefix.name+"/"+format.name, func(t *testing.T) {
				input := prefix.prefix + format.data + prefix.suffix
				got, err := Parse(input)
				if err != nil {
					t.Fatalf("Parse(%q) error = %v", input, err)
				}
				if got.Raw != input || got.Symbology != prefix.sym || !slices.Equal(got.Elements, format.want) {
					t.Errorf("Parse(%q) = %+v, want raw input, symbology %v and elements %+v", input, got, prefix.sym, format.want)
				}
			})
		}
	}
}

func TestParseBracketNotationCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Element
		wantErr error
	}{
		{name: "unclosed bracket remains tolerated", input: "(10LOT", want: []Element{{AI: "10", Value: "LOT"}}},
		{name: "stray close remains data", input: "(10)LOT)", want: []Element{{AI: "10", Value: "LOT)"}}},
		{name: "AI lookup remains unchanged", input: "(999)1", want: []Element{{AI: "99", Value: "91"}}},
		{name: "unknown AI", input: "(88)1", wantErr: ErrUnknownAI},
		{name: "truncated data", input: "(01)123", wantErr: ErrTruncatedData},
		{name: "empty value", input: "(10)", wantErr: ErrInvalidData},
		{name: "empty following value", input: "(10)LOT(17)", wantErr: ErrTruncatedData},
		{name: "repeated AIM remains unsupported", input: "]d2]C1(10)LOT", wantErr: ErrUnknownAI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Parse(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr == nil && !slices.Equal(got.Elements, tt.want) {
				t.Errorf("Elements = %+v, want %+v", got.Elements, tt.want)
			}
		})
	}
}

func TestParseParenthesesWithOptions(t *testing.T) {
	input := "]d2010415000002112610LOT(90)X\x1D21S(1)"
	want := []Element{{AI: "01", Value: "04150000021126"}, {AI: "10", Value: "LOT(90)X"}, {AI: "21", Value: "S(1)"}}
	options := []ParseOptions{{}, {ValidateAssociations: true, ValidateCheckDigits: true}}
	var reused Barcode
	for _, opts := range options {
		got, err := ParseWithOptions(input, opts)
		if err != nil {
			t.Fatalf("ParseWithOptions(%+v) error = %v", opts, err)
		}
		if got.Raw != input || got.Symbology != SymDataMatrix || !slices.Equal(got.Elements, want) {
			t.Errorf("ParseWithOptions(%+v) = %+v, want %+v", opts, got, want)
		}
		reused.Reset()
		if err := ParseIntoWithOptions(input, &reused, opts); err != nil {
			t.Fatalf("ParseIntoWithOptions(%+v) error = %v", opts, err)
		}
		if reused.Raw != input || reused.Symbology != SymDataMatrix || !slices.Equal(reused.Elements, want) {
			t.Errorf("ParseIntoWithOptions(%+v) = %+v, want %+v", opts, reused, want)
		}
	}
}

func BenchmarkParseIntoRawParentheses(b *testing.B) {
	input := "010415000002112610LOT(90)X\x1D21S(1)"
	var barcode Barcode
	for i := 0; i < b.N; i++ {
		barcode.Reset()
		if err := ParseInto(input, &barcode); err != nil {
			b.Fatal(err)
		}
	}
}
