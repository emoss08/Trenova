package secretutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyPrefix_ReadsTheVendorPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		secret string
		want   string
	}{
		{"anthropic", "sk-ant-api03-AbCdEfGhIjKlMnOpQrStUvWxYz012345", "sk-ant-"},
		{"openai project", "sk-proj-AbCdEfGhIjKlMnOpQrStUvWxYz012345", "sk-proj-"},
		{"openai service account", "sk-svcacct-AbCdEfGhIjKlMnOpQrStUvWxYz0123", "sk-svcacct-"},
		{"openai legacy", "sk-AbCdEfGhIjKlMnOpQrStUvWxYz012345", "sk-"},
		{"openrouter", "sk-or-v1-0123456789abcdef0123456789abcdef", "sk-or-"},
		{"groq", "gsk_AbCdEfGhIjKlMnOpQrStUvWxYz012345", "gsk_"},
		{"voyage", "pa-AbCdEfGhIjKlMnOpQrStUvWxYz012345", "pa-"},
		{"nvidia", "nvapi-AbCdEfGhIjKlMnOpQrStUvWxYz012345", "nvapi-"},
		{"google", "AIzaSyAbCdEfGhIjKlMnOpQrStUvWxYz01234", "AIza"},
		{"surrounding space", "  gsk_AbCdEfGhIjKlMnOpQrStUvWxYz012345\n", "gsk_"},
		{"no prefix", "0123456789abcdef0123456789abcdef", ""},
		{"upper case start", "ABCD-0123456789abcdef0123456789", ""},
		{"first segment too long", "averylongword-0123456789abcdef0123", ""},
		{"too short to keep anything hidden", "sk-abc", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, KeyPrefix(tc.secret))
		})
	}
}

func TestLastFour_KeepsOnlyTheEndOfALongEnoughSecret(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "2345", LastFour("sk-ant-api03-AbCdEfGh012345"))
	assert.Equal(t, "2345", LastFour(" sk-ant-api03-AbCdEfGh012345 \n"))
	assert.Empty(t, LastFour("short"), "a short secret keeps nothing, since four would be most of it")
	assert.Empty(t, LastFour(""))
}

func TestFingerprintOf_NeverHoldsMoreThanThePrefixAndLastFour(t *testing.T) {
	t.Parallel()

	secret := "sk-proj-AbCdEfGhIjKlMnOpQrStUvWxYz9876"
	fingerprint := FingerprintOf(secret)

	assert.Equal(t, Fingerprint{Prefix: "sk-proj-", LastFour: "9876"}, fingerprint)
	assert.Less(t, len(fingerprint.Prefix)+len(fingerprint.LastFour), len(secret)/2)
}
