package quickbooks_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVerifier = "verifier-token-value"

func sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testVerifier))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	t.Parallel()

	body := fixture(t, "webhook_cloudevents.json")
	require.NoError(t, quickbooks.VerifySignature(testVerifier, sign(body), body))

	tampered := append([]byte{}, body...)
	tampered[10] = 'X'
	require.ErrorIs(t, quickbooks.VerifySignature(testVerifier, sign(body), tampered), quickbooks.ErrInvalidSignature)
	require.ErrorIs(t, quickbooks.VerifySignature(testVerifier, "", body), quickbooks.ErrMissingSignature)
	require.ErrorIs(t, quickbooks.VerifySignature(testVerifier, "%%%", body), quickbooks.ErrMissingSignature)
	require.ErrorIs(t, quickbooks.VerifySignature("", sign(body), body), quickbooks.ErrVerifierRequired)
}

func TestParseRealmIDsFromCloudEvents(t *testing.T) {
	t.Parallel()

	realms, err := quickbooks.ParseRealmIDs(fixture(t, "webhook_cloudevents.json"))
	require.NoError(t, err)
	assert.Equal(t, []string{"9341452431742015", "4620816365201234567"}, realms)
}

func TestParseRealmIDsFromLegacyEnvelope(t *testing.T) {
	t.Parallel()

	realms, err := quickbooks.ParseRealmIDs(fixture(t, "webhook_legacy.json"))
	require.NoError(t, err)
	assert.Equal(t, []string{"9341452431742015"}, realms)
}

func TestParseRealmIDsRejectsGarbage(t *testing.T) {
	t.Parallel()

	_, err := quickbooks.ParseRealmIDs([]byte("not json"))
	require.ErrorIs(t, err, quickbooks.ErrUnexpectedPayload)
	_, err = quickbooks.ParseRealmIDs([]byte("[{"))
	require.ErrorIs(t, err, quickbooks.ErrUnexpectedPayload)

	realms, err := quickbooks.ParseRealmIDs([]byte("[]"))
	require.NoError(t, err)
	assert.Empty(t, realms)
}
