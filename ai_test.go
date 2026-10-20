package gs1

import "testing"

func TestAITableIntegrity(t *testing.T) {
	for key, spec := range aiTable {
		if key != spec.AI {
			t.Errorf("map key %q does not match spec.AI %q", key, spec.AI)
		}
		if spec.name() == "" {
			t.Errorf("AI %q has empty Name", key)
		}
		if spec.FixedLen > 0 && spec.FixedLen != spec.MaxLen {
			t.Errorf("AI %q: fixed-length AI has FixedLen=%d != MaxLen=%d", key, spec.FixedLen, spec.MaxLen)
		}
		if spec.FixedLen == 0 && spec.MaxLen == 0 {
			t.Errorf("AI %q: variable-length AI has MaxLen=0", key)
		}
	}
}

func TestLookupAIAt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pos     int
		wantAI  string
		wantLen int
		wantOK  bool
	}{
		{name: "2-digit GTIN", input: "0104150000021125", pos: 0, wantAI: "01", wantLen: 2, wantOK: true},
		{name: "2-digit lot", input: "10ABC123", pos: 0, wantAI: "10", wantLen: 2, wantOK: true},
		{name: "2-digit expiry", input: "17250630", pos: 0, wantAI: "17", wantLen: 2, wantOK: true},
		{name: "3-digit additional", input: "240PROD001", pos: 0, wantAI: "240", wantLen: 3, wantOK: true},
		{name: "3-digit NHRN", input: "713BR12345", pos: 0, wantAI: "713", wantLen: 3, wantOK: true},
		{name: "3-digit GDTI", input: "253123456789012DOC", pos: 0, wantAI: "253", wantLen: 3, wantOK: true},
		{name: "3-digit GINC", input: "401GINC123", pos: 0, wantAI: "401", wantLen: 3, wantOK: true},
		{name: "4-digit GRAI", input: "8003012345678901", pos: 0, wantAI: "8003", wantLen: 4, wantOK: true},
		{name: "4-digit GIAI", input: "8004ASSET123", pos: 0, wantAI: "8004", wantLen: 4, wantOK: true},
		{name: "4-digit GSRN provider", input: "801712345678901234", pos: 0, wantAI: "8017", wantLen: 4, wantOK: true},
		{name: "4-digit weight", input: "3102123456", pos: 0, wantAI: "3102", wantLen: 4, wantOK: true},
		{name: "at offset", input: "0104150000021125172506301", pos: 16, wantAI: "17", wantLen: 2, wantOK: true},
		{name: "3-digit GLN", input: "4141234567890123", pos: 0, wantAI: "414", wantLen: 3, wantOK: true},
		{name: "3-digit ship-to GLN", input: "4101234567890123", pos: 0, wantAI: "410", wantLen: 3, wantOK: true},
		{name: "3-digit party GLN", input: "4171234567890123", pos: 0, wantAI: "417", wantLen: 3, wantOK: true},
		{name: "3-digit GLN extension", input: "254EXTENSION", pos: 0, wantAI: "254", wantLen: 3, wantOK: true},
		{name: "4-digit UIC", input: "70401ABC", pos: 0, wantAI: "7040", wantLen: 4, wantOK: true},
		{name: "3-digit GSIN", input: "40212345678901234567", pos: 0, wantAI: "402", wantLen: 3, wantOK: true},
		{name: "4-digit gross kg", input: "3302123456", pos: 0, wantAI: "3302", wantLen: 4, wantOK: true},
		{name: "4-digit gross lb", input: "3400123456", pos: 0, wantAI: "3400", wantLen: 4, wantOK: true},
		{name: "2-digit custom 90", input: "90CUSTOM", pos: 0, wantAI: "90", wantLen: 2, wantOK: true},
		{name: "2-digit custom 99", input: "99DATA", pos: 0, wantAI: "99", wantLen: 2, wantOK: true},
		{name: "unknown AI", input: "8812345", pos: 0, wantAI: "", wantLen: 0, wantOK: false},
		{name: "too short", input: "0", pos: 0, wantAI: "", wantLen: 0, wantOK: false},
		{name: "empty at pos", input: "01", pos: 2, wantAI: "", wantLen: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, aiLen, ok := lookupAIAt(tt.input, tt.pos)
			if ok != tt.wantOK {
				t.Errorf("lookupAIAt() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if !ok {
				return
			}
			if spec.AI != tt.wantAI {
				t.Errorf("lookupAIAt() AI = %q, want %q", spec.AI, tt.wantAI)
			}
			if aiLen != tt.wantLen {
				t.Errorf("lookupAIAt() len = %d, want %d", aiLen, tt.wantLen)
			}
		})
	}
}

func TestLookupAI(t *testing.T) {
	tests := []struct {
		code       string
		wantName   string
		wantFormat string
		wantOK     bool
	}{
		{code: "01", wantName: "GTIN", wantFormat: "N14", wantOK: true},
		{code: "00", wantName: "SSCC", wantFormat: "N18", wantOK: true},
		{code: "10", wantName: "Batch/Lot", wantFormat: "X..20", wantOK: true},
		{code: "17", wantName: "Expiration Date", wantFormat: "N6", wantOK: true},
		{code: "30", wantName: "Count", wantFormat: "N..8", wantOK: true},
		{code: "3102", wantName: "Net Weight kg", wantFormat: "N6", wantOK: true},
		{code: "91", wantName: "Internal", wantFormat: "X..90", wantOK: true},
		{code: "713", wantName: "NHRN Brazil", wantFormat: "X..20", wantOK: true},
		{code: "410", wantName: "Ship to / Deliver to GLN", wantFormat: "N13", wantOK: true},
		{code: "417", wantName: "Party GLN", wantFormat: "N13", wantOK: true},
		{code: "254", wantName: "GLN extension component", wantFormat: "X..20", wantOK: true},
		{code: "7040", wantName: "GS1 UIC with extension", wantFormat: "N1 X1 X1 X1", wantOK: true},
		{code: "253", wantName: "GDTI", wantFormat: "N13 [X..17]", wantOK: true},
		{code: "401", wantName: "GINC", wantFormat: "X..30", wantOK: true},
		{code: "8003", wantName: "GRAI", wantFormat: "N1 N13 [X..16]", wantOK: true},
		{code: "8004", wantName: "GIAI", wantFormat: "X..30", wantOK: true},
		{code: "8017", wantName: "GSRN – Provider", wantFormat: "N18", wantOK: true},
		{code: "88", wantOK: false},
		{code: "", wantOK: false},
		{code: "0", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got, ok := LookupAI(tt.code)
			if ok != tt.wantOK {
				t.Fatalf("LookupAI(%q) ok = %v, want %v", tt.code, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.Code != tt.code {
				t.Errorf("Code = %q, want %q", got.Code, tt.code)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", got.Format, tt.wantFormat)
			}
		})
	}
}

func TestLookupAICoversTable(t *testing.T) {
	for code := range aiTable {
		if _, ok := LookupAI(code); !ok {
			t.Errorf("LookupAI(%q) = false for table entry", code)
		}
	}
}
