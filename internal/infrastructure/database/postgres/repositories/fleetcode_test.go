package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestFleetCodeRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	fc := ts.Fixture.MustRow("FleetCode.fc_1").(*fleetcode.FleetCode)

	repo := repositories.NewFleetCodeRepository(repositories.FleetCodeRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list", func(t *testing.T) {
		opts := &repoports.ListFleetCodeOptions{
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

	t.Run("list fleet codes with manager details", func(t *testing.T) {
		opts := &repoports.ListFleetCodeOptions{
			IncludeManagerDetails: true,
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
		require.NotEmpty(t, result.Items[0].Manager)
	})

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetFleetCodeByIDOptions{
			ID:    fc.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get fleet code by id with manager details", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetFleetCodeByIDOptions{
			ID:                    fc.ID,
			OrgID:                 org.ID,
			BuID:                  bu.ID,
			IncludeManagerDetails: true,
		})

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.NotEmpty(t, entity.Manager)
	})

	t.Run("get with invalid id", func(t *testing.T) {
		fleetCode, err := repo.GetByID(ctx, repoports.GetFleetCodeByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "fleet code not found")
		require.Nil(t, fleetCode)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &fleetcode.FleetCode{
			Name:           "Test Fleet Code",
			Description:    "Test Fleet Code Description",
			Status:         domain.StatusActive,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("create tractor failure", func(t *testing.T) {
		// Test Data
		newEntity := &fleetcode.FleetCode{
			Name:           "Test Fleet Code 2",
			Description:    "Test Fleet Code Description 2",
			Status:         domain.StatusActive,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
			ManagerID:      pulid.Must("invalid-id"),
		}

		results, err := repo.Create(ctx, newEntity)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update", func(t *testing.T) {
		fc.Description = "Test Fleet Code 2"
		testutils.TestRepoUpdate(ctx, t, repo, fc)
	})

	t.Run("update fleet code version lock failure", func(t *testing.T) {
		fc.Description = "Test Fleet Code 3"
		fc.Version = 0

		results, err := repo.Update(ctx, fc)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update fleet code with invalid information", func(t *testing.T) {
		fc.Description = "Test Fleet Code 4"
		fc.ManagerID = pulid.Must("invalid-id")

		results, err := repo.Update(ctx, fc)

		require.Error(t, err)
		require.Nil(t, results)
	})
}
