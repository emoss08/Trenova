package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestShipmentRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	smt := ts.Fixture.MustRow("Shipment.test_shipment").(*shipment.Shipment)

	repo := repositories.NewShipmentRepository(repositories.ShipmentRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list shipments", func(t *testing.T) {
		opts := &repoports.ListShipmentOptions{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list shipments with query", func(t *testing.T) {
		opts := &repoports.ListShipmentOptions{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Query: "12",
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list shipments with details", func(t *testing.T) {
		opts := &repoports.ListShipmentOptions{
			ShipmentOptions: repoports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		result, err := repo.List(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.Items)
		require.NotEmpty(t, result.Items[0].Customer)
		require.NotEmpty(t, result.Items[0].Moves)
		require.NotEmpty(t, result.Items[0].Moves[0].Stops) // Include the movement stops
		require.NotEmpty(t, result.Items[0].TractorType)
		require.NotEmpty(t, result.Items[0].TrailerType)
		require.NotEmpty(t, result.Items[0].Commodities)
	})

	t.Run("get shipment by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetShipmentByIDOptions{
			ID:    smt.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get tractor with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetShipmentByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, entity)
	})

	t.Run("get shipment by id with details", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetShipmentByIDOptions{
			ShipmentOptions: repoports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
			ID:    smt.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.Customer)
		require.NotEmpty(t, entity.Moves)
		require.NotEmpty(t, entity.Moves[0].Stops)
		require.NotEmpty(t, entity.TractorType)
		require.NotEmpty(t, entity.TrailerType)
		require.NotEmpty(t, entity.Commodities)
	})

	// t.Run("get tractor by id with equipment details", func(t *testing.T) {
	// 	entity, err := repo.GetByID(ctx, repoports.GetTractorByIDOptions{
	// 		ID:                      trk.ID,
	// 		OrgID:                   org.ID,
	// 		BuID:                    bu.ID,
	// 		IncludeEquipmentDetails: true,
	// 	})

	// 	require.NoError(t, err)
	// 	require.NotNil(t, entity)
	// 	require.NotEmpty(t, entity.EquipmentManufacturer)
	// 	require.NotEmpty(t, entity.EquipmentType)
	// })

	// t.Run("get tractor by id with fleet details", func(t *testing.T) {
	// 	entity, err := repo.GetByID(ctx, repoports.GetTractorByIDOptions{
	// 		ID:                  trk.ID,
	// 		OrgID:               org.ID,
	// 		BuID:                bu.ID,
	// 		IncludeFleetDetails: true,
	// 	})

	// 	require.NoError(t, err)
	// 	require.NotNil(t, entity)
	// 	require.NotEmpty(t, entity.FleetCode)
	// })

	// t.Run("get tractor by id failure", func(t *testing.T) {
	// 	result, err := repo.GetByID(ctx, repoports.GetTractorByIDOptions{
	// 		ID:    "invalid-id",
	// 		OrgID: org.ID,
	// 		BuID:  bu.ID,
	// 	})

	// 	require.Error(t, err)
	// 	require.Nil(t, result)
	// })

	// t.Run("create tractor", func(t *testing.T) {
	// 	// Test Data
	// 	newEntity := &tractor.Tractor{
	// 		Code:                    "TRN-1",
	// 		PrimaryWorkerID:         wrk3.ID,
	// 		EquipmentTypeID:         et.ID,
	// 		EquipmentManufacturerID: em.ID,
	// 		BusinessUnitID:          bu.ID,
	// 		OrganizationID:          org.ID,
	// 	}

	// 	testutils.TestRepoCreate(ctx, t, repo, newEntity)
	// })

	// t.Run("create tractor failure", func(t *testing.T) {
	// 	// Test Data
	// 	newEntity := &tractor.Tractor{
	// 		Code:                    "TRN-2",
	// 		PrimaryWorkerID:         wrk.ID,
	// 		EquipmentTypeID:         "invalid-id",
	// 		EquipmentManufacturerID: em.ID,
	// 		BusinessUnitID:          bu.ID,
	// 		OrganizationID:          org.ID,
	// 	}

	// 	results, err := repo.Create(ctx, newEntity)

	// 	require.Error(t, err)
	// 	require.Nil(t, results)
	// })

	// t.Run("update tractor", func(t *testing.T) {
	// 	trk.Code = "TRN-3"
	// 	testutils.TestRepoUpdate(ctx, t, repo, trk)
	// })

	// t.Run("update tractor version lock failure", func(t *testing.T) {
	// 	trk.Code = "TRN-4"
	// 	trk.Version = 0

	// 	results, err := repo.Update(ctx, trk)

	// 	require.Error(t, err)
	// 	require.Nil(t, results)
	// })

	// t.Run("update tractor with invalid information", func(t *testing.T) {
	// 	trk.Code = "TRN-5"
	// 	trk.EquipmentTypeID = "invalid-id"

	// 	results, err := repo.Update(ctx, trk)

	// 	require.Error(t, err)
	// 	require.Nil(t, results)
	// })
}
