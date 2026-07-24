package billingqueueservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateStatus_AgentActorCannotApprove(t *testing.T) {
	svc := billingqueueservice.New(billingqueueservice.Params{
		Logger: zap.NewNop(),
	})

	agentActor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeAgent,
		PrincipalID:    services.AgentPrincipalID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	item, err := svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
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
