package cloudsignup_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/cloudsignup"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func validSignup() *cloudsignup.CloudSignup {
	return &cloudsignup.CloudSignup{
		EmailAddress:    "Owner@Example.com",
		EmailNormalized: "owner@example.com",
		Name:            "Owner Person",
		CompanyName:     "Example Freight",
		PasswordHash:    "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		TokenHash:       strings.Repeat("ab", 32),
		Status:          cloudsignup.StatusPending,
		ExpiresAt:       2_000,
	}
}

func TestCloudSignupValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*cloudsignup.CloudSignup)
		field  string
	}{
		{name: "valid"},
		{name: "email format", mutate: func(c *cloudsignup.CloudSignup) { c.EmailAddress = "nope" }, field: "emailAddress"},
		{name: "name required", mutate: func(c *cloudsignup.CloudSignup) { c.Name = "" }, field: "name"},
		{name: "company required", mutate: func(c *cloudsignup.CloudSignup) { c.CompanyName = "" }, field: "companyName"},
		{name: "password required", mutate: func(c *cloudsignup.CloudSignup) { c.PasswordHash = "" }, field: "PasswordHash"},
		{
			name:   "token hash shape",
			mutate: func(c *cloudsignup.CloudSignup) { c.TokenHash = strings.Repeat("Z", 64) },
			field:  "TokenHash",
		},
		{name: "status invalid", mutate: func(c *cloudsignup.CloudSignup) { c.Status = "verified" }, field: "status"},
		{name: "attempts negative", mutate: func(c *cloudsignup.CloudSignup) { c.Attempts = -1 }, field: "attempts"},
		{name: "expiry required", mutate: func(c *cloudsignup.CloudSignup) { c.ExpiresAt = 0 }, field: "expiresAt"},
		{
			name:   "user agent too long",
			mutate: func(c *cloudsignup.CloudSignup) { c.UserAgent = strings.Repeat("a", 513) },
			field:  "userAgent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			signup := validSignup()
			if tt.mutate != nil {
				tt.mutate(signup)
			}

			multiErr := errortypes.NewMultiError()
			signup.Validate(multiErr)

			if tt.field == "" {
				assert.False(t, multiErr.HasErrors(), multiErr.Error())
				return
			}

			fields := make([]string, 0, len(multiErr.Errors))
			for _, e := range multiErr.Errors {
				fields = append(fields, e.Field)
			}
			assert.Contains(t, fields, tt.field)
		})
	}
}

func TestCloudSignupState(t *testing.T) {
	t.Parallel()

	signup := validSignup()
	assert.True(t, signup.IsPending())
	assert.False(t, signup.IsExpired(1_999))
	assert.True(t, signup.IsExpired(2_000))

	assert.False(t, cloudsignup.StatusPending.IsTerminal())
	assert.True(t, cloudsignup.StatusProvisioned.IsTerminal())
	assert.True(t, cloudsignup.StatusExpired.IsTerminal())
	assert.True(t, cloudsignup.StatusRejected.IsTerminal())
	for _, status := range cloudsignup.AllStatuses() {
		assert.True(t, status.IsValid())
	}
}
