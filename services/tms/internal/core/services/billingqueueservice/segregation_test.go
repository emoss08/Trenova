package billingqueueservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateStatus_AgentActorCannotApprove(t *testing.T) {
	svc := billingqueueservice.New(billingqueueservice.Params{
		Logger: zap.NewNop(),
	})

	agentActor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    permissionports.AgentPrincipalID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	item, err := svc.UpdateStatus(t.Context(), &billingqueueports.UpdateBillingQueueStatusRequest{
		ItemID:    pulid.MustNew("bqi_"),
		NewStatus: billingqueue.StatusApproved,
		TenantInfo: pagination.TenantInfo{
			OrgID: agentActor.OrganizationID,
			BuID:  agentActor.BusinessUnitID,
		},
	}, agentActor)

	require.Error(t, err, "agent principals must not be able to approve billing queue items")
	require.Nil(t, item)
}
