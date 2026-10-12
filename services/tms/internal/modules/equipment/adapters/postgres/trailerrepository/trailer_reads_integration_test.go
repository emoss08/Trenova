//go:build integration

package trailerrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/trailerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetByID_ReadsTheOwnerOperatorAndWhereTheTrailerWasLeft(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment, Force: true})
	require.NoError(t, err)

	var seeded struct {
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
		TrailerID      pulid.ID `bun:"trailer_id"`
		WorkerID       pulid.ID `bun:"worker_id"`
		LocationID     pulid.ID `bun:"location_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("trailers AS tr").
		ColumnExpr("tr.organization_id, tr.business_unit_id, tr.id AS trailer_id").
		ColumnExpr("(SELECT w.id FROM workers AS w WHERE w.organization_id = tr.organization_id AND w.business_unit_id = tr.business_unit_id ORDER BY w.id LIMIT 1) AS worker_id").
		ColumnExpr("(SELECT l.id FROM locations AS l WHERE l.organization_id = tr.organization_id AND l.business_unit_id = tr.business_unit_id ORDER BY l.id LIMIT 1) AS location_id").
		OrderExpr("tr.id").
		Limit(1).
		Scan(ctx, &seeded))
	require.False(t, seeded.WorkerID.IsNil())
	require.False(t, seeded.LocationID.IsNil())

	_, err = db.NewUpdate().
		Table("trailers").
		Set("ownership_type = ?", domaintypes.OwnershipTypeOwnerOperator).
		Set("owner_worker_id = ?", seeded.WorkerID).
		Where("id = ?", seeded.TrailerID).
		Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewUpdate().
		Table("locations").
		Set("latitude = ?", 35.4676).
		Set("longitude = ?", -97.5164).
		Where("id = ?", seeded.LocationID).
		Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&equipmentcontinuity.EquipmentContinuity{
		ID:                pulid.MustNew("ec_"),
		OrganizationID:    seeded.OrganizationID,
		BusinessUnitID:    seeded.BusinessUnitID,
		EquipmentType:     equipmentcontinuity.EquipmentTypeTrailer,
		EquipmentID:       seeded.TrailerID,
		CurrentLocationID: seeded.LocationID,
		SourceType:        equipmentcontinuity.SourceTypeManualLocate,
		IsCurrent:         true,
	}).Exec(ctx)
	require.NoError(t, err)

	repo := trailerrepository.New(trailerrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	got, err := repo.GetByID(ctx, trailerports.GetTrailerByIDRequest{
		ID:         seeded.TrailerID,
		TenantInfo: pagination.TenantInfo{OrgID: seeded.OrganizationID, BuID: seeded.BusinessUnitID},
		TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
			IncludeOwnerWorker:       true,
			IncludeLastKnownLocation: true,
		},
	})
	require.NoError(t, err)

	require.NotNil(t, got.OwnerWorker)
	assert.Equal(t, seeded.WorkerID, got.OwnerWorker.ID)
	assert.Equal(t, seeded.LocationID, got.LastKnownLocationID)
	require.NotNil(t, got.LastKnownLocationLatitude)
	assert.InDelta(t, 35.4676, *got.LastKnownLocationLatitude, 0.0001)
	require.NotNil(t, got.LastKnownLocationLongitude)
	assert.InDelta(t, -97.5164, *got.LastKnownLocationLongitude, 0.0001)
}
