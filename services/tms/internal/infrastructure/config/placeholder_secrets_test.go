package config

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/rendererconfig"
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

func TestProductionRendererNeedsCredentialsOnlyWhileConfigured(t *testing.T) {
	t.Parallel()

	const url = "http://gotenberg:3000"
	for name, renderer := range map[string]*rendererconfig.Config{
		"no password": {GotenbergURL: url, GotenbergUsername: "trenova"},
		"no username": {GotenbergURL: url, GotenbergPassword: stagedGotenbergPassword},
		"placeholder": {
			GotenbergURL:      url,
			GotenbergUsername: "trenova",
			GotenbergPassword: "change-me",
		},
	} {
		require.ErrorIs(
			t,
			renderer.ValidateProduction(),
			rendererconfig.ErrProductionCredentialsRequired,
			name,
		)
	}

	require.NoError(t, (&rendererconfig.Config{}).ValidateProduction())
	require.NoError(t, (&rendererconfig.Config{
		GotenbergURL:      url,
		GotenbergUsername: "trenova",
		GotenbergPassword: stagedGotenbergPassword,
	}).ValidateProduction())
}
