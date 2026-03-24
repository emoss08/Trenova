package glaccountservice

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func newTestDBConnection(t *testing.T) (*postgres.Connection, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() { bunDB.Close() })
	return postgres.NewTestConnection(bunDB), mock
}

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

func TestParentAccountActiveRule(t *testing.T) {
	t.Parallel()

	t.Run("skips when no parent", func(t *testing.T) {
		t.Parallel()
		conn, _ := newTestDBConnection(t)
		entity := newTestGLAccount()

		rule := createParentAccountActiveRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("passes when parent is active", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		parentID := pulid.MustNew("gla_")
		entity.ParentID = parentID

		rows := sqlmock.NewRows([]string{"id", "status"}).
			AddRow(parentID.String(), string(domaintypes.StatusActive))
		mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

		rule := createParentAccountActiveRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when parent not found", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.ParentID = pulid.MustNew("gla_")

		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("sql: no rows in result set"))

		rule := createParentAccountActiveRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "parentId", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "Parent account not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when parent is inactive", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		parentID := pulid.MustNew("gla_")
		entity.ParentID = parentID

		rows := sqlmock.NewRows([]string{"id", "status"}).
			AddRow(parentID.String(), string(domaintypes.StatusInactive))
		mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

		rule := createParentAccountActiveRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "parentId", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "Parent account must be active")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCircularReferenceRule(t *testing.T) {
	t.Parallel()

	t.Run("skips when no parent", func(t *testing.T) {
		t.Parallel()
		conn, _ := newTestDBConnection(t)
		entity := newTestGLAccount()

		rule := createCircularReferenceRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("fails when self-referencing", func(t *testing.T) {
		t.Parallel()
		conn, _ := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.ParentID = entity.ID

		rule := createCircularReferenceRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "parentId", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "Account cannot be its own parent")
	})

	t.Run("passes with valid parent chain ending in nil", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		parentID := pulid.MustNew("gla_")
		entity.ParentID = parentID

		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

		rule := createCircularReferenceRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("passes when parent chain ends with empty parent_id", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		parentID := pulid.MustNew("gla_")
		grandparentID := pulid.MustNew("gla_")
		entity.ParentID = parentID

		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(grandparentID.String()))

		emptyStr := ""
		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(emptyStr))

		rule := createCircularReferenceRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when circular reference found in chain", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		parentID := pulid.MustNew("gla_")
		entity.ParentID = parentID

		mock.ExpectQuery(`SELECT`).
			WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(entity.ID.String()))

		rule := createCircularReferenceRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "parentId", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "Circular reference detected")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("passes when parent query returns error", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		parentID := pulid.MustNew("gla_")
		entity.ParentID = parentID

		mock.ExpectQuery(`SELECT`).
			WillReturnError(errors.New("sql: no rows in result set"))

		rule := createCircularReferenceRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		assert.False(t, multiErr.HasErrors())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when hierarchy exceeds max depth", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		parentID := pulid.MustNew("gla_")
		entity.ParentID = parentID

		ids := make([]pulid.ID, maxHierarchyDepth+1)
		ids[0] = parentID
		for i := 1; i <= maxHierarchyDepth; i++ {
			ids[i] = pulid.MustNew("gla_")
		}

		for i := range maxHierarchyDepth {
			mock.ExpectQuery(`SELECT`).
				WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(ids[i+1].String()))
		}

		rule := createCircularReferenceRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeCreate)

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "parentId", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "Account hierarchy is too deep")
		require.NoError(t, mock.ExpectationsWereMet())
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

func TestDeactivationProtectionRule(t *testing.T) {
	t.Parallel()

	t.Run("skips when status is active", func(t *testing.T) {
		t.Parallel()
		conn, _ := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.Status = domaintypes.StatusActive

		rule := createDeactivationProtectionRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeUpdate)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("passes deactivation with no children and zero balance", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.Status = domaintypes.StatusInactive
		entity.CurrentBalance = 0

		mock.ExpectQuery(`SELECT count`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		rule := createDeactivationProtectionRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeUpdate)

		assert.False(t, multiErr.HasErrors())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails deactivation with active children", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.Status = domaintypes.StatusInactive
		entity.CurrentBalance = 0

		mock.ExpectQuery(`SELECT count`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		rule := createDeactivationProtectionRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeUpdate)

		require.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "status", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "active child accounts")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails deactivation with non-zero balance", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.Status = domaintypes.StatusInactive
		entity.CurrentBalance = 5000

		mock.ExpectQuery(`SELECT count`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		rule := createDeactivationProtectionRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeUpdate)

		require.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "currentBalance", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "non-zero balance")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails deactivation with active children and non-zero balance", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.Status = domaintypes.StatusInactive
		entity.CurrentBalance = 1000

		mock.ExpectQuery(`SELECT count`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		rule := createDeactivationProtectionRule(conn)
		multiErr := runRule(t, rule, entity, validationframework.ModeUpdate)

		require.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 2)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when count query fails", func(t *testing.T) {
		t.Parallel()
		conn, mock := newTestDBConnection(t)
		entity := newTestGLAccount()
		entity.Status = domaintypes.StatusInactive

		mock.ExpectQuery(`SELECT count`).
			WillReturnError(errors.New("db error"))

		rule := createDeactivationProtectionRule(conn)
		ctx := t.Context()
		multiErr := errortypes.NewMultiError()
		valCtx := newValCtx(entity, validationframework.ModeUpdate)
		err := rule.Validate(ctx, entity, valCtx, multiErr)

		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
