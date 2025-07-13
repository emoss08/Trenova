package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestCommodityRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	comm := ts.Fixture.MustRow("Commodity.test_commodity").(*commodity.Commodity)

	repo := repositories.NewCommodityRepository(repositories.CommodityRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list commodities", func(t *testing.T) {
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

	t.Run("list commodities with query", func(t *testing.T) {
		opts := &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
			Query: "Test",
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("get commodity by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetCommodityByIDOptions{
			ID:    comm.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get commodity with invalid id", func(t *testing.T) {
		l, err := repo.GetByID(ctx, repoports.GetCommodityByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err)
		require.Nil(t, l)
	})

	t.Run("create commodity", func(t *testing.T) {
		// Test Data
		l := &commodity.Commodity{
			Name:           "Test commodity 2",
			Description:    "1234 Main St",
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, l)
	})

	t.Run("create commodity failure", func(t *testing.T) {
		// Test Data
		l := &commodity.Commodity{
			Name:                "Test commodity 2",
			Description:         "1234 Main St",
			HazardousMaterialID: pulid.Must("invalid-id"),
			BusinessUnitID:      bu.ID,
			OrganizationID:      org.ID,
		}

		results, err := repo.Create(ctx, l)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update commodity", func(t *testing.T) {
		comm.Name = "Test Commodity 3"
		testutils.TestRepoUpdate(ctx, t, repo, comm)
	})

	t.Run("update commodity version lock failure", func(t *testing.T) {
		comm.Name = "Test Commodity 3"
		comm.Version = 0

		results, err := repo.Update(ctx, comm)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update commodity with invalid information", func(t *testing.T) {
		comm.Name = "Test commodity 3"
		comm.HazardousMaterialID = pulid.Must("invalid-id")

		results, err := repo.Update(ctx, comm)

		require.Error(t, err)
		require.Nil(t, results)
	})
}
