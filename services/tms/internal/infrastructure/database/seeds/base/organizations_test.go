package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireDevelopmentFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *config.Config
		refused bool
	}{
		{name: "no config", cfg: nil},
		{name: "development", cfg: &config.Config{App: config.AppConfig{Env: config.EnvDevelopment}}},
		{name: "test", cfg: &config.Config{App: config.AppConfig{Env: config.EnvTest}}},
		{
			name:    "production",
			cfg:     &config.Config{App: config.AppConfig{Env: config.EnvProduction}},
			refused: true,
		},
		{
			name:    "staging",
			cfg:     &config.Config{App: config.AppConfig{Env: config.EnvStaging}},
			refused: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := requireDevelopmentFixtures(seedhelpers.NewSeedContext(nil, nil, tt.cfg))
			if tt.refused {
				require.ErrorIs(t, err, ErrKnownCredentialsOutsideDevelopment)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSeedsWithKnownCredentialsAreDevelopmentOnly(t *testing.T) {
	t.Parallel()

	for _, seed := range []interface{ Environments() []common.Environment }{
		NewAdminAccountSeed(),
		NewOrganizationRolesSeed(),
	} {
		assert.ElementsMatch(
			t,
			[]common.Environment{common.EnvDevelopment, common.EnvTest},
			seed.Environments(),
		)
	}
}
