//go:build integration

package carriersettlementcontrolrepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol/carriersettlementcontroldb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateKeepsSundayPeriodEndAndNoPayDelay(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID pulid.ID
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	cc := carriersettlementcontroldb.CarrierSettlementControls
	_, err := cc.Where(keel.True()).Delete(ctx, db)
	require.NoError(t, err)

	control := carriersettlementcontrol.NewDefault(orgID, buID)
	control.PeriodEndDayOfWeek = 0
	control.PayDelayDays = 0
	require.NoError(t, cc.Create(ctx, db, control))

	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	got, err := repo.GetOrCreate(ctx, tenant)
	require.NoError(t, err)
	assert.Equal(t, control.ID, got.ID)
	assert.Equal(t, 0, got.PeriodEndDayOfWeek)
	assert.Equal(t, 0, got.PayDelayDays)
}

func TestGetOrCreateWritesTheDefaultsForANewTenant(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID pulid.ID
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	_, err := carriersettlementcontroldb.CarrierSettlementControls.Where(keel.True()).
		Delete(ctx, db)
	require.NoError(t, err)

	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	got, err := repo.GetOrCreate(ctx, tenant)
	require.NoError(t, err)

	want := carriersettlementcontrol.NewDefault(orgID, buID)
	assert.Equal(t, want.PayTrigger, got.PayTrigger)
	assert.Equal(t, want.PayPeriodFrequency, got.PayPeriodFrequency)
	assert.Equal(t, 6, got.PeriodEndDayOfWeek)
	assert.Equal(t, 5, got.PayDelayDays)
	assert.True(t, got.RequireSeparateApprover)
	assert.False(t, got.HoldUntilInvoiceMatched)
}
