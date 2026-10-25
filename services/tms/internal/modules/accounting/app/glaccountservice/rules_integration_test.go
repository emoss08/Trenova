//go:build integration

package glaccountservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rulesFixture struct {
	db            *accountingtest.Database
	accountTypeID pulid.ID
}

func newRulesFixture(t *testing.T) *rulesFixture {
	t.Helper()

	db := accountingtest.NewDatabase(t)
	at := &accounttype.AccountType{
		Status:   domaintypes.StatusActive,
		Code:     "AST",
		Name:     "Assets",
		Category: accounttype.CategoryAsset,
	}
	require.NoError(t, accounttypedb.AccountTypes.Create(db.A.Ctx, db.Admin, at))

	return &rulesFixture{db: db, accountTypeID: at.ID}
}

func (f *rulesFixture) account(
	t *testing.T,
	code string,
	status domaintypes.Status,
	parentID pulid.ID,
) *glaccount.GLAccount {
	t.Helper()

	account := &glaccount.GLAccount{
		Status:        status,
		AccountTypeID: f.accountTypeID,
		ParentID:      parentID,
		AccountCode:   code,
		Name:          "Account " + code,
	}
	require.NoError(t, glaccountdb.GLAccounts.Create(f.db.A.Ctx, f.db.Admin, account))

	return account
}

func (f *rulesFixture) run(
	t *testing.T,
	rule validationframework.TenantedRule[*glaccount.GLAccount],
	entity *glaccount.GLAccount,
	mode validationframework.ValidationMode,
) *errortypes.MultiError {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	require.NoError(t, rule.Validate(f.db.A.Ctx, entity, &validationframework.TenantedValidationContext{
		Mode:           mode,
		OrganizationID: f.db.A.Info.OrgID,
		BusinessUnitID: f.db.A.Info.BuID,
		EntityID:       entity.ID,
	}, multiErr))

	return multiErr
}

func (f *rulesFixture) draft(code string, parentID pulid.ID) *glaccount.GLAccount {
	return &glaccount.GLAccount{
		ID:            pulid.MustNew("gla_"),
		Status:        domaintypes.StatusActive,
		AccountTypeID: f.accountTypeID,
		ParentID:      parentID,
		AccountCode:   code,
		Name:          "Draft " + code,
	}
}

func TestParentAccountActiveRule(t *testing.T) {
	f := newRulesFixture(t)
	rule := createParentAccountActiveRule(f.db.Conn)
	active := f.account(t, "1000", domaintypes.StatusActive, pulid.Nil)
	inactive := f.account(t, "1100", domaintypes.StatusInactive, pulid.Nil)

	t.Run("skips when no parent", func(t *testing.T) {
		assert.False(t, f.run(t, rule, f.draft("2000", pulid.Nil), validationframework.ModeCreate).HasErrors())
	})

	t.Run("passes when parent is active", func(t *testing.T) {
		assert.False(t, f.run(t, rule, f.draft("2001", active.ID), validationframework.ModeCreate).HasErrors())
	})

	t.Run("fails when parent not found", func(t *testing.T) {
		multiErr := f.run(t, rule, f.draft("2002", pulid.MustNew("gla_")), validationframework.ModeCreate)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "parentId", multiErr.Errors[0].Field)
	})

	t.Run("fails when parent is inactive", func(t *testing.T) {
		multiErr := f.run(t, rule, f.draft("2003", inactive.ID), validationframework.ModeCreate)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Errors[0].Message, "active")
	})

	t.Run("does not see another tenant's parent", func(t *testing.T) {
		other := &glaccount.GLAccount{
			Status:        domaintypes.StatusActive,
			AccountTypeID: f.accountTypeID,
			AccountCode:   "1000",
			Name:          "Other tenant",
		}
		at := &accounttype.AccountType{
			Status:   domaintypes.StatusActive,
			Code:     "AST",
			Name:     "Assets",
			Category: accounttype.CategoryAsset,
		}
		require.NoError(t, accounttypedb.AccountTypes.Create(f.db.B.Ctx, f.db.Admin, at))
		other.AccountTypeID = at.ID
		require.NoError(t, glaccountdb.GLAccounts.Create(f.db.B.Ctx, f.db.Admin, other))

		multiErr := f.run(t, rule, f.draft("2004", other.ID), validationframework.ModeCreate)
		assert.True(t, multiErr.HasErrors())
	})
}

func TestCircularReferenceRule(t *testing.T) {
	f := newRulesFixture(t)
	rule := createCircularReferenceRule(f.db.Conn)

	t.Run("skips when no parent", func(t *testing.T) {
		assert.False(t, f.run(t, rule, f.draft("3000", pulid.Nil), validationframework.ModeCreate).HasErrors())
	})

	t.Run("fails when self-referencing", func(t *testing.T) {
		entity := f.draft("3001", pulid.Nil)
		entity.ParentID = entity.ID
		multiErr := f.run(t, rule, entity, validationframework.ModeCreate)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "parentId", multiErr.Errors[0].Field)
	})

	t.Run("passes with a parent chain that ends", func(t *testing.T) {
		root := f.account(t, "3100", domaintypes.StatusActive, pulid.Nil)
		child := f.account(t, "3110", domaintypes.StatusActive, root.ID)
		assert.False(t, f.run(t, rule, f.draft("3111", child.ID), validationframework.ModeCreate).HasErrors())
	})

	t.Run("passes when the parent does not exist", func(t *testing.T) {
		assert.False(t, f.run(t, rule, f.draft("3112", pulid.MustNew("gla_")), validationframework.ModeCreate).HasErrors())
	})

	t.Run("fails when the chain leads back to the account", func(t *testing.T) {
		root := f.account(t, "3200", domaintypes.StatusActive, pulid.Nil)
		child := f.account(t, "3210", domaintypes.StatusActive, root.ID)
		root.ParentID = child.ID
		multiErr := f.run(t, rule, root, validationframework.ModeUpdate)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Errors[0].Message, "Circular")
	})

	t.Run("fails when the hierarchy exceeds the maximum depth", func(t *testing.T) {
		parent := pulid.Nil
		for i := range maxHierarchyDepth + 1 {
			parent = f.account(t, "33"+string(rune('A'+i)), domaintypes.StatusActive, parent).ID
		}
		multiErr := f.run(t, rule, f.draft("3399", parent), validationframework.ModeCreate)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Errors[0].Message, "too deep")
	})
}

func TestDeactivationProtectionRule(t *testing.T) {
	f := newRulesFixture(t)
	rule := createDeactivationProtectionRule(f.db.Conn)

	t.Run("skips when status is active", func(t *testing.T) {
		account := f.account(t, "4000", domaintypes.StatusActive, pulid.Nil)
		f.account(t, "4010", domaintypes.StatusActive, account.ID)
		assert.False(t, f.run(t, rule, account, validationframework.ModeUpdate).HasErrors())
	})

	t.Run("passes deactivation with no children and zero balance", func(t *testing.T) {
		account := f.account(t, "4100", domaintypes.StatusActive, pulid.Nil)
		f.account(t, "4110", domaintypes.StatusInactive, account.ID)
		account.Status = domaintypes.StatusInactive
		assert.False(t, f.run(t, rule, account, validationframework.ModeUpdate).HasErrors())
	})

	t.Run("fails deactivation with active children", func(t *testing.T) {
		account := f.account(t, "4200", domaintypes.StatusActive, pulid.Nil)
		f.account(t, "4210", domaintypes.StatusActive, account.ID)
		account.Status = domaintypes.StatusInactive
		multiErr := f.run(t, rule, account, validationframework.ModeUpdate)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "status", multiErr.Errors[0].Field)
	})

	t.Run("fails deactivation with non-zero balance", func(t *testing.T) {
		account := f.account(t, "4300", domaintypes.StatusActive, pulid.Nil)
		account.Status = domaintypes.StatusInactive
		account.CurrentBalance = 500
		multiErr := f.run(t, rule, account, validationframework.ModeUpdate)
		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "currentBalance", multiErr.Errors[0].Field)
	})
}
