package carrierassignmentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errStopAfterRevoke = errors.New("stop after revoking the tracking links")

type recordingTrackingLinks struct {
	checkcallports.TrackingLinkRepository
	revoked []*checkcallports.RevokeTrackingLinksRequest
}

func (r *recordingTrackingLinks) RevokeForMove(
	_ context.Context,
	req *checkcallports.RevokeTrackingLinksRequest,
) (int, error) {
	r.revoked = append(r.revoked, req)
	return 1, errStopAfterRevoke
}

func assignedCarrier(moveID pulid.ID) *shipment.CarrierAssignment {
	return &shipment.CarrierAssignment{
		ID:             pulid.MustNew("casn_"),
		ShipmentMoveID: moveID,
		CarrierID:      pulid.MustNew("car_"),
		Status:         shipmentvocab.CarrierAssignmentStatusConfirmed,
	}
}

func assertRevokedForMove(
	t *testing.T,
	links *recordingTrackingLinks,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) {
	t.Helper()

	require.Len(t, links.revoked, 1)
	assert.Equal(t, moveID, links.revoked[0].MoveID)
	assert.Equal(t, tenantInfo, links.revoked[0].TenantInfo)
	assert.NotZero(t, links.revoked[0].At)
}

func TestCancelRevokesTheCarrierTrackingLinksInTheSameTransaction(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipmentvocab.MoveStatusAssigned)
	existing := assignedCarrier(f.moveID)
	links := &recordingTrackingLinks{}
	f.svc.trackingLinks = links

	f.repo.On("GetActiveByMoveID", mock.Anything, f.tenantInfo, f.moveID).
		Return(existing, nil).
		Once()
	f.repo.On("Update", mock.Anything, mock.Anything).Return(existing, nil).Once()
	f.expectShipmentRead()
	f.controls.ExpectedCalls = nil

	err := f.svc.Cancel(t.Context(), &carrierassignmentports.CancelCarrierAssignmentRequest{
		TenantInfo:     f.tenantInfo,
		ShipmentMoveID: f.moveID,
		Reason:         "Carrier fell off",
	})

	require.ErrorIs(t, err, errStopAfterRevoke)
	assertRevokedForMove(t, links, f.tenantInfo, f.moveID)
}

func TestReplacingTheCarrierRevokesTheOldCarriersTrackingLinks(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipmentvocab.MoveStatusAssigned)
	entity := qualifiedCarrier(timeutils.NowUnix())
	entity.ID = pulid.MustNew("car_")
	existing := assignedCarrier(f.moveID)
	links := &recordingTrackingLinks{}
	f.svc.trackingLinks = links

	f.holds.EXPECT().HasActiveDispatchHold(mock.Anything, mock.Anything).Return(false, nil).Once()
	f.assignments.EXPECT().
		GetByMoveID(mock.Anything, f.tenantInfo, f.moveID).
		Return(nil, nil).
		Once()
	f.repo.On("GetActiveByMoveID", mock.Anything, f.tenantInfo, f.moveID).
		Return(existing, nil).
		Once()
	f.repo.On("Update", mock.Anything, mock.Anything).Return(existing, nil).Once()
	f.expectCarrier(entity)
	f.expectShipmentRead()
	f.controls.ExpectedCalls = nil

	req := f.assignRequest(entity.ID)
	req.Replace = true
	_, err := f.svc.AssignToMove(t.Context(), req)

	require.ErrorIs(t, err, errStopAfterRevoke)
	assertRevokedForMove(t, links, f.tenantInfo, f.moveID)
}
