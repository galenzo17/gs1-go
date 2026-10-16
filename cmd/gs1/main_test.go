package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	gs1 "github.com/galenzo17/gs1-go"
)

func exec(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestParseText(t *testing.T) {
	code, out, _ := exec(t, "", "parse", "0104150000021126172506301012345")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"(01) 04150000021126", "GTIN", "(17) 250630", "(10) 12345", "Batch/Lot"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestParseWeightText(t *testing.T) {
	code, out, _ := exec(t, "", "parse", "3102001250")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "12.50 kg") {
		t.Errorf("output = %q, want formatted weight", out)
	}
}

func TestParseWeightJSON(t *testing.T) {
	code, out, _ := exec(t, "", "parse", "-json", "3405001250")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got parseOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(got.Elements) != 1 || got.Elements[0].Measure == nil {
		t.Fatalf("elements = %+v, want one measured element", got.Elements)
	}
	measure := got.Elements[0].Measure
	if measure.Raw != "001250" || measure.Value != 0.0125 || measure.Unit != "lb" {
		t.Errorf("measure = %+v, want raw 001250, value 0.0125, unit lb", measure)
	}
}

func TestParseJSONWithISODates(t *testing.T) {
	code, out, _ := exec(t, "", "parse", "-json", "-iso", "010415000002112617250200020415000002112613250601")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got parseOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(got.Elements) != 4 {
		t.Fatalf("elements = %d, want 4", len(got.Elements))
	}
	if got.Elements[1].Date != "2025-02-28" {
		t.Errorf("date = %q, want 2025-02-28", got.Elements[1].Date)
	}
	if got.PackagingDate != "2025-06-01" {
		t.Errorf("packagingDate = %q, want 2025-06-01", got.PackagingDate)
	}
}

func TestParseJSONIncludesTypedAccessors(t *testing.T) {
	input := "0104150000021126" + "0204150000021126" + "13250601" + "3742\x1D" + "40212345678901234567" + "4141234567890123"
	code, out, _ := exec(t, "", "parse", "-json", input)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}

	var got parseOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if got.ContentGTIN != "04150000021126" || got.CountOfTradeItems != "42" || got.GLN != "1234567890123" || got.GSIN != "12345678901234567" || got.PackagingDate != "250601" {
		t.Errorf("typed accessors = %+v", got)
	}
}

func TestParseDueDateWarning(t *testing.T) {
	code, out, errOut := exec(t, "", "parse", "-json", "12250630")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got parseOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Code != gs1.WarnDueDateAsExpiry {
		t.Errorf("warnings = %+v", got.Warnings)
	}
	if !strings.Contains(errOut, "AI (12) is a due date") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestParseInvalid(t *testing.T) {
	code, _, errOut := exec(t, "", "parse", "8812345")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut, "unknown application identifier") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestParseStrictAssociations(t *testing.T) {
	code, _, errOut := exec(t, "", "parse", "-strict", "0204150000021126")
	if code != 1 || !strings.Contains(errOut, "requires AI (37)") {
		t.Errorf("strict association exit %d stderr %q", code, errOut)
	}
	if code, _, _ := exec(t, "", "parse", "-strict", "0001234567890123456702041500000211263720"); code != 0 {
		t.Errorf("valid strict association exit %d, want 0", code)
	}
}

func TestParseIndependentValidationFlags(t *testing.T) {
	options := []struct {
		name        string
		flags       []string
		strict      bool
		checkDigits bool
	}{
		{name: "lenient"},
		{name: "strict", flags: []string{"-strict"}, strict: true},
		{name: "check digits", flags: []string{"-check-digits"}, checkDigits: true},
		{name: "both", flags: []string{"-strict", "-check-digits"}, strict: true, checkDigits: true},
	}
	inputs := []struct {
		name           string
		input          string
		badAssociation bool
		badCheckDigit  bool
	}{
		{name: "valid", input: "0104150000021126"},
		{name: "checksum only", input: "0104150000021127", badCheckDigit: true},
		{name: "content GTIN checksum only", input: "0001234567890123456702041500000211273720", badCheckDigit: true},
		{name: "association only", input: "0204150000021126", badAssociation: true},
		{name: "both errors", input: "0204150000021127", badAssociation: true, badCheckDigit: true},
	}
	for _, option := range options {
		for _, input := range inputs {
			t.Run(option.name+"/"+input.name, func(t *testing.T) {
				args := append([]string{"parse"}, option.flags...)
				args = append(args, input.input)
				code, out, errOut := exec(t, "", args...)
				wantError := ""
				if option.strict && input.badAssociation {
					wantError = "invalid AI association"
				} else if option.checkDigits && input.badCheckDigit {
					wantError = "invalid check digit"
				}
				if wantError == "" {
					if code != 0 || out == "" || errOut != "" {
						t.Errorf("exit %d, stdout %q, stderr %q, want parsed output", code, out, errOut)
					}
				} else if code != 1 || out != "" || !strings.Contains(errOut, wantError) {
					t.Errorf("exit %d, stdout %q, stderr %q, want %q", code, out, errOut, wantError)
				}
			})
		}
	}
}

func TestParseCheckDigitStreamJSONContinues(t *testing.T) {
	inputs := []string{"0104150000021126", "0104150000021127", "]E496385074"}
	code, out, errOut := exec(t, strings.Join(inputs, "\n")+"\n", "parse", "-json", "-strict", "-check-digits")
	if code != 1 || errOut != "" {
		t.Errorf("exit %d, stderr %q, want exit 1 and inline JSON errors", code, errOut)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != len(inputs) {
		t.Fatalf("JSON records = %d, want %d: %s", len(lines), len(inputs), out)
	}
	for i, line := range lines {
		var got parseOutput
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatalf("record %d: invalid JSON: %v", i, err)
		}
		if got.Raw != inputs[i] {
			t.Errorf("record %d: Raw = %q, want %q", i, got.Raw, inputs[i])
		}
		if i == 1 {
			if !strings.Contains(got.Error, "invalid check digit") || len(got.Elements) != 0 {
				t.Errorf("error record = %+v, want checksum error without elements", got)
			}
		} else if got.Error != "" || len(got.Elements) != 1 || got.Elements[0].AI != "01" {
			t.Errorf("record %d = %+v, want one successful GTIN", i, got)
		}
	}
}

func TestParseValidateRegulator(t *testing.T) {
	full := "0104150000021126172506302112345ABC\x1D10LOT42X"
	if code, _, _ := exec(t, "", "parse", "-validate", "anvisa", full); code != 0 {
		t.Errorf("ANVISA pass: exit %d", code)
	}
	code, _, errOut := exec(t, "", "parse", "-validate", "anvisa", "01041500000211261725063010LOT42X")
	if code != 1 || !strings.Contains(errOut, "requires AI (21)") {
		t.Errorf("ANVISA fail: exit %d stderr %q", code, errOut)
	}
	if code, _, _ := exec(t, "", "parse", "-validate", "nope", full); code != 2 {
		t.Errorf("unknown regulator: exit %d, want 2", code)
	}
}

func TestParseStream(t *testing.T) {
	stdin := "0104150000021126172506301012345\n\n8812345\n01041500000211261725063010LOT2\n"
	code, out, errOut := exec(t, stdin, "parse")
	if code != 1 {
		t.Errorf("exit %d, want 1 (one bad line)", code)
	}
	if strings.Count(out, "(01) 04150000021126") != 2 {
		t.Errorf("expected two parsed barcodes:\n%s", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("expected separator between records")
	}
	if !strings.Contains(errOut, "unknown application identifier") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestParseStreamJSONReportsErrorsInline(t *testing.T) {
	code, out, _ := exec(t, "8812345\n", "parse", "-json")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	var got parseOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if got.Error == "" || got.Raw != "8812345" {
		t.Errorf("got %+v", got)
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{name: "raw", args: []string{"encode", "(10)LOT42(01)04150000021126(21)SERIAL"}, wantOut: "\x1D0104150000021126" + "10LOT42\x1D21SERIAL\n"},
		{name: "hri reorders", args: []string{"encode", "-hri", "(10)LOT42(01)04150000021126(21)SERIAL"}, wantOut: "(01)04150000021126(10)LOT42(21)SERIAL\n"},
		{name: "zpl", args: []string{"encode", "-zpl", "(10)LOT42(01)04150000021126(21)SERIAL"}, wantOut: ">80104150000021126" + "10LOT42>821SERIAL\n"},
		{name: "zpl escapes greater-than", args: []string{"encode", "-zpl", "(01)04150000021126(10)A>B"}, wantOut: ">80104150000021126" + "10A><B\n"},
		{name: "raw scan input", args: []string{"encode", "-hri", "]d20104150000021126" + "10LOT42"}, wantOut: "(01)04150000021126(10)LOT42\n"},
		{name: "logistic label", args: []string{"encode", "-hri", "(00)790123451234500116(02)49012345008794(37)12(10)ABC123"}, wantOut: "(00)790123451234500116(02)49012345008794(37)12(10)ABC123\n"},
		{name: "association", args: []string{"encode", "(02)49012345008794(37)12"}, wantCode: 1, wantErr: "AI (37) requires AI (00)"},
		{name: "check digit", args: []string{"encode", "(01)04150000021127"}, wantCode: 1, wantErr: "invalid check digit"},
		{name: "unknown AI", args: []string{"encode", "(999)1"}, wantCode: 1, wantErr: "unknown application identifier"},
		{name: "malformed HRI", args: []string{"encode", "(01"}, wantCode: 1, wantErr: "invalid data"},
		{name: "empty value", args: []string{"encode", "(01)(10)LOT"}, wantCode: 1, wantErr: "has no value"},
		{name: "both flags", args: []string{"encode", "-hri", "-zpl", "(01)04150000021126"}, wantCode: 2, wantErr: "mutually exclusive"},
		{name: "too many arguments", args: []string{"encode", "(01)04150000021126", "(10)X"}, wantCode: 2, wantErr: "at most one"},
		{name: "stdin stream", args: []string{"encode", "-hri"}, stdin: "(10)A(01)04150000021126\n\n(21)S\n(00)790123451234500116\n", wantCode: 1,
			wantOut: "(01)04150000021126(10)A\n(00)790123451234500116\n", wantErr: "AI (21) requires"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := exec(t, tt.stdin, tt.args...)
			if code != tt.wantCode || out != tt.wantOut || !strings.Contains(errOut, tt.wantErr) {
				t.Errorf("exit %d stdout %q stderr %q, want exit %d stdout %q stderr containing %q", code, out, errOut, tt.wantCode, tt.wantOut, tt.wantErr)
			}
		})
	}
}

func TestEncodeRoundTripsThroughParse(t *testing.T) {
	code, raw, _ := exec(t, "", "encode", "(10)LOT42(01)04150000021126(17)250630")
	if code != 0 {
		t.Fatalf("encode exit %d", code)
	}
	code, out, _ := exec(t, "", "parse", "-strict", "-check-digits", "-json", strings.TrimSpace(raw))
	if code != 0 {
		t.Fatalf("parse exit %d", code)
	}
	var got parseOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Elements) != 3 || got.Elements[0].AI != "01" || got.Elements[1].AI != "17" || got.Elements[2].Value != "LOT42" {
		t.Errorf("parse(encode()) elements = %+v", got.Elements)
	}
}

func TestGTIN(t *testing.T) {
	if code, out, _ := exec(t, "", "gtin", "04150000021126"); code != 0 || !strings.Contains(out, "valid") {
		t.Errorf("valid GTIN: exit %d out %q", code, out)
	}
	if code, _, _ := exec(t, "", "gtin", "04150000021127"); code != 1 {
		t.Errorf("bad check digit: exit %d, want 1", code)
	}
	if code, _, _ := exec(t, "", "gtin", "12"); code != 2 {
		t.Errorf("bad length: exit %d, want 2", code)
	}
}

func TestSSCC(t *testing.T) {
	code, out, _ := exec(t, "", "sscc", "-ext", "3", "-gcp", "7801234", "-serial", "000000009")
	if code != 0 || strings.TrimSpace(out) != "378012340000000095" {
		t.Errorf("SSCC: exit %d out %q", code, out)
	}
	if code, _, _ := exec(t, "", "sscc", "-ext", "3", "-gcp", "7", "-serial", "000000009"); code != 1 {
		t.Errorf("invalid SSCC: exit %d, want 1", code)
	}
}

func TestAI(t *testing.T) {
	code, out, _ := exec(t, "", "ai", "3102")
	if code != 0 || !strings.Contains(out, "Net Weight kg") || !strings.Contains(out, "N6") {
		t.Errorf("exit %d out %q", code, out)
	}
	if code, _, _ := exec(t, "", "ai", "88"); code != 1 {
		t.Errorf("unknown AI: exit %d, want 1", code)
	}
}

func TestUsageAndVersion(t *testing.T) {
	if code, _, errOut := exec(t, ""); code != 2 || !strings.Contains(errOut, "Usage") {
		t.Errorf("no args: exit %d stderr %q", code, errOut)
	}
	if code, out, _ := exec(t, "", "help"); code != 0 || !strings.Contains(out, "Usage") {
		t.Errorf("help: exit %d", code)
	}
	if code, _, _ := exec(t, "", "bogus"); code != 2 {
		t.Errorf("unknown command: exit %d", code)
	}
	if code, out, _ := exec(t, "", "version"); code != 0 || !strings.HasPrefix(out, "gs1 ") {
		t.Errorf("version: exit %d out %q", code, out)
	}
}
