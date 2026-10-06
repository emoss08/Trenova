package emailservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestSendRefusesOutboundEmailThePlanRestricts(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityEmailOutbound)}

	msg, err := svc.Send(t.Context(), &services.SendEmailRequest{
		TenantInfo: plantest.Tenant(),
		Purpose:    email.PurposeBilling,
		To:         []string{"ap@example.com"},
	})

	require.Nil(t, msg)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestRequireOutboundAlwaysAllowsAuthenticationEmail(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityEmailOutbound)}

	require.NoError(t, svc.requireOutbound(t.Context(), plantest.Tenant(), email.PurposeAuthentication))
	require.Error(t, svc.requireOutbound(t.Context(), plantest.Tenant(), email.PurposeGeneral))
}
