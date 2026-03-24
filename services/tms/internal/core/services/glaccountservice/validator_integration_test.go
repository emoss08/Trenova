//go:build integration

package glaccountservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type testFixture struct {
	ctx   context.Context
	db    *bun.DB
	conn  *postgres.Connection
	v     *Validator
	orgID pulid.ID
	buID  pulid.ID
}

func setupFixture(t *testing.T) *testFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)

	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{
			SystemUserPassword: "integration-system-password",
		},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{
		Environment: common.EnvDevelopment,
		Force:       true,
	})
	require.NoError(t, err)

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	err = db.NewSelect().
		TableExpr("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org)
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	v := NewTestValidatorWithDB(conn)

	return &testFixture{
		ctx:   ctx,
		db:    db,
		conn:  conn,
		v:     v,
		orgID: org.ID,
		buID:  org.BusinessUnitID,
	}
}

func (f *testFixture) getAccountTypeID(t *testing.T) pulid.ID {
	t.Helper()

	var atID pulid.ID
	err := f.db.NewSelect().
		Model((*accounttype.AccountType)(nil)).
		Column("id").
		Where("organization_id = ?", f.orgID).
		Where("business_unit_id = ?", f.buID).
		Limit(1).
		Scan(f.ctx, &atID)
	require.NoError(t, err)

	return atID
}

func (f *testFixture) getExistingAccount(t *testing.T) *glaccount.GLAccount {
	t.Helper()

	account := new(glaccount.GLAccount)
	err := f.db.NewSelect().
		Model(account).
		Where("organization_id = ?", f.orgID).
		Where("business_unit_id = ?", f.buID).
		Where("is_system = ?", false).
		Where("parent_id IS NOT NULL").
		Limit(1).
		Scan(f.ctx)
	require.NoError(t, err)

	return account
}

func (f *testFixture) getSystemAccount(t *testing.T) *glaccount.GLAccount {
	t.Helper()

	account := new(glaccount.GLAccount)
	err := f.db.NewSelect().
		Model(account).
		Where("organization_id = ?", f.orgID).
		Where("business_unit_id = ?", f.buID).
		Where("is_system = ?", true).
		Limit(1).
		Scan(f.ctx)
	require.NoError(t, err)

	return account
}

func (f *testFixture) getParentWithChildren(t *testing.T) *glaccount.GLAccount {
	t.Helper()

	account := new(glaccount.GLAccount)
	err := f.db.NewSelect().
		Model(account).
		Where("gla.organization_id = ?", f.orgID).
		Where("gla.business_unit_id = ?", f.buID).
		Where("EXISTS (SELECT 1 FROM gl_accounts c WHERE c.parent_id = gla.id AND c.status = 'Active')").
		Limit(1).
		Scan(f.ctx)
	require.NoError(t, err)

	return account
}

func (f *testFixture) newValidEntity(t *testing.T) *glaccount.GLAccount {
	t.Helper()

	return &glaccount.GLAccount{
		BusinessUnitID: f.buID,
		OrganizationID: f.orgID,
		Status:         domaintypes.StatusActive,
		AccountTypeID:  f.getAccountTypeID(t),
		AccountCode:    "9999",
		Name:           "Test Account",
	}
}

// --- Create Validation Tests ---

func TestIntegration_ValidateCreate_Success(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	multiErr := f.v.ValidateCreate(f.ctx, entity)
	assert.Nil(t, multiErr)
}

func TestIntegration_ValidateCreate_MissingRequiredFields(t *testing.T) {
	f := setupFixture(t)

	entity := &glaccount.GLAccount{
		BusinessUnitID: f.buID,
		OrganizationID: f.orgID,
	}

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_InvalidStatus(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	entity.Status = "BadStatus"

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_DuplicateAccountCode(t *testing.T) {
	f := setupFixture(t)

	existing := f.getExistingAccount(t)

	entity := f.newValidEntity(t)
	entity.AccountCode = existing.AccountCode

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_AccountTypeNotFound(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	entity.AccountTypeID = pulid.MustNew("at_")

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_ParentNotFound(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	entity.ParentID = pulid.MustNew("gla_")

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_ParentExists(t *testing.T) {
	f := setupFixture(t)

	parent := f.getExistingAccount(t)

	entity := f.newValidEntity(t)
	entity.ParentID = parent.ID

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	assert.Nil(t, multiErr)
}

func TestIntegration_ValidateCreate_ParentInactive(t *testing.T) {
	f := setupFixture(t)

	parent := f.newValidEntity(t)
	parent.ID = pulid.MustNew("gla_")
	parent.AccountCode = "8800"
	parent.Status = domaintypes.StatusInactive
	_, err := f.db.NewInsert().Model(parent).Exec(f.ctx)
	require.NoError(t, err)

	entity := f.newValidEntity(t)
	entity.AccountCode = "8801"
	entity.ParentID = parent.ID

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_NegativeDebitBalance(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	entity.DebitBalance = -500

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_NegativeCreditBalance(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	entity.CreditBalance = -100

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_BothNegativeBalances(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	entity.DebitBalance = -100
	entity.CreditBalance = -200

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_SelfReferencing(t *testing.T) {
	f := setupFixture(t)

	entityID := pulid.MustNew("gla_")

	entity := f.newValidEntity(t)
	entity.ID = entityID
	entity.ParentID = entityID

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

// --- Update Validation Tests ---

func TestIntegration_ValidateUpdate_Success(t *testing.T) {
	f := setupFixture(t)

	existing := f.getExistingAccount(t)

	entity := f.newValidEntity(t)
	entity.ID = existing.ID
	entity.AccountCode = existing.AccountCode
	entity.Version = existing.Version

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	assert.Nil(t, multiErr)
}

func TestIntegration_ValidateUpdate_SystemAccountBlocked(t *testing.T) {
	f := setupFixture(t)

	sysAccount := f.getSystemAccount(t)

	entity := f.newValidEntity(t)
	entity.ID = sysAccount.ID
	entity.AccountCode = sysAccount.AccountCode
	entity.IsSystem = true
	entity.Version = sysAccount.Version

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateUpdate_DeactivateWithActiveChildren(t *testing.T) {
	f := setupFixture(t)

	parent := f.getParentWithChildren(t)

	entity := f.newValidEntity(t)
	entity.ID = parent.ID
	entity.AccountCode = parent.AccountCode
	entity.Status = domaintypes.StatusInactive
	entity.Version = parent.Version

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateUpdate_DeactivateWithNonZeroBalance(t *testing.T) {
	f := setupFixture(t)

	entity := f.newValidEntity(t)
	entity.ID = pulid.MustNew("gla_")
	entity.AccountCode = "8900"
	entity.Status = domaintypes.StatusInactive
	entity.CurrentBalance = 5000
	entity.Version = 1

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateUpdate_DeactivateClean(t *testing.T) {
	f := setupFixture(t)

	account := f.newValidEntity(t)
	account.ID = pulid.MustNew("gla_")
	account.AccountCode = "8901"
	_, err := f.db.NewInsert().Model(account).Exec(f.ctx)
	require.NoError(t, err)

	entity := f.newValidEntity(t)
	entity.ID = account.ID
	entity.AccountCode = account.AccountCode
	entity.Status = domaintypes.StatusInactive
	entity.CurrentBalance = 0
	entity.Version = 1

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	assert.Nil(t, multiErr)
}

func TestIntegration_ValidateUpdate_DuplicateAccountCode(t *testing.T) {
	f := setupFixture(t)

	account1 := f.newValidEntity(t)
	account1.ID = pulid.MustNew("gla_")
	account1.AccountCode = "8910"
	_, err := f.db.NewInsert().Model(account1).Exec(f.ctx)
	require.NoError(t, err)

	account2 := f.newValidEntity(t)
	account2.ID = pulid.MustNew("gla_")
	account2.AccountCode = "8911"
	_, err = f.db.NewInsert().Model(account2).Exec(f.ctx)
	require.NoError(t, err)

	entity := f.newValidEntity(t)
	entity.ID = account2.ID
	entity.AccountCode = "8910"
	entity.Version = 1

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateUpdate_SameAccountCodeOwnRecord(t *testing.T) {
	f := setupFixture(t)

	existing := f.getExistingAccount(t)

	entity := f.newValidEntity(t)
	entity.ID = existing.ID
	entity.AccountCode = existing.AccountCode
	entity.Version = existing.Version

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	assert.Nil(t, multiErr)
}

// --- Circular Reference Detection ---

func TestIntegration_ValidateUpdate_CircularReference(t *testing.T) {
	f := setupFixture(t)

	accountA := f.newValidEntity(t)
	accountA.ID = pulid.MustNew("gla_")
	accountA.AccountCode = "8920"
	_, err := f.db.NewInsert().Model(accountA).Exec(f.ctx)
	require.NoError(t, err)

	accountB := f.newValidEntity(t)
	accountB.ID = pulid.MustNew("gla_")
	accountB.AccountCode = "8921"
	accountB.ParentID = accountA.ID
	_, err = f.db.NewInsert().Model(accountB).Exec(f.ctx)
	require.NoError(t, err)

	entity := f.newValidEntity(t)
	entity.ID = accountA.ID
	entity.AccountCode = "8920"
	entity.ParentID = accountB.ID
	entity.Version = 1

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIntegration_ValidateCreate_ValidParentChain(t *testing.T) {
	f := setupFixture(t)

	grandparent := f.newValidEntity(t)
	grandparent.ID = pulid.MustNew("gla_")
	grandparent.AccountCode = "8930"
	_, err := f.db.NewInsert().Model(grandparent).Exec(f.ctx)
	require.NoError(t, err)

	parent := f.newValidEntity(t)
	parent.ID = pulid.MustNew("gla_")
	parent.AccountCode = "8931"
	parent.ParentID = grandparent.ID
	_, err = f.db.NewInsert().Model(parent).Exec(f.ctx)
	require.NoError(t, err)

	entity := f.newValidEntity(t)
	entity.AccountCode = "8932"
	entity.ParentID = parent.ID

	multiErr := f.v.ValidateCreate(f.ctx, entity)
	assert.Nil(t, multiErr)
}

func TestIntegration_ValidateUpdate_DeactivateWithChildrenAndBalance(t *testing.T) {
	f := setupFixture(t)

	parent := f.getParentWithChildren(t)

	entity := f.newValidEntity(t)
	entity.ID = parent.ID
	entity.AccountCode = parent.AccountCode
	entity.Status = domaintypes.StatusInactive
	entity.CurrentBalance = 1000
	entity.Version = parent.Version

	multiErr := f.v.ValidateUpdate(f.ctx, entity)
	require.NotNil(t, multiErr)

	errCount := 0
	for _, e := range multiErr.Errors {
		if e.Field == "status" || e.Field == "currentBalance" {
			errCount++
		}
	}
	assert.Equal(t, 2, errCount)
}
