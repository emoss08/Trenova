package domainvalidation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLocale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"english", "en", false},
		{"spanish", "es", false},
		{"traditional chinese", "zh-TW", false},
		{"simplified chinese", "zh-CN", false},
		{"empty defers to the column default", "", false},
		{"a language we do not ship", "fr", true},
		{"case must match the catalog filename", "zh-tw", true},
		{"not a string", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateLocale(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestValidateLocaleNamesTheSupportedSet(t *testing.T) {
	t.Parallel()

	err := ValidateLocale("fr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zh-TW",
		"the message should tell the caller what is actually accepted")
}
