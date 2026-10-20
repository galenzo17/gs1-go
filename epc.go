package gs1

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// EPCScheme is a family of EPC Tag Data Standard identifiers that map to a
// GS1 key.
type EPCScheme uint8

const (
	// EPCUnknown is the zero value.
	EPCUnknown EPCScheme = iota
	// EPCSGTIN is a serialized GTIN: AI (01) with AI (21).
	EPCSGTIN
	// EPCSSCC is a Serial Shipping Container Code: AI (00).
	EPCSSCC
	// EPCSGLN is a GLN with an optional extension: AI (414) with AI (254).
	EPCSGLN
	// EPCGRAI is a Global Returnable Asset Identifier: AI (8003).
	EPCGRAI
	// EPCGIAI is a Global Individual Asset Identifier: AI (8004).
	EPCGIAI
)

// String returns the scheme name used in EPC URIs, e.g. "sgtin".
func (s EPCScheme) String() string {
	switch s {
	case EPCSGTIN:
		return "sgtin"
	case EPCSSCC:
		return "sscc"
	case EPCSGLN:
		return "sgln"
	case EPCGRAI:
		return "grai"
	case EPCGIAI:
		return "giai"
	}
	return "unknown"
}

// EPC is a decoded EPC Tag Data Standard identifier. The fields follow the
// TDS naming: CompanyPrefix is the GS1 Company Prefix, Reference is the
// scheme's reference (item reference with its indicator digit, serial
// reference with its extension digit, location reference, asset type, or
// individual asset reference) and Serial is the serial component of SGTIN,
// the extension of SGLN, or the serial of GRAI.
type EPC struct {
	Scheme EPCScheme
	// Bits is the tag length of the binary coding scheme (96, 198, 195, 170
	// or 202), or 0 when decoded from a pure-identity URI.
	Bits int
	// Filter is the filter value of a tag encoding, or -1 for a
	// pure-identity URI.
	Filter        int
	CompanyPrefix string
	Reference     string
	Serial        string
}

// URI returns the pure-identity EPC URI, e.g. "urn:epc:id:sgtin:0614141.812345.6789".
func (e EPC) URI() string {
	parts := []string{e.CompanyPrefix}
	switch e.Scheme {
	case EPCSGTIN, EPCGRAI, EPCSGLN:
		parts = append(parts, e.Reference, urnEscape(e.Serial))
	case EPCSSCC:
		parts = append(parts, e.Reference)
	case EPCGIAI:
		parts = append(parts, urnEscape(e.Reference))
	}
	return "urn:epc:id:" + e.Scheme.String() + ":" + strings.Join(parts, ".")
}

// Elements returns the GS1 element string equivalent of the identifier,
// with check digits computed: (01)+(21) for SGTIN, (00) for SSCC,
// (414)+(254) for SGLN, (8003) for GRAI and (8004) for GIAI, so RFID reads
// land in the same shape as barcode scans.
func (e EPC) Elements() []Element {
	switch e.Scheme {
	case EPCSGTIN:
		gtin := e.Reference[:1] + e.CompanyPrefix + e.Reference[1:]
		check, _ := computeCheckDigit(gtin)
		return []Element{{AI: "01", Value: gtin + string(check)}, {AI: "21", Value: e.Serial}}
	case EPCSSCC:
		sscc := e.Reference[:1] + e.CompanyPrefix + e.Reference[1:]
		check, _ := computeCheckDigit(sscc)
		return []Element{{AI: "00", Value: sscc + string(check)}}
	case EPCSGLN:
		gln := e.CompanyPrefix + e.Reference
		check, _ := computeCheckDigit(gln)
		elements := []Element{{AI: "414", Value: gln + string(check)}}
		if e.Serial != "0" {
			elements = append(elements, Element{AI: "254", Value: e.Serial})
		}
		return elements
	case EPCGRAI:
		key := e.CompanyPrefix + e.Reference
		check, _ := computeCheckDigit(key)
		return []Element{{AI: "8003", Value: "0" + key + string(check) + e.Serial}}
	case EPCGIAI:
		return []Element{{AI: "8004", Value: e.CompanyPrefix + e.Reference}}
	}
	return nil
}

// ParseEPC decodes the hexadecimal memory bank content of an RFID tag
// encoded with the EPC Tag Data Standard into a Barcode whose Elements are
// the equivalent GS1 element string. Supported coding schemes are SGTIN-96,
// SGTIN-198, SSCC-96, SGLN-96, SGLN-195, GRAI-96, GRAI-170, GIAI-96 and
// GIAI-202. Symbology is SymEPC and Raw is the input.
func ParseEPC(hexTag string) (Barcode, error) {
	epc, err := DecodeEPC(hexTag)
	if err != nil {
		return Barcode{}, err
	}
	return Barcode{Raw: hexTag, Symbology: SymEPC, Elements: epc.Elements()}, nil
}

// ParseEPCURI decodes an EPC tag URI ("urn:epc:tag:sgtin-96:3.0614141.812345.6789")
// or pure-identity URI ("urn:epc:id:sgtin:0614141.812345.6789") into a
// Barcode, like ParseEPC.
func ParseEPCURI(uri string) (Barcode, error) {
	epc, err := DecodeEPCURI(uri)
	if err != nil {
		return Barcode{}, err
	}
	return Barcode{Raw: uri, Symbology: SymEPC, Elements: epc.Elements()}, nil
}

// epcCoding describes the binary layout of one coding scheme per TDS
// section 14: header, fields and the partition table.
type epcCoding struct {
	scheme     EPCScheme
	name       string // tag URI scheme name, e.g. "sgtin-96"
	bits       int
	header     byte
	partitions [7]epcPartition
	// refDigits returns the digit count of the reference for a partition,
	// or 0 when the reference is a plain decimal (GIAI).
	refDigits func(p int) int
	// serialBits is the length of the trailing serial or extension field;
	// 0 when the scheme has none.
	serialBits int
	// alphanumeric selects 7-bit character encoding for the last field.
	alphanumeric bool
}

// epcPartition gives the bit lengths of the company prefix and the
// reference for one partition value (TDS tables 14-2 to 14-9).
type epcPartition struct {
	companyBits, refBits int
}

// companyDigits is the GS1 Company Prefix length for a partition value.
func companyDigits(p int) int { return 12 - p }

var (
	sgtinPartitions = [7]epcPartition{{40, 4}, {37, 7}, {34, 10}, {30, 14}, {27, 17}, {24, 20}, {20, 24}}
	ssccPartitions  = [7]epcPartition{{40, 18}, {37, 21}, {34, 24}, {30, 28}, {27, 31}, {24, 34}, {20, 38}}
	sglnPartitions  = [7]epcPartition{{40, 1}, {37, 4}, {34, 7}, {30, 11}, {27, 14}, {24, 17}, {20, 21}}
	graiPartitions  = [7]epcPartition{{40, 4}, {37, 7}, {34, 10}, {30, 14}, {27, 17}, {24, 20}, {20, 24}}
	giaiPartitions  = [7]epcPartition{{40, 42}, {37, 45}, {34, 48}, {30, 52}, {27, 55}, {24, 58}, {20, 62}}
	giai202Parts    = [7]epcPartition{{40, 148}, {37, 151}, {34, 154}, {30, 158}, {27, 161}, {24, 164}, {20, 168}}
)

var epcCodings = []epcCoding{
	{scheme: EPCSGTIN, name: "sgtin-96", bits: 96, header: 0x30, partitions: sgtinPartitions, refDigits: func(p int) int { return 1 + p }, serialBits: 38},
	{scheme: EPCSGTIN, name: "sgtin-198", bits: 198, header: 0x36, partitions: sgtinPartitions, refDigits: func(p int) int { return 1 + p }, serialBits: 140, alphanumeric: true},
	{scheme: EPCSSCC, name: "sscc-96", bits: 96, header: 0x31, partitions: ssccPartitions, refDigits: func(p int) int { return 5 + p }},
	{scheme: EPCSGLN, name: "sgln-96", bits: 96, header: 0x32, partitions: sglnPartitions, refDigits: func(p int) int { return p }, serialBits: 41},
	{scheme: EPCSGLN, name: "sgln-195", bits: 195, header: 0x39, partitions: sglnPartitions, refDigits: func(p int) int { return p }, serialBits: 140, alphanumeric: true},
	{scheme: EPCGRAI, name: "grai-96", bits: 96, header: 0x33, partitions: graiPartitions, refDigits: func(p int) int { return p }, serialBits: 38},
	{scheme: EPCGRAI, name: "grai-170", bits: 170, header: 0x37, partitions: graiPartitions, refDigits: func(p int) int { return p }, serialBits: 112, alphanumeric: true},
	{scheme: EPCGIAI, name: "giai-96", bits: 96, header: 0x34, partitions: giaiPartitions, refDigits: func(int) int { return 0 }},
	{scheme: EPCGIAI, name: "giai-202", bits: 202, header: 0x38, partitions: giai202Parts, refDigits: func(int) int { return 0 }, alphanumeric: true},
}

// DecodeEPC decodes the hexadecimal content of an EPC memory bank. The
// input may be padded with zero bits to a 16-bit word boundary, as tags
// store it; any other trailing bits are rejected.
func DecodeEPC(hexTag string) (EPC, error) {
	data, err := decodeHex(hexTag)
	if err != nil {
		return EPC{}, err
	}
	if len(data) == 0 {
		return EPC{}, ErrEmptyInput
	}
	coding, ok := epcCodingByHeader(data[0])
	if !ok {
		return EPC{}, fmt.Errorf("%w: header 0x%02X", ErrUnsupportedEPCScheme, data[0])
	}
	if len(data)*8 < coding.bits {
		return EPC{}, fmt.Errorf("%w: %s needs %d bits, got %d", ErrInvalidData, coding.name, coding.bits, len(data)*8)
	}
	if len(data)*8 > coding.bits+15 {
		return EPC{}, fmt.Errorf("%w: %s is %d bits, got %d", ErrInvalidData, coding.name, coding.bits, len(data)*8)
	}
	return decodeEPCBinary(coding, data)
}

// decodeEPCBinary reads the fields after the header per the coding
// scheme's partition table.
func decodeEPCBinary(coding epcCoding, data []byte) (EPC, error) {
	r := bitReader{data: data, pos: 8}
	filter := r.readSmall(3)
	partition := r.readSmall(3)
	if partition > 6 {
		return EPC{}, fmt.Errorf("%w: partition value %d", ErrInvalidData, partition)
	}
	layout := coding.partitions[partition]
	epc := EPC{Scheme: coding.scheme, Bits: coding.bits, Filter: filter}

	var err error
	if epc.CompanyPrefix, err = r.decimal(layout.companyBits, companyDigits(partition), "company prefix"); err != nil {
		return EPC{}, err
	}
	switch {
	case coding.scheme == EPCGIAI && coding.alphanumeric:
		epc.Reference, err = r.sevenBit(layout.refBits, "individual asset reference")
	case coding.scheme == EPCGIAI:
		epc.Reference = r.unpadded(layout.refBits)
	default:
		epc.Reference, err = r.decimal(layout.refBits, coding.refDigits(partition), "reference")
	}
	if err != nil {
		return EPC{}, err
	}
	switch {
	case coding.serialBits == 0:
	case coding.alphanumeric:
		epc.Serial, err = r.sevenBit(coding.serialBits, "serial")
	default:
		epc.Serial = r.unpadded(coding.serialBits)
	}
	if err != nil {
		return EPC{}, err
	}
	if !r.restIsZero() {
		return EPC{}, fmt.Errorf("%w: %s has non-zero bits after position %d", ErrInvalidData, coding.name, r.pos)
	}
	return epc, epc.checkAILengths()
}

// checkAILengths rejects identifiers whose element string would exceed the
// GS1 AI limits: 20 characters for the (21) serial and the (254) extension,
// 16 for the GRAI serial, 30 for the whole GIAI. A 12-digit company prefix
// in GIAI-202 leaves room for 21 encoded characters, but TDS allows 18.
func (e EPC) checkAILengths() error {
	switch e.Scheme {
	case EPCSGTIN, EPCSGLN:
		if len(e.Serial) > 20 {
			return fmt.Errorf("%w: %s serial %q exceeds 20 characters", ErrInvalidData, e.Scheme, e.Serial)
		}
	case EPCGRAI:
		if len(e.Serial) > 16 {
			return fmt.Errorf("%w: grai serial %q exceeds 16 characters", ErrInvalidData, e.Serial)
		}
	case EPCGIAI:
		if len(e.CompanyPrefix)+len(e.Reference) > 30 {
			return fmt.Errorf("%w: giai exceeds 30 characters", ErrInvalidData)
		}
	}
	return nil
}

// checkCoding verifies that a tag URI's values fit the binary coding
// scheme it names: numeric serials and the GIAI-96 reference must be
// decimal without leading zeros and fit their bit field for the partition
// implied by the company prefix length; alphanumeric fields are limited to
// the characters the field can hold.
func (c epcCoding) checkCoding(e EPC) error {
	partition := 12 - len(e.CompanyPrefix)
	layout := c.partitions[partition]
	switch {
	case c.scheme == EPCGIAI && c.alphanumeric:
		return checkSevenBitFits(e.Reference, layout.refBits, c.name, "individual asset reference")
	case c.scheme == EPCGIAI:
		return checkDecimalFits(e.Reference, layout.refBits, c.name, "individual asset reference")
	case c.serialBits == 0:
		return nil
	case c.alphanumeric:
		return checkSevenBitFits(e.Serial, c.serialBits, c.name, "serial")
	}
	return checkDecimalFits(e.Serial, c.serialBits, c.name, "serial")
}

func checkDecimalFits(value string, bits int, coding, field string) error {
	if !isDigits(value) || (len(value) > 1 && value[0] == '0') {
		return fmt.Errorf("%w: %s %s %q must be a decimal without leading zeros", ErrInvalidData, coding, field, value)
	}
	v, err := strconv.ParseUint(value, 10, 64)
	if err != nil || v>>bits != 0 {
		return fmt.Errorf("%w: %s %s %q does not fit in %d bits", ErrInvalidData, coding, field, value, bits)
	}
	return nil
}

func checkSevenBitFits(value string, bits int, coding, field string) error {
	if len(value) > bits/7 {
		return fmt.Errorf("%w: %s %s %q exceeds %d characters", ErrInvalidData, coding, field, value, bits/7)
	}
	return nil
}

func epcCodingByHeader(header byte) (epcCoding, bool) {
	for _, c := range epcCodings {
		if c.header == header {
			return c, true
		}
	}
	return epcCoding{}, false
}

func epcCodingByName(name string) (epcCoding, bool) {
	for _, c := range epcCodings {
		if c.name == name {
			return c, true
		}
	}
	return epcCoding{}, false
}

// decodeHex accepts upper- or lowercase hexadecimal, with or without a 0x
// prefix, and rejects odd lengths and other characters.
func decodeHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s = s[2:]
	}
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("%w: hexadecimal EPC has odd length %d", ErrInvalidData, len(s))
	}
	out, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%w: non-hexadecimal character in EPC", ErrInvalidData)
	}
	return out, nil
}

// bitReader reads big-endian bit fields from tag memory.
type bitReader struct {
	data []byte
	pos  int
}

// bit returns the next bit; bits beyond the data read as zero.
func (r *bitReader) bit() byte {
	var v byte
	if r.pos < len(r.data)*8 && r.data[r.pos/8]&(0x80>>(r.pos%8)) != 0 {
		v = 1
	}
	r.pos++
	return v
}

// read returns the next n bits (n <= 64) as an integer.
func (r *bitReader) read(n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = v<<1 | uint64(r.bit())
	}
	return v
}

// readSmall returns the next n bits (n <= 8) as an int.
func (r *bitReader) readSmall(n int) int {
	v := 0
	for i := 0; i < n; i++ {
		v = v<<1 | int(r.bit())
	}
	return v
}

// decimal reads an n-bit integer and renders it with exactly digits
// digits. A field with zero digits, such as the location reference of a
// 12-digit company prefix, must be zero and renders as "". Values that do
// not fit are invalid per TDS.
func (r *bitReader) decimal(bits, digits int, field string) (string, error) {
	v := r.read(bits)
	s := strconv.FormatUint(v, 10)
	if digits == 0 {
		if v != 0 {
			return "", fmt.Errorf("%w: %s must be zero for a 12-digit company prefix, got %d", ErrInvalidData, field, v)
		}
		return "", nil
	}
	if len(s) > digits {
		return "", fmt.Errorf("%w: %s value %d exceeds %d digits", ErrInvalidData, field, v, digits)
	}
	return strings.Repeat("0", digits-len(s)) + s, nil
}

// unpadded reads an n-bit integer as a decimal without leading zeros.
func (r *bitReader) unpadded(bits int) string {
	return strconv.FormatUint(r.read(bits), 10)
}

// sevenBit reads a string of 7-bit ISO 646 characters padded with zero
// bits on the right (TDS section 14.4.2), stopping at the first NUL.
func (r *bitReader) sevenBit(bits int, field string) (string, error) {
	var b strings.Builder
	ended := false
	for i := 0; i+7 <= bits; i += 7 {
		var c byte
		for j := 0; j < 7; j++ {
			c = c<<1 | r.bit()
		}
		switch {
		case c == 0:
			ended = true
		case ended:
			return "", fmt.Errorf("%w: %s has a character after the terminating zero", ErrInvalidData, field)
		case !epcSerialByte(c):
			return "", fmt.Errorf("%w: %s contains character 0x%02X outside GS1 character set 82", ErrInvalidData, field, c)
		default:
			b.WriteByte(c)
		}
	}
	r.pos += bits % 7
	if b.Len() == 0 {
		return "", fmt.Errorf("%w: %s is empty", ErrInvalidData, field)
	}
	return b.String(), nil
}

func (r *bitReader) restIsZero() bool {
	for r.pos < len(r.data)*8 {
		if r.bit() != 0 {
			return false
		}
	}
	return true
}

// DecodeEPCURI decodes an EPC tag URI ("urn:epc:tag:<scheme>-<bits>:<filter>.<fields>")
// or pure-identity URI ("urn:epc:id:<scheme>:<fields>"). Serial components
// may carry URN %XX escapes.
func DecodeEPCURI(uri string) (EPC, error) {
	uri = strings.TrimSpace(uri)
	if rest, ok := strings.CutPrefix(uri, "urn:epc:tag:"); ok {
		return decodeEPCTagURI(rest)
	}
	if rest, ok := strings.CutPrefix(uri, "urn:epc:id:"); ok {
		return decodeEPCIdentityURI(rest)
	}
	return EPC{}, fmt.Errorf("%w: EPC URI must start with the urn:epc:tag or urn:epc:id namespace", ErrInvalidData)
}

func decodeEPCTagURI(rest string) (EPC, error) {
	name, fields, ok := strings.Cut(rest, ":")
	if !ok {
		return EPC{}, fmt.Errorf("%w: tag URI needs a scheme and fields", ErrInvalidData)
	}
	coding, ok := epcCodingByName(name)
	if !ok {
		return EPC{}, fmt.Errorf("%w: %q", ErrUnsupportedEPCScheme, name)
	}
	filterText, fields, ok := strings.Cut(fields, ".")
	if !ok {
		return EPC{}, fmt.Errorf("%w: tag URI needs a filter value", ErrInvalidData)
	}
	if len(filterText) != 1 || filterText[0] < '0' || filterText[0] > '7' {
		return EPC{}, fmt.Errorf("%w: filter value %q", ErrInvalidData, filterText)
	}
	epc, err := decodeEPCFields(coding.scheme, fields)
	if err != nil {
		return EPC{}, err
	}
	if err := coding.checkCoding(epc); err != nil {
		return EPC{}, err
	}
	epc.Bits = coding.bits
	epc.Filter = int(filterText[0] - '0')
	return epc, nil
}

func decodeEPCIdentityURI(rest string) (EPC, error) {
	name, fields, ok := strings.Cut(rest, ":")
	if !ok {
		return EPC{}, fmt.Errorf("%w: pure-identity URI needs a scheme and fields", ErrInvalidData)
	}
	var scheme EPCScheme
	for s := EPCSGTIN; s <= EPCGIAI; s++ {
		if s.String() == name {
			scheme = s
		}
	}
	if scheme == EPCUnknown {
		return EPC{}, fmt.Errorf("%w: %q", ErrUnsupportedEPCScheme, name)
	}
	epc, err := decodeEPCFields(scheme, fields)
	if err != nil {
		return EPC{}, err
	}
	epc.Filter = -1
	return epc, nil
}

// decodeEPCFields splits the dot-separated identifier fields of a URI and
// checks the digit totals of TDS section 7: company prefix 6 to 12 digits,
// and company prefix plus reference of 13 digits for SGTIN, 17 for SSCC and
// 12 for SGLN and GRAI. Serial components are unescaped, may contain dots,
// and must stay in GS1 character set 82.
func decodeEPCFields(scheme EPCScheme, fields string) (EPC, error) {
	want := 3
	if scheme == EPCSSCC || scheme == EPCGIAI {
		want = 2
	}
	// Only the separators between fields split; a serial may itself
	// contain dots, which the TDT grammar allows.
	parts := strings.SplitN(fields, ".", want)
	if len(parts) != want {
		return EPC{}, fmt.Errorf("%w: %s URI needs %d fields, got %d", ErrInvalidData, scheme, want, len(parts))
	}
	epc := EPC{Scheme: scheme, CompanyPrefix: parts[0]}
	if len(epc.CompanyPrefix) < 6 || len(epc.CompanyPrefix) > 12 || !isDigits(epc.CompanyPrefix) {
		return EPC{}, fmt.Errorf("%w: company prefix %q must be 6 to 12 digits", ErrInvalidData, epc.CompanyPrefix)
	}
	var err error
	if scheme == EPCGIAI {
		if epc.Reference, err = decodeEPCSerial(parts[1], "individual asset reference"); err != nil {
			return EPC{}, err
		}
		return epc, epc.checkAILengths()
	}
	epc.Reference = parts[1]
	total := map[EPCScheme]int{EPCSGTIN: 13, EPCSSCC: 17, EPCSGLN: 12, EPCGRAI: 12}[scheme]
	// The reference is empty when a 12-digit company prefix leaves no room
	// for it (SGLN, GRAI).
	if (epc.Reference != "" && !isDigits(epc.Reference)) || len(epc.CompanyPrefix)+len(epc.Reference) != total {
		return EPC{}, fmt.Errorf("%w: %s company prefix and reference must total %d digits", ErrInvalidData, scheme, total)
	}
	if want == 3 {
		if epc.Serial, err = decodeEPCSerial(parts[2], "serial"); err != nil {
			return EPC{}, err
		}
	}
	return epc, epc.checkAILengths()
}

// decodeEPCSerial unescapes a serial or asset reference component and
// checks it against GS1 character set 82, the alphabet TDS allows.
func decodeEPCSerial(component, field string) (string, error) {
	value, err := urnUnescape(component)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%w: %s is empty", ErrInvalidData, field)
	}
	for i := 0; i < len(value); i++ {
		if !epcSerialByte(value[i]) {
			return "", fmt.Errorf("%w: %s contains character %q outside GS1 character set 82", ErrInvalidData, field, value[i])
		}
	}
	return value, nil
}

// epcSerialByte reports whether c belongs to GS1 AI encodable character set
// 82, the alphabet of EPC serial components (TDS section 6.3.1).
func epcSerialByte(c byte) bool {
	switch {
	case c >= '0' && c <= '9', c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
		return true
	}
	return strings.IndexByte("!\"%&'()*+,-./:;<=>?_", c) >= 0
}

// urnUnescape decodes %XX escapes of EPC URIs.
func urnUnescape(s string) (string, error) {
	if !strings.Contains(s, "%") {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			b.WriteByte(s[i])
			continue
		}
		if i+2 >= len(s) {
			return "", fmt.Errorf("%w: bad escape in %q", ErrInvalidData, s)
		}
		decoded, err := hex.DecodeString(s[i+1 : i+3])
		if err != nil {
			return "", fmt.Errorf("%w: bad escape in %q", ErrInvalidData, s)
		}
		b.Write(decoded)
		i += 2
	}
	return b.String(), nil
}

// urnEscape applies the TDS escapes for the characters that are not
// allowed verbatim in an EPC URI serial component.
func urnEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '%', '&', '/', '<', '>', '?':
			fmt.Fprintf(&b, "%%%02X", c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
