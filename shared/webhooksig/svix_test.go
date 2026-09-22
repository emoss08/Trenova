package webhooksig_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "whsec_" // + the base64 key appended below

func signedDelivery(t *testing.T, body []byte, at time.Time) webhooksig.SvixParams {
	t.Helper()

	key := []byte("a-signing-key-of-some-length")
	encoded := base64.StdEncoding.EncodeToString(key)
	id := "msg_2b1c"
	stamp := strconv.FormatInt(at.Unix(), 10)

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + stamp + "."))
	mac.Write(body)

	return webhooksig.SvixParams{
		Secret:    testSecret + encoded,
		ID:        id,
		Timestamp: stamp,
		Signature: "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil)),
		Body:      body,
		Now:       at,
	}
}

func TestVerifySvix_AcceptsAGenuineDelivery(t *testing.T) {
	t.Parallel()

	now := time.Unix(1784131200, 0)
	require.NoError(t, webhooksig.VerifySvix(signedDelivery(t, []byte(`{"ok":true}`), now)))
}

// The body is what the signature is for. A delivery whose body was swapped
// after signing is the attack the check exists to stop.
func TestVerifySvix_RefusesAnAlteredBody(t *testing.T) {
	t.Parallel()

	p := signedDelivery(t, []byte(`{"ok":true}`), time.Unix(1784131200, 0))
	p.Body = []byte(`{"ok":false}`)

	require.ErrorIs(t, webhooksig.VerifySvix(p), webhooksig.ErrNoMatch)
}

/*
The reason this package exists.

The verifier it replaces checked the HMAC and nothing else, so a delivery
captured once stayed valid for as long as the signing secret did — replayable
forever, by anyone who had ever seen one. The signature covers the timestamp, so
an attacker cannot move it; reading it is what bounds the window.
*/
func TestVerifySvix_RefusesAReplayOutsideTheWindow(t *testing.T) {
	t.Parallel()

	signedAt := time.Unix(1784131200, 0)
	p := signedDelivery(t, []byte(`{"ok":true}`), signedAt)

	// The same bytes, with the same valid signature, an hour later.
	p.Now = signedAt.Add(time.Hour)

	err := webhooksig.VerifySvix(p)
	require.ErrorIs(t, err, webhooksig.ErrStale)
	assert.Contains(t, err.Error(), "1h0m0s out")
}

// A timestamp from the future is as much a sign of a forged header as an old
// one is of a replay, so the window is checked both ways.
func TestVerifySvix_RefusesADeliveryFromTheFuture(t *testing.T) {
	t.Parallel()

	signedAt := time.Unix(1784131200, 0)
	p := signedDelivery(t, []byte(`{"ok":true}`), signedAt)
	p.Now = signedAt.Add(-time.Hour)

	require.ErrorIs(t, webhooksig.VerifySvix(p), webhooksig.ErrStale)
}

// Ordinary clock drift and a provider's own retry latency must not drop a
// genuine delivery.
func TestVerifySvix_ToleratesOrdinaryDrift(t *testing.T) {
	t.Parallel()

	signedAt := time.Unix(1784131200, 0)
	p := signedDelivery(t, []byte(`{"ok":true}`), signedAt)
	p.Now = signedAt.Add(2 * time.Minute)

	require.NoError(t, webhooksig.VerifySvix(p))
}

// A secret roll puts every signature still in rotation on the delivery, space
// separated. Dropping those messages would be an outage every rotation.
func TestVerifySvix_AcceptsOneOfSeveralRotatedSignatures(t *testing.T) {
	t.Parallel()

	p := signedDelivery(t, []byte(`{"ok":true}`), time.Unix(1784131200, 0))
	p.Signature = "v1,Zm9v " + p.Signature + " v1,YmFy"

	require.NoError(t, webhooksig.VerifySvix(p))
}

func TestVerifySvix_RefusesMissingHeaders(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"secret", "id", "timestamp", "signature"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()

			p := signedDelivery(t, []byte(`{}`), time.Unix(1784131200, 0))
			switch field {
			case "secret":
				p.Secret = ""
			case "id":
				p.ID = ""
			case "timestamp":
				p.Timestamp = ""
			case "signature":
				p.Signature = ""
			}

			require.ErrorIs(t, webhooksig.VerifySvix(p), webhooksig.ErrMissingHeaders)
		})
	}
}

func TestVerifySvix_RefusesANonNumericTimestamp(t *testing.T) {
	t.Parallel()

	p := signedDelivery(t, []byte(`{}`), time.Unix(1784131200, 0))
	p.Timestamp = "yesterday"

	require.ErrorIs(t, webhooksig.VerifySvix(p), webhooksig.ErrBadTimestamp)
}
