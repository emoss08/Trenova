// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestLocationCategoryRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	lcategory := ts.Fixture.MustRow("LocationCategory.location_category_1").(*location.LocationCategory)

	repo := repositories.NewLocationCategoryRepository(
		repositories.LocationCategoryRepositoryParams{
			Logger: logger.NewLogger(testutils.NewTestConfig()),
			DB:     ts.DB,
		},
	)

	t.Run("list", func(t *testing.T) {
		opts := &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list with query", func(t *testing.T) {
		opts := &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
			Query: "Company",
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetLocationCategoryByIDOptions{
			ID:    lcategory.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetLocationCategoryByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "location category not found")
		require.Nil(t, entity)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &location.LocationCategory{
			Name:                "Test Location Category",
			Description:         "Test Location Category Description",
			Type:                location.CategoryWarehouse,
			FacilityType:        location.FacilityTypeColdStorage,
			Color:               "#000000",
			HasSecureParking:    false,
			RequiresAppointment: false,
			AllowsOvernight:     false,
			HasRestroom:         false,
			BusinessUnitID:      bu.ID,
			OrganizationID:      org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("update", func(t *testing.T) {
		lcategory.Description = "Test Location Category 2"
		testutils.TestRepoUpdate(ctx, t, repo, lcategory)
	})

	t.Run("update shipment version lock failure", func(t *testing.T) {
		lcategory.Description = "Test Location Category 2"
		lcategory.Version = 0

		results, err := repo.Update(ctx, lcategory)

		require.Error(t, err)
		require.Nil(t, results)
	})
}
