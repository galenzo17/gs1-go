package gs1

//go:generate go run ./internal/gen/aitable

// dataType is the character set of one AI data component, in GS1 Syntax
// Dictionary notation.
type dataType int

const (
	dataNumeric      dataType = iota // N: digits 0-9 only
	dataAlphanumeric                 // X: GS1 AI encodable character set 82
	dataCSET39                       // Y: GS1 AI encodable character set 39
	dataBase64URL                    // Z: base64url alphabet
)

// component is one typed part of an AI data field. The dictionary splits
// fields such as "N3,iso3166 X..9" into components that are applied in
// order, each consuming a prefix of the data.
type component struct {
	Type     dataType
	Len      int      // exact length, or the maximum length when Variable
	Variable bool     // "N..20": 1 to Len characters; only the last component varies
	Optional bool     // "[N3]": applied only when data remains
	Linters  []string // GS1 Syntax Tests names such as csum, yymmdd, iso3166
}

// aiSpec describes one Application Identifier as generated from the GS1
// Barcode Syntax Dictionary. FixedLen, MinLen, MaxLen and DataType summarize
// the components for the parser's hot path.
type aiSpec struct {
	AI           string      // AI code, e.g. "01", "10", "3100"
	Title        string      // data title from the dictionary, e.g. "BATCH/LOT"
	Predefined   bool        // predefined length (table 7.8.5-2): no FNC1 separator needed
	DLAttribute  bool        // permitted as a GS1 Digital Link data attribute
	Components   []component // format specification, in order
	Req          [][]string  // mandatory associations: each group lists alternatives, "01+21" means both
	Ex           []string    // invalid pairings, with "n" as a digit wildcard
	DLPKey       bool        // GS1 Digital Link primary key
	DLQualifiers string      // ordered qualifier AIs for a primary key, "|" separates alternatives
	FixedLen     int         // total length when every component is fixed and mandatory; 0 otherwise
	MinLen       int         // minimum data length
	MaxLen       int         // maximum data length
	DataType     dataType    // dataNumeric when every component is numeric; dataAlphanumeric otherwise
}

// aiNames holds data titles for the AIs this package documented before the
// table was generated, so CLI and JSON output keep their established names.
// Every other AI reports the dictionary title.
var aiNames = map[string]string{
	"00": "SSCC", "01": "GTIN", "02": "Content GTIN", "03": "Made-to-Order GTIN",
	"10": "Batch/Lot", "11": "Production Date", "12": "Due Date", "13": "Packaging Date",
	"15": "Best Before Date", "16": "Sell By Date", "17": "Expiration Date",
	"20": "Variant", "21": "Serial Number", "22": "Consumer Product Variant",
	"30": "Count", "37": "Count of Trade Items",
	"240": "Additional Product ID", "241": "Customer Part Number", "242": "Made-to-Order Variant",
	"243": "Packaging Component Number", "250": "Secondary Serial Number", "251": "Reference to Source",
	"253": "GDTI", "254": "GLN extension component", "255": "GCN",
	"400": "Order Number", "401": "GINC", "402": "GSIN", "403": "Routing Code",
	"410": "Ship to / Deliver to GLN", "411": "Bill to / Invoice to GLN", "412": "Purchased from GLN",
	"413": "Ship for / Deliver for GLN", "414": "GLN", "415": "Invoicing party GLN",
	"416": "Production / service location GLN", "417": "Party GLN",
	"7040": "GS1 UIC with extension",
	"710":  "NHRN Germany", "711": "NHRN France", "712": "NHRN Spain", "713": "NHRN Brazil",
	"714": "NHRN Portugal", "715": "NHRN United States", "716": "NHRN Italy", "717": "NHRN Serbia",
	"8003": "GRAI", "8004": "GIAI", "8006": "ITIP", "8010": "CPID", "8011": "CPID Serial",
	"8013": "GMN", "8017": "GSRN – Provider", "8018": "GSRN – Recipient", "8026": "ITIP Content",
	"8110": "Coupon Code (North America)", "8112": "Paperless Coupon Code (North America)",
	"8200": "Product URL",
}

func init() {
	for prefix, name := range map[string]string{
		"310": "Net Weight kg", "320": "Net Weight lb", "330": "Gross Weight kg", "340": "Gross Weight lb",
	} {
		for d := '0'; d <= '5'; d++ {
			aiNames[prefix+string(d)] = name
		}
	}
	for ai := 90; ai <= 99; ai++ {
		aiNames[itoa(ai)] = "Internal"
	}
}

// name returns the established data title for the AI, or the dictionary
// title when this package has none.
func (s *aiSpec) name() string {
	if name, ok := aiNames[s.AI]; ok {
		return name
	}
	if s.Title != "" {
		return s.Title
	}
	return "AI " + s.AI
}

// lookupAIAt finds the AI spec at the given position in the input string.
// GS1 AIs are prefix-free, so the first match among 2-, 3- and 4-digit keys
// is the only match. Returns the spec and the number of characters consumed
// for the AI code, or false if no match was found.
func lookupAIAt(input string, pos int) (*aiSpec, int, bool) {
	remaining := len(input) - pos

	if remaining >= 2 {
		if spec, ok := aiTable[input[pos:pos+2]]; ok {
			return spec, 2, true
		}
	}
	if remaining >= 3 {
		if spec, ok := aiTable[input[pos:pos+3]]; ok {
			return spec, 3, true
		}
	}
	if remaining >= 4 {
		if spec, ok := aiTable[input[pos:pos+4]]; ok {
			return spec, 4, true
		}
	}

	return nil, 0, false
}

// AI describes a GS1 Application Identifier as listed in the GS1 Barcode
// Syntax Dictionary. Format uses the dictionary notation: "N14" is exactly
// 14 digits, "X..20" is up to 20 characters from character set 82, components
// are separated by spaces and optional components are bracketed, as in
// "N13 [X..17]".
type AI struct {
	Code       string // AI code, e.g. "01", "10", "3102"
	Name       string // data title, e.g. "GTIN", "Batch/Lot"
	Title      string // data title as printed in the dictionary, e.g. "BATCH/LOT"
	Format     string // format specifier, e.g. "N14", "X..20", "N3 X..9"
	Predefined bool   // predefined length: no FNC1 separator is needed after the data
}

// LookupAI returns the specification for an Application Identifier code
// such as "01" or "3102". The boolean is false for codes the dictionary
// does not define.
func LookupAI(code string) (AI, bool) {
	spec, ok := aiTable[code]
	if !ok {
		return AI{}, false
	}
	return AI{Code: spec.AI, Name: spec.name(), Title: spec.Title, Format: spec.format(), Predefined: spec.Predefined}, true
}

// SyntaxDictionaryRelease returns the release date of the GS1 Barcode Syntax
// Dictionary that the AI table was generated from, e.g. "2026-01-27".
func SyntaxDictionaryRelease() string {
	return syntaxDictionaryRelease
}

// format renders the components in GS1 Syntax Dictionary notation without
// linter names.
func (s *aiSpec) format() string {
	out := make([]byte, 0, 8*len(s.Components))
	for i, c := range s.Components {
		if i > 0 {
			out = append(out, ' ')
		}
		if c.Optional {
			out = append(out, '[')
		}
		out = append(out, c.Type.letter())
		if c.Variable {
			out = append(out, '.', '.')
		}
		out = append(out, itoa(c.Len)...)
		if c.Optional {
			out = append(out, ']')
		}
	}
	return string(out)
}

func (t dataType) letter() byte {
	switch t {
	case dataNumeric:
		return 'N'
	case dataCSET39:
		return 'Y'
	case dataBase64URL:
		return 'Z'
	}
	return 'X'
}

// itoa converts a small non-negative int to decimal without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [4]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
