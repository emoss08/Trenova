package emailservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestSendRefusesOutboundEmailWhenAdministrationIsOff(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureAdministration)}

	msg, err := svc.Send(t.Context(), &emailports.SendEmailRequest{
		TenantInfo: plantest.Tenant(),
		Type:       email.TypeInvoice,
		Purpose:    email.PurposeBilling,
		To:         []string{"ap@example.com"},
	})

	require.Nil(t, msg)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestRequireOutboundAlwaysAllowsAuthenticationEmail(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureAdministration)}

	require.NoError(t, svc.requireOutbound(t.Context(), email.PurposeAuthentication))
	require.Error(t, svc.requireOutbound(t.Context(), email.PurposeGeneral))
}
