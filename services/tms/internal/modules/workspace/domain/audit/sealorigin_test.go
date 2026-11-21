package audit_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/stretchr/testify/assert"
)

func TestASealSaysWhereItsHistoryWasVerifiedAndTheHashCoversIt(t *testing.T) {
	t.Parallel()

	secret := []byte("seal-origin-test-key")
	cases := []struct {
		origin   audit.Origin
		keyID    string
		version  int16
		status   audit.VerificationStatus
		unsigned bool
	}{
		{audit.OriginHere, "k1", audit.SealHashVersionSigned, audit.VerificationVerified, false},
		{audit.OriginHere, "", audit.SealHashVersionUnsigned, audit.VerificationVerified, true},
		{audit.OriginSourceVerified, "k1", audit.SealHashVersionSourceVerified,
			audit.VerificationSourceVerified, false},
		{audit.OriginSourceVerified, "", audit.SealHashVersionSourceVerifiedUnsigned,
			audit.VerificationSourceVerified, true},
		{audit.OriginUnverified, "k1", audit.SealHashVersionImported, audit.VerificationImported, false},
		{audit.OriginUnverified, "", audit.SealHashVersionImportedUnsigned,
			audit.VerificationImported, true},
	}
	for _, tc := range cases {
		seal := &audit.Seal{Seq: 4, FromTS: 10, ToTS: 20, RowCount: 2}
		seal.SignFrom(tc.origin, tc.keyID, secret)
		assert.Equal(t, tc.version, seal.HashVersion)
		assert.Equal(t, tc.origin, seal.Origin())
		assert.Equal(t, tc.unsigned, seal.Unsigned())
		assert.Equal(t, tc.status, seal.Origin().Status())
		assert.True(t, tc.status.Holds())

		key := secret
		if tc.unsigned {
			key = nil
		}
		hash := seal.Hash
		seal.HashVersion = audit.SealHashVersionSigned
		if tc.unsigned {
			seal.HashVersion = audit.SealHashVersionUnsigned
		}
		if tc.origin != audit.OriginHere {
			assert.NotEqual(t, hash, seal.ComputeHash(key), "stripping the mark breaks the seal")
		}
	}
	assert.Equal(t, audit.OriginUnverified, audit.OriginSourceVerified.Weaker(audit.OriginUnverified))
	assert.Equal(t, audit.OriginSourceVerified, audit.OriginHere.Weaker(audit.OriginSourceVerified))
	assert.False(t, audit.VerificationBroken.Holds())
}
