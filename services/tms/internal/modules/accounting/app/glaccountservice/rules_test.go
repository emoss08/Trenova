package glaccountservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGLAccount() *glaccount.GLAccount {
	return &glaccount.GLAccount{
		ID:             pulid.MustNew("gla_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		AccountTypeID:  pulid.MustNew("at_"),
		AccountCode:    "1000",
		Name:           "Cash",
	}
}

func newValCtx(
	entity *glaccount.GLAccount,
	mode validationframework.ValidationMode,
) *validationframework.TenantedValidationContext {
	return &validationframework.TenantedValidationContext{
		Mode:           mode,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		EntityID:       entity.ID,
	}
}

func runRule(
	t *testing.T,
	rule validationframework.TenantedRule[*glaccount.GLAccount],
	entity *glaccount.GLAccount,
	mode validationframework.ValidationMode,
) *errortypes.MultiError {
	t.Helper()
	ctx := t.Context()
	multiErr := errortypes.NewMultiError()
	valCtx := newValCtx(entity, mode)
	err := rule.Validate(ctx, entity, valCtx, multiErr)
	require.NoError(t, err)
	return multiErr
}

func TestSystemAccountProtectionRule(t *testing.T) {
	t.Parallel()

	t.Run("passes for non-system account", func(t *testing.T) {
		t.Parallel()
		entity := newTestGLAccount()
		entity.IsSystem = false

		rule := createSystemAccountProtectionRule()
		multiErr := runRule(t, rule, entity, validationframework.ModeUpdate)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("fails for system account", func(t *testing.T) {
		t.Parallel()
		entity := newTestGLAccount()
		entity.IsSystem = true

		rule := createSystemAccountProtectionRule()
		multiErr := runRule(t, rule, entity, validationframework.ModeUpdate)

		require.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "isSystem", multiErr.Errors[0].Field)
		assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
	})
}

func TestBalanceConsistencyRule(t *testing.T) {
	t.Parallel()

	t.Run("passes with zero balances", func(t *testing.T) {
		t.Parallel()
		entity := newTestGLAccount()

		rule := createBalanceConsistencyRule()
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("passes with positive balances", func(t *testing.T) {
		t.Parallel()
		entity := newTestGLAccount()
		entity.DebitBalance = 1000
		entity.CreditBalance = 500

		rule := createBalanceConsistencyRule()
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("fails with negative debit balance", func(t *testing.T) {
		t.Parallel()
		entity := newTestGLAccount()
		entity.DebitBalance = -100

		rule := createBalanceConsistencyRule()
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "debitBalance", multiErr.Errors[0].Field)
	})

	t.Run("fails with negative credit balance", func(t *testing.T) {
		t.Parallel()
		entity := newTestGLAccount()
		entity.CreditBalance = -200

		rule := createBalanceConsistencyRule()
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "creditBalance", multiErr.Errors[0].Field)
	})

	t.Run("fails with both negative balances", func(t *testing.T) {
		t.Parallel()
		entity := newTestGLAccount()
		entity.DebitBalance = -100
		entity.CreditBalance = -200

		rule := createBalanceConsistencyRule()
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 2)
	})
}
