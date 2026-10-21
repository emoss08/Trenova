package tenant

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckNewPasswordAppliesThePasswordPolicy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		password string
		email    string
		message  string
	}{
		{"too short", "abcd-1234", "", "Password must be at least 12 characters"},
		{"too long", strings.Repeat("ab1-", 19), "", "Password must be 72 bytes or fewer"},
		{"only spaces", strings.Repeat(" ", 13), "", "Password cannot be only spaces"},
		{"common", "password1234", "", "This password is too common. Choose something harder to guess"},
		{
			"contains the email",
			"x-dana.whitfield-x",
			"dana.whitfield@example.com",
			"Password must not contain the account's email address",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := CheckNewPassword("newPassword", tc.password, tc.email)
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, "newPassword", multiErr.Errors[0].Field)
			assert.Equal(t, tc.message, multiErr.Errors[0].Message)
		})
	}

	require.NoError(t, CheckNewPassword("newPassword", "Tr4ck-the-l0ads!", "dana@example.com"))
}

func TestGeneratePasswordRefusesMoreThanBcryptReads(t *testing.T) {
	t.Parallel()

	user := &User{}
	_, err := user.GeneratePassword(strings.Repeat("ab1-", 19))
	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "password", validationErr.Field)

	hashed, err := user.GeneratePassword(strings.Repeat("ab1-", 18))
	require.NoError(t, err)
	assert.NotEmpty(t, hashed)
}
