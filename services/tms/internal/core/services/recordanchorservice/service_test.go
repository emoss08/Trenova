package recordanchorservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/recordanchorservice"
	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gatedPermissions struct {
	services.PermissionEngine
	denied map[string]bool
}

func (g *gatedPermissions) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	return &services.PermissionCheckResult{Allowed: !g.denied[req.Resource]}, nil
}

type fakeTracking struct {
	snapshots map[pulid.ID]*shipmenttracking.Snapshot
}

func (f *fakeTracking) TrackingSnapshots(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
	_ string,
) (map[pulid.ID]*shipmenttracking.Snapshot, error) {
	out := make(map[pulid.ID]*shipmenttracking.Snapshot, len(ids))
	for _, id := range ids {
		if snapshot, ok := f.snapshots[id]; ok {
			out[id] = snapshot
		}
	}

	return out, nil
}

type fakeTelematics struct {
	repositories.TelematicsRepository
	states []*telematics.WorkerHOSState
}

func (f *fakeTelematics) ListWorkerHOSStates(
	_ context.Context,
	_ *repositories.ListWorkerHOSStatesRequest,
) ([]*telematics.WorkerHOSState, error) {
	return f.states, nil
}

const now = int64(1791600000)

func TestAnchor_ReadsEachRecordAgainInTheOrderGiven(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	goneID := pulid.MustNew("shp_")
	invoiceID := pulid.MustNew("inv_")
	customerID := pulid.MustNew("cus_")
	driverID := pulid.MustNew("wrk_")

	service := recordanchorservice.NewWithDependencies(&recordanchorservice.Dependencies{
		Permissions: &gatedPermissions{denied: map[string]bool{
			permission.ResourceInvoice.String(): true,
		}},
		Tracking: &fakeTracking{snapshots: map[pulid.ID]*shipmenttracking.Snapshot{
			shipmentID: {
				ProNumber: "SEED-DET-001",
				Status:    "InTransit",
				Customer:  "Acme Manufacturing",
				Position: &shipmenttracking.PositionSnapshot{
					FormattedLocation: "Amarillo, TX",
					RecordedAt:        now - 2*3600,
					Stale:             true,
				},
			},
		}},
		Invoices:   &stubInvoices{},
		Telematics: &fakeTelematics{},
		Now:        func() int64 { return now },
	})

	anchors := service.Anchor(t.Context(), &services.RecordAnchorRequest{
		Actor: &services.RequestActor{UserID: pulid.MustNew("usr_")},
		Records: []agent.EntityRef{
			{ID: shipmentID.String(), Label: "the acme load"},
			{ID: goneID.String()},
			{ID: invoiceID.String(), Label: "INV-1"},
			{ID: customerID.String(), Label: "Acme Manufacturing"},
			{ID: driverID.String(), Label: "Jane Doe"},
			{ID: shipmentID.String()},
			{ID: "not-a-record"},
		},
		Timezone: "UTC",
	})

	require.Len(t, anchors, 5)

	assert.Equal(t, shipmentID.String(), anchors[0].ID)
	assert.Equal(t, "SEED-DET-001", anchors[0].Label, "the record's own name replaces the label seen")
	require.NotEmpty(t, anchors[0].Facts)
	assert.Equal(t, "InTransit for Acme Manufacturing", anchors[0].Facts[0].Value)
	position := anchors[0].Facts[len(anchors[0].Facts)-1]
	assert.Equal(t, "last position", position.Name)
	assert.Equal(t, now-2*3600, position.SeenAt)
	assert.True(t, position.Stale)

	assert.Equal(t, goneID.String(), anchors[1].ID)
	assert.Contains(t, anchors[1].Note, "no longer exists")

	assert.Equal(t, "INV-1", anchors[2].Label)
	assert.Contains(t, anchors[2].Note, "can no longer read")
	assert.Empty(t, anchors[2].Facts)

	assert.Equal(t, "Acme Manufacturing", anchors[3].Label)
	assert.Empty(t, anchors[3].Facts, "a kind with nothing moving is named, not read")
	assert.Empty(t, anchors[3].Note)

	assert.Equal(t, "Jane Doe", anchors[4].Label)
	require.Len(t, anchors[4].Facts, 1)
	assert.Contains(t, anchors[4].Facts[0].Value, "unknown")
}

func TestAnchor_NothingToReadIsNothing(t *testing.T) {
	t.Parallel()

	service := recordanchorservice.NewWithDependencies(&recordanchorservice.Dependencies{})
	assert.Nil(t, service.Anchor(t.Context(), &services.RecordAnchorRequest{}))
	assert.Nil(t, service.Anchor(t.Context(), nil))
}

type stubInvoices struct {
	repositories.InvoiceRepository
}

func (s *stubInvoices) GetByIDs(
	_ context.Context,
	_ repositories.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	return nil, nil
}
