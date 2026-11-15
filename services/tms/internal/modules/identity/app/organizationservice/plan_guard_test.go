package organizationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestSSOConfigurationRefusesWhenSecurityAndIdentityIsOff(t *testing.T) {
	t.Parallel()

	svc := &service{features: plantest.Without(platformcatalog.FeatureIdentity)}
	tenant := plantest.Tenant()

	okta, err := svc.UpsertOktaSSOConfig(t.Context(), tenant, &tenantports.OktaSSOConfig{})
	require.Nil(t, okta)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	microsoft, err := svc.UpsertMicrosoftSSOConfig(t.Context(), tenant, &tenantports.MicrosoftSSOConfig{})
	require.Nil(t, microsoft)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
