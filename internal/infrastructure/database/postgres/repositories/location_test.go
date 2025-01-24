package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestLocationRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	loc := ts.Fixture.MustRow("Location.test_location").(*location.Location)
	usState := ts.Fixture.MustRow("UsState.ca").(*usstate.UsState)
	locCategory := testutils.FixtureMustRow("LocationCategory.location_category_1").(*location.LocationCategory)

	repo := repositories.NewLocationRepository(repositories.LocationRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list locations", func(t *testing.T) {
		opts := &repoports.ListLocationOptions{
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

	t.Run("list locations with query", func(t *testing.T) {
		opts := &repoports.ListLocationOptions{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
				Query: "Test Location",
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list locations with categories", func(t *testing.T) {
		opts := &repoports.ListLocationOptions{
			IncludeCategory: true,
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
		require.NotEmpty(t, result.Items[0].LocationCategory)
	})

	t.Run("list locations with state", func(t *testing.T) {
		opts := &repoports.ListLocationOptions{
			IncludeState: true,
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
		require.NotEmpty(t, result.Items[0].State)
	})

	t.Run("get location by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetLocationByIDOptions{
			ID:    loc.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get location with invalid id", func(t *testing.T) {
		l, err := repo.GetByID(ctx, repoports.GetLocationByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "location not found")
		require.Nil(t, l)
	})

	t.Run("get location by id with category", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetLocationByIDOptions{
			ID:              loc.ID,
			OrgID:           org.ID,
			BuID:            bu.ID,
			IncludeCategory: true,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.LocationCategory)
	})

	t.Run("get location by id with state", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetLocationByIDOptions{
			ID:           loc.ID,
			OrgID:        org.ID,
			BuID:         bu.ID,
			IncludeState: true,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.State)
	})

	t.Run("get location by id failure", func(t *testing.T) {
		result, err := repo.GetByID(ctx, repoports.GetLocationByIDOptions{
			ID:           "invalid-id",
			OrgID:        org.ID,
			BuID:         bu.ID,
			IncludeState: true,
		})

		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("create location", func(t *testing.T) {
		// Test Data
		l := &location.Location{
			Name:               "Test Location 2",
			AddressLine1:       "1234 Main St",
			Code:               "TEST000001",
			City:               "Los Angeles",
			PostalCode:         "90001",
			Status:             domain.StatusActive,
			StateID:            usState.ID,
			LocationCategoryID: locCategory.ID,
			BusinessUnitID:     bu.ID,
			OrganizationID:     org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, l)
	})

	t.Run("create location failure", func(t *testing.T) {
		// Test Data
		l := &location.Location{
			Name:               "Test Location 2",
			AddressLine1:       "1234 Main St",
			Code:               "TEST000001",
			City:               "Los Angeles",
			PostalCode:         "90001",
			Status:             domain.StatusActive,
			StateID:            "invalid-id",
			LocationCategoryID: locCategory.ID,
			BusinessUnitID:     bu.ID,
			OrganizationID:     org.ID,
		}

		results, err := repo.Create(ctx, l)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update location", func(t *testing.T) {
		loc.Name = "Test Location 3"
		testutils.TestRepoUpdate(ctx, t, repo, loc)
	})

	t.Run("update location version lock failure", func(t *testing.T) {
		loc.Name = "Test Location 3"
		loc.Version = 0

		results, err := repo.Update(ctx, loc)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update location with invalid information", func(t *testing.T) {
		loc.Name = "Test Location 3"
		loc.LocationCategoryID = "invalid-id"

		results, err := repo.Update(ctx, loc)

		require.Error(t, err)
		require.Nil(t, results)
	})
}
