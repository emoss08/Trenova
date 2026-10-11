package userservice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMySettingsRequest_Locale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		locale  string
		wantErr bool
	}{
		{"a language we ship", "es", false},
		{"traditional chinese", "zh-TW", false},
		{"omitted leaves the current language alone", "", false},
		{"a language we do not ship is rejected", "fr", true},
		{"a malformed tag is rejected", "not-a-locale", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := UpdateMySettingsRequest{
				Timezone:   "America/New_York",
				TimeFormat: domaintypes.TimeFormat12Hour,
				Locale:     tt.locale,
			}

			err := req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "language",
					"the message should name the field the form has to highlight")
				return
			}
			assert.NoError(t, err)
		})
	}
}
