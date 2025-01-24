package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
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
		opts := &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: &ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetFleetCodeByIDOptions{
			ID:    fc.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
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

	t.Run("update", func(t *testing.T) {
		fc.Description = "Test Fleet Code 2"
		testutils.TestRepoUpdate(ctx, t, repo, fc)
	})
}
