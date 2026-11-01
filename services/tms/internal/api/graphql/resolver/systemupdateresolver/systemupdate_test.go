package systemupdateresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/updateconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/versionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newDeps() *Deps {
	return &Deps{
		Core: &resolvercore.Core{},
		VersionService: versionservice.New(versionservice.Params{
			Logger:       zap.NewNop(),
			AppConfig:    &appconfig.Config{Version: "4.12.0"},
			UpdateConfig: &updateconfig.Config{Enabled: false},
		}),
	}
}

func TestSystemUpdateStatus_ReportsTheRunningVersion(t *testing.T) {
	t.Parallel()

	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	status, err := (&QueryResolver{newDeps()}).SystemUpdateStatus(ctx)
	require.NoError(t, err)

	assert.Equal(t, "4.12.0", status.CurrentVersion)
	assert.False(t, status.UpdateAvailable)
}

func TestSystemUpdateStatus_RequiresASignedInCaller(t *testing.T) {
	t.Parallel()

	_, err := (&QueryResolver{newDeps()}).SystemUpdateStatus(t.Context())

	var authnErr *errortypes.AuthenticationError
	require.ErrorAs(t, err, &authnErr)
}
