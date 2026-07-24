package webhooks

import (
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	timestamp := "2026-01-20T06:39:05.683Z"
	body := []byte(`{"eventId":"evt-1"}`)

	signature, err := Sign(secret, timestamp, body)
	require.NoError(t, err)
	assert.Regexp(t, `^v1=[0-9a-f]{64}$`, signature)

	require.NoError(t, VerifySignature(secret, timestamp, body, signature))
}

func TestSignSecretRequired(t *testing.T) {
	t.Parallel()

	_, err := Sign("", "ts", []byte("body"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureSecretRequired)

	err = VerifySignature(
		"  ",
		"ts",
		[]byte("body"),
		"v1=0000000000000000000000000000000000000000000000000000000000000000",
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureSecretRequired)
}

func TestVerifySignatureTamperedBody(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	timestamp := "2026-01-20T06:39:05.683Z"

	signature, err := Sign(secret, timestamp, []byte(`{"eventId":"evt-1"}`))
	require.NoError(t, err)

	err = VerifySignature(secret, timestamp, []byte(`{"eventId":"evt-2"}`), signature)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func TestVerifySignatureTamperedTimestamp(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	body := []byte(`{"eventId":"evt-1"}`)

	signature, err := Sign(secret, "2026-01-20T06:39:05.683Z", body)
	require.NoError(t, err)

	err = VerifySignature(secret, "2026-01-20T07:00:00.000Z", body, signature)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func TestVerifySignatureMalformedHeader(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	body := []byte("body")

	tests := []struct {
		name   string
		header string
	}{
		{name: "empty", header: ""},
		{name: "whitespace only", header: "   "},
		{name: "missing prefix", header: "deadbeef"},
		{name: "wrong prefix", header: "v2=deadbeef"},
		{name: "invalid hex", header: "v1=not-hex-digits"},
		{name: "truncated digest", header: "v1=deadbeef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := VerifySignature(secret, "ts", body, tt.header)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrSignatureFormatInvalid)
		})
	}
}

func TestVerifySignatureHeaderList(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	timestamp := "2026-01-20T06:39:05.683Z"
	body := []byte(`{"eventId":"evt-1"}`)

	signature, err := Sign(secret, timestamp, body)
	require.NoError(t, err)

	header := "v0=abc, " + signature
	require.NoError(t, VerifySignature(secret, timestamp, body, header))
}

func TestVerifySignatureRawSecretFallback(t *testing.T) {
	t.Parallel()

	secret := "not_base64_!!"
	timestamp := "1737355145683"
	body := []byte(`{"eventId":"evt-1"}`)

	signature, err := Sign(secret, timestamp, body)
	require.NoError(t, err)
	require.NoError(t, VerifySignature(secret, timestamp, body, signature))
}

func TestVerifySignatureBase64AndRawDiffer(t *testing.T) {
	t.Parallel()

	rawKey := []byte("super-secret-key")
	encoded := base64.StdEncoding.EncodeToString(rawKey)
	timestamp := "2026-01-20T06:39:05.683Z"
	body := []byte("body")

	fromEncoded, err := Sign(encoded, timestamp, body)
	require.NoError(t, err)

	err = VerifySignature("different-secret-entirely!", timestamp, body, fromEncoded)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func TestVerifySignatureWithToleranceRFC3339(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	now := time.Date(2026, 1, 20, 6, 40, 0, 0, time.UTC)
	body := []byte(`{"eventId":"evt-1"}`)

	timestamp := "2026-01-20T06:39:05.683Z"
	signature, err := Sign(secret, timestamp, body)
	require.NoError(t, err)

	require.NoError(
		t,
		VerifySignatureWithTolerance(secret, timestamp, body, signature, now, 5*time.Minute),
	)

	err = VerifySignatureWithTolerance(secret, timestamp, body, signature, now, 30*time.Second)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTimestampSkewExceeded)
}

func TestVerifySignatureWithToleranceUnixMillis(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	eventTime := time.Date(2026, 1, 20, 6, 39, 5, 0, time.UTC)
	now := eventTime.Add(2 * time.Minute)
	body := []byte(`{"eventId":"evt-1"}`)

	timestamp := strconv.FormatInt(eventTime.UnixMilli(), 10)
	signature, err := Sign(secret, timestamp, body)
	require.NoError(t, err)

	require.NoError(
		t,
		VerifySignatureWithTolerance(secret, timestamp, body, signature, now, 5*time.Minute),
	)

	err = VerifySignatureWithTolerance(secret, timestamp, body, signature, now, time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTimestampSkewExceeded)
}

func TestVerifySignatureWithToleranceFutureTimestamp(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	now := time.Date(2026, 1, 20, 6, 39, 0, 0, time.UTC)
	body := []byte(`{"eventId":"evt-1"}`)

	timestamp := now.Add(10 * time.Minute).Format(time.RFC3339)
	signature, err := Sign(secret, timestamp, body)
	require.NoError(t, err)

	err = VerifySignatureWithTolerance(secret, timestamp, body, signature, now, 5*time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTimestampSkewExceeded)
}

func TestVerifySignatureWithToleranceMalformedTimestamp(t *testing.T) {
	t.Parallel()

	secret := base64.StdEncoding.EncodeToString([]byte("super-secret-key"))
	body := []byte(`{"eventId":"evt-1"}`)

	err := VerifySignatureWithTolerance(
		secret,
		"not-a-timestamp",
		body,
		"v1=0000000000000000000000000000000000000000000000000000000000000000",
		time.Now(),
		5*time.Minute,
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureFormatInvalid)
}
