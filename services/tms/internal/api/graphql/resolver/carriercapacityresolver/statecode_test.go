package carriercapacityresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateCode_ReadsATwoLetterCodeInAnyCase(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]string{"TX": "TX", " ga ": "GA", "nY": "NY", "": "", "  ": ""} {
		got, err := stateCode("originStateCode", &raw)
		require.NoError(t, err, raw)
		assert.Equal(t, want, got, raw)
	}

	none, err := stateCode("originStateCode", nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestStateCode_RefusesWhatIsNotACode(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"Texas", "T", "T1", "us_01M6FCW4EB35K62PBPYCK3QKQY", "É!"} {
		_, err := stateCode("destinationStateCode", &raw)
		require.Error(t, err, raw)
	}
}
