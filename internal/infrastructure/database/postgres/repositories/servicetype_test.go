package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestServiceTypeRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	stype := ts.Fixture.MustRow("ServiceType.std_service_type").(*servicetype.ServiceType)

	repo := repositories.NewServiceTypeRepository(repositories.ServiceTypeRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list", func(t *testing.T) {
		opts := &repoports.ListServiceTypeRequest{
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

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetServiceTypeByIDOptions{
			ID:    stype.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetServiceTypeByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "service type not found")
		require.Nil(t, entity)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &servicetype.ServiceType{
			Code:           "TEST",
			Description:    "Test Service Type",
			Color:          "#000000",
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("update", func(t *testing.T) {
		stype.Description = "Test Service Type 2"
		testutils.TestRepoUpdate(ctx, t, repo, stype)
	})
}
