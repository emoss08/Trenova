package accountingconnectionauth

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAccountingConnectionsRefuseWhenAccountingIntegrationsAreOff(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger: zap.NewNop(),
		Kit: accountingconnectionkit.New(accountingconnectionkit.Params{
			Logger:   zap.NewNop(),
			Features: plantest.Without(platformcatalog.FeatureAccountingIntegration),
		}),
	})
	tenant := plantest.Tenant()

	start, err := svc.StartAuthorization(
		t.Context(),
		&accountingconnectionports.StartAccountingAuthorizationRequest{
			TenantInfo:      tenant,
			IntegrationType: integration.TypeQuickBooksOnline,
		},
	)
	require.Nil(t, start)
	require.True(t, errortypes.IsPlanRestrictionError(err))

	completion, err := svc.CompleteAuthorization(
		t.Context(),
		&accountingconnectionports.CompleteAccountingAuthorizationRequest{
			TenantInfo:      tenant,
			IntegrationType: integration.TypeXero,
		},
	)
	require.Nil(t, completion)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
