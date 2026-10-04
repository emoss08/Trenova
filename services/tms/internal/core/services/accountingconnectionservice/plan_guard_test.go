package accountingconnectionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestAccountingConnectionsRefuseWhenThePlanRestrictsIntegrations(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityIntegrations)}
	tenant := plantest.Tenant()

	start, err := svc.StartAuthorization(t.Context(), &services.StartAccountingAuthorizationRequest{
		TenantInfo:      tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.Nil(t, start)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	completion, err := svc.CompleteAuthorization(t.Context(), &services.CompleteAccountingAuthorizationRequest{
		TenantInfo:      tenant,
		IntegrationType: integration.TypeXero,
	})
	require.Nil(t, completion)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
