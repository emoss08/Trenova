package seedhelpers_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/assert"
)

func TestBusinessUnitOptions_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid options pass validation", func(t *testing.T) {
		opts := &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "TEST",
		}

		err := opts.Validate()
		assert.NoError(t, err)
	})

	t.Run("nil options returns error", func(t *testing.T) {
		var opts *seedhelpers.BusinessUnitOptions

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrNilValue)
	})

	t.Run("empty name returns error", func(t *testing.T) {
		opts := &seedhelpers.BusinessUnitOptions{
			Name: "",
			Code: "TEST",
		}

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("empty code returns error", func(t *testing.T) {
		opts := &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "",
		}

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "code")
	})
}

func TestOrganizationOptions_Validate(t *testing.T) {
	t.Parallel()

	validOpts := func() *seedhelpers.OrganizationOptions {
		return &seedhelpers.OrganizationOptions{
			BusinessUnitID: "bu_123",
			Name:           "Test Org",
			ScacCode:       "TEST",
			City:           "Test City",
			StateID:        "st_123",
			Timezone:       "America/Los_Angeles",
			DOTNumber:      "1234567",
			BucketName:     "test-bucket",
		}
	}

	t.Run("valid options pass validation", func(t *testing.T) {
		opts := validOpts()

		err := opts.Validate()
		assert.NoError(t, err)
	})

	t.Run("nil options returns error", func(t *testing.T) {
		var opts *seedhelpers.OrganizationOptions

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrNilValue)
	})

	t.Run("empty business unit ID returns error", func(t *testing.T) {
		opts := validOpts()
		opts.BusinessUnitID = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "business unit ID")
	})

	t.Run("empty name returns error", func(t *testing.T) {
		opts := validOpts()
		opts.Name = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("empty SCAC code returns error", func(t *testing.T) {
		opts := validOpts()
		opts.ScacCode = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "SCAC")
	})

	t.Run("empty city returns error", func(t *testing.T) {
		opts := validOpts()
		opts.City = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "city")
	})

	t.Run("empty state ID returns error", func(t *testing.T) {
		opts := validOpts()
		opts.StateID = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "state ID")
	})

	t.Run("empty timezone returns error", func(t *testing.T) {
		opts := validOpts()
		opts.Timezone = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "timezone")
	})

	t.Run("empty DOT number returns error", func(t *testing.T) {
		opts := validOpts()
		opts.DOTNumber = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "DOT")
	})

	t.Run("empty bucket name returns error", func(t *testing.T) {
		opts := validOpts()
		opts.BucketName = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "bucket")
	})
}

func TestUserOptions_Validate(t *testing.T) {
	t.Parallel()

	validOpts := func() *seedhelpers.UserOptions {
		return &seedhelpers.UserOptions{
			OrganizationID: "org_123",
			BusinessUnitID: "bu_123",
			Name:           "Test User",
			Username:       "testuser",
			Email:          "test@example.com",
			Timezone:       "America/Los_Angeles",
			Status:         domaintypes.StatusActive,
		}
	}

	t.Run("valid options pass validation", func(t *testing.T) {
		opts := validOpts()

		err := opts.Validate()
		assert.NoError(t, err)
	})

	t.Run("nil options returns error", func(t *testing.T) {
		var opts *seedhelpers.UserOptions

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrNilValue)
	})

	t.Run("empty organization ID returns error", func(t *testing.T) {
		opts := validOpts()
		opts.OrganizationID = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "organization ID")
	})

	t.Run("empty business unit ID returns error", func(t *testing.T) {
		opts := validOpts()
		opts.BusinessUnitID = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "business unit ID")
	})

	t.Run("empty name returns error", func(t *testing.T) {
		opts := validOpts()
		opts.Name = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("empty username returns error", func(t *testing.T) {
		opts := validOpts()
		opts.Username = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("empty email returns error", func(t *testing.T) {
		opts := validOpts()
		opts.Email = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("empty timezone returns error", func(t *testing.T) {
		opts := validOpts()
		opts.Timezone = ""

		err := opts.Validate()
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
		assert.Contains(t, err.Error(), "timezone")
	})

	t.Run("password is optional", func(t *testing.T) {
		opts := validOpts()
		opts.Password = ""

		err := opts.Validate()
		assert.NoError(t, err)
	})
}
