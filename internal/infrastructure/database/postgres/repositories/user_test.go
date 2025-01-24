package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestUserRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	usr := ts.Fixture.MustRow("User.test_user").(*user.User)

	repo := repositories.NewUserRepository(repositories.UserRepositoryParams{
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
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetUserByIDOptions{
			UserID: usr.ID,
			OrgID:  org.ID,
			BuID:   bu.ID,
		})
	})

	t.Run("get with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetUserByIDOptions{
			UserID: "invalid-id",
			OrgID:  org.ID,
			BuID:   bu.ID,
		})

		require.Error(t, err, "entity not found")
		require.Nil(t, entity)
	})

	t.Run("find by email", func(t *testing.T) {
		entity, err := repo.FindByEmail(ctx, usr.EmailAddress)
		require.NoError(t, err)
		require.NotNil(t, entity)
		require.Equal(t, entity.ID, usr.ID)
	})
}
