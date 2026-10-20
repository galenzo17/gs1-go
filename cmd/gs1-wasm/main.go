//go:build js && wasm

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/galenzo17/gs1-go"
)

type elementJSON struct {
	AI      string       `json:"ai"`
	Name    string       `json:"name,omitempty"`
	Value   string       `json:"value"`
	Measure *measureJSON `json:"measure,omitempty"`
	Date    string       `json:"date,omitempty"`
}

// measureJSON mirrors the CLI JSON measure object so both entry points
// expose implied-decimal weights the same way.
type measureJSON struct {
	Raw      string  `json:"raw"`
	Value    float64 `json:"value"`
	Scaled   int64   `json:"scaled"`
	Decimals int     `json:"decimals"`
	Unit     string  `json:"unit"`
}

type parseResultJSON struct {
	Raw               string        `json:"raw"`
	Elements          []elementJSON `json:"elements"`
	EPC               *epcJSON      `json:"epc,omitempty"`
	GTIN              string        `json:"gtin"`
	Lot               string        `json:"lot"`
	Serial            string        `json:"serial"`
	ContentGTIN       string        `json:"contentGtin,omitempty"`
	CountOfTradeItems string        `json:"countOfTradeItems,omitempty"`
	GLN               string        `json:"gln,omitempty"`
	GSIN              string        `json:"gsin,omitempty"`
	PackagingDate     string        `json:"packagingDate,omitempty"`
	ExpirationDate    string        `json:"expirationDate,omitempty"`
	ProductionDate    string        `json:"productionDate,omitempty"`
	BestBeforeDate    string        `json:"bestBeforeDate,omitempty"`
	Symbology         string        `json:"symbology"`
	Warnings          []gs1.Warning `json:"warnings,omitempty"`
}

// epcJSON is the decoded EPC identifier attached to parseEPC results.
type epcJSON struct {
	Scheme        string `json:"scheme"`
	Bits          int    `json:"bits,omitempty"`
	Filter        *int   `json:"filter,omitempty"`
	CompanyPrefix string `json:"companyPrefix"`
	Reference     string `json:"reference"`
	Serial        string `json:"serial,omitempty"`
	URI           string `json:"uri"`
}

// dateAIs are the AI codes that contain YYMMDD dates.
var dateAIs = map[string]string{
	"17": "expirationDate",
	"11": "productionDate",
	"15": "bestBeforeDate",
	"13": "packagingDate",
}

func parse(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("parse requires 1 argument")
	}
	input := args[0].String()

	// Parse optional date formatting and independent semantic checks.
	dateFormat := "raw"
	dayZero := gs1.DayZeroLastDay
	var parseOpts gs1.ParseOptions
	if len(args) >= 2 && args[1].Type() == js.TypeObject {
		opts := args[1]
		if df := opts.Get("dateFormat"); df.Type() == js.TypeString {
			dateFormat = df.String()
		}
		if dz := opts.Get("dayZero"); dz.Type() == js.TypeString {
			if dz.String() == "first" {
				dayZero = gs1.DayZeroFirstDay
			}
		}
		if s := opts.Get("strict"); s.Type() == js.TypeBoolean {
			parseOpts.ValidateAssociations = s.Bool()
		}
		if c := opts.Get("validateCheckDigits"); c.Type() == js.TypeBoolean {
			parseOpts.ValidateCheckDigits = c.Bool()
		}
		if g := opts.Get("assumeBareGTIN8"); g.Type() == js.TypeBoolean {
			parseOpts.AssumeBareGTIN8 = g.Bool()
		}
	}

	b, err := gs1.ParseWithOptions(input, parseOpts)
	if err != nil {
		return errorResult(err.Error())
	}

	dateOpts := gs1.DateOptions{DayZero: dayZero}
	result := parseResultJSON{
		Raw:               b.Raw,
		Elements:          elementsJSON(b.Elements, dateFormat == "iso", dateOpts),
		GTIN:              b.GTIN(),
		Lot:               b.Lot(),
		Serial:            b.SerialNumber(),
		ContentGTIN:       b.ContentGTIN(),
		CountOfTradeItems: b.CountOfTradeItems(),
		GLN:               b.GLN(),
		GSIN:              b.GSIN(),
		PackagingDate:     "",
		Symbology:         b.Symbology.String(),
		Warnings:          b.Warnings(),
	}
	// Resolve date fields based on options.
	for ai, field := range dateAIs {
		v, ok := b.Get(ai)
		if !ok {
			continue
		}
		var dateStr string
		if dateFormat == "iso" {
			t, err := gs1.ParseDateWithOptions(v, dateOpts)
			if err == nil {
				dateStr = t.Format("2006-01-02")
			}
		} else {
			dateStr = v
		}
		switch field {
		case "expirationDate":
			result.ExpirationDate = dateStr
		case "productionDate":
			result.ProductionDate = dateStr
		case "bestBeforeDate":
			result.BestBeforeDate = dateStr
		case "packagingDate":
			result.PackagingDate = dateStr
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return errorResult(err.Error())
	}
	return js.Global().Get("JSON").Call("parse", string(data))
}

// parseEPC decodes an RFID tag (hexadecimal EPC memory content, tag URI or
// pure-identity URI) and returns the same result shape as parse with an
// extra epc object, or {error}.
func parseEPC(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("parseEPC requires 1 argument")
	}
	input := args[0].String()
	var epc gs1.EPC
	var err error
	if strings.HasPrefix(strings.TrimSpace(input), "urn:") {
		epc, err = gs1.DecodeEPCURI(input)
	} else {
		epc, err = gs1.DecodeEPC(input)
	}
	if err != nil {
		return errorResult(err.Error())
	}
	b := gs1.Barcode{Raw: input, Symbology: gs1.SymEPC, Elements: epc.Elements()}
	result := parseResultJSON{
		Raw:       b.Raw,
		Elements:  elementsJSON(b.Elements, false, gs1.DateOptions{}),
		GTIN:      b.GTIN(),
		Lot:       b.Lot(),
		Serial:    b.SerialNumber(),
		GLN:       b.GLN(),
		Symbology: b.Symbology.String(),
		EPC:       &epcJSON{Scheme: epc.Scheme.String(), Bits: epc.Bits, CompanyPrefix: epc.CompanyPrefix, Reference: epc.Reference, Serial: epc.Serial, URI: epc.URI()},
	}
	if epc.Filter >= 0 {
		filter := epc.Filter
		result.EPC.Filter = &filter
	}
	data, err := json.Marshal(result)
	if err != nil {
		return errorResult(err.Error())
	}
	return js.Global().Get("JSON").Call("parse", string(data))
}

// elementsJSON mirrors the CLI JSON element objects: AI name, decoded
// measure for four-digit measure AIs and, when iso is set, the ISO 8601 date
// of date AIs.
func elementsJSON(elements []gs1.Element, iso bool, dateOpts gs1.DateOptions) []elementJSON {
	out := make([]elementJSON, len(elements))
	for i, e := range elements {
		out[i] = elementJSON{AI: e.AI, Value: e.Value}
		if ai, ok := gs1.LookupAI(e.AI); ok {
			out[i].Name = ai.Name
		}
		if m, ok := measureForElement(e); ok {
			out[i].Measure = &measureJSON{Raw: m.Raw, Value: m.Value, Scaled: m.Scaled, Decimals: m.Decimals, Unit: m.Unit}
		}
		if iso && isDateAI(e.AI) {
			if t, err := gs1.ParseDateWithOptions(e.Value, dateOpts); err == nil {
				out[i].Date = t.Format("2006-01-02")
			}
		}
	}
	return out
}

// isDateAI reports whether the AI carries a YYMMDD date, the same set the
// CLI renders per element under -iso.
func isDateAI(ai string) bool {
	switch ai {
	case "11", "12", "13", "15", "17":
		return true
	}
	return false
}

// stringArgs returns the first len(names) arguments as strings, or an error
// naming the first argument that is missing or not a JavaScript string, so
// numbers and {error} objects are rejected instead of being stringified.
func stringArgs(fn string, args []js.Value, names ...string) ([]string, error) {
	if len(args) < len(names) {
		return nil, fmt.Errorf("%s requires %d arguments: %s", fn, len(names), strings.Join(names, ", "))
	}
	out := make([]string, len(names))
	for i, name := range names {
		if args[i].Type() != js.TypeString {
			return nil, fmt.Errorf("%s: %s must be a string", fn, name)
		}
		out[i] = args[i].String()
	}
	return out, nil
}

// measureForElement decodes the implied decimals of a four-digit measure AI
// through the same accessor the Go API exposes.
func measureForElement(e gs1.Element) (gs1.Measure, bool) {
	if len(e.AI) != 4 {
		return gs1.Measure{}, false
	}
	return gs1.Barcode{Elements: []gs1.Element{e}}.Measure(e.AI[:3])
}

// newSSCC builds an SSCC from an extension digit, a company prefix and a
// serial reference, returning the 18-digit string or {error}.
func newSSCC(_ js.Value, args []js.Value) any {
	in, err := stringArgs("newSSCC", args, "extensionDigit", "companyPrefix", "serialReference")
	if err != nil {
		return errorResult(err.Error())
	}
	if len(in[0]) != 1 {
		return errorResult("newSSCC: extension digit must be a single digit")
	}
	sscc, err := gs1.NewSSCC(in[0][0], in[1], in[2])
	if err != nil {
		return errorResult(err.Error())
	}
	return sscc
}

// newGTIN13 builds a GTIN-13 from a company prefix and an item reference.
func newGTIN13(_ js.Value, args []js.Value) any {
	in, err := stringArgs("newGTIN13", args, "companyPrefix", "itemReference")
	if err != nil {
		return errorResult(err.Error())
	}
	gtin, err := gs1.NewGTIN13(in[0], in[1])
	if err != nil {
		return errorResult(err.Error())
	}
	return gtin
}

// newGTIN14 builds a case-level GTIN-14 from an indicator digit and a
// GTIN-13.
func newGTIN14(_ js.Value, args []js.Value) any {
	in, err := stringArgs("newGTIN14", args, "indicator", "gtin13")
	if err != nil {
		return errorResult(err.Error())
	}
	if len(in[0]) != 1 {
		return errorResult("newGTIN14: indicator must be a single digit")
	}
	gtin, err := gs1.NewGTIN14(in[0][0], in[1])
	if err != nil {
		return errorResult(err.Error())
	}
	return gtin
}

// encode builds the element string for a barcode generator from an array of
// {ai, value} objects, from bracket notation (split literally by ParseHRI)
// or from a raw scan string, and returns its raw, HRI and ZPL forms, or
// {error} when the elements violate GS1 rules.
func encode(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("encode requires 1 argument: elements array or bracket notation string")
	}
	elements, err := elementsArgument(args[0])
	if err != nil {
		return errorResult(err.Error())
	}
	raw, err := gs1.Encode(elements)
	if err != nil {
		return errorResult(err.Error())
	}
	zpl, err := gs1.EncodeZPL(elements)
	if err != nil {
		return errorResult(err.Error())
	}
	ordered, err := gs1.Parse(raw)
	if err != nil {
		return errorResult(err.Error())
	}
	result := js.Global().Get("Object").New()
	result.Set("raw", raw)
	result.Set("hri", ordered.HRI())
	result.Set("zpl", zpl)
	return result
}

// elementsArgument converts the JavaScript encode argument into elements.
func elementsArgument(arg js.Value) ([]gs1.Element, error) {
	switch arg.Type() {
	case js.TypeString:
		input := arg.String()
		if strings.HasPrefix(input, "(") {
			return gs1.ParseHRI(input)
		}
		parsed, err := gs1.Parse(input)
		if err != nil {
			return nil, err
		}
		return parsed.Elements, nil
	case js.TypeObject:
		if !js.Global().Get("Array").Call("isArray", arg).Bool() {
			return nil, errors.New("encode expects an array of {ai, value} objects or a string")
		}
		n := arg.Length()
		elements := make([]gs1.Element, 0, n)
		for i := 0; i < n; i++ {
			item := arg.Index(i)
			if item.Type() != js.TypeObject {
				return nil, fmt.Errorf("encode element %d is not an object", i)
			}
			ai, value := item.Get("ai"), item.Get("value")
			if ai.Type() != js.TypeString || value.Type() != js.TypeString {
				return nil, fmt.Errorf("encode element %d needs string ai and value", i)
			}
			elements = append(elements, gs1.Element{AI: ai.String(), Value: value.String()})
		}
		return elements, nil
	default:
		return nil, errors.New("encode expects an array of {ai, value} objects or a string")
	}
}

func validateGTIN(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return false
	}
	return gs1.ValidateGTIN(args[0].String()) == nil
}

// validateKey validates any numeric GS1 key and reports the families its
// length admits, mirroring the gs1 key CLI command.
func validateKey(_ js.Value, args []js.Value) any {
	result := js.Global().Get("Object").New()
	if len(args) < 1 {
		result.Set("valid", false)
		result.Set("keyTypes", js.Global().Get("Array").New())
		result.Set("error", "validateKey requires 1 argument")
		return result
	}
	types, err := gs1.ValidateKey(args[0].String())
	names := js.Global().Get("Array").New()
	for _, t := range types {
		names.Call("push", t.String())
	}
	result.Set("valid", err == nil)
	result.Set("keyTypes", names)
	if err != nil {
		result.Set("error", err.Error())
	}
	return result
}

func validateRegulatory(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return "validateRegulatory requires 2 arguments: input, regulator"
	}
	input := args[0].String()
	regName := args[1].String()

	b, err := gs1.Parse(input)
	if err != nil {
		return err.Error()
	}

	regulators := map[string]gs1.Regulator{
		"anvisa":   gs1.ANVISA,
		"anmat":    gs1.ANMAT,
		"snfa":     gs1.SNFA,
		"cofepris": gs1.COFEPRIS,
	}

	reg, ok := regulators[regName]
	if !ok {
		return "unknown regulator: " + regName
	}

	if err := reg.Validate(b); err != nil {
		return err.Error()
	}
	return js.Null()
}

func errorResult(msg string) any {
	obj := js.Global().Get("Object").New()
	obj.Set("error", msg)
	return obj
}

func main() {
	ns := js.Global().Get("Object").New()
	ns.Set("parse", js.FuncOf(parse))
	ns.Set("parseEPC", js.FuncOf(parseEPC))
	ns.Set("encode", js.FuncOf(encode))
	ns.Set("validateGTIN", js.FuncOf(validateGTIN))
	ns.Set("validateKey", js.FuncOf(validateKey))
	ns.Set("validateRegulatory", js.FuncOf(validateRegulatory))
	ns.Set("newSSCC", js.FuncOf(newSSCC))
	ns.Set("newGTIN13", js.FuncOf(newGTIN13))
	ns.Set("newGTIN14", js.FuncOf(newGTIN14))
	js.Global().Set("gs1", ns)

	// Keep the Go runtime alive.
	<-make(chan struct{})
}
