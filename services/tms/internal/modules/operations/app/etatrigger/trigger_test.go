package etatrigger

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRequester struct {
	shipments []pulid.ID
	tenants   []pagination.TenantInfo
}

func (r *recordingRequester) RequestRefresh(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	r.tenants = append(r.tenants, tenantInfo)
	r.shipments = append(r.shipments, shipmentID)
}

func TestOnShipmentEvent_RequestsRefreshForStopAndAssignmentChanges(t *testing.T) {
	t.Parallel()

	requester := &recordingRequester{}
	trigger := New(Params{Requester: requester})
	orgID, buID, shipmentID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("shp_")

	require.NoError(t, trigger.OnShipmentEvent(t.Context(), &shipmentevent.Event{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     shipmentID,
		Type:           shipmentevent.TypeMoveArrived,
	}))
	require.NoError(t, trigger.OnShipmentEvent(t.Context(), &shipmentevent.Event{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     shipmentID,
		Type:           shipmentevent.TypeCommentPosted,
	}))
	require.NoError(t, trigger.OnShipmentEvent(t.Context(), nil))

	assert.Equal(t, []pulid.ID{shipmentID}, requester.shipments)
	assert.Equal(t, pagination.TenantInfo{OrgID: orgID, BuID: buID}, requester.tenants[0])
}

func TestCheckCallRecorded_RequestsRefresh(t *testing.T) {
	t.Parallel()

	requester := &recordingRequester{}
	trigger := New(Params{Requester: requester})
	shipmentID := pulid.MustNew("shp_")

	trigger.CheckCallRecorded(
		t.Context(),
		pagination.TenantInfo{},
		shipmentID,
		pulid.MustNew("smv_"),
	)

	assert.Equal(t, []pulid.ID{shipmentID}, requester.shipments)
}

func TestRefreshes(t *testing.T) {
	t.Parallel()

	assert.True(t, Refreshes(shipmentevent.TypeStopCompleted))
	assert.True(t, Refreshes(shipmentevent.TypeDriverAssigned))
	assert.False(t, Refreshes(shipmentevent.TypeHoldPlaced))
}
