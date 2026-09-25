package gs1

import (
	"errors"
	"testing"
)

func TestEncode(t *testing.T) {
	elements := []Element{
		{AI: "10", Value: "LOT42"},
		{AI: "01", Value: "04150000021126"},
		{AI: "17", Value: "250630"},
		{AI: "21", Value: "SERIAL"},
	}
	want := "\x1D01041500000211261725063010LOT42\x1D21SERIAL"
	got, err := Encode(elements)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got != want {
		t.Fatalf("Encode() = %q, want %q", got, want)
	}
	b, err := Parse(got)
	if err != nil {
		t.Fatalf("Parse(Encode()) error = %v", err)
	}
	if got := b.HRI(); got != "(01)04150000021126(17)250630(10)LOT42(21)SERIAL" {
		t.Errorf("HRI() = %q", got)
	}
	if got := b.String(); got != b.HRI() {
		t.Errorf("String() = %q, want HRI %q", got, b.HRI())
	}
}

func TestEncodeErrors(t *testing.T) {
	tests := []struct {
		name     string
		elements []Element
		wantErr  error
	}{
		{name: "empty", wantErr: ErrEmptyInput},
		{name: "unknown AI", elements: []Element{{AI: "999", Value: "x"}}, wantErr: ErrUnknownAI},
		{name: "wrong fixed length", elements: []Element{{AI: "17", Value: "2506"}}, wantErr: ErrInvalidData},
		{name: "non-numeric", elements: []Element{{AI: "01", Value: "0415000002112A"}}, wantErr: ErrInvalidData},
		{name: "FNC1 in value", elements: []Element{{AI: "10", Value: "LOT\x1D42"}}, wantErr: ErrInvalidData},
		{name: "control character", elements: []Element{{AI: "10", Value: "LOT\x001"}}, wantErr: ErrInvalidData},
		{name: "space-only value", elements: []Element{{AI: "21", Value: " "}}, wantErr: ErrInvalidData},
		{name: "trailing space", elements: []Element{{AI: "10", Value: "LOT "}}, wantErr: ErrInvalidData},
		{name: "parentheses", elements: []Element{{AI: "10", Value: "LOT(1)"}}, wantErr: ErrInvalidData},
		{name: "duplicate AI", elements: []Element{{AI: "10", Value: "A"}, {AI: "10", Value: "B"}}, wantErr: ErrInvalidData},
		{name: "empty value", elements: []Element{{AI: "10", Value: ""}}, wantErr: ErrInvalidData},
		{name: "invalid GTIN check digit", elements: []Element{{AI: "01", Value: "04150000021127"}}, wantErr: ErrInvalidCheckDigit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Encode(tt.elements)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Encode() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncodeLogisticLabel(t *testing.T) {
	elements := []Element{
		{AI: "00", Value: "106141411234567897"},
		{AI: "02", Value: "04150000021126"},
		{AI: "37", Value: "20"},
		{AI: "10", Value: "LOT42"},
		{AI: "414", Value: "0614141123452"},
	}
	encoded, err := Encode(elements)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(encoded)
	if err != nil {
		t.Fatalf("Parse(Encode()) error = %v", err)
	}
	if len(parsed.Elements) != len(elements) {
		t.Fatalf("Parse(Encode()) got %d elements, want %d", len(parsed.Elements), len(elements))
	}
	wantEncoded := "\x1D00106141411234567897020415000002112641406141411234523720\x1D10LOT42"
	if encoded != wantEncoded {
		t.Fatalf("Encode() = %q, want %q", encoded, wantEncoded)
	}
	wantHRI := "(00)106141411234567897(02)04150000021126(414)0614141123452(37)20(10)LOT42"
	if parsed.HRI() != wantHRI {
		t.Fatalf("Parse(Encode()) HRI = %q, want %q", parsed.HRI(), wantHRI)
	}
}

func FuzzEncodeRoundTrip(f *testing.F) {
	f.Add("LOT42", "SERIAL")
	f.Fuzz(func(t *testing.T, lot, serial string) {
		if len(lot) == 0 || len(lot) > 20 || len(serial) == 0 || len(serial) > 20 {
			t.Skip()
		}
		for _, value := range []string{lot, serial} {
			if validateEncodableValue(value, "10") != nil {
				t.Skip()
			}
		}
		encoded, err := Encode([]Element{{AI: "10", Value: lot}, {AI: "21", Value: serial}})
		if err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		parsed, err := Parse(encoded)
		if err != nil {
			t.Fatalf("Parse(Encode()) error = %v", err)
		}
		if parsed.Lot() != lot || parsed.SerialNumber() != serial {
			t.Fatalf("round trip changed values: lot=%q serial=%q", parsed.Lot(), parsed.SerialNumber())
		}
	})
}

func TestEncodeSeparatesFixedLengthNonPredefinedAI(t *testing.T) {
	got, err := Encode([]Element{
		{AI: "402", Value: "12345678901234567"},
		{AI: "10", Value: "LOT1"},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	want := "\x1D40212345678901234567\x1D10LOT1"
	if got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}
