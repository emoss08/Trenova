//go:build integration

package dashcontrolrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol/dashcontroldb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func freshTenant(t *testing.T) (context.Context, *keel.DB, pagination.TenantInfo) {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID pulid.ID
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	_, err := dashcontroldb.DashControls.Where(keel.True()).Delete(ctx, db)
	require.NoError(t, err)

	return ctx, db, tenant
}

func TestGetOrCreateWritesTheDigestDefaultsForANewTenant(t *testing.T) {
	ctx, db, tenant := freshTenant(t)

	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	got, err := repo.GetOrCreate(ctx, tenant)
	require.NoError(t, err)

	assert.Equal(t, dashcontrol.DigestImmediate, got.DriverDigestCadence)
	assert.Equal(t, dashcontrol.DefaultDriverDigestWeekday, got.DriverDigestWeekday)
	assert.True(t, got.EnableDetentionAlerts)
	assert.Equal(t, dashcontrol.DefaultDetentionAlertMinutes, got.DetentionAlertThresholdMinutes)
}

func TestUpdatePersistsTheDigestCadenceAndSunday(t *testing.T) {
	ctx, db, tenant := freshTenant(t)

	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	current, err := repo.GetOrCreate(ctx, tenant)
	require.NoError(t, err)

	current.DriverDigestCadence = dashcontrol.DigestWeekly
	current.DriverDigestWeekday = 0
	current.EnableDetentionAlerts = false
	current.DetentionAlertThresholdMinutes = 45

	updated, err := repo.Update(ctx, current)
	require.NoError(t, err)

	assert.Equal(t, dashcontrol.DigestWeekly, updated.DriverDigestCadence)
	assert.Equal(t, int16(0), updated.DriverDigestWeekday)
	assert.False(t, updated.EnableDetentionAlerts)
	assert.Equal(t, int16(45), updated.DetentionAlertThresholdMinutes)

	stored, err := dashcontroldb.DashControls.Where(keel.True()).One(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, dashcontrol.DigestWeekly, stored.DriverDigestCadence)
	assert.Equal(t, int16(0), stored.DriverDigestWeekday)
}
