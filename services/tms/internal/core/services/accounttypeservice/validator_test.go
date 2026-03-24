package accounttypeservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
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

	entity := &accounttype.AccountType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "AST",
		Name:           "Asset",
		Category:       accounttype.CategoryAsset,
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	assert.Nil(t, multiErr)
}

func TestValidateCreate_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateCreate_InvalidCategory(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "BAD",
		Name:           "Bad Category",
		Category:       "InvalidCategory",
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateCreate_InvalidStatus(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         "InvalidStatus",
		Code:           "AST",
		Name:           "Asset",
		Category:       accounttype.CategoryAsset,
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateCreate_CodeTooShort(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "AB",
		Name:           "Asset",
		Category:       accounttype.CategoryAsset,
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateCreate_InvalidColor(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "AST",
		Name:           "Asset",
		Category:       accounttype.CategoryAsset,
		Color:          "not-a-color",
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateCreate_ValidColor(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "AST",
		Name:           "Asset",
		Category:       accounttype.CategoryAsset,
		Color:          "#FF5733",
	}

	multiErr := v.ValidateCreate(t.Context(), entity)
	assert.Nil(t, multiErr)
}

func TestValidateUpdate_Success(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		ID:             pulid.MustNew("at_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "AST",
		Name:           "Asset",
		Category:       accounttype.CategoryAsset,
		Version:        1,
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)
	assert.Nil(t, multiErr)
}

func TestValidateUpdate_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()

	entity := &accounttype.AccountType{
		ID:             pulid.MustNew("at_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Name:           "",
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestValidateCreate_AllCategories(t *testing.T) {
	t.Parallel()

	categories := []accounttype.Category{
		accounttype.CategoryAsset,
		accounttype.CategoryLiability,
		accounttype.CategoryEquity,
		accounttype.CategoryRevenue,
		accounttype.CategoryCostOfRevenue,
		accounttype.CategoryExpense,
	}

	for _, cat := range categories {
		t.Run(string(cat), func(t *testing.T) {
			t.Parallel()
			v := NewTestValidator()

			entity := &accounttype.AccountType{
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Status:         domaintypes.StatusActive,
				Code:           "TST",
				Name:           "Test",
				Category:       cat,
			}

			multiErr := v.ValidateCreate(t.Context(), entity)
			assert.Nil(t, multiErr)
		})
	}
}
