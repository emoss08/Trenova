package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestHazmatExpirationRepository(t *testing.T) {
	florida := ts.Fixture.MustRow("UsState.fl").(*usstate.UsState)

	repo := repositories.NewHazmatExpirationRepository(
		repositories.HazmatExpirationRepositoryParams{
			Logger: logger.NewLogger(testutils.NewTestConfig()),
			DB:     ts.DB,
		},
	)

	t.Run("get by state id", func(t *testing.T) {
		expiration, err := repo.GetHazmatExpirationByStateID(ctx, florida.ID)
		require.NoError(t, err)
		require.NotNil(t, expiration)
	})
}
