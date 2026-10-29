package netsuite_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCertificateID = "Zm9vYmFyLWNlcnRpZmljYXRl"

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

func decodeSegment(t *testing.T, segment string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &out))
	return out
}

func verifyES256(t *testing.T, key *ecdsa.PublicKey, assertion string) []string {
	t.Helper()
	segments := strings.Split(assertion, ".")
	require.Len(t, segments, 3)
	signature, err := base64.RawURLEncoding.DecodeString(segments[2])
	require.NoError(t, err)
	require.Len(t, signature, 64, "ES256 is raw r||s, not DER")
	digest := sha256.Sum256([]byte(segments[0] + "." + segments[1]))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	assert.True(t, ecdsa.Verify(key, digest[:], r, s), "the signature verifies")
	return segments
}

func TestClientAssertionHeaderAndClaims(t *testing.T) {
	t.Parallel()

	key := newKey(t)
	client, err := netsuite.NewOAuthClient(testOAuthConfig())
	require.NoError(t, err)
	now := time.Date(2026, 10, 13, 14, 0, 0, 0, time.UTC)

	assertion, err := client.ClientAssertion(netsuite.AssertionRequest{
		AccountID:     testAccount,
		CertificateID: testCertificateID,
		PrivateKey:    key,
		Now:           now,
	})
	require.NoError(t, err)
	segments := verifyES256(t, &key.PublicKey, assertion)

	header := decodeSegment(t, segments[0])
	assert.Equal(t, map[string]any{"alg": "ES256", "typ": "JWT", "kid": testCertificateID}, header)

	claims := decodeSegment(t, segments[1])
	assert.Equal(t, testClientID, claims["iss"])
	assert.Equal(t, "rest_webservices", claims["scope"])
	assert.Equal(t,
		"https://1234567-sb1.suitetalk.api.netsuite.com/services/rest/auth/oauth2/v1/token",
		claims["aud"])
	assert.InDelta(t, float64(now.Unix()), claims["iat"], 0)
	assert.InDelta(t, float64(now.Add(5*time.Minute).Unix()), claims["exp"], 0)
	jti, ok := claims["jti"].(string)
	require.True(t, ok)
	assert.Len(t, jti, 32)

	again, err := client.ClientAssertion(netsuite.AssertionRequest{
		AccountID: testAccount, CertificateID: testCertificateID, PrivateKey: key, Now: now,
	})
	require.NoError(t, err)
	assert.NotEqual(t, jti, decodeSegment(t, strings.Split(again, ".")[1])["jti"])

	other := newKey(t)
	digest := sha256.Sum256([]byte(segments[0] + "." + segments[1]))
	raw, err := base64.RawURLEncoding.DecodeString(segments[2])
	require.NoError(t, err)
	assert.False(t, ecdsa.Verify(&other.PublicKey, digest[:],
		new(big.Int).SetBytes(raw[:32]), new(big.Int).SetBytes(raw[32:])))
}

func TestClientAssertionValidates(t *testing.T) {
	t.Parallel()

	client, err := netsuite.NewOAuthClient(testOAuthConfig())
	require.NoError(t, err)
	key := newKey(t)

	_, err = client.ClientAssertion(netsuite.AssertionRequest{
		AccountID: testAccount, PrivateKey: key,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidCertificate)
	_, err = client.ClientAssertion(netsuite.AssertionRequest{
		AccountID: testAccount, CertificateID: "has space", PrivateKey: key,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidCertificate)
	_, err = client.ClientAssertion(netsuite.AssertionRequest{
		AccountID: testAccount, CertificateID: testCertificateID,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidPrivateKey)
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	_, err = client.ClientAssertion(netsuite.AssertionRequest{
		AccountID: testAccount, CertificateID: testCertificateID, PrivateKey: p384,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidPrivateKey)
	_, err = client.ClientAssertion(netsuite.AssertionRequest{
		AccountID: "bad id", CertificateID: testCertificateID, PrivateKey: key,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidAccountID)
}

func TestTokenFromCertificate(t *testing.T) {
	t.Parallel()

	key := newKey(t)
	now := time.Date(2026, 10, 13, 14, 0, 0, 0, time.UTC)
	var form map[string][]string
	client, base := newOAuthClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"), "the assertion is the only credential")
		form = assertTokenRequest(t, r, testTokenPath)
		_, _ = w.Write(fixture(t, "token_client_credentials.json"))
	}, netsuite.WithClock(func() time.Time { return now }))

	token, err := client.TokenFromCertificate(t.Context(), netsuite.CertificateGrant{
		AccountID:     testAccount,
		CertificateID: testCertificateID,
		PrivateKey:    key,
	})
	require.NoError(t, err)
	assert.Contains(t, token.AccessToken, "m2m-access")
	assert.Empty(t, token.RefreshToken)
	assert.Equal(t, time.Hour, token.ExpiresIn)

	assert.Equal(t, []string{"client_credentials"}, form["grant_type"])
	assert.Equal(t, []string{"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		form["client_assertion_type"])
	require.Len(t, form["client_assertion"], 1)
	segments := verifyES256(t, &key.PublicKey, form["client_assertion"][0])
	claims := decodeSegment(t, segments[1])
	assert.Equal(t, base+testTokenPath, claims["aud"])
	assert.InDelta(t, float64(now.Unix()), claims["iat"], 0)
}

func TestTokenFromCertificateRejectedAsInvalidClient(t *testing.T) {
	t.Parallel()

	client, _ := newOAuthClient(t, serveFixture(t, http.StatusBadRequest, "invalid_client.json"))
	_, err := client.TokenFromCertificate(t.Context(), netsuite.CertificateGrant{
		AccountID: testAccount, CertificateID: testCertificateID, PrivateKey: newKey(t),
	})
	assert.True(t, netsuiteerrors.IsInvalidClient(err))
}

func TestNewCertificateRoundTrips(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 13, 14, 0, 0, 500, time.UTC)
	cert, err := netsuite.NewCertificate(netsuite.CertificateRequest{
		Subject: "Trenova Acme Freight",
		Now:     now,
	})
	require.NoError(t, err)

	block, _ := pem.Decode(cert.CertificatePEM)
	require.NotNil(t, block)
	assert.Equal(t, "CERTIFICATE", block.Type)
	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.Equal(t, "Trenova Acme Freight", parsed.Subject.CommonName)
	assert.Equal(t, parsed.Subject.String(), parsed.Issuer.String(), "self-signed")
	require.NoError(t, parsed.CheckSignature(
		parsed.SignatureAlgorithm, parsed.RawTBSCertificate, parsed.Signature))
	assert.Equal(t, now.Truncate(time.Second), parsed.NotBefore)
	assert.Equal(t, now.Truncate(time.Second).Add(2*365*24*time.Hour), parsed.NotAfter)
	assert.Equal(t, parsed.NotAfter, cert.ExpiresAt)
	assert.Equal(t, x509.ECDSA, parsed.PublicKeyAlgorithm)

	key, err := netsuite.ParsePrivateKey(cert.PrivateKeyPEM)
	require.NoError(t, err)
	public, ok := parsed.PublicKey.(*ecdsa.PublicKey)
	require.True(t, ok)
	assert.True(t, key.PublicKey.Equal(public), "the key matches the certificate")

	client, err := netsuite.NewOAuthClient(testOAuthConfig())
	require.NoError(t, err)
	assertion, err := client.ClientAssertion(netsuite.AssertionRequest{
		AccountID: testAccount, CertificateID: testCertificateID, PrivateKey: key,
	})
	require.NoError(t, err)
	verifyES256(t, public, assertion)

	short, err := netsuite.NewCertificate(netsuite.CertificateRequest{
		Subject: "Trenova", Now: now, Validity: 30 * 24 * time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, now.Truncate(time.Second).Add(30*24*time.Hour), short.ExpiresAt)
}

func TestNewCertificateValidates(t *testing.T) {
	t.Parallel()

	_, err := netsuite.NewCertificate(netsuite.CertificateRequest{Subject: " "})
	require.ErrorIs(t, err, netsuiteerrors.ErrSubjectRequired)
	_, err = netsuite.NewCertificate(netsuite.CertificateRequest{
		Subject: strings.Repeat("a", 65),
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrSubjectRequired)
	_, err = netsuite.NewCertificate(netsuite.CertificateRequest{
		Subject: "Trenova", Validity: 3 * 365 * 24 * time.Hour,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidValidity)
	_, err = netsuite.NewCertificate(netsuite.CertificateRequest{
		Subject: "Trenova", Validity: -time.Hour,
	})
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidValidity)
}

func TestParsePrivateKey(t *testing.T) {
	t.Parallel()

	key := newKey(t)
	sec1, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	parsed, err := netsuite.ParsePrivateKey(
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: sec1}))
	require.NoError(t, err)
	assert.True(t, key.Equal(parsed))

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcs8, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	_, err = netsuite.ParsePrivateKey(
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}))
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidPrivateKey)

	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	pkcs8, err = x509.MarshalPKCS8PrivateKey(p384)
	require.NoError(t, err)
	_, err = netsuite.ParsePrivateKey(
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}))
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidPrivateKey)

	for _, raw := range [][]byte{nil, []byte("not pem"),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte{1}}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{1, 2}})} {
		_, err = netsuite.ParsePrivateKey(raw)
		require.ErrorIs(t, err, netsuiteerrors.ErrInvalidPrivateKey)
	}
}
