package gs1

import (
	"math"
	"strconv"
)

// Measure is a numeric GS1 measurement with its original representation.
// Scaled is the integer value before applying Decimals decimal places.
type Measure struct {
	Value    float64
	Raw      string
	Scaled   int64
	Decimals int
	Unit     string
}

// Measure returns the first measurement whose four-digit AI starts with the
// supplied three-digit prefix. Supported prefixes are 310 (kg), 320 (lb),
// 330 (kg), and 340 (lb).
func (b Barcode) Measure(prefix string) (Measure, bool) {
	unit, ok := measureUnit(prefix)
	if !ok {
		return Measure{}, false
	}
	for _, element := range b.Elements {
		if len(element.AI) != 4 || element.AI[:3] != prefix {
			continue
		}
		decimals := int(element.AI[3] - '0')
		if decimals > 5 {
			continue
		}
		scaled, err := strconv.ParseInt(element.Value, 10, 64)
		if err != nil {
			continue
		}
		value := float64(scaled) / math.Pow10(decimals)
		return Measure{Value: value, Raw: element.Value, Scaled: scaled, Decimals: decimals, Unit: unit}, true
	}
	return Measure{}, false
}

// NetWeightKg returns the first net-weight measurement in kilograms.
func (b Barcode) NetWeightKg() (Measure, bool) { return b.Measure("310") }

// NetWeightLb returns the first net-weight measurement in pounds.
func (b Barcode) NetWeightLb() (Measure, bool) { return b.Measure("320") }

// GrossWeightKg returns the first gross-weight measurement in kilograms.
func (b Barcode) GrossWeightKg() (Measure, bool) { return b.Measure("330") }

// GrossWeightLb returns the first gross-weight measurement in pounds.
func (b Barcode) GrossWeightLb() (Measure, bool) { return b.Measure("340") }

func measureUnit(prefix string) (string, bool) {
	switch prefix {
	case "310", "330":
		return "kg", true
	case "320", "340":
		return "lb", true
	default:
		return "", false
	}
}
