package emailservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func invoiceEmail() *emailports.SendEmailRequest {
	return &emailports.SendEmailRequest{
		TenantInfo: plantest.Tenant(),
		Type:       email.TypeInvoice,
		Purpose:    email.PurposeBilling,
		To:         []string{"ap@customer.example"},
	}
}

func TestSend_TrenovaSupportSendsNothingOutsideTheOrganization(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureAdministration)}
	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)

	byRequest := invoiceEmail()
	byRequest.TenantInfo.UserID = principal
	msg, err := svc.Send(t.Context(), byRequest)
	require.Nil(t, msg)
	var refusal *errortypes.AuthorizationError
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Message, "outside your organization")

	req := invoiceEmail()
	scoped := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         principal,
	})
	_, err = svc.Send(scoped, req)
	require.ErrorAs(t, err, &refusal, "a send caused in the session's scope is the session's send")
}

func TestSend_APersonOrTheSystemIsNotJudgedBySupportAccess(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureAdministration)}

	for _, userID := range []pulid.ID{pulid.MustNew("usr_"), pulid.Nil} {
		req := invoiceEmail()
		req.TenantInfo.UserID = userID
		_, err := svc.Send(t.Context(), req)
		require.True(t, errortypes.IsPlanRestrictionError(err),
			"the send reaches the next check, which refuses only for the plan: %v", err)
	}
}
