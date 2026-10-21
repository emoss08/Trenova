package encryptionservice

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptStringWithLegacyReadsBoundAndUnboundValues(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()

	bound, err := svc.EncryptStringWithAAD("sk-bound", aad)
	require.NoError(t, err)
	plaintext, legacy, err := svc.DecryptStringWithLegacy(bound, aad)
	require.NoError(t, err)
	assert.Equal(t, "sk-bound", plaintext)
	assert.False(t, legacy)

	unbound, err := svc.EncryptString("sk-legacy")
	require.NoError(t, err)
	plaintext, legacy, err = svc.DecryptStringWithLegacy(unbound, aad)
	require.NoError(t, err)
	assert.Equal(t, "sk-legacy", plaintext)
	assert.True(t, legacy)
}

func TestDecryptStringWithLegacyNeverFallsBackForAValueBoundElsewhere(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()
	elsewhere := aad
	elsewhere.OrganizationID = pulid.MustNew("org_")

	foreign, err := svc.EncryptStringWithAAD("sk-foreign", elsewhere)
	require.NoError(t, err)

	_, legacy, err := svc.DecryptStringWithLegacy(foreign, aad)
	require.ErrorIs(t, err, ErrInvalidEnvelope)
	assert.False(t, legacy)

	_, _, err = svc.DecryptStringWithLegacy("not-an-envelope", aad)
	require.ErrorIs(t, err, ErrInvalidEnvelope)
}

func TestDecryptAndRebindBindsOnlyLegacyValues(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()

	unbound, err := svc.EncryptString("sk-legacy")
	require.NoError(t, err)
	plaintext, rebound, err := svc.DecryptAndRebind(unbound, aad)
	require.NoError(t, err)
	assert.Equal(t, "sk-legacy", plaintext)
	require.NotEmpty(t, rebound)
	assert.True(t, EnvelopeHasAAD(rebound, aad))
	assert.False(t, EnvelopeHasAAD(rebound, AAD{}))
	again, err := svc.DecryptStringWithAAD(rebound, aad)
	require.NoError(t, err)
	assert.Equal(t, "sk-legacy", again)

	bound, err := svc.EncryptStringWithAAD("sk-bound", aad)
	require.NoError(t, err)
	_, rebound, err = svc.DecryptAndRebind(bound, aad)
	require.NoError(t, err)
	assert.Empty(t, rebound)
}

func TestEnvelopeHasAAD(t *testing.T) {
	t.Parallel()

	svc := testService()
	aad := testAAD()
	bound, err := svc.EncryptStringWithAAD("value", aad)
	require.NoError(t, err)

	assert.True(t, EnvelopeHasAAD(bound, aad))
	assert.False(t, EnvelopeHasAAD(bound, AAD{}))
	assert.False(t, EnvelopeHasAAD("plain", aad))
	assert.False(t, EnvelopeHasAAD(envelopePrefix+"!!!", aad))
}
