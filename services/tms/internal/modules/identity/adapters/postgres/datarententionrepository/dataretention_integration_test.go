//go:build integration

package datarententionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpsertSavesTheDriverQualificationRetentionOnInsertAndUpdate(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	tctx := data.Ctx(ctx)
	info := data.Info()
	get := tenantports.GetDataRetentionRequest{OrgID: info.OrgID, BuID: info.BuID}

	entity := func(days int) *dataretention.DataRetention {
		return &dataretention.DataRetention{
			OrganizationID:                     info.OrgID,
			BusinessUnitID:                     info.BuID,
			AIAuditRetentionPeriod:             dataretention.DefaultAIAuditRetentionDays,
			DriverQualificationRetentionPeriod: days,
		}
	}

	if existing, err := repo.Get(tctx, get); err == nil {
		existing.DriverQualificationRetentionPeriod = 1095
		_, err = repo.Upsert(tctx, existing)
		require.NoError(t, err)
	} else {
		_, err = repo.Upsert(tctx, entity(1095))
		require.NoError(t, err)
	}

	_, err := repo.Upsert(tctx, entity(1500))
	require.NoError(t, err)

	saved, err := repo.Get(tctx, get)
	require.NoError(t, err)
	assert.Equal(t, 1500, saved.DriverQualificationRetentionPeriod)
}
