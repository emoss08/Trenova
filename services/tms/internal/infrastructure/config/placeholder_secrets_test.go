package config

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/systemconfig"
	"github.com/stretchr/testify/require"
)

func TestProductionRefusesPlaceholderSecrets(t *testing.T) {
	t.Parallel()

	require.ErrorIs(
		t,
		(&databaseconfig.Config{SSLMode: "require", Password: "change-me"}).ValidateProduction(),
		databaseconfig.ErrProductionPasswordRequired,
	)
	require.ErrorIs(
		t,
		(&storageconfig.Config{SecretKey: "change-me"}).ValidateProduction(),
		storageconfig.ErrProductionCredentialsRequired,
	)
	require.ErrorIs(
		t,
		(&systemconfig.Config{SystemUserPassword: "change-me"}).ValidateProduction(),
		systemconfig.ErrProductionSystemUserPasswordRequired,
	)

	search := &searchconfig.Config{Enabled: true}
	search.Meilisearch.APIKey = "change-me"
	require.ErrorIs(t, search.ValidateProduction(), searchconfig.ErrProductionAPIKeyRequired)

	search.Enabled = false
	require.NoError(t, search.ValidateProduction())
}
