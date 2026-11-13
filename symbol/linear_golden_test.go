package symbol

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// goldenCase is one zint-generated case: the symbology, the data given to
// zint and the module row it produced.
type goldenCase struct{ sym, data, row string }

// readGoldenCases parses testdata/linear-golden.txt (see gen-linear-golden.sh).
func readGoldenCases(t *testing.T) []goldenCase {
	t.Helper()
	f, err := os.Open("testdata/linear-golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	var cases []goldenCase
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != "case" {
			t.Fatalf("unexpected line %q", line)
		}
		if !sc.Scan() {
			t.Fatalf("case %s %s: missing module row", fields[1], fields[2])
		}
		cases = append(cases, goldenCase{fields[1], fields[2], sc.Text()})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return cases
}

// TestLinearGolden compares every symbol with the module row zint produces
// for the same data.
func TestLinearGolden(t *testing.T) {
	build := map[string]func(string) (*Linear, error){
		"ean13": EAN13,
		"ean8":  EAN8,
		"upca":  UPCA,
		"upce":  UPCE,
		"itf14": func(s string) (*Linear, error) { return ITF14(s, ITF14Options{}) },
	}
	cases := readGoldenCases(t)
	if len(cases) < 50 {
		t.Fatalf("only %d golden cases", len(cases))
	}
	for _, c := range cases {
		fn, ok := build[c.sym]
		if !ok {
			t.Fatalf("unknown symbology %q", c.sym)
		}
		t.Run(c.sym+"/"+c.data, func(t *testing.T) {
			data := c.data
			if c.sym == "itf14" {
				// zint derives the ITF-14 check digit from thirteen digits.
				data = withCheck(t, data)
			}
			l, err := fn(data)
			if err != nil {
				t.Fatal(err)
			}
			got, want := modules(l), hexModules(t, c.row)
			if len(want) < len(got) || !strings.HasPrefix(want, got) || strings.Trim(want[len(got):], "0") != "" {
				t.Errorf("module row differs from zint\n got %s\nwant %s", got, want)
			}
			if c.sym == "upce" && !strings.HasPrefix(l.HRI(), data) {
				t.Errorf("HRI %s does not start with %s", l.HRI(), data)
			}
		})
	}
}

// hexModules expands a zint --dump row into a string of '0' and '1'. Each
// hex digit is four modules, most significant bit first; zint writes two
// digits per byte and a single digit when only the high nibble is left.
func hexModules(t *testing.T, row string) string {
	t.Helper()
	var b strings.Builder
	for _, h := range strings.Fields(row) {
		if len(h) != 1 && len(h) != 2 {
			t.Fatalf("bad hex token %q in %q", h, row)
		}
		for _, c := range []byte(h) {
			var n byte
			switch {
			case c >= '0' && c <= '9':
				n = c - '0'
			case c >= 'A' && c <= 'F':
				n = c - 'A' + 10
			case c >= 'a' && c <= 'f':
				n = c - 'a' + 10
			default:
				t.Fatalf("bad hex token %q in %q", h, row)
			}
			for i := 3; i >= 0; i-- {
				if n&(1<<i) != 0 {
					b.WriteByte('1')
				} else {
					b.WriteByte('0')
				}
			}
		}
	}
	return b.String()
}
