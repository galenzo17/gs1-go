//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/galenzo17/gs1-go"
)

type elementJSON struct {
	AI      string       `json:"ai"`
	Name    string       `json:"name,omitempty"`
	Value   string       `json:"value"`
	Measure *measureJSON `json:"measure,omitempty"`
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

	result := parseResultJSON{
		Raw:               b.Raw,
		Elements:          make([]elementJSON, len(b.Elements)),
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
	for i, e := range b.Elements {
		result.Elements[i] = elementJSON{AI: e.AI, Value: e.Value}
		if ai, ok := gs1.LookupAI(e.AI); ok {
			result.Elements[i].Name = ai.Name
		}
		if m, ok := measureForElement(e); ok {
			result.Elements[i].Measure = &measureJSON{Raw: m.Raw, Value: m.Value, Scaled: m.Scaled, Decimals: m.Decimals, Unit: m.Unit}
		}
	}

	// Resolve date fields based on options.
	dateOpts := gs1.DateOptions{DayZero: dayZero}
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
	if len(args) < 3 {
		return errorResult("newSSCC requires 3 arguments: extensionDigit, companyPrefix, serialReference")
	}
	extension := args[0].String()
	if len(extension) != 1 {
		return errorResult("newSSCC: extension digit must be a single digit")
	}
	sscc, err := gs1.NewSSCC(extension[0], args[1].String(), args[2].String())
	if err != nil {
		return errorResult(err.Error())
	}
	return sscc
}

// newGTIN13 builds a GTIN-13 from a company prefix and an item reference.
func newGTIN13(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return errorResult("newGTIN13 requires 2 arguments: companyPrefix, itemReference")
	}
	gtin, err := gs1.NewGTIN13(args[0].String(), args[1].String())
	if err != nil {
		return errorResult(err.Error())
	}
	return gtin
}

// newGTIN14 builds a case-level GTIN-14 from an indicator digit and a
// GTIN-13.
func newGTIN14(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return errorResult("newGTIN14 requires 2 arguments: indicator, gtin13")
	}
	indicator := args[0].String()
	if len(indicator) != 1 {
		return errorResult("newGTIN14: indicator must be a single digit")
	}
	gtin, err := gs1.NewGTIN14(indicator[0], args[1].String())
	if err != nil {
		return errorResult(err.Error())
	}
	return gtin
}

func validateGTIN(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return false
	}
	return gs1.ValidateGTIN(args[0].String()) == nil
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
	ns.Set("validateGTIN", js.FuncOf(validateGTIN))
	ns.Set("validateRegulatory", js.FuncOf(validateRegulatory))
	ns.Set("newSSCC", js.FuncOf(newSSCC))
	ns.Set("newGTIN13", js.FuncOf(newGTIN13))
	ns.Set("newGTIN14", js.FuncOf(newGTIN14))
	js.Global().Set("gs1", ns)

	// Keep the Go runtime alive.
	<-make(chan struct{})
}
