// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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
