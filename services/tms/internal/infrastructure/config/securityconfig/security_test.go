package securityconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryCodePepperKeyIsStableAndKeyedByThePepper(t *testing.T) {
	t.Parallel()

	configured := &Config{
		MFA: MFAConfig{RecoveryCodePepper: "4d1f8a2b6c9e3071a5b8c2d4e6f8091a3b5c7d9e"},
	}
	first, err := configured.RecoveryCodePepperKey()
	require.NoError(t, err)
	second, err := configured.RecoveryCodePepperKey()
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Len(t, first, 32)

	fromSession := &Config{
		Session: SessionConfig{Secret: "a-very-long-secret-that-is-at-least-32-characters-long"},
	}
	derived, err := fromSession.RecoveryCodePepperKey()
	require.NoError(t, err)
	assert.NotEqual(t, first, derived)
}

func TestValidateProductionLocalKey_RequiresLength(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, validateProductionLocalKey("short"), ErrProductionLocalEncryptionKeyRequired)
	require.NoError(t, validateProductionLocalKey("3f9b1c7e5a2d48f6b0c9e1a7d4f2b8c6"))
}
