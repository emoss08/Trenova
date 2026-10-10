package organizationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestSSOConfigurationRefusesWhenThePlanRestrictsSSO(t *testing.T) {
	t.Parallel()

	svc := &service{plans: plantest.Restricting(t, platformplan.CapabilitySSO)}
	tenant := plantest.Tenant()

	okta, err := svc.UpsertOktaSSOConfig(t.Context(), tenant, &tenantports.OktaSSOConfig{})
	require.Nil(t, okta)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	microsoft, err := svc.UpsertMicrosoftSSOConfig(t.Context(), tenant, &tenantports.MicrosoftSSOConfig{})
	require.Nil(t, microsoft)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
