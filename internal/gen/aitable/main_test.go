package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    []entry
		wantErr bool
	}{
		{
			name: "predefined key with linters and dlpkey",
			line: "00         *?  N18,csum,gcppos2                  dlpkey                                           # SSCC",
			want: []entry{{AI: "00", Title: "SSCC", Predefined: true, DLAttribute: true,
				Components: []component{{Type: 'N', Len: 18, Linters: []string{"csum", "gcppos2"}}}, DLPKey: true}},
		},
		{
			name: "attributes with qualifiers",
			line: "01         *?  N14,csum,gcppos2                  ex=255,37 dlpkey=22,10,21|235                    # GTIN",
			want: []entry{{AI: "01", Title: "GTIN", Predefined: true, DLAttribute: true,
				Components: []component{{Type: 'N', Len: 14, Linters: []string{"csum", "gcppos2"}}},
				Ex:         []string{"255", "37"}, DLPKey: true, DLQualifiers: "22,10,21|235"}},
		},
		{
			name: "no flags column",
			line: "21             X..20                             req=01,03,8006 ex=235                            # SERIAL",
			want: []entry{{AI: "21", Title: "SERIAL",
				Components: []component{{Type: 'X', Len: 20, Variable: true}},
				Req:        [][]string{{"01", "03", "8006"}}, Ex: []string{"235"}}},
		},
		{
			name: "range with pattern exclusion",
			line: "3100-3102  *?  N6                                req=01,02 ex=310n                                # NET WEIGHT (kg)",
			want: []entry{
				{AI: "3100", Title: "NET WEIGHT (kg)", Predefined: true, DLAttribute: true, Components: []component{{Type: 'N', Len: 6}}, Req: [][]string{{"01", "02"}}, Ex: []string{"310n"}},
				{AI: "3101", Title: "NET WEIGHT (kg)", Predefined: true, DLAttribute: true, Components: []component{{Type: 'N', Len: 6}}, Req: [][]string{{"01", "02"}}, Ex: []string{"310n"}},
				{AI: "3102", Title: "NET WEIGHT (kg)", Predefined: true, DLAttribute: true, Components: []component{{Type: 'N', Len: 6}}, Req: [][]string{{"01", "02"}}, Ex: []string{"310n"}},
			},
		},
		{
			name: "optional components and repeated req",
			line: "253         ?  N13,csum,gcppos1 [X..17]          dlpkey req=01 req=21+10                          # GDTI",
			want: []entry{{AI: "253", Title: "GDTI", DLAttribute: true,
				Components: []component{{Type: 'N', Len: 13, Linters: []string{"csum", "gcppos1"}}, {Type: 'X', Len: 17, Variable: true, Optional: true}},
				Req:        [][]string{{"01"}, {"21+10"}}, DLPKey: true}},
		},
		{
			name: "bracketed component with linter outside the bracket",
			line: "423         ?  N3,iso3166 [N3],iso3166 [N3],iso3166  req=01,02 ex=426   # COUNTRY - INITIAL PROCESS",
			want: []entry{{AI: "423", Title: "COUNTRY - INITIAL PROCESS", DLAttribute: true,
				Components: []component{{Type: 'N', Len: 3, Linters: []string{"iso3166"}}, {Type: 'N', Len: 3, Optional: true, Linters: []string{"iso3166"}}, {Type: 'N', Len: 3, Optional: true, Linters: []string{"iso3166"}}},
				Req:        [][]string{{"01", "02"}}, Ex: []string{"426"}}},
		},
		{
			name: "title containing a hash",
			line: "7030        ?  N3,iso3166999 X..27               req=01,02                                        # PROCESSOR # 0",
			want: []entry{{AI: "7030", Title: "PROCESSOR # 0", DLAttribute: true,
				Components: []component{{Type: 'N', Len: 3, Linters: []string{"iso3166999"}}, {Type: 'X', Len: 27, Variable: true}},
				Req:        [][]string{{"01", "02"}}}},
		},
		{
			name: "no title and base64url type",
			line: "8030        ?  Z..90  req=00,01+21,253",
			want: []entry{{AI: "8030", DLAttribute: true,
				Components: []component{{Type: 'Z', Len: 90, Variable: true}},
				Req:        [][]string{{"00", "01+21", "253"}}}},
		},
		{
			name: "two-digit range",
			line: "91-93       ?  X..90                                                                              # INTERNAL",
			want: []entry{
				{AI: "91", Title: "INTERNAL", DLAttribute: true, Components: []component{{Type: 'X', Len: 90, Variable: true}}},
				{AI: "92", Title: "INTERNAL", DLAttribute: true, Components: []component{{Type: 'X', Len: 90, Variable: true}}},
				{AI: "93", Title: "INTERNAL", DLAttribute: true, Components: []component{{Type: 'X', Len: 90, Variable: true}}},
			},
		},
		{name: "missing specification", line: "01", wantErr: true},
		{name: "unknown type", line: "01  *  Q14  # BAD", wantErr: true},
		{name: "unknown attribute", line: "01  *  N14 foo=bar  # BAD", wantErr: true},
		{name: "bad pattern", line: "01  *  N14 ex=1  # BAD", wantErr: true},
		{name: "unbalanced bracket", line: "01  *  [N14  # BAD", wantErr: true},
		{name: "mandatory after variable", line: "01  *  N..14 X2  # BAD", wantErr: true},
		{name: "mandatory after optional", line: "01  *  N14 [X2] X2  # BAD", wantErr: true},
		{name: "descending range", line: "3105-3100  *  N6  # BAD", wantErr: true},
		{name: "five-digit AI", line: "12345  *  N6  # BAD", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseLine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLine() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseDictionary(t *testing.T) {
	src := `# GS1 Barcode Syntax Dictionary
# Release: 2026-01-27
#
00         *?  N18,csum,gcppos2                  dlpkey                                           # SSCC
3100-3101  *?  N6                                req=01,02 ex=310n                                # NET WEIGHT (kg)
`
	release, entries, err := parseDictionary(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if release != "2026-01-27" || len(entries) != 3 {
		t.Fatalf("release %q, %d entries", release, len(entries))
	}
	if _, _, err := parseDictionary(strings.NewReader("00  *  N18\n")); err == nil {
		t.Error("missing release line accepted")
	}
	if _, _, err := parseDictionary(strings.NewReader("# Release: x\n00  *  N18\n00  *  N18\n")); err == nil {
		t.Error("duplicate AI accepted")
	}
	if _, _, err := parseDictionary(strings.NewReader("# Release: x\n")); err == nil {
		t.Error("empty dictionary accepted")
	}
}

func TestDerive(t *testing.T) {
	tests := []struct {
		name       string
		components []component
		want       derived
	}{
		{name: "fixed numeric", components: []component{{Type: 'N', Len: 14}}, want: derived{FixedLen: 14, MinLen: 14, MaxLen: 14, Numeric: true}},
		{name: "variable alphanumeric", components: []component{{Type: 'X', Len: 20, Variable: true}}, want: derived{MinLen: 1, MaxLen: 20}},
		{name: "numeric prefix with optional", components: []component{{Type: 'N', Len: 13}, {Type: 'X', Len: 17, Variable: true, Optional: true}}, want: derived{MinLen: 13, MaxLen: 30}},
		{name: "fixed compound", components: []component{{Type: 'N', Len: 1}, {Type: 'X', Len: 1}, {Type: 'X', Len: 1}, {Type: 'X', Len: 1}}, want: derived{FixedLen: 4, MinLen: 4, MaxLen: 4}},
		{name: "numeric with optional numeric", components: []component{{Type: 'N', Len: 18}, {Type: 'N', Len: 2, Variable: true, Optional: true}}, want: derived{MinLen: 18, MaxLen: 20, Numeric: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := derive(tt.components); got != tt.want {
				t.Errorf("derive() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGenerateIsValidGo(t *testing.T) {
	entries, err := parseLine("01         *?  N14,csum,gcppos2                  ex=255,37 dlpkey=22,10,21|235                    # GTIN")
	if err != nil {
		t.Fatal(err)
	}
	src, err := generate("2026-01-27", entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"DO NOT EDIT", `syntaxDictionaryRelease = "2026-01-27"`, `"01": {AI: "01", Title: "GTIN", Predefined: true, DLAttribute: true`, `FixedLen: 14, MinLen: 14, MaxLen: 14, DataType: dataNumeric`} {
		if !strings.Contains(string(src), want) {
			t.Errorf("generated source lacks %q:\n%s", want, src)
		}
	}
}

// TestCommittedTableIsCurrent regenerates the table from the vendored
// dictionary and compares it with the committed ai_table.go, so a stale
// table fails go test as well as the CI diff check.
func TestCommittedTableIsCurrent(t *testing.T) {
	text, err := os.ReadFile("gs1-syntax-dictionary.txt")
	if err != nil {
		t.Fatal(err)
	}
	release, entries, err := parseDictionary(bytes.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	want, err := generate(release, entries)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join("..", "..", "..", "ai_table.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("ai_table.go is stale: run make generate")
	}
}
