//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/galenzo17/gs1-go"
)

type elementJSON struct {
	AI    string `json:"ai"`
	Value string `json:"value"`
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

	// Parse options from second argument: { dateFormat?: "raw"|"iso", dayZero?: "last"|"first", strict?: boolean }
	dateFormat := "raw"
	dayZero := gs1.DayZeroLastDay
	strict := false
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
			strict = s.Bool()
		}
	}

	b, err := gs1.ParseWithOptions(input, gs1.ParseOptions{ValidateAssociations: strict})
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
	}
	for i, e := range b.Elements {
		result.Elements[i] = elementJSON{AI: e.AI, Value: e.Value}
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
	js.Global().Set("gs1", ns)

	// Keep the Go runtime alive.
	<-make(chan struct{})
}
