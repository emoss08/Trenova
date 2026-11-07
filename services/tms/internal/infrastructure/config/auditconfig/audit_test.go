package auditconfig

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditConfig_ExportTTLDefaultsToAWeek(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 168*time.Hour, (&Config{}).GetExportTTL())
	assert.Equal(t, 2*time.Hour, (&Config{ExportTTL: 2 * time.Hour}).GetExportTTL())
}

func TestAuditConfig_RefusesAShortFingerprintPepper(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&Config{}).Validate(), "no pepper fingerprints by last four only")
	require.ErrorIs(t, (&Config{FingerprintPepper: "short"}).Validate(), ErrFingerprintPepperTooShort)
	require.NoError(t, (&Config{FingerprintPepper: strings.Repeat("p", 32)}).Validate())
}
