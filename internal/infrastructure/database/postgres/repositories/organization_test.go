package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestOrganizationRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	usState := ts.Fixture.MustRow("UsState.ca").(*usstate.UsState)

	repo := repositories.NewOrganizationRepository(repositories.OrganizationRepositoryParams{
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
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetOrgByIDOptions{
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get by id with state", func(t *testing.T) {
		opts := repoports.GetOrgByIDOptions{
			OrgID:        org.ID,
			BuID:         bu.ID,
			IncludeState: true,
		}

		result, err := repo.GetByID(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.State)
	})

	t.Run("get by id with business unit", func(t *testing.T) {
		opts := repoports.GetOrgByIDOptions{
			OrgID:     org.ID,
			BuID:      bu.ID,
			IncludeBu: true,
		}

		result, err := repo.GetByID(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.BusinessUnit)
	})

	t.Run("get with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetOrgByIDOptions{
			OrgID: "invalid-id",
			BuID:  bu.ID,
		})

		require.Error(t, err, "organization not found")
		require.Nil(t, entity)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &organization.Organization{
			Name:           "Test Organization",
			BusinessUnitID: bu.ID,
			City:           "Los Angeles",
			PostalCode:     "90001",
			StateID:        usState.ID,
			ScacCode:       "TEST",
			OrgType:        organization.TypeCarrier,
			AddressLine1:   "1234 Main St",
			PrimaryContact: "John Doe",
			PrimaryEmail:   "john.doe@trenova.com",
			PrimaryPhone:   "123-456-7890",
			TaxID:          "1234567890",
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("update", func(t *testing.T) {
		org.Name = "Test Organization 2"
		testutils.TestRepoUpdate(ctx, t, repo, org)
	})

	t.Run("update organization version lock failure", func(t *testing.T) {
		org.Name = "Test Organization 3"
		org.Version = 0

		results, err := repo.Update(ctx, org)

		require.Error(t, err)
		require.Nil(t, results)
	})

	t.Run("update organization with invalid information", func(t *testing.T) {
		org.Name = "Test Organization 3"
		org.StateID = "invalid-id"

		results, err := repo.Update(ctx, org)

		require.Error(t, err)
		require.Nil(t, results)
	})
}
