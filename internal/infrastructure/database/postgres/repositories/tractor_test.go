package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestTractorRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	trk := ts.Fixture.MustRow("Tractor.tractor_1").(*tractor.Tractor)
	wrk := ts.Fixture.MustRow("Worker.worker_1").(*worker.Worker)
	wrk3 := ts.Fixture.MustRow("Worker.worker_3").(*worker.Worker)
	et := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)
	em := ts.Fixture.MustRow("EquipmentManufacturer.freightliner_manufacturer").(*equipmentmanufacturer.EquipmentManufacturer)

	repo := repositories.NewTractorRepository(repositories.TractorRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list tractors", func(t *testing.T) {
		opts := &repoports.ListTractorRequest{
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

	t.Run("list tractors with query", func(t *testing.T) {
		opts := &repoports.ListTractorRequest{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Query: "TRN-001",
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list tractors with workers", func(t *testing.T) {
		opts := &repoports.ListTractorRequest{
			FilterOptions: repoports.TractorFilterOptions{
				IncludeWorkerDetails: true,
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
		require.NotEmpty(t, result.Items[0].PrimaryWorker)
		require.NotEmpty(t, result.Items[0].SecondaryWorker)
	})

	t.Run("list tractors with equipment details", func(t *testing.T) {
		opts := &repoports.ListTractorRequest{
			FilterOptions: repoports.TractorFilterOptions{
				IncludeEquipmentDetails: true,
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
		require.NotEmpty(t, result.Items[0].EquipmentType)
		require.NotEmpty(t, result.Items[0].EquipmentManufacturer)
	})

	t.Run("list tractors with fleet details", func(t *testing.T) {
		opts := &repoports.ListTractorRequest{
			FilterOptions: repoports.TractorFilterOptions{
				IncludeFleetDetails: true,
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
		require.NotEmpty(t, result.Items[0].FleetCode)
	})

	t.Run("get tractor by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, &repoports.GetTractorByIDRequest{
			TractorID: trk.ID,
			OrgID:     org.ID,
			BuID:      bu.ID,
		})
	})

	t.Run("get tractor with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, &repoports.GetTractorByIDRequest{
			TractorID: "invalid-id",
			OrgID:     org.ID,
			BuID:      bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, entity)
	})

	t.Run("get tractor by id with workers", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, &repoports.GetTractorByIDRequest{
			TractorID: trk.ID,
			OrgID:     org.ID,
			BuID:      bu.ID,
			FilterOptions: repoports.TractorFilterOptions{
				IncludeWorkerDetails: true,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.PrimaryWorker)
		require.NotEmpty(t, entity.SecondaryWorker)
	})

	t.Run("get tractor by id with equipment details", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, &repoports.GetTractorByIDRequest{
			TractorID: trk.ID,
			OrgID:     org.ID,
			BuID:      bu.ID,
			FilterOptions: repoports.TractorFilterOptions{
				IncludeEquipmentDetails: true,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.EquipmentManufacturer)
		require.NotEmpty(t, entity.EquipmentType)
	})

	t.Run("get tractor by id with fleet details", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, &repoports.GetTractorByIDRequest{
			TractorID: trk.ID,
			OrgID:     org.ID,
			BuID:      bu.ID,
			FilterOptions: repoports.TractorFilterOptions{
				IncludeFleetDetails: true,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.FleetCode)
	})

	t.Run("get tractor by id failure", func(t *testing.T) {
		result, err := repo.GetByID(ctx, &repoports.GetTractorByIDRequest{
			TractorID: "invalid-id",
			OrgID:     org.ID,
			BuID:      bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("create tractor", func(t *testing.T) {
		// Test Data
		newEntity := &tractor.Tractor{
			Code:                    "TRN-1",
			PrimaryWorkerID:         wrk3.ID,
			EquipmentTypeID:         et.ID,
			EquipmentManufacturerID: em.ID,
			BusinessUnitID:          bu.ID,
			OrganizationID:          org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("create tractor failure", func(t *testing.T) {
		// Test Data
		newEntity := &tractor.Tractor{
			Code:                    "TRN-2",
			PrimaryWorkerID:         wrk.ID,
			EquipmentTypeID:         "invalid-id",
			EquipmentManufacturerID: em.ID,
			BusinessUnitID:          bu.ID,
			OrganizationID:          org.ID,
		}

		results, err := repo.Create(ctx, newEntity)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update tractor", func(t *testing.T) {
		trk.Code = "TRN-3"
		testutils.TestRepoUpdate(ctx, t, repo, trk)
	})

	t.Run("update tractor version lock failure", func(t *testing.T) {
		trk.Code = "TRN-4"
		trk.Version = 0

		results, err := repo.Update(ctx, trk)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update tractor with invalid information", func(t *testing.T) {
		trk.Code = "TRN-5"
		trk.EquipmentTypeID = "invalid-id"

		results, err := repo.Update(ctx, trk)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("get tractor by primary worker id", func(t *testing.T) {
		entity, err := repo.GetByPrimaryWorkerID(ctx, repoports.GetTractorByPrimaryWorkerIDRequest{
			WorkerID: wrk.ID,
			OrgID:    org.ID,
			BuID:     wrk.BusinessUnitID,
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.ID)
		require.Equal(t, entity.PrimaryWorkerID, wrk.ID)
		require.Equal(t, entity.BusinessUnitID, wrk.BusinessUnitID)
		require.Equal(t, entity.OrganizationID, org.ID)
	})

	t.Run("get tractor by primary worker id failure", func(t *testing.T) {
		entity, err := repo.GetByPrimaryWorkerID(ctx, repoports.GetTractorByPrimaryWorkerIDRequest{
			WorkerID: "invalid-id",
			OrgID:    org.ID,
			BuID:     wrk.BusinessUnitID,
		})

		require.Error(t, err)
		require.Nil(t, entity)
	})
}
