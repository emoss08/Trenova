package glaccountservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}

func TestValidateCreate_Success(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &glaccount.GLAccount{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		AccountTypeID:  pulid.MustNew("at_"),
		AccountCode:    "1000",
		Name:           "Cash",
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	assert.Nil(t, multiErr)
}

func TestValidateCreate_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &glaccount.GLAccount{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateUpdate_Success(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &glaccount.GLAccount{
		ID:             pulid.MustNew("gla_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		AccountTypeID:  pulid.MustNew("at_"),
		AccountCode:    "1000",
		Name:           "Cash",
		Version:        1,
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)
	assert.Nil(t, multiErr)
}

func TestValidateUpdate_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &glaccount.GLAccount{
		ID:             pulid.MustNew("gla_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Name:           "",
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateCreate_InvalidStatus(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &glaccount.GLAccount{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         "InvalidStatus",
		AccountTypeID:  pulid.MustNew("at_"),
		AccountCode:    "1000",
		Name:           "Cash",
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}
