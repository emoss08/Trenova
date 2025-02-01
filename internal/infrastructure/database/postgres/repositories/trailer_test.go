package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestTrailerRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	trail := ts.Fixture.MustRow("Trailer.test_trailer").(*trailer.Trailer)
	et := ts.Fixture.MustRow("EquipmentType.tractor_equip_type").(*equipmenttype.EquipmentType)
	em := ts.Fixture.MustRow("EquipmentManufacturer.freightliner_manufacturer").(*equipmentmanufacturer.EquipmentManufacturer)

	repo := repositories.NewTrailerRepository(repositories.TrailerRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list trailers", func(t *testing.T) {
		opts := &repoports.ListTrailerOptions{
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

	t.Run("list trailers with query", func(t *testing.T) {
		opts := &repoports.ListTrailerOptions{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Query: "TRL-001",
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list trailers with equipment details", func(t *testing.T) {
		opts := &repoports.ListTrailerOptions{
			IncludeEquipmentDetails: true,
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

	t.Run("list trailers with fleet details", func(t *testing.T) {
		opts := &repoports.ListTrailerOptions{
			IncludeFleetDetails: true,
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

	t.Run("get trailer by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetTrailerByIDOptions{
			ID:    trail.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get trailer with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetTrailerByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, entity)
	})

	t.Run("get trailer by id with equipment details", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetTrailerByIDOptions{
			ID:                      trail.ID,
			OrgID:                   org.ID,
			BuID:                    bu.ID,
			IncludeEquipmentDetails: true,
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.EquipmentType)
		require.NotEmpty(t, entity.EquipmentManufacturer)
	})

	t.Run("get trailer by id with fleet details", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetTrailerByIDOptions{
			ID:                  trail.ID,
			OrgID:               org.ID,
			BuID:                bu.ID,
			IncludeFleetDetails: true,
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.FleetCode)
	})

	t.Run("get trailer by id failure", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetTrailerByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("create trailer", func(t *testing.T) {
		// Test Data
		newEntity := &trailer.Trailer{
			Code:                    "TRL-1",
			EquipmentTypeID:         et.ID,
			EquipmentManufacturerID: em.ID,
			BusinessUnitID:          bu.ID,
			OrganizationID:          org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("create tractor failure", func(t *testing.T) {
		// Test Data
		newEntity := &trailer.Trailer{
			Code:                    "TRL-2",
			EquipmentTypeID:         "invalid-id",
			EquipmentManufacturerID: em.ID,
			BusinessUnitID:          bu.ID,
			OrganizationID:          org.ID,
		}

		results, err := repo.Create(ctx, newEntity)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update trailer", func(t *testing.T) {
		trail.Code = "TRL-3"
		testutils.TestRepoUpdate(ctx, t, repo, trail)
	})

	t.Run("update trailer version lock failure", func(t *testing.T) {
		trail.Code = "TRL-4"
		trail.Version = 0

		results, err := repo.Update(ctx, trail)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update tractor with invalid information", func(t *testing.T) {
		trail.Code = "TRL-5"
		trail.EquipmentTypeID = "invalid-id"

		results, err := repo.Update(ctx, trail)

		require.Error(t, err)
		require.Nil(t, results)
	})
}
