package util_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/util"
	"github.com/stretchr/testify/assert"
)

type testTimeZone struct {
	Timezone string `validate:"timezone"`
}

func TestValidateTimeZone(t *testing.T) {
	validator, err := util.NewValidator()
	if err != nil {
		assert.FailNow(t, "Failed to create validator")
	}

	tests := []struct {
		name    string
		payload testTimeZone // Use the struct designed for testing
		wantErr bool
	}{
		{
			name:    "valid timezone",
			payload: testTimeZone{Timezone: "America/New_York"},
			wantErr: false,
		},
		{
			name:    "invalid timezone",
			payload: testTimeZone{Timezone: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = validator.Validate(tt.payload)
			if tt.wantErr {
				assert.Error(t, err, "Expected an error for test case: %s", tt.name)
			} else {
				assert.NoError(t, err, "Did not expect an error for test case: %s", tt.name)
			}
		})
	}
}

type testEmailAddress struct {
	Recipients string `validate:"commaSeparatedEmails"`
}

func TestValidateEmailAddress(t *testing.T) {
	validator, err := util.NewValidator()
	if err != nil {
		assert.FailNow(t, "Failed to create validator")
	}

	tests := []struct {
		name    string
		payload testEmailAddress
		wantErr bool
	}{
		{
			name: "valid email addresses",
			payload: testEmailAddress{
				Recipients: "test@gmail.com,test2@gmail.com",
			},
			wantErr: false,
		},
		{
			name:    "invalid email addresses",
			payload: testEmailAddress{Recipients: "invalid#email.com,invalid2$email.com"},
			wantErr: true,
		},

		{
			name:    "mixed valid and invalid email addresses",
			payload: testEmailAddress{Recipients: "test@gmail.com,invalid#email.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = validator.Validate(tt.payload)
			if tt.wantErr {
				assert.Error(t, err, "Expected an error for test case: %s", tt.name)
			} else {
				assert.NoError(t, err, "Did not expect an error for test case: %s", tt.name)
			}
		})
	}
}
