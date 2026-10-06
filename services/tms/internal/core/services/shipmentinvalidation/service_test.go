package shipmentinvalidation

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingRealtime struct {
	requests []*services.PublishResourceInvalidationRequest
}

func (r *recordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *services.PublishResourceInvalidationRequest,
) error {
	r.requests = append(r.requests, req)
	return nil
}

type recordingEpochs struct {
	bumps []pulid.ID
	err   error
}

func (r *recordingEpochs) BumpEpoch(_ context.Context, organizationID, _ pulid.ID) error {
	r.bumps = append(r.bumps, organizationID)
	return r.err
}

func TestInvalidateShipmentsBumpsAndPublishes(t *testing.T) {
	t.Parallel()

	realtime := &recordingRealtime{}
	epochs := &recordingEpochs{}
	svc := NewWithDependencies(realtime, epochs, zap.NewNop())

	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	recordID := pulid.MustNew("shp_")

	svc.InvalidateShipments(
		t.Context(),
		services.ShipmentInvalidationByUser(tenant, recordID, "updated"),
	)

	require.Len(t, realtime.requests, 1)
	assert.Equal(t, Resource, realtime.requests[0].Resource)
	assert.Equal(t, recordID, realtime.requests[0].RecordID)
	assert.Equal(t, tenant.UserID, realtime.requests[0].ActorUserID)
	assert.Equal(t, []pulid.ID{tenant.OrgID}, epochs.bumps)
}

func TestInvalidateShipmentsPublishesWhenTheEpochCannotMove(t *testing.T) {
	t.Parallel()

	realtime := &recordingRealtime{}
	svc := NewWithDependencies(realtime, &recordingEpochs{err: errors.New("down")}, zap.NewNop())

	svc.InvalidateShipments(t.Context(), &services.ShipmentInvalidation{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Action:         "bulk_created",
	})
	assert.Len(t, realtime.requests, 1)

	svc.InvalidateShipments(t.Context(), nil)
	svc.InvalidateShipments(t.Context(), &services.ShipmentInvalidation{Action: "x"})
	assert.Len(t, realtime.requests, 1)
}
