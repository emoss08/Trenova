package referencedataguard_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/platformconfig"
	"github.com/emoss08/trenova/internal/modules/edi/app/referencedataguard"
	"github.com/emoss08/trenova/internal/modules/kernel/app/editioninfo"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuard_CloudDeniesEveryoneWhenNoStewardIsConfigured(t *testing.T) {
	t.Parallel()

	guard, err := referencedataguard.FromPlatform(&platformconfig.Config{
		Mode: platformconfig.ModeCloud,
	}, editioninfo.NewStatic("cloud", true))
	require.NoError(t, err)

	err = guard.RequireSteward(pulid.MustNew("org_"))
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestGuard_CloudAllowsOnlyListedStewards(t *testing.T) {
	t.Parallel()

	steward := pulid.MustNew("org_")
	guard, err := referencedataguard.FromPlatform(&platformconfig.Config{
		Mode:                  platformconfig.ModeCloud,
		ReferenceDataStewards: []string{" " + steward.String() + " "},
	}, editioninfo.NewStatic("cloud", true))
	require.NoError(t, err)

	require.NoError(t, guard.RequireSteward(steward))

	err = guard.RequireSteward(pulid.MustNew("org_"))
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestGuard_SelfHostedAllowsOperatorOrganizationsByDefault(t *testing.T) {
	t.Parallel()

	guard, err := referencedataguard.FromPlatform(&platformconfig.Config{
		Mode: platformconfig.ModeSelfHosted,
	}, editioninfo.SelfHosted())
	require.NoError(t, err)

	require.NoError(t, guard.RequireSteward(pulid.MustNew("org_")))
}

func TestGuard_SelfHostedHonoursAnExplicitList(t *testing.T) {
	t.Parallel()

	steward := pulid.MustNew("org_")
	guard, err := referencedataguard.FromPlatform(&platformconfig.Config{
		Mode:                  platformconfig.ModeSelfHosted,
		ReferenceDataStewards: []string{steward.String()},
	}, editioninfo.SelfHosted())
	require.NoError(t, err)

	require.NoError(t, guard.RequireSteward(steward))
	require.Error(t, guard.RequireSteward(pulid.MustNew("org_")))
}

func TestGuard_WithoutAnEditionBehavesAsSelfHosted(t *testing.T) {
	t.Parallel()

	guard, err := referencedataguard.FromPlatform(&platformconfig.Config{}, nil)
	require.NoError(t, err)
	require.NoError(t, guard.RequireSteward(pulid.MustNew("org_")))
}

func TestGuard_FailsClosed(t *testing.T) {
	t.Parallel()

	var guard *referencedataguard.Guard
	require.Error(t, guard.RequireSteward(pulid.MustNew("org_")))

	open, err := referencedataguard.FromPlatform(&platformconfig.Config{}, editioninfo.SelfHosted())
	require.NoError(t, err)
	require.Error(t, open.RequireSteward(pulid.Nil))
}

func TestGuard_RejectsMalformedStewardIDs(t *testing.T) {
	t.Parallel()

	_, err := referencedataguard.FromPlatform(&platformconfig.Config{
		ReferenceDataStewards: []string{"not-an-id"},
	}, editioninfo.SelfHosted())
	require.Error(t, err)
}

func TestGuard_SharedTenancyRequiresListedStewardsWhateverTheMode(t *testing.T) {
	t.Parallel()

	guard, err := referencedataguard.FromPlatform(&platformconfig.Config{
		Mode: platformconfig.ModeSelfHosted,
	}, editioninfo.NewStatic("cloud", true))
	require.NoError(t, err)

	require.Error(t, guard.RequireSteward(pulid.MustNew("org_")))
}
