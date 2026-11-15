package supportaccess_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireRefused(t *testing.T, err error, reason supportaccess.RefusalReason) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, reason, supportaccess.ReasonOf(err))
}

func TestVerifyAssertion_AcceptsAnAssertionTheControlPlaneSignedForThisInstance(t *testing.T) {
	t.Parallel()

	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)

	got, err := supportaccess.VerifyAssertion(sign(t, key, claimsAt(now)), keySetOf(key), now)
	require.NoError(t, err)
	assert.Equal(t, "jane.doe@trenova.app", got.StaffEmail)
	assert.Equal(t, "Jane Doe", got.StaffName)
	assert.Equal(t, testOrg, got.OrganizationID.String())
	assert.Equal(t, "a1b2c3d4e5f6g7h8i9j0", got.ID)
}

func TestVerifyAssertion_ComparesTheEmailAfterLowerCasingASCII(t *testing.T) {
	t.Parallel()

	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)
	claims := claimsAt(now)
	claims["sub"] = "Jane.Doe@Trenova.APP"

	got, err := supportaccess.VerifyAssertion(sign(t, key, claims), keySetOf(key), now)
	require.NoError(t, err)
	assert.Equal(t, "jane.doe@trenova.app", got.StaffEmail)
}

func TestVerifyAssertion_RefusesWhatTheInstanceMustNotAccept(t *testing.T) {
	t.Parallel()

	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)

	tests := []struct {
		name   string
		mutate func(map[string]any)
		reason supportaccess.RefusalReason
	}{
		{"for another instance", func(c map[string]any) { c["aud"] = "inst_other" }, supportaccess.RefusalAudience},
		{"with an audience list", func(c map[string]any) { c["aud"] = []string{testInstance} }, supportaccess.RefusalClaims},
		{"from another issuer", func(c map[string]any) { c["iss"] = "someone-else" }, supportaccess.RefusalIssuer},
		{"expired", func(c map[string]any) {
			c["iat"] = now.Add(-2 * time.Minute).Unix()
			c["exp"] = now.Add(-time.Minute).Unix()
		}, supportaccess.RefusalExpired},
		{"not yet valid", func(c map[string]any) {
			c["iat"] = now.Add(time.Minute).Unix()
			c["exp"] = now.Add(90 * time.Second).Unix()
		}, supportaccess.RefusalNotYetValid},
		{"before its not-before", func(c map[string]any) {
			c["nbf"] = now.Add(time.Minute).Unix()
		}, supportaccess.RefusalNotYetValid},
		{"living longer than a minute", func(c map[string]any) {
			c["exp"] = now.Add(10 * time.Minute).Unix()
		}, supportaccess.RefusalLifetime},
		{"with no issue time", func(c map[string]any) { delete(c, "iat") }, supportaccess.RefusalLifetime},
		{"for a non-staff domain", func(c map[string]any) { c["sub"] = "jane@example.com" }, supportaccess.RefusalStaffEmail},
		{"for a subdomain of the staff domain", func(c map[string]any) {
			c["sub"] = "jane@evil.trenova.app"
		}, supportaccess.RefusalStaffEmail},
		{"with a Cyrillic look-alike local part", func(c map[string]any) {
			c["sub"] = "jаne.doe@trenova.app"
		}, supportaccess.RefusalStaffEmail},
		{"with a short assertion id", func(c map[string]any) { c["jti"] = "short" }, supportaccess.RefusalClaims},
		{"with an organization that is not an id", func(c map[string]any) {
			c["org"] = "org_' OR 1=1 --xxxxxxxxxxxxxxx"
		}, supportaccess.RefusalClaims},
		{"with no staff name", func(c map[string]any) { c["name"] = "  " }, supportaccess.RefusalClaims},
		{"with a control character in the name", func(c map[string]any) {
			c["name"] = "Jane‮Doe"
		}, supportaccess.RefusalClaims},
		{"with an unknown claim", func(c map[string]any) { c["role"] = "admin" }, supportaccess.RefusalClaims},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			claims := claimsAt(now)
			tt.mutate(claims)
			_, err := supportaccess.VerifyAssertion(sign(t, key, claims), keySetOf(key), now)
			requireRefused(t, err, tt.reason)
		})
	}
}

func TestVerifyAssertion_RefusesASignatureByAnyKeyButTheDeliveredOne(t *testing.T) {
	t.Parallel()

	delivered := newSigningKey(t, testKeyID)
	impostor := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)

	_, err := supportaccess.VerifyAssertion(sign(t, impostor, claimsAt(now)), keySetOf(delivered), now)
	requireRefused(t, err, supportaccess.RefusalSignature)

	unknown := newSigningKey(t, "retired-key")
	_, err = supportaccess.VerifyAssertion(sign(t, unknown, claimsAt(now)), keySetOf(delivered), now)
	requireRefused(t, err, supportaccess.RefusalUnknownKey)
}

func TestVerifyAssertion_AcceptsEitherKeyWhileTheyRotate(t *testing.T) {
	t.Parallel()

	old := newSigningKey(t, "cp-old")
	next := newSigningKey(t, "cp-next")
	now := time.Unix(1_800_000_000, 0)
	keys := keySetOf(old, next)

	_, err := supportaccess.VerifyAssertion(sign(t, old, claimsAt(now)), keys, now)
	require.NoError(t, err)
	_, err = supportaccess.VerifyAssertion(sign(t, next, claimsAt(now)), keys, now)
	require.NoError(t, err)
}

func TestVerifyAssertion_RefusesAlgorithmConfusion(t *testing.T) {
	t.Parallel()

	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)
	payload, err := sonic.Marshal(claimsAt(now))
	require.NoError(t, err)

	t.Run("an unsigned token", func(t *testing.T) {
		t.Parallel()
		header := `{"alg":"none","kid":"` + testKeyID + `","typ":"` + supportaccess.AssertionType + `"}`
		token := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." +
			base64.RawURLEncoding.EncodeToString(payload) + "."
		_, verr := supportaccess.VerifyAssertion(token, keySetOf(key), now)
		requireRefused(t, verr, supportaccess.RefusalAlgorithm)
	})

	t.Run("an HMAC keyed with the public key", func(t *testing.T) {
		t.Parallel()
		signer, serr := jose.NewSigner(
			jose.SigningKey{
				Algorithm: jose.HS256,
				Key:       jose.JSONWebKey{Key: []byte(key.pub), KeyID: testKeyID},
			},
			(&jose.SignerOptions{}).WithType(supportaccess.AssertionType),
		)
		require.NoError(t, serr)
		_, verr := supportaccess.VerifyAssertion(serialize(t, signer, claimsAt(now)), keySetOf(key), now)
		requireRefused(t, verr, supportaccess.RefusalAlgorithm)
	})

	t.Run("a key embedded in the header", func(t *testing.T) {
		t.Parallel()
		signer, serr := jose.NewSigner(
			jose.SigningKey{Algorithm: jose.EdDSA, Key: key.priv},
			(&jose.SignerOptions{EmbedJWK: true}).WithType(supportaccess.AssertionType),
		)
		require.NoError(t, serr)
		_, verr := supportaccess.VerifyAssertion(serialize(t, signer, claimsAt(now)), keySetOf(key), now)
		requireRefused(t, verr, supportaccess.RefusalHeader)
	})

	t.Run("a token of another type", func(t *testing.T) {
		t.Parallel()
		signer, serr := jose.NewSigner(
			jose.SigningKey{Algorithm: jose.EdDSA, Key: jose.JSONWebKey{Key: key.priv, KeyID: testKeyID}},
			(&jose.SignerOptions{}).WithType("JWT"),
		)
		require.NoError(t, serr)
		_, verr := supportaccess.VerifyAssertion(serialize(t, signer, claimsAt(now)), keySetOf(key), now)
		requireRefused(t, verr, supportaccess.RefusalHeader)
	})
}

func TestVerifyAssertion_RefusesAMalformedOrOversizedToken(t *testing.T) {
	t.Parallel()

	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)

	for _, token := range []string{"", "a.b", strings.Repeat("a", supportaccess.MaxAssertionBytes+1) + ".b.c"} {
		_, err := supportaccess.VerifyAssertion(token, keySetOf(key), now)
		requireRefused(t, err, supportaccess.RefusalMalformed)
	}
}

func TestVerifyAssertion_RefusesEverythingBeforeKeysArrive(t *testing.T) {
	t.Parallel()

	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)

	_, err := supportaccess.VerifyAssertion(sign(t, key, claimsAt(now)), nil, now)
	requireRefused(t, err, supportaccess.RefusalNotConfigured)
}
