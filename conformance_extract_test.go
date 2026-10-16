package gs1

import (
	"reflect"
	"testing"
)

func TestParseCArgsExtractsLiteralsAndIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		args     []string
		literals []bool
	}{
		{name: "identifier and strings", src: `OK, "(01)123", "^01123")`, args: []string{"OK", "(01)123", "^01123"}, literals: []bool{false, true, true}},
		{name: "adjacent literals", src: "OK, \"]C1\" \"\\x1D\" \"99X\",\n\t\tQR, \"^99X\")", args: []string{"OK", "]C1\x1d99X", "QR", "^99X"}, literals: []bool{false, true, false, true}},
		{name: "whitespace inside strings is kept", src: `OK, " AB C ", X)`, args: []string{"OK", " AB C ", "X"}, literals: []bool{false, true, false}},
		{name: "empty string", src: `ERR, "", NONE, "")`, args: []string{"ERR", "", "NONE", ""}, literals: []bool{false, true, false, true}},
		{name: "escapes", src: `OK, "a\"b\\c\t\x41\x1d")`, args: []string{"OK", "a\"b\\c\tA\x1d"}, literals: []bool{false, true}},
		{name: "trailing comment", src: "OK, \"x\")\t// comment\n", args: []string{"OK", "x"}, literals: []bool{false, true}},
		{name: "comment between arguments", src: "OK, /* why */ \"x\",\n// note\n\"y\")", args: []string{"OK", "x", "y"}, literals: []bool{false, true, true}},
		{name: "nested parentheses in identifier", src: `sizeof(x), "y")`, args: []string{"sizeof(x)", "y"}, literals: []bool{false, true}},
		{name: "table row", src: `"(00)123", GS1_LINTER_OK }`, args: []string{"(00)123", "GS1_LINTER_OK"}, literals: []bool{true, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, literals, _, ok := parseCArgs(tt.src, 0)
			if !ok {
				t.Fatal("parseCArgs() failed")
			}
			if !reflect.DeepEqual(args, tt.args) || !reflect.DeepEqual(literals, tt.literals) {
				t.Errorf("parseCArgs() = %q %v, want %q %v", args, literals, tt.args, tt.literals)
			}
		})
	}
	if _, _, _, ok := parseCArgs(`OK, "unterminated`, 0); ok {
		t.Error("unterminated string accepted")
	}
}

func TestFindCallsSkipsMacroDefinitionsAndPrefixes(t *testing.T) {
	src := `
#define test_parseAIdata(err, d, e) do {					\
	do_test_parseAIdata(ctx, __FILE__, __LINE__, gs1_encoder_e##err, d, e);	\
} while (0)

	test_parseAIdata(OK, "(01)12345678901231", "^0112345678901231");
	do_test_parseAIdata(ctx, "x", "y");
	test_parseAIdata(AI_UNKNOWN, "(1A)12345", "");
	test_parseAIdata(OK, inbuf, expbuf);
`
	calls := findCalls(src, "test_parseAIdata", 3)
	if len(calls) != 3 {
		t.Fatalf("findCalls() = %d calls, want 3: %+v", len(calls), calls)
	}
	if calls[0].line != 6 || calls[0].args[1] != "(01)12345678901231" || !calls[0].literal(1) {
		t.Errorf("first call = %+v", calls[0])
	}
	if calls[2].literal(1) {
		t.Errorf("variable argument reported as literal: %+v", calls[2])
	}
}

func TestFindTableRows(t *testing.T) {
	src := `
void test_ai_linters(void) {
	static const struct linter_test_s tests[] = {
		{ "(00)123456789012345675",				GS1_LINTER_OK },
		{ "(10) ",						GS1_LINTER_INVALID_CSET82_CHARACTER },
	};
}

void test_other(void) {
	{ "(99)X", GS1_LINTER_OK },
}
`
	rows := findTableRows(src, "test_ai_linters")
	if len(rows) != 2 || rows[0].args[0] != "(00)123456789012345675" || rows[1].args[0] != "(10) " || rows[1].args[1] != "GS1_LINTER_INVALID_CSET82_CHARACTER" {
		t.Errorf("findTableRows() = %+v", rows)
	}
}

func TestUnhat(t *testing.T) {
	tests := map[string]string{
		"^0112345^10ABC": "\x1d0112345\x1d10ABC",
		"\\^TESTING":     "^TESTING",
		"plain":          "plain",
	}
	for in, want := range tests {
		if got := unhat(in); got != want {
			t.Errorf("unhat(%q) = %q, want %q", in, got, want)
		}
	}
}
