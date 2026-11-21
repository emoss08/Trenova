package supportaccess_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sharedKeySetOf(keys ...signingKey) *supportaccess.KeySet {
	set := keySetOf(keys...)
	set.InstanceID = ""
	return set
}

func refusalOf(t *testing.T, err error) supportaccess.RefusalReason {
	t.Helper()
	var refusal *supportaccess.RefusalError
	require.ErrorAs(t, err, &refusal)
	return refusal.Reason
}

func TestVerifyTenantAssertion_HandsTheAudienceOnForTheTenantCheck(t *testing.T) {
	t.Parallel()
	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)
	claims := claimsAt(now)
	claims["aud"] = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"

	got, err := supportaccess.VerifyTenantAssertion(sign(t, key, claims), sharedKeySetOf(key), now)
	require.NoError(t, err)
	assert.Equal(t, "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10", got.Audience)
}

func TestVerifyTenantAssertion_RefusesEachEarlierCheckAsTheDedicatedOneDoes(t *testing.T) {
	t.Parallel()
	key, other := newSigningKey(t, testKeyID), newSigningKey(t, "cp-other")
	now := time.Unix(1_800_000_000, 0)

	_, err := supportaccess.VerifyTenantAssertion(sign(t, other, claimsAt(now)), sharedKeySetOf(key), now)
	assert.Equal(t, supportaccess.RefusalUnknownKey, refusalOf(t, err))

	issuer := claimsAt(now)
	issuer["iss"] = "someone-else"
	_, err = supportaccess.VerifyTenantAssertion(sign(t, key, issuer), sharedKeySetOf(key), now)
	assert.Equal(t, supportaccess.RefusalIssuer, refusalOf(t, err))

	long := claimsAt(now)
	long["exp"] = now.Unix() + 120
	_, err = supportaccess.VerifyTenantAssertion(sign(t, key, long), sharedKeySetOf(key), now)
	assert.Equal(t, supportaccess.RefusalLifetime, refusalOf(t, err))

	_, err = supportaccess.VerifyTenantAssertion(
		sign(t, key, claimsAt(now)), sharedKeySetOf(key), now.Add(2*time.Minute),
	)
	assert.Equal(t, supportaccess.RefusalExpired, refusalOf(t, err))
}

func TestEachTenancyRefusesTheOthersKeySet(t *testing.T) {
	t.Parallel()
	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)
	token := sign(t, key, claimsAt(now))

	_, err := supportaccess.VerifyTenantAssertion(token, keySetOf(key), now)
	assert.Equal(t, supportaccess.RefusalNotConfigured, refusalOf(t, err))
	_, err = supportaccess.VerifyAssertion(token, sharedKeySetOf(key), now)
	assert.Equal(t, supportaccess.RefusalNotConfigured, refusalOf(t, err))
}
