package webhooksig_test

import (
	"encoding/base64"
	"testing"

	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func basic(credentials string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
}

/*
Postmark does not sign inbound webhooks. What it offers instead is HTTP basic
auth on the webhook URL — https://user:pass@host/… — which it sends back as an
Authorization header on every delivery. That header is the whole proof, so it
is checked exactly and in constant time.
*/
func TestVerifyBasicAuth(t *testing.T) {
	const secret = "postmark:correct-horse-battery-staple"

	cases := []struct {
		name   string
		header string
		want   error
	}{
		{"accepts the configured credentials", basic(secret), nil},
		{
			"accepts the scheme in any case",
			"basic " + base64.StdEncoding.EncodeToString([]byte(secret)),
			nil,
		},
		{"refuses a wrong password", basic("postmark:wrong"), webhooksig.ErrNoMatch},
		{
			"refuses a wrong user",
			basic("someone:correct-horse-battery-staple"),
			webhooksig.ErrNoMatch,
		},
		{
			"refuses a password that only starts the same",
			basic("postmark:correct-horse"),
			webhooksig.ErrNoMatch,
		},
		{"refuses a missing header", "", webhooksig.ErrMissingHeaders},
		{"refuses another scheme", "Bearer " + secret, webhooksig.ErrNoMatch},
		{"refuses a header that is not base64", "Basic !!!", webhooksig.ErrNoMatch},
		{"refuses credentials with no colon", basic("postmark"), webhooksig.ErrNoMatch},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := webhooksig.VerifyBasicAuth(secret, tc.header)
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestVerifyBasicAuth_RefusesEverythingWithoutAConfiguredSecret(t *testing.T) {
	assert.ErrorIs(t, webhooksig.VerifyBasicAuth("", basic(":")), webhooksig.ErrMissingHeaders)
}

func TestParseBasicCredentials(t *testing.T) {
	user, pass, ok := webhooksig.ParseBasicCredentials("postmark:s3cret:with:colons")
	require.True(t, ok)
	assert.Equal(t, "postmark", user)
	assert.Equal(t, "s3cret:with:colons", pass)

	for _, bad := range []string{"", "postmark", ":pass", "user:"} {
		_, _, ok = webhooksig.ParseBasicCredentials(bad)
		assert.Falsef(t, ok, "%q is not a usable credential pair", bad)
	}
}
