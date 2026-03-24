package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		req := &LoginRequest{
			EmailAddress: "test@example.com",
			Password:     "password123",
		}
		assert.NoError(t, req.Validate())
	})

	t.Run("missing email", func(t *testing.T) {
		t.Parallel()
		req := &LoginRequest{
			Password: "password123",
		}
		err := req.Validate()
		require.Error(t, err)
	})

	t.Run("invalid email format", func(t *testing.T) {
		t.Parallel()
		req := &LoginRequest{
			EmailAddress: "not-an-email",
			Password:     "password123",
		}
		err := req.Validate()
		require.Error(t, err)
	})

	t.Run("missing password", func(t *testing.T) {
		t.Parallel()
		req := &LoginRequest{
			EmailAddress: "test@example.com",
		}
		err := req.Validate()
		require.Error(t, err)
	})

	t.Run("both missing", func(t *testing.T) {
		t.Parallel()
		req := &LoginRequest{}
		err := req.Validate()
		require.Error(t, err)
	})
}
