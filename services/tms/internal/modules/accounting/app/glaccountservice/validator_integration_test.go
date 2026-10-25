//go:build integration

package glaccountservice

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testFixture struct {
	ctx   context.Context
	db    *keel.DB
	conn  *postgres.Connection
	v     *Validator
	orgID pulid.ID
	buID  pulid.ID
}

func setupFixture(t *testing.T) *testFixture {
	t.Helper()

	db := accountingtest.NewDevelopmentDatabase(t)

	return &testFixture{
		ctx:   db.A.Ctx,
		db:    db.Admin,
		conn:  db.Conn,
		v:     NewTestValidatorWithDB(db.Conn),
		orgID: db.A.Info.OrgID,
		buID:  db.A.Info.BuID,
	}
}

func (f *testFixture) getAccountTypeID(t *testing.T) pulid.ID {
	t.Helper()

	at := accounttypedb.AccountTypes
	accountType, err := at.Pick(at.ID).Order(at.ID.Asc()).First(f.ctx, f.db)
	require.NoError(t, err)

	return accountType.ID
}

func (f *testFixture) firstAccount(t *testing.T, preds ...keel.Pred) *glaccount.GLAccount {
	t.Helper()

	gla := glaccountdb.GLAccounts
	account, err := gla.Where(preds...).Order(gla.ID.Asc()).First(f.ctx, f.db)
	require.NoError(t, err)

	return account
}

func (f *testFixture) getExistingAccount(t *testing.T) *glaccount.GLAccount {
	t.Helper()

	gla := glaccountdb.GLAccounts
	return f.firstAccount(t, gla.IsSystem.IsFalse(), gla.ParentID.IsNotNull())
}

func (f *testFixture) getSystemAccount(t *testing.T) *glaccount.GLAccount {
	t.Helper()

	return f.firstAccount(t, glaccountdb.GLAccounts.IsSystem.IsTrue())
}

func (f *testFixture) getParentWithChildren(t *testing.T) *glaccount.GLAccount {
	t.Helper()

	gla := glaccountdb.GLAccounts
	children := gla.Children()
	return f.firstAccount(t, children.Any(children.Status.Eq(domaintypes.StatusActive)))
}

func (f *testFixture) insert(t *testing.T, account *glaccount.GLAccount) {
	t.Helper()

	require.NoError(t, glaccountdb.GLAccounts.Create(f.ctx, f.db, account))
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
	f.insert(t, parent)

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
	f.insert(t, account)

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
	f.insert(t, account1)

	account2 := f.newValidEntity(t)
	account2.ID = pulid.MustNew("gla_")
	account2.AccountCode = "8911"
	f.insert(t, account2)

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
	f.insert(t, accountA)

	accountB := f.newValidEntity(t)
	accountB.ID = pulid.MustNew("gla_")
	accountB.AccountCode = "8921"
	accountB.ParentID = accountA.ID
	f.insert(t, accountB)

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
	f.insert(t, grandparent)

	parent := f.newValidEntity(t)
	parent.ID = pulid.MustNew("gla_")
	parent.AccountCode = "8931"
	parent.ParentID = grandparent.ID
	f.insert(t, parent)

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
