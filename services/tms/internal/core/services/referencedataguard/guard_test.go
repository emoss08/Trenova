package referencedataguard_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/referencedataguard"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuard_CloudDeniesEveryoneWhenNoStewardIsConfigured(t *testing.T) {
	t.Parallel()

	guard, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		Mode: config.PlatformModeCloud,
	})
	require.NoError(t, err)

	err = guard.RequireSteward(pulid.MustNew("org_"))
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestGuard_CloudAllowsOnlyListedStewards(t *testing.T) {
	t.Parallel()

	steward := pulid.MustNew("org_")
	guard, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		Mode:                  config.PlatformModeCloud,
		ReferenceDataStewards: []string{" " + steward.String() + " "},
	})
	require.NoError(t, err)

	require.NoError(t, guard.RequireSteward(steward))

	err = guard.RequireSteward(pulid.MustNew("org_"))
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestGuard_SelfHostedAllowsOperatorOrganizationsByDefault(t *testing.T) {
	t.Parallel()

	guard, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		Mode: config.PlatformModeSelfHosted,
	})
	require.NoError(t, err)

	require.NoError(t, guard.RequireSteward(pulid.MustNew("org_")))
}

func TestGuard_SelfHostedHonoursAnExplicitList(t *testing.T) {
	t.Parallel()

	steward := pulid.MustNew("org_")
	guard, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		Mode:                  config.PlatformModeSelfHosted,
		ReferenceDataStewards: []string{steward.String()},
	})
	require.NoError(t, err)

	require.NoError(t, guard.RequireSteward(steward))
	require.Error(t, guard.RequireSteward(pulid.MustNew("org_")))
}

func TestGuard_FailsClosed(t *testing.T) {
	t.Parallel()

	var guard *referencedataguard.Guard
	require.Error(t, guard.RequireSteward(pulid.MustNew("org_")))

	open, err := referencedataguard.FromPlatform(&config.PlatformConfig{})
	require.NoError(t, err)
	require.Error(t, open.RequireSteward(pulid.Nil))
}

func TestGuard_RejectsMalformedStewardIDs(t *testing.T) {
	t.Parallel()

	_, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		ReferenceDataStewards: []string{"not-an-id"},
	})
	require.Error(t, err)
}

func TestGuard_ControlPlaneBackedInstancesRequireListedStewards(t *testing.T) {
	t.Parallel()

	guard, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		Mode:         config.PlatformModeSelfHosted,
		ControlPlane: config.PlatformControlPlaneConfig{Enabled: true},
	})
	require.NoError(t, err)

	require.Error(t, guard.RequireSteward(pulid.MustNew("org_")))
}
