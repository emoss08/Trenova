package assignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestUnassign_PublishesMoveUnassignedEvent(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	svc := newUnassignService(t, unassignServiceParams{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
		MoveID:     moveID,
	})
	recorder := &agenteventstest.Recorder{}
	svc.agentEvents = recorder

	err := svc.Unassign(t.Context(), &repositories.UnassignShipmentMoveRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
	})

	require.NoError(t, err)
	events := recorder.Published()
	require.Len(t, events, 1)
	require.Equal(t, agent.EventShipmentMoveUnassigned, events[0].Kind)
	require.Equal(t, moveID, events[0].SubjectID)
	require.Equal(t, tenantInfo.OrgID, events[0].TenantInfo.OrgID)
	require.Equal(t, tenantInfo.BuID, events[0].TenantInfo.BuID)
}
