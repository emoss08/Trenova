package organizationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestSSOConfigurationRefusesWhenThePlanRestrictsSSO(t *testing.T) {
	t.Parallel()

	svc := &service{plans: plantest.Restricting(t, platformplan.CapabilitySSO)}
	tenant := plantest.Tenant()

	okta, err := svc.UpsertOktaSSOConfig(t.Context(), tenant, &services.OktaSSOConfig{})
	require.Nil(t, okta)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	microsoft, err := svc.UpsertMicrosoftSSOConfig(t.Context(), tenant, &services.MicrosoftSSOConfig{})
	require.Nil(t, microsoft)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
