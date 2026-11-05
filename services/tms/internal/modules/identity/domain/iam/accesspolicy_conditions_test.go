package iam

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestAccessPolicyConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		conditions map[string]string
		invalid    bool
	}{
		{name: "none"},
		{name: "a known key", conditions: map[string]string{"principalType": "api_key"}},
		{name: "a misspelt key", conditions: map[string]string{"mfaRequred": "true"}, invalid: true},
		{name: "a numeric minimum", conditions: map[string]string{"authenticatorAalMin": "2"}},
		{
			name:       "an unreadable minimum",
			conditions: map[string]string{"federationFalMin": "two"},
			invalid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			(&AccessPolicy{Conditions: tt.conditions}).validateConditions(multiErr)

			assert.Equal(t, tt.invalid, multiErr.HasErrors())
		})
	}
}
