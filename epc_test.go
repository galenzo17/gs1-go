package gs1

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestDecodeEPCReferenceVectors(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		want     EPC
		uri      string
		elements []Element
	}{
		{
			// GS1 EPC Tag Data Standard, SGTIN-96 worked example.
			name:     "SGTIN-96 TDS example",
			hex:      "3074257BF7194E4000001A85",
			want:     EPC{Scheme: EPCSGTIN, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "812345", Serial: "6789"},
			uri:      "urn:epc:id:sgtin:0614141.812345.6789",
			elements: []Element{{AI: "01", Value: "80614141123458"}, {AI: "21", Value: "6789"}},
		},
		{
			name:     "SGTIN-96 lowercase hex with 0x prefix and filter 1",
			hex:      "0x3034257bf7194e4000001a85",
			want:     EPC{Scheme: EPCSGTIN, Bits: 96, Filter: 1, CompanyPrefix: "0614141", Reference: "812345", Serial: "6789"},
			uri:      "urn:epc:id:sgtin:0614141.812345.6789",
			elements: []Element{{AI: "01", Value: "80614141123458"}, {AI: "21", Value: "6789"}},
		},
		{
			// GS1 EPC Tag Data Standard, SSCC-96 worked example.
			name:     "SSCC-96 TDS example",
			hex:      "3174257BF4499602D2000000",
			want:     EPC{Scheme: EPCSSCC, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "1234567890"},
			uri:      "urn:epc:id:sscc:0614141.1234567890",
			elements: []Element{{AI: "00", Value: "106141412345678908"}},
		},
		{
			name:     "SSCC-96 partition 6",
			hex:      "3178E61C883950F59A000000",
			want:     EPC{Scheme: EPCSSCC, Bits: 96, Filter: 3, CompanyPrefix: "235634", Reference: "35321345434"},
			uri:      "urn:epc:id:sscc:235634.35321345434",
			elements: []Element{{AI: "00", Value: "323563453213454346"}},
		},
		{
			name:     "GRAI-170 with 16-character serial, zero padded to 176 bits",
			hex:      "377A6BB0C1BDA6D9B664D1AB266D1AB266D1AB266D00",
			want:     EPC{Scheme: EPCGRAI, Bits: 170, Filter: 3, CompanyPrefix: "634563", Reference: "456347", Serial: "3624523452345234"},
			uri:      "urn:epc:id:grai:634563.456347.3624523452345234",
			elements: []Element{{AI: "8003", Value: "063456345634723624523452345234"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeEPC(tt.hex)
			if err != nil {
				t.Fatalf("DecodeEPC() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("DecodeEPC() = %+v, want %+v", got, tt.want)
			}
			if got.URI() != tt.uri {
				t.Errorf("URI() = %q, want %q", got.URI(), tt.uri)
			}
			assertElements(t, got.Elements(), tt.elements)

			b, err := ParseEPC(tt.hex)
			if err != nil || b.Symbology != SymEPC || b.Raw != tt.hex {
				t.Fatalf("ParseEPC() = %+v, %v", b, err)
			}
			assertElements(t, b.Elements, tt.elements)
			if err := b.Validate(); err != nil {
				t.Errorf("Validate() of decoded elements = %v", err)
			}
			if _, err := ParseWithOptions(mustEncode(t, b.Elements), ParseOptions{ValidateAssociations: true, ValidateCheckDigits: true}); err != nil {
				t.Errorf("strict parse of encoded elements = %v", err)
			}
		})
	}
}

func assertElements(t *testing.T, got, want []Element) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("elements = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("element[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func mustEncode(t *testing.T, elements []Element) string {
	t.Helper()
	raw, err := Encode(elements)
	if err != nil {
		t.Fatalf("Encode(%+v) error = %v", elements, err)
	}
	return raw
}

func TestDecodeEPCErrors(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		wantErr error
	}{
		{name: "empty", hex: "", wantErr: ErrEmptyInput},
		{name: "odd length", hex: "307", wantErr: ErrInvalidData},
		{name: "non-hex", hex: "3G74257BF7194E4000001A85", wantErr: ErrInvalidData},
		{name: "SGLN-96 non-zero location reference with 12-digit prefix", hex: "32405D21DB9FF60000003039", wantErr: ErrInvalidData},
		{name: "unsupported header GID-96", hex: "350000000000000000000000", wantErr: ErrUnsupportedEPCScheme},
		{name: "unsupported header SGCN-96", hex: "3F0000000000000000000000", wantErr: ErrUnsupportedEPCScheme},
		{name: "too short", hex: "3074257BF7194E40", wantErr: ErrInvalidData},
		{name: "too long", hex: "3074257BF7194E4000001A850000", wantErr: ErrInvalidData},
		{name: "partition 7", hex: "307C257BF7194E4000001A85", wantErr: ErrInvalidData},
		{name: "company prefix exceeds digits", hex: "30FFFFFFFFFFFFFFFFFFFFFF", wantErr: ErrInvalidData},
		{name: "SSCC-96 non-zero padding", hex: "3174257BF4499602D2000001", wantErr: ErrInvalidData},
		{name: "GRAI-170 non-zero padding", hex: "377A6BB0C1BDA6D9B664D1AB266D1AB266D1AB266D01", wantErr: ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeEPC(tt.hex)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DecodeEPC(%q) error = %v, want %v", tt.hex, err, tt.wantErr)
			}
			if _, err := ParseEPC(tt.hex); !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseEPC(%q) error = %v, want %v", tt.hex, err, tt.wantErr)
			}
		})
	}
}

func TestDecodeEPCURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		want     EPC
		elements []Element
		wantErr  error
	}{
		{name: "sgtin-96 tag URI", uri: "urn:epc:tag:sgtin-96:3.0614141.812345.6789",
			want:     EPC{Scheme: EPCSGTIN, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "812345", Serial: "6789"},
			elements: []Element{{AI: "01", Value: "80614141123458"}, {AI: "21", Value: "6789"}}},
		{name: "sgtin pure identity", uri: "urn:epc:id:sgtin:0614141.812345.6789",
			want:     EPC{Scheme: EPCSGTIN, Bits: 0, Filter: -1, CompanyPrefix: "0614141", Reference: "812345", Serial: "6789"},
			elements: []Element{{AI: "01", Value: "80614141123458"}, {AI: "21", Value: "6789"}}},
		{name: "sgtin-198 escaped serial", uri: "urn:epc:tag:sgtin-198:3.0614141.812345.32a%2Fb",
			want:     EPC{Scheme: EPCSGTIN, Bits: 198, Filter: 3, CompanyPrefix: "0614141", Reference: "812345", Serial: "32a/b"},
			elements: []Element{{AI: "01", Value: "80614141123458"}, {AI: "21", Value: "32a/b"}}},
		{name: "sscc-96 tag URI", uri: "urn:epc:tag:sscc-96:3.0614141.1234567890",
			want:     EPC{Scheme: EPCSSCC, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "1234567890"},
			elements: []Element{{AI: "00", Value: "106141412345678908"}}},
		{name: "sgln-96 with extension", uri: "urn:epc:tag:sgln-96:3.0614141.12345.400",
			want:     EPC{Scheme: EPCSGLN, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "12345", Serial: "400"},
			elements: []Element{{AI: "414", Value: "0614141123452"}, {AI: "254", Value: "400"}}},
		{name: "sgln without extension", uri: "urn:epc:id:sgln:0614141.12345.0",
			want:     EPC{Scheme: EPCSGLN, Filter: -1, CompanyPrefix: "0614141", Reference: "12345", Serial: "0"},
			elements: []Element{{AI: "414", Value: "0614141123452"}}},
		{name: "sgln 12-digit company prefix", uri: "urn:epc:id:sgln:061414112345..0",
			want:     EPC{Scheme: EPCSGLN, Filter: -1, CompanyPrefix: "061414112345", Reference: "", Serial: "0"},
			elements: []Element{{AI: "414", Value: "0614141123452"}}},
		{name: "grai-96", uri: "urn:epc:tag:grai-96:3.0614141.12345.400",
			want:     EPC{Scheme: EPCGRAI, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "12345", Serial: "400"},
			elements: []Element{{AI: "8003", Value: "00614141123452400"}}},
		{name: "giai-96", uri: "urn:epc:tag:giai-96:3.0614141.12345400",
			want:     EPC{Scheme: EPCGIAI, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "12345400"},
			elements: []Element{{AI: "8004", Value: "061414112345400"}}},
		{name: "serial with dots", uri: "urn:epc:id:sgtin:0614141.812345.1.2.3",
			want:     EPC{Scheme: EPCSGTIN, Filter: -1, CompanyPrefix: "0614141", Reference: "812345", Serial: "1.2.3"},
			elements: []Element{{AI: "01", Value: "80614141123458"}, {AI: "21", Value: "1.2.3"}}},
		{name: "giai pure identity alphanumeric", uri: "urn:epc:id:giai:0614141.32a%2Fb",
			want:     EPC{Scheme: EPCGIAI, Filter: -1, CompanyPrefix: "0614141", Reference: "32a/b"},
			elements: []Element{{AI: "8004", Value: "061414132a/b"}}},
		{name: "not an EPC URI", uri: "https://example.com/01/80614141123458", wantErr: ErrInvalidData},
		{name: "unsupported tag scheme", uri: "urn:epc:tag:gid-96:0.1.2", wantErr: ErrUnsupportedEPCScheme},
		{name: "unsupported identity scheme", uri: "urn:epc:id:gdti:0614141.12345.ABC", wantErr: ErrUnsupportedEPCScheme},
		{name: "missing fields", uri: "urn:epc:tag:sgtin-96", wantErr: ErrInvalidData},
		{name: "bad filter", uri: "urn:epc:tag:sgtin-96:9.0614141.812345.6789", wantErr: ErrInvalidData},
		{name: "wrong field count", uri: "urn:epc:tag:sgtin-96:3.0614141.812345", wantErr: ErrInvalidData},
		{name: "digit total", uri: "urn:epc:id:sgtin:0614141.81234.6789", wantErr: ErrInvalidData},
		{name: "short company prefix", uri: "urn:epc:id:sscc:06141.12345678901", wantErr: ErrInvalidData},
		{name: "non-numeric reference", uri: "urn:epc:id:sgtin:0614141.8123A5.6789", wantErr: ErrInvalidData},
		{name: "empty serial", uri: "urn:epc:id:sgtin:0614141.812345.", wantErr: ErrInvalidData},
		{name: "bad escape", uri: "urn:epc:id:sgtin:0614141.812345.6%ZZ", wantErr: ErrInvalidData},
		{name: "sgtin-96 alphanumeric serial", uri: "urn:epc:tag:sgtin-96:3.0614141.812345.ABC", wantErr: ErrInvalidData},
		{name: "sgtin-96 serial with leading zero", uri: "urn:epc:tag:sgtin-96:3.0614141.812345.0123", wantErr: ErrInvalidData},
		{name: "sgtin-96 serial beyond 38 bits", uri: "urn:epc:tag:sgtin-96:3.0614141.812345.274877906944", wantErr: ErrInvalidData},
		{name: "sgtin-96 serial at 38-bit maximum", uri: "urn:epc:tag:sgtin-96:3.0614141.812345.274877906943",
			want:     EPC{Scheme: EPCSGTIN, Bits: 96, Filter: 3, CompanyPrefix: "0614141", Reference: "812345", Serial: "274877906943"},
			elements: []Element{{AI: "01", Value: "80614141123458"}, {AI: "21", Value: "274877906943"}}},
		{name: "sgtin-198 serial over 20 characters", uri: "urn:epc:tag:sgtin-198:3.0614141.812345.ABCDEFGHIJKLMNOPQRSTU", wantErr: ErrInvalidData},
		{name: "sgtin pure identity serial over 20 characters", uri: "urn:epc:id:sgtin:0614141.812345.ABCDEFGHIJKLMNOPQRSTU", wantErr: ErrInvalidData},
		{name: "sgln-96 alphanumeric extension", uri: "urn:epc:tag:sgln-96:3.0614141.12345.4a", wantErr: ErrInvalidData},
		{name: "grai serial over 16 characters", uri: "urn:epc:id:grai:0614141.12345.ABCDEFGHIJKLMNOPQ", wantErr: ErrInvalidData},
		{name: "grai-96 serial beyond 38 bits", uri: "urn:epc:tag:grai-96:3.0614141.12345.274877906944", wantErr: ErrInvalidData},
		{name: "giai-96 alphanumeric reference", uri: "urn:epc:tag:giai-96:3.0614141.32a%2Fb", wantErr: ErrInvalidData},
		{name: "giai-96 reference with leading zero", uri: "urn:epc:tag:giai-96:3.0614141.0123", wantErr: ErrInvalidData},
		{name: "giai-96 reference beyond 58 bits", uri: "urn:epc:tag:giai-96:3.0614141.288230376151711744", wantErr: ErrInvalidData},
		{name: "giai over 30 characters", uri: "urn:epc:id:giai:0614141.ABCDEFGHIJKLMNOPQRSTUVWX", wantErr: ErrInvalidData},
		{name: "giai-202 reference beyond 30 characters total", uri: "urn:epc:tag:giai-202:3.061414112345.ABCDEFGHIJKLMNOPQRS", wantErr: ErrInvalidData},
		{name: "control character in serial", uri: "urn:epc:id:giai:000000.000%09", wantErr: ErrInvalidData},
		{name: "space in serial", uri: "urn:epc:id:sgtin:0614141.812345.6%2089", wantErr: ErrInvalidData},
		{name: "hash in serial", uri: "urn:epc:id:sgtin:0614141.812345.6#9", wantErr: ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeEPCURI(tt.uri)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DecodeEPCURI() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				if _, err := ParseEPCURI(tt.uri); !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseEPCURI() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if got != tt.want {
				t.Errorf("DecodeEPCURI() = %+v, want %+v", got, tt.want)
			}
			assertElements(t, got.Elements(), tt.elements)
			b, err := ParseEPCURI(tt.uri)
			if err != nil || b.Symbology != SymEPC || b.Raw != tt.uri {
				t.Fatalf("ParseEPCURI() = %+v, %v", b, err)
			}
			if err := b.Validate(); err != nil {
				t.Errorf("Validate() = %v", err)
			}
			if again, err := DecodeEPCURI(got.URI()); err != nil || again.CompanyPrefix != got.CompanyPrefix || again.Reference != got.Reference || again.Serial != got.Serial {
				t.Errorf("DecodeEPCURI(URI()) = %+v, %v, want the same identifier", again, err)
			}
		})
	}
}

func TestDecodeEPCRejectsOverlongGIAI202(t *testing.T) {
	// Partition 0 leaves 148 bits (21 characters) for the reference, but the
	// GIAI is limited to 30 characters including the 12-digit prefix.
	coding, _ := epcCodingByName("giai-202")
	hexTag := encodeTestEPC(t, coding, 3, 0, "061414112345", "ABCDEFGHIJKLMNOPQRS", "")
	if _, err := DecodeEPC(hexTag); !errors.Is(err, ErrInvalidData) {
		t.Errorf("DecodeEPC(%s) error = %v, want ErrInvalidData", hexTag, err)
	}
	hexTag = encodeTestEPC(t, coding, 3, 0, "061414112345", "ABCDEFGHIJKLMNOPQR", "")
	if got, err := DecodeEPC(hexTag); err != nil || got.Reference != "ABCDEFGHIJKLMNOPQR" {
		t.Errorf("DecodeEPC(%s) = %+v, %v, want an 18-character reference", hexTag, got, err)
	}
}

func TestEPCURIEscapesSerial(t *testing.T) {
	e := EPC{Scheme: EPCSGTIN, CompanyPrefix: "0614141", Reference: "812345", Serial: `a/b"c%d&e<f>g?h`}
	want := "urn:epc:id:sgtin:0614141.812345.a%2Fb%22c%25d%26e%3Cf%3Eg%3Fh"
	if got := e.URI(); got != want {
		t.Errorf("URI() = %q, want %q", got, want)
	}
	back, err := DecodeEPCURI(want)
	if err != nil || back.Serial != e.Serial {
		t.Errorf("DecodeEPCURI(URI()) serial = %q, %v", back.Serial, err)
	}
}

// bitWriter is a test-only encoder used to exercise every partition of
// every coding scheme independently of the reference vectors.
type bitWriter struct {
	bits []byte
}

func (w *bitWriter) write(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits = append(w.bits, byte(v>>uint(i)&1))
	}
}

func (w *bitWriter) writeSevenBit(s string, n int) {
	for i := 0; i < len(s); i++ {
		w.write(uint64(s[i]), 7)
	}
	w.write(0, n-7*len(s))
}

func (w *bitWriter) hex() string {
	for len(w.bits)%16 != 0 {
		w.bits = append(w.bits, 0)
	}
	var b strings.Builder
	for i := 0; i < len(w.bits); i += 4 {
		n := w.bits[i]<<3 | w.bits[i+1]<<2 | w.bits[i+2]<<1 | w.bits[i+3]
		b.WriteByte("0123456789ABCDEF"[n])
	}
	return b.String()
}

func encodeTestEPC(t *testing.T, coding epcCoding, filter, partition int, company, reference, serial string) string {
	t.Helper()
	var w bitWriter
	w.write(uint64(coding.header), 8)
	w.write(uint64(filter), 3)
	w.write(uint64(partition), 3)
	layout := coding.partitions[partition]
	cp, _ := strconv.ParseUint(company, 10, 64)
	w.write(cp, layout.companyBits)
	if coding.alphanumeric && coding.scheme == EPCGIAI {
		w.writeSevenBit(reference, layout.refBits)
	} else {
		ref, _ := strconv.ParseUint(reference, 10, 64)
		w.write(ref, layout.refBits)
	}
	if coding.serialBits > 0 {
		if coding.alphanumeric {
			w.writeSevenBit(serial, coding.serialBits)
		} else {
			s, _ := strconv.ParseUint(serial, 10, 64)
			w.write(s, coding.serialBits)
		}
	}
	if coding.scheme == EPCSSCC {
		w.write(0, 24)
	}
	if len(w.bits) != coding.bits {
		t.Fatalf("%s: encoded %d bits, want %d", coding.name, len(w.bits), coding.bits)
	}
	return w.hex()
}

func TestDecodeEPCEveryPartition(t *testing.T) {
	for _, coding := range epcCodings {
		for partition := 0; partition <= 6; partition++ {
			company := strings.Repeat("9", companyDigits(partition)-1) + "7"
			company = "0" + company[1:] // keep a realistic leading zero
			var reference, serial string
			switch coding.scheme {
			case EPCSGTIN, EPCSSCC, EPCSGLN, EPCGRAI:
				n := coding.refDigits(partition)
				reference = strings.Repeat("3", n)
				if n > 0 {
					reference = "1" + reference[1:]
				}
			case EPCGIAI:
				if coding.alphanumeric {
					reference = "A-1"
				} else {
					reference = "42"
				}
			}
			if coding.serialBits > 0 {
				if coding.alphanumeric {
					serial = "x!Z9"
				} else {
					serial = "12345"
				}
			}
			t.Run(coding.name+"/"+strconv.Itoa(partition), func(t *testing.T) {
				hexTag := encodeTestEPC(t, coding, 2, partition, company, reference, serial)
				got, err := DecodeEPC(hexTag)
				if err != nil {
					t.Fatalf("DecodeEPC(%s) error = %v", hexTag, err)
				}
				want := EPC{Scheme: coding.scheme, Bits: coding.bits, Filter: 2, CompanyPrefix: company, Reference: reference, Serial: serial}
				if got != want {
					t.Errorf("DecodeEPC(%s) = %+v, want %+v", hexTag, got, want)
				}
				b, err := ParseEPC(hexTag)
				if err != nil {
					t.Fatal(err)
				}
				if err := b.Validate(); err != nil {
					t.Errorf("Validate() = %v", err)
				}
				tagURI := "urn:epc:tag:" + coding.name + ":2." + strings.TrimPrefix(got.URI(), "urn:epc:id:"+coding.scheme.String()+":")
				fromURI, err := DecodeEPCURI(tagURI)
				if err != nil || fromURI != got {
					t.Errorf("DecodeEPCURI(%s) = %+v, %v, want %+v", tagURI, fromURI, err, got)
				}
			})
		}
	}
}

func TestEPCSchemeString(t *testing.T) {
	for s, want := range map[EPCScheme]string{EPCUnknown: "unknown", EPCSGTIN: "sgtin", EPCSSCC: "sscc", EPCSGLN: "sgln", EPCGRAI: "grai", EPCGIAI: "giai", EPCScheme(99): "unknown"} {
		if got := s.String(); got != want {
			t.Errorf("EPCScheme(%d).String() = %q, want %q", s, got, want)
		}
	}
	if (EPC{Scheme: EPCUnknown}).Elements() != nil {
		t.Error("Elements() of an unknown scheme should be nil")
	}
}

func FuzzDecodeEPC(f *testing.F) {
	f.Add("3074257BF7194E4000001A85")
	f.Add("3174257BF4499602D2000000")
	f.Add("377A6BB0C1BDA6D9B664D1AB266D1AB266D1AB266D00")
	f.Add("32")
	f.Add("")
	f.Add("urn:epc:tag:sgtin-96:3.0614141.812345.6789")
	f.Add("urn:epc:id:giai:0614141.32a%2Fb")
	// Found by fuzzing: a control character and an escaped dot in a serial.
	f.Add("urn:epc:id:giai:000000.000%09")
	f.Add("urn:epc:id:giai:000000.%2E")
	f.Fuzz(func(t *testing.T, input string) {
		if b, err := ParseEPC(input); err == nil {
			if len(b.Elements) == 0 || b.Symbology != SymEPC {
				t.Fatalf("ParseEPC(%q) = %+v without error", input, b)
			}
			if err := b.Validate(); err != nil {
				t.Fatalf("ParseEPC(%q) produced invalid elements: %v", input, err)
			}
		}
		if e, err := DecodeEPCURI(input); err == nil {
			if again, err := DecodeEPCURI(e.URI()); err != nil || again.Elements()[0] != e.Elements()[0] {
				t.Fatalf("DecodeEPCURI(%q).URI() does not round trip: %v", input, err)
			}
		}
	})
}
