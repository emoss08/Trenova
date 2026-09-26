package shipmentmoveservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPreviewStopActual_ReadsWithoutWriting(t *testing.T) {
	t.Parallel()

	move, origin, _ := stopActualMoveForTest()
	move.ShipmentID = pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	occurredAt := int64(1790000000)

	moveRepo := mocks.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetMoveByIDRequest) bool {
			return req.MoveID == move.ID && req.ExpandMoveDetails && !req.ForUpdate
		})).
		Return(move, nil).
		Once()
	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, move.ID).
		Return(nil, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		repo:           moveRepo,
		assignmentRepo: assignmentRepo,
		holdRepo:       mocks.NewMockShipmentHoldRepository(t),
	}

	plan, err := svc.PreviewStopActual(t.Context(), &repositories.RecordStopActualRequest{
		TenantInfo: tenantInfo,
		MoveID:     move.ID,
		StopID:     origin.ID,
		Action:     repositories.StopActualActionArrive,
		OccurredAt: &occurredAt,
	})
	require.NoError(t, err)

	assert.Equal(t, shipment.MoveStatusAssigned, plan.Before.Status)
	assert.Equal(t, shipment.MoveStatusInTransit, plan.After.Status)
	assert.Nil(t, plan.Before.Stops[0].ActualArrival)
	require.NotNil(t, plan.After.Stops[0].ActualArrival)
	assert.Equal(t, occurredAt, *plan.After.Stops[0].ActualArrival)
}

func TestPreviewStopActual_RefusesWhatRecordingWouldRefuse(t *testing.T) {
	t.Parallel()

	move, _, destination := stopActualMoveForTest()
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	moveRepo := mocks.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(move, nil).Once()

	svc := &service{l: zap.NewNop(), repo: moveRepo}

	_, err := svc.PreviewStopActual(t.Context(), &repositories.RecordStopActualRequest{
		TenantInfo: tenantInfo,
		MoveID:     move.ID,
		StopID:     destination.ID,
		Action:     repositories.StopActualActionArrive,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}
