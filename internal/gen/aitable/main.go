// Command aitable generates ai_table.go from the GS1 Barcode Syntax
// Dictionary vendored next to it. It is run by go generate from the root
// package and has no dependencies outside the standard library.
//
// The dictionary describes every Application Identifier as a sequence of
// typed components with optional linter names, flags for predefined-length
// AIs and GS1 Digital Link attributes, and association attributes (req=,
// ex=, dlpkey). This program parses that syntax and emits the aiSpec table
// consumed by the parser, encoder and association validator.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// component is one typed part of an AI data field.
type component struct {
	Type     byte // 'N', 'X', 'Y' or 'Z'
	Len      int
	Variable bool
	Optional bool
	Linters  []string
}

// entry is one Application Identifier after range expansion.
type entry struct {
	AI           string
	Title        string
	Predefined   bool
	DLAttribute  bool
	Components   []component
	Req          [][]string
	Ex           []string
	DLPKey       bool
	DLQualifiers string
}

func main() {
	dict := flag.String("dict", "", "path to gs1-syntax-dictionary.txt (default: next to this program)")
	out := flag.String("out", "", "output Go file (default: ai_table.go at the module root)")
	flag.Parse()

	dir := programDir()
	if *dict == "" {
		*dict = filepath.Join(dir, "gs1-syntax-dictionary.txt")
	}
	if *out == "" {
		*out = filepath.Join(dir, "..", "..", "..", "ai_table.go")
	}

	text, err := os.ReadFile(*dict)
	if err != nil {
		fatal(err)
	}
	release, entries, err := parseDictionary(bytes.NewReader(text))
	if err != nil {
		fatal(err)
	}
	src, err := generate(release, entries)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*out, src, 0o600); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "aitable: wrote %d AIs from release %s to %s\n", len(entries), release, *out)
}

// programDir locates this source file so the generator works both through
// go generate (cwd is the package directory) and from the module root.
func programDir() string {
	if _, err := os.Stat("gs1-syntax-dictionary.txt"); err == nil {
		return "."
	}
	return filepath.Join("internal", "gen", "aitable")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "aitable:", err)
	os.Exit(1)
}

var (
	componentRE = regexp.MustCompile(`^(\[)?([NXYZ])(\.\.)?([0-9]+)(\])?(?:,([A-Za-z0-9,]+))?$`)
	rangeRE     = regexp.MustCompile(`^([0-9]+)-([0-9]+)$`)
	aiRE        = regexp.MustCompile(`^[0-9]{2,4}$`)
	releaseRE   = regexp.MustCompile(`^#\s*Release:\s*(\S+)`)
)

// parseDictionary reads the dictionary text and returns the release date
// and every AI, with ranges such as 3100-3105 expanded.
func parseDictionary(r io.Reader) (string, []entry, error) {
	var release string
	var entries []entry
	seen := make(map[string]bool)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if m := releaseRE.FindStringSubmatch(trimmed); m != nil && release == "" {
				release = m[1]
			}
			continue
		}
		parsed, err := parseLine(trimmed)
		if err != nil {
			return "", nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		for _, e := range parsed {
			if seen[e.AI] {
				return "", nil, fmt.Errorf("line %d: duplicate AI %s", lineNo, e.AI)
			}
			seen[e.AI] = true
			entries = append(entries, e)
		}
	}
	if err := sc.Err(); err != nil {
		return "", nil, err
	}
	if release == "" {
		return "", nil, errors.New("dictionary header has no Release line")
	}
	if len(entries) == 0 {
		return "", nil, errors.New("dictionary has no entries")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].AI < entries[j].AI })
	return release, entries, nil
}

// parseLine parses one dictionary entry:
//
//	AIs  [Flags]  Specification  [Attributes...]  [# Title]
func parseLine(line string) ([]entry, error) {
	var title string
	if i := strings.IndexByte(line, '#'); i >= 0 {
		title = strings.TrimSpace(line[i+1:])
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, fmt.Errorf("expected AI and specification, got %q", line)
	}
	codes, err := expandAIs(fields[0])
	if err != nil {
		return nil, err
	}
	fields = fields[1:]

	base := entry{Title: title}
	if isFlags(fields[0]) {
		base.Predefined = strings.Contains(fields[0], "*")
		base.DLAttribute = strings.Contains(fields[0], "?")
		fields = fields[1:]
	}
	for _, field := range fields {
		var err error
		if field == "dlpkey" || strings.Contains(field, "=") {
			err = base.applyAttribute(field)
		} else {
			err = base.appendComponent(field)
		}
		if err != nil {
			return nil, err
		}
	}
	if len(base.Components) == 0 {
		return nil, fmt.Errorf("AI %s has no format specification", codes[0])
	}
	entries := make([]entry, 0, len(codes))
	for _, code := range codes {
		e := base
		e.AI = code
		entries = append(entries, e)
	}
	return entries, nil
}

// applyAttribute parses one key-value attribute or the solitary dlpkey key.
func (e *entry) applyAttribute(field string) error {
	switch {
	case field == "dlpkey":
		e.DLPKey = true
	case strings.HasPrefix(field, "dlpkey="):
		e.DLPKey = true
		e.DLQualifiers = strings.TrimPrefix(field, "dlpkey=")
	case strings.HasPrefix(field, "req="):
		alternatives := strings.Split(strings.TrimPrefix(field, "req="), ",")
		for _, alt := range alternatives {
			if err := validatePatternGroup(alt); err != nil {
				return fmt.Errorf("req=%s: %w", alt, err)
			}
		}
		e.Req = append(e.Req, alternatives)
	case strings.HasPrefix(field, "ex="):
		for _, pattern := range strings.Split(strings.TrimPrefix(field, "ex="), ",") {
			if err := validatePattern(pattern); err != nil {
				return fmt.Errorf("ex=%s: %w", pattern, err)
			}
			e.Ex = append(e.Ex, pattern)
		}
	default:
		return fmt.Errorf("unknown attribute %q", field)
	}
	return nil
}

// appendComponent parses one format component and checks the dictionary's
// ordering rules: only the final component may vary in length and no
// mandatory component may follow an optional one.
func (e *entry) appendComponent(field string) error {
	c, err := parseComponent(field)
	if err != nil {
		return err
	}
	if len(e.Components) > 0 {
		prev := e.Components[len(e.Components)-1]
		if prev.Variable && !c.Optional {
			return fmt.Errorf("mandatory component %q follows a variable-length component", field)
		}
		if prev.Optional && !c.Optional {
			return fmt.Errorf("mandatory component %q follows an optional component", field)
		}
	}
	e.Components = append(e.Components, c)
	return nil
}

// isFlags reports whether a token is the optional flags column, which is
// allocated from punctuation only and never contains letters or digits.
func isFlags(token string) bool {
	for i := 0; i < len(token); i++ {
		c := token[i]
		if c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			return false
		}
	}
	return true
}

func expandAIs(token string) ([]string, error) {
	if aiRE.MatchString(token) {
		return []string{token}, nil
	}
	m := rangeRE.FindStringSubmatch(token)
	if m == nil || len(m[1]) != len(m[2]) {
		return nil, fmt.Errorf("invalid AI or range %q", token)
	}
	from, _ := strconv.Atoi(m[1])
	to, _ := strconv.Atoi(m[2])
	if to < from {
		return nil, fmt.Errorf("descending AI range %q", token)
	}
	codes := make([]string, 0, to-from+1)
	for n := from; n <= to; n++ {
		codes = append(codes, fmt.Sprintf("%0*d", len(m[1]), n))
	}
	return codes, nil
}

func parseComponent(token string) (component, error) {
	m := componentRE.FindStringSubmatch(token)
	if m == nil {
		return component{}, fmt.Errorf("invalid component %q", token)
	}
	if (m[1] == "") != (m[5] == "") {
		return component{}, fmt.Errorf("unbalanced brackets in component %q", token)
	}
	length, _ := strconv.Atoi(m[4])
	if length == 0 {
		return component{}, fmt.Errorf("zero length in component %q", token)
	}
	c := component{Type: m[2][0], Len: length, Variable: m[3] != "", Optional: m[1] != ""}
	if m[6] != "" {
		c.Linters = strings.Split(m[6], ",")
	}
	return c, nil
}

func validatePatternGroup(group string) error {
	for _, pattern := range strings.Split(group, "+") {
		if err := validatePattern(pattern); err != nil {
			return err
		}
	}
	return nil
}

func validatePattern(pattern string) error {
	if len(pattern) < 2 || len(pattern) > 4 {
		return fmt.Errorf("invalid AI pattern %q", pattern)
	}
	for i := 0; i < len(pattern); i++ {
		if c := pattern[i]; c != 'n' && (c < '0' || c > '9') {
			return fmt.Errorf("invalid AI pattern %q", pattern)
		}
	}
	return nil
}

// derived holds the summary lengths the parser uses on its hot path.
type derived struct {
	FixedLen, MinLen, MaxLen int
	Numeric                  bool
}

func derive(components []component) derived {
	d := derived{Numeric: true}
	fixed := true
	for _, c := range components {
		if c.Type != 'N' {
			d.Numeric = false
		}
		if c.Variable || c.Optional {
			fixed = false
		}
		d.MaxLen += c.Len
		if !c.Optional {
			if c.Variable {
				d.MinLen++
			} else {
				d.MinLen += c.Len
			}
		}
	}
	if fixed {
		d.FixedLen = d.MaxLen
	}
	return d
}

var typeNames = map[byte]string{'N': "dataNumeric", 'X': "dataAlphanumeric", 'Y': "dataCSET39", 'Z': "dataBase64URL"}

func generate(release string, entries []entry) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by internal/gen/aitable from the GS1 Barcode Syntax Dictionary release %s. DO NOT EDIT.\n\n", release)
	b.WriteString("package gs1\n\n")
	b.WriteString("// syntaxDictionaryRelease is the release of the GS1 Barcode Syntax Dictionary\n// that aiTable was generated from.\n")
	fmt.Fprintf(&b, "const syntaxDictionaryRelease = %q\n\n", release)
	b.WriteString("// aiTable maps every Application Identifier in the dictionary to its\n// specification. Keys are 2 to 4 digits and prefix-free.\n")
	b.WriteString("var aiTable = map[string]*aiSpec{\n")
	for _, e := range entries {
		writeEntry(&b, e)
	}
	b.WriteString("}\n")
	return format.Source(b.Bytes())
}

func writeEntry(b *bytes.Buffer, e entry) {
	d := derive(e.Components)
	fmt.Fprintf(b, "\t%q: {AI: %q, Title: %q", e.AI, e.AI, e.Title)
	if e.Predefined {
		b.WriteString(", Predefined: true")
	}
	if e.DLAttribute {
		b.WriteString(", DLAttribute: true")
	}
	writeComponents(b, e.Components)
	writeAssociations(b, e)
	fmt.Fprintf(b, ", FixedLen: %d, MinLen: %d, MaxLen: %d", d.FixedLen, d.MinLen, d.MaxLen)
	if d.Numeric {
		b.WriteString(", DataType: dataNumeric")
	} else {
		b.WriteString(", DataType: dataAlphanumeric")
	}
	b.WriteString("},\n")
}

func writeComponents(b *bytes.Buffer, components []component) {
	b.WriteString(", Components: []component{")
	for i, c := range components {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "{Type: %s, Len: %d", typeNames[c.Type], c.Len)
		if c.Variable {
			b.WriteString(", Variable: true")
		}
		if c.Optional {
			b.WriteString(", Optional: true")
		}
		if len(c.Linters) > 0 {
			b.WriteString(", Linters: ")
			writeStrings(b, c.Linters)
		}
		b.WriteString("}")
	}
	b.WriteString("}")
}

func writeAssociations(b *bytes.Buffer, e entry) {
	if len(e.Req) > 0 {
		b.WriteString(", Req: [][]string{")
		for i, group := range e.Req {
			if i > 0 {
				b.WriteString(", ")
			}
			writeStrings(b, group)
		}
		b.WriteString("}")
	}
	if len(e.Ex) > 0 {
		b.WriteString(", Ex: ")
		writeStrings(b, e.Ex)
	}
	if e.DLPKey {
		b.WriteString(", DLPKey: true")
		if e.DLQualifiers != "" {
			fmt.Fprintf(b, ", DLQualifiers: %q", e.DLQualifiers)
		}
	}
}

func writeStrings(b *bytes.Buffer, values []string) {
	b.WriteString("[]string{")
	for i, v := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%q", v)
	}
	b.WriteString("}")
}
