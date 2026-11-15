package integrationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateConfigRefusesEnablingAnIntegrationWhenAdministrationIsOff(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), features: plantest.Without(platformcatalog.FeatureAdministration)}

	resp, err := svc.UpdateConfig(
		t.Context(),
		plantest.Tenant(),
		integration.TypeGoogleMaps,
		&integrationports.UpdateConfigRequest{Enabled: true},
		pulid.MustNew("usr_"),
	)

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestTestConnectionRefusesWhenAdministrationIsOff(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), features: plantest.Without(platformcatalog.FeatureAdministration)}

	resp, err := svc.TestConnection(
		t.Context(),
		plantest.Tenant(),
		integration.TypeSamsara,
		pulid.MustNew("usr_"),
	)

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestEnablingAnIntegrationNeedsItsOwnAddOn(t *testing.T) {
	t.Parallel()

	for typ, feature := range map[integration.Type]platformcatalog.FeatureKey{
		integration.TypeSamsara:    platformcatalog.FeatureSamsaraIntegration,
		integration.TypeGoogleMaps: platformcatalog.FeatureGoogleMapsIntegration,
	} {
		svc := &Service{l: zap.NewNop(), features: plantest.Without(feature)}

		resp, err := svc.UpdateConfig(
			t.Context(),
			plantest.Tenant(),
			typ,
			&integrationports.UpdateConfigRequest{Enabled: true},
			pulid.MustNew("usr_"),
		)

		require.Nil(t, resp, string(typ))
		require.True(t, errortypes.IsPlanRestrictionError(err), string(typ))
	}
}

func TestAnIntegrationOfTheBasePlanIsOnWithoutAnyAddOn(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), features: plantest.BaseOnly()}

	on, err := svc.integrationOnPlan(t.Context(), integration.TypeOANDAExchangeRates)
	require.NoError(t, err)
	require.True(t, on)

	on, err = svc.integrationOnPlan(t.Context(), integration.TypeGoogleMaps)
	require.NoError(t, err)
	require.False(t, on)
}
