package gs1

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// updateDivergences rewrites testdata/gs1/known-divergences.txt from the
// current outcomes, keeping the reasons already recorded and proposing one
// for new entries. Use it after vendoring a new release or changing the
// parser, then review the diff:
//
//	go test -run TestConformance . -args -conformance.update
var updateDivergences = flag.Bool("conformance.update", false, "rewrite testdata/gs1/known-divergences.txt from the current outcomes")

// The conformance suite runs the test vectors that GS1 publishes with the
// Barcode Syntax Engine and the Barcode Syntax Dictionary through this
// parser. GS1 ships them inside C unit tests, so the vendored sources under
// testdata/gs1 are scanned for the test calls and every vector becomes a
// subtest. testdata/gs1/README.md documents the sources and the divergence
// baseline.

// conformanceVector is one reference case: an input, the outcome the
// reference implementation expects, and for some suites the expected
// element string.
type conformanceVector struct {
	suite    string
	line     int
	input    string
	expectOK bool
	expected string // raw AI data with "^" as FNC1, when the vector states it
	skip     string // reason the vector is not applicable to this package
}

// key identifies a vector in the divergence baseline independently of line
// numbers, which change between vendored releases.
func (v conformanceVector) key() string {
	return v.suite + "\t" + strconv.Quote(v.input)
}

// TestConformance runs every vendored vector and compares the set of
// diverging vectors with testdata/gs1/known-divergences.txt. A vector that
// diverges without being listed is a regression; a listed vector that now
// agrees is a stale entry. Both fail the test so the conformance rate only
// changes deliberately. The rate is logged per suite for CI output.
func TestConformance(t *testing.T) {
	vectors := loadConformanceVectors(t)
	baseline := loadKnownDivergences(t)

	type tally struct{ total, agree, skipped int }
	tallies := make(map[string]*tally)
	var suites []string
	diverging := make(map[string]conformanceVector)

	for _, v := range vectors {
		tl, ok := tallies[v.suite]
		if !ok {
			tl = &tally{}
			tallies[v.suite] = tl
			suites = append(suites, v.suite)
		}
		tl.total++
		if v.skip != "" {
			tl.skipped++
			continue
		}
		if err := runConformanceVector(v); err == nil {
			tl.agree++
		} else {
			diverging[v.key()] = v
			if _, known := baseline[v.key()]; !known {
				t.Errorf("new divergence %s line %d %q: %v", v.suite, v.line, v.input, err)
			}
		}
	}
	for key, reason := range baseline {
		if _, still := diverging[key]; !still {
			t.Errorf("stale divergence entry (now agrees, remove it): %s # %s", key, reason)
		}
	}
	if *updateDivergences {
		writeKnownDivergences(t, vectors, diverging, baseline)
	}

	sort.Strings(suites)
	total, agree, skipped := 0, 0, 0
	for _, suite := range suites {
		tl := tallies[suite]
		applicable := tl.total - tl.skipped
		t.Logf("conformance %-32s %4d/%-4d agree (%5.1f%%), %d not applicable", suite, tl.agree, applicable, pct(tl.agree, applicable), tl.skipped)
		total += applicable
		agree += tl.agree
		skipped += tl.skipped
	}
	t.Logf("conformance %-32s %4d/%-4d agree (%5.1f%%), %d not applicable, %d known divergences", "TOTAL", agree, total, pct(agree, total), skipped, len(baseline))
	if agree == 0 {
		t.Fatal("no conformance vectors agreed; the vendored sources were probably not parsed")
	}
}

func pct(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return 100 * float64(part) / float64(whole)
}

// runConformanceVector returns nil when this package agrees with the
// reference outcome, or an error describing the divergence.
func runConformanceVector(v conformanceVector) error {
	switch v.suite {
	case "engine/parseAIdata":
		// Bracket notation to AI data. Compare the element split with the
		// expected raw string; the reference keeps input order and this
		// package does too when parsing.
		b, err := Parse(v.input)
		return compareOutcome(v, err, func() error { return compareElements(b, v.expected) })
	case "engine/processAIdata", "engine/linters":
		// Raw AI data validated with every linter. Check digits are the one
		// linter this package applies on request.
		_, err := ParseWithOptions(unhat(v.input), ParseOptions{ValidateCheckDigits: true})
		return compareOutcome(v, err, nil)
	case "engine/validateAIs":
		_, err := ParseWithOptions(v.input, ParseOptions{ValidateAssociations: true})
		return compareOutcome(v, err, nil)
	case "engine/scanData":
		// The reference always validates the primary check digit of
		// EAN/UPC scan data.
		b, err := ParseWithOptions(v.input, ParseOptions{ValidateCheckDigits: true})
		return compareOutcome(v, err, func() error { return compareElements(b, v.expected) })
	}
	if strings.HasPrefix(v.suite, "dictionary/lint_") {
		linter := strings.TrimPrefix(v.suite, "dictionary/lint_")
		fn, ok := conformanceLinters[linter]
		if !ok {
			return fmt.Errorf("no implementation for linter %s", linter)
		}
		return compareOutcome(v, fn(v.input), nil)
	}
	return fmt.Errorf("unknown suite %s", v.suite)
}

// compareOutcome checks accept/reject agreement and, on an expected accept,
// runs the optional payload comparison.
func compareOutcome(v conformanceVector, err error, payload func() error) error {
	if v.expectOK {
		if err != nil {
			return fmt.Errorf("reference accepts, this package rejects: %w", err)
		}
		if payload != nil {
			return payload()
		}
		return nil
	}
	if err == nil {
		return errors.New("reference rejects, this package accepts")
	}
	return nil
}

// compareElements checks that b carries the same elements as the expected
// raw AI data, written with "^" as FNC1.
func compareElements(b Barcode, expected string) error {
	if expected == "" {
		return nil
	}
	want, err := Parse(unhat(expected))
	if err != nil {
		return fmt.Errorf("expected data %q does not parse: %w", expected, err)
	}
	if len(want.Elements) != len(b.Elements) {
		return fmt.Errorf("elements = %+v, want %+v", b.Elements, want.Elements)
	}
	for i := range want.Elements {
		if want.Elements[i] != b.Elements[i] {
			return fmt.Errorf("elements = %+v, want %+v", b.Elements, want.Elements)
		}
	}
	return nil
}

// unhat converts the reference's "^" FNC1 placeholder to the GS character
// this package uses. A "\^" escape is a literal caret.
func unhat(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '^':
			out.WriteByte('^')
			i++
		case s[i] == '^':
			out.WriteByte(byte(fnc1))
		default:
			out.WriteByte(s[i])
		}
	}
	return out.String()
}

// conformanceLinters maps the dictionary linters this package implements to
// the function that checks a single component value.
var conformanceLinters = map[string]func(string) error{
	"csum": ValidateCheckDigit,
	"yymmdd": func(s string) error {
		if len(s) == 6 && s[4:] == "00" {
			return fmt.Errorf("%w: day 00 not allowed by yymmdd", ErrInvalidDate)
		}
		_, err := ParseDate(s)
		return err
	},
	"yymmd0": func(s string) error {
		_, err := ParseDate(s)
		return err
	},
	"csetnumeric": func(s string) error { return checkCharset(s, dataNumeric) },
	"cset82": func(s string) error {
		for i := 0; i < len(s); i++ {
			if !inCSET82(s[i]) {
				return fmt.Errorf("%w: byte %q at %d is not in character set 82", ErrInvalidData, s[i], i)
			}
		}
		return nil
	},
	"cset39": func(s string) error { return checkCharset(s, dataCSET39) },
	"cset64": validateBase64URL,
	"zero": func(s string) error {
		if !isZeros(s) {
			return fmt.Errorf("%w: not zero", ErrInvalidData)
		}
		return nil
	},
}

func checkCharset(s string, t dataType) error {
	for i := 0; i < len(s); i++ {
		if !inCharset(s[i], t) {
			return fmt.Errorf("%w: byte %q at %d is not in %s", ErrInvalidData, s[i], i, t.description())
		}
	}
	return nil
}

// loadConformanceVectors scans the vendored C sources for the reference
// test calls.
func loadConformanceVectors(t *testing.T) []conformanceVector {
	t.Helper()
	vectors := append(loadEngineVectors(t), loadDictionaryVectors(t)...)
	if len(vectors) < 500 {
		t.Fatalf("only %d vectors extracted; the vendored sources changed shape", len(vectors))
	}
	return vectors
}

func loadEngineVectors(t *testing.T) []conformanceVector {
	t.Helper()
	var vectors []conformanceVector
	engine := filepath.Join("testdata", "gs1", "syntax-engine")
	ai := readTestdata(t, filepath.Join(engine, "ai.c"))
	scan := readTestdata(t, filepath.Join(engine, "scandata.c"))

	for _, c := range findCalls(ai, "test_parseAIdata", 3) {
		if c.literal(1) && c.literal(2) {
			vectors = append(vectors, conformanceVector{suite: "engine/parseAIdata", line: c.line, input: c.args[1], expectOK: c.args[0] == "OK", expected: c.args[2]})
		}
	}
	for _, c := range findCalls(ai, "test_processAIdata", 2) {
		if c.literal(1) {
			vectors = append(vectors, conformanceVector{suite: "engine/processAIdata", line: c.line, input: c.args[1], expectOK: c.args[0] == "OK"})
		}
	}
	for _, c := range findCalls(ai, "test_validateAIs", 3) {
		if c.literal(2) {
			vectors = append(vectors, conformanceVector{suite: "engine/validateAIs", line: c.line, input: c.args[2], expectOK: c.args[0] == "OK"})
		}
	}
	for _, c := range findTableRows(ai, "test_ai_linters") {
		vectors = append(vectors, conformanceVector{suite: "engine/linters", line: c.line, input: c.args[0], expectOK: c.args[1] == "GS1_LINTER_OK"})
	}
	return append(vectors, loadScanDataVectors(scan)...)
}

// loadScanDataVectors extracts the scan data vectors and marks the ones
// outside this package's scope as not applicable.
func loadScanDataVectors(scan string) []conformanceVector {
	var vectors []conformanceVector
	for _, c := range findCalls(scan, "test_testProcessScanData", 4) {
		if !c.literal(1) || !c.literal(3) {
			continue
		}
		v := conformanceVector{suite: "engine/scanData", line: c.line, input: c.args[1], expectOK: c.args[0] == "OK", expected: c.args[3]}
		switch {
		case v.expectOK && !strings.HasPrefix(v.expected, "^"):
			v.skip = "non-GS1 symbology or Digital Link payload is returned verbatim by the reference; this package parses AI data only"
		case !v.expectOK && !isGS1SymbologyInput(v.input):
			v.skip = "rejection concerns a symbology identifier this package does not model"
		}
		vectors = append(vectors, v)
	}
	return vectors
}

func loadDictionaryVectors(t *testing.T) []conformanceVector {
	t.Helper()
	var vectors []conformanceVector
	dict := filepath.Join("testdata", "gs1", "syntax-dictionary")
	entries, err := os.ReadDir(dict)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "lint_") || !strings.HasSuffix(e.Name(), ".c") {
			continue
		}
		src := readTestdata(t, filepath.Join(dict, e.Name()))
		suite := "dictionary/" + strings.TrimSuffix(e.Name(), ".c")
		for _, c := range findCalls(src, "UNIT_TEST_PASS", 2) {
			vectors = append(vectors, conformanceVector{suite: suite, line: c.line, input: c.args[1], expectOK: true})
		}
		for _, c := range findCalls(src, "UNIT_TEST_FAIL", 4) {
			vectors = append(vectors, conformanceVector{suite: suite, line: c.line, input: c.args[1]})
		}
	}
	return vectors
}

// isGS1SymbologyInput reports whether scan data starts with a symbology
// identifier that carries GS1 AI data and that this package recognizes.
func isGS1SymbologyInput(s string) bool {
	for _, id := range []string{"]C1", "]d2", "]Q3", "]e0", "]J1", "]E0", "]E4", "]E1", "]E2", "]I1", "]I2"} {
		if strings.HasPrefix(s, id) {
			return true
		}
	}
	return false
}

func readTestdata(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("vendored vectors missing: %v", err)
	}
	return string(data)
}

// loadKnownDivergences reads testdata/gs1/known-divergences.txt, one
// "<suite>\t<quoted input>\t# reason" entry per line.
func loadKnownDivergences(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "gs1", "known-divergences.txt"))
	if err != nil {
		t.Fatal(err)
	}
	baseline := make(map[string]string)
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			t.Fatalf("known-divergences.txt line %d: expected <suite>\\t<quoted input>\\t# reason", lineNo)
		}
		if _, err := strconv.Unquote(parts[1]); err != nil {
			t.Fatalf("known-divergences.txt line %d: input is not a Go quoted string: %v", lineNo, err)
		}
		reason := ""
		if len(parts) == 3 {
			reason = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(parts[2]), "#"))
		}
		key := parts[0] + "\t" + parts[1]
		if _, dup := baseline[key]; dup {
			t.Fatalf("known-divergences.txt line %d: duplicate entry %s", lineNo, key)
		}
		baseline[key] = reason
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return baseline
}

// writeKnownDivergences rewrites the baseline from the current outcomes in
// source order, keeping recorded reasons and proposing one for new entries
// from the linters of the AI involved.
func writeKnownDivergences(t *testing.T, vectors []conformanceVector, diverging map[string]conformanceVector, baseline map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("# Vectors whose outcome differs from the GS1 reference, one per line:\n")
	b.WriteString("# <suite>\t<quoted input>\t# reason\n")
	b.WriteString("# Regenerate with: go test -run TestConformance . -args -conformance.update\n")
	written := make(map[string]bool)
	for _, v := range vectors {
		key := v.key()
		if _, ok := diverging[key]; !ok || written[key] {
			continue
		}
		written[key] = true
		reason, ok := baseline[key]
		if !ok || reason == "" {
			reason = proposeReason(v)
		}
		fmt.Fprintf(&b, "%s\t# %s\n", key, reason)
	}
	path := filepath.Join("testdata", "gs1", "known-divergences.txt")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d divergences to %s", len(written), path)
}

// proposeReason names the dictionary linters of the vector's first AI that
// this package does not apply while parsing, which explains most
// divergences in the linter suites.
func proposeReason(v conformanceVector) string {
	input := strings.TrimPrefix(v.input, "^")
	var ai string
	if strings.HasPrefix(input, "(") {
		if end := strings.IndexByte(input, ')'); end > 0 {
			ai = input[1:end]
		}
	} else if spec, _, ok := lookupAIAt(input, 0); ok {
		ai = spec.AI
	}
	spec, ok := aiTable[ai]
	if !ok {
		return "TODO classify"
	}
	var pending []string
	for _, c := range spec.Components {
		for _, l := range c.Linters {
			switch l {
			case "csum", "zero", "cset82":
				continue
			}
			pending = append(pending, l)
		}
	}
	if len(pending) == 0 {
		return "TODO classify"
	}
	return "linter " + strings.Join(pending, ", ") + " not applied while parsing"
}

// cCall is one extracted macro call with its arguments: identifiers as
// written, string literals unescaped and concatenated.
type cCall struct {
	line     int
	args     []string
	literals []bool // whether each argument was a string literal
}

// literal reports whether argument i exists and was a string literal, as
// opposed to a variable the reference fills at run time.
func (c cCall) literal(i int) bool {
	return i < len(c.literals) && c.literals[i]
}

// findCalls extracts every "name(" call with at least minArgs arguments
// from C source, tolerating line breaks, adjacent string literals and
// trailing comments. Calls inside the macro definition itself (followed by
// "do {") and in comments are ignored.
func findCalls(src, name string, minArgs int) []cCall {
	var calls []cCall
	for pos := 0; ; {
		i := strings.Index(src[pos:], name+"(")
		if i < 0 {
			break
		}
		start := pos + i
		pos = start + len(name) + 1
		if start > 0 && (isIdentByte(src[start-1]) || src[start-1] == '#') {
			continue
		}
		args, literals, end, ok := parseCArgs(src, pos)
		if !ok || len(args) < minArgs || isMacroDefinition(src, end) {
			continue
		}
		calls = append(calls, cCall{line: lineOf(src, start), args: args, literals: literals})
		pos = end
	}
	return calls
}

// findTableRows extracts "{ "string", IDENT }" rows from the static table
// inside the named test function.
func findTableRows(src, function string) []cCall {
	i := strings.Index(src, "void "+function+"(")
	if i < 0 {
		return nil
	}
	body := src[i:]
	if end := strings.Index(body, "\n}\n"); end >= 0 {
		body = body[:end]
	}
	var rows []cCall
	for pos := 0; ; {
		j := strings.Index(body[pos:], "{ \"")
		if j < 0 {
			break
		}
		start := pos + j + 1
		args, literals, end, ok := parseCArgs(body, start)
		if !ok || len(args) != 2 {
			pos = start
			continue
		}
		rows = append(rows, cCall{line: lineOf(src, i+start), args: args, literals: literals})
		pos = end
	}
	return rows
}

func isMacroDefinition(src string, end int) bool {
	rest := strings.TrimLeft(src[end:], " \t")
	return strings.HasPrefix(rest, "do {")
}

func isIdentByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func lineOf(src string, pos int) int {
	return 1 + strings.Count(src[:pos], "\n")
}

// parseCArgs parses a comma-separated argument list starting after "(" or
// "{" and returns the arguments, whether each was a string literal, and the
// position after the closing delimiter. Identifiers are returned trimmed;
// adjacent string literals are unescaped and joined with their content
// intact; nested parentheses are kept verbatim.
func parseCArgs(src string, pos int) ([]string, []bool, int, bool) {
	var p cArgParser
	for pos < len(src) {
		c := src[pos]
		if c == '"' {
			lit, next, ok := parseCString(src, pos)
			if !ok {
				return nil, nil, pos, false
			}
			p.current.WriteString(lit)
			p.hasString = true
			pos = next
			continue
		}
		if c == '/' {
			if next, ok := skipCComment(src, pos); ok {
				pos = next
				continue
			}
		}
		if p.consume(c) {
			return p.args, p.literals, pos + 1, true
		}
		pos++
	}
	return nil, nil, pos, false
}

// cArgParser accumulates the arguments of one C call.
type cArgParser struct {
	args      []string
	literals  []bool
	current   strings.Builder
	hasString bool
	depth     int
}

// consume handles one byte outside string literals and comments and
// reports whether it closed the argument list.
func (p *cArgParser) consume(c byte) bool {
	if isCSpace(c) {
		// Whitespace before an argument or between adjacent literals is
		// not part of the string; inside an identifier it is kept.
		if !p.hasString && p.current.Len() > 0 {
			p.current.WriteByte(' ')
		}
		return false
	}
	switch c {
	case '(', '{':
		p.depth++
	case ')', '}':
		if p.depth == 0 {
			p.flush()
			return true
		}
		p.depth--
	case ',':
		if p.depth == 0 {
			p.flush()
			return false
		}
	}
	p.current.WriteByte(c)
	return false
}

func isCSpace(c byte) bool { return c == ' ' || c == '\n' || c == '\t' }

func (p *cArgParser) flush() {
	arg := p.current.String()
	if !p.hasString {
		arg = strings.TrimSpace(arg)
	}
	if p.hasString || arg != "" {
		p.args = append(p.args, arg)
		p.literals = append(p.literals, p.hasString)
	}
	p.current.Reset()
	p.hasString = false
}

// skipCComment returns the position after a // or /* */ comment starting
// at pos, or false when pos does not start a comment.
func skipCComment(src string, pos int) (int, bool) {
	switch {
	case strings.HasPrefix(src[pos:], "//"):
		if nl := strings.IndexByte(src[pos:], '\n'); nl >= 0 {
			return pos + nl, true
		}
		return len(src), true
	case strings.HasPrefix(src[pos:], "/*"):
		if end := strings.Index(src[pos+2:], "*/"); end >= 0 {
			return pos + end + 4, true
		}
		return len(src), true
	}
	return pos, false
}

// parseCString unescapes one C string literal starting at the opening
// quote and returns the position after the closing quote.
func parseCString(src string, pos int) (string, int, bool) {
	var out strings.Builder
	pos++ // opening quote
	for pos < len(src) {
		c := src[pos]
		switch c {
		case '"':
			return out.String(), pos + 1, true
		case '\\':
			if pos+1 >= len(src) {
				return "", pos, false
			}
			pos++
			switch e := src[pos]; e {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			case '0':
				out.WriteByte(0)
			case 'x':
				n := 0
				digits := 0
				for pos+1 < len(src) && digits < 2 && isHexByte(src[pos+1]) {
					pos++
					n = n*16 + hexValue(src[pos])
					digits++
				}
				if digits == 0 {
					return "", pos, false
				}
				out.WriteByte(byte(n))
			default:
				out.WriteByte(e) // \\ \" \' and \?
			}
		default:
			out.WriteByte(c)
		}
		pos++
	}
	return "", pos, false
}

func isHexByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	}
	return int(c-'A') + 10
}

// TestLogisticLabelFixtures parses the element strings in
// testdata/labels/logistic-labels.txt with association and check-digit
// validation and checks that every carrier of a label yields the same
// elements as its HRI line.
func TestLogisticLabelFixtures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "labels", "logistic-labels.txt"))
	if err != nil {
		t.Fatal(err)
	}
	opts := ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true}
	var reference []Element
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			reference = nil // a comment starts a new label
			continue
		}
		carrier, data, ok := strings.Cut(line, "\t")
		if !ok {
			t.Fatalf("line %d: expected <carrier>\\t<data>", lineNo)
		}
		data = strings.ReplaceAll(data, "{GS}", string(rune(fnc1)))
		b, err := ParseWithOptions(data, opts)
		if err != nil {
			t.Errorf("line %d (%s): %v", lineNo, carrier, err)
			continue
		}
		if reference == nil {
			reference = append([]Element(nil), b.Elements...)
			continue
		}
		if !sameElementSet(reference, b.Elements) {
			t.Errorf("line %d (%s): elements %+v differ from the label's first line %+v", lineNo, carrier, b.Elements, reference)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
}

func sameElementSet(a, b []Element) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[Element]int, len(a))
	for _, e := range a {
		seen[e]++
	}
	for _, e := range b {
		if seen[e] == 0 {
			return false
		}
		seen[e]--
	}
	return true
}
