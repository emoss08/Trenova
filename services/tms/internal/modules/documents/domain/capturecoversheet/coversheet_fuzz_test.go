//go:build fuzz

package capturecoversheet

import (
	"strings"
	"testing"
)

func FuzzCoverSheetTokenReadsOnlyWhatASheetPrints(f *testing.F) {
	for _, seed := range []string{
		"TRNV-CS1:abc123", " TRNV-CS1:abc123 \n", "TRNV-CS1:", "trnv-cs1:abc", "TRNV-CS2:abc",
		"https://example.com", "", "TRNV-CS1:TRNV-CS1:abc",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, payload string) {
		token, ok := CoverSheetToken(payload)

		if !ok {
			if token != "" {
				t.Fatalf("%q was refused but read %q", payload, token)
			}
			return
		}
		if token == "" {
			t.Fatalf("%q read an empty token", payload)
		}
		if CoverSheetPayload(token) != strings.TrimSpace(payload) {
			t.Fatalf("%q read %q, which prints as %q", payload, token, CoverSheetPayload(token))
		}
		if again, ok := CoverSheetToken(CoverSheetPayload(token)); !ok || again != token {
			t.Fatalf("%q read %q, which reads back as %q", payload, token, again)
		}
	})
}
