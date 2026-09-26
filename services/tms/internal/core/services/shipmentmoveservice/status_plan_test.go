package shipmentmoveservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

// The preview reads the moves without locking them and holds no writer: the
// mocks fail on an UpdateStatus, an UpdateDerivedState or a lock, which is
// how a preview that wrote would show itself.
func TestPreviewUpdateStatus_ProjectsTheMovesAndTheirShipment(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	moveRepo := mocks.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && !req.ForUpdate
		})).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusAssigned,
			Version:    4,
		}, nil).
		Once()
	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().GetByMoveID(mock.Anything, tenantInfo, moveID).Return(nil, nil).Once()
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{
			ID:             shipmentID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ProNumber:      "S-1001",
			Status:         shipment.StatusAssigned,
			Moves: []*shipment.ShipmentMove{
				{ID: moveID, ShipmentID: shipmentID, Status: shipment.MoveStatusAssigned},
			},
		}, nil).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             testDBConnection{},
		repo:           moveRepo,
		assignmentRepo: assignmentRepo,
		shipmentRepo:   shipmentRepo,
		holdRepo:       mocks.NewMockShipmentHoldRepository(t),
		controlRepo:    controlRepo,
		coordinator:    shipmentstateCoordinator(),
	}

	plan, err := svc.PreviewUpdateStatus(t.Context(), &repositories.BulkUpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveIDs:    []pulid.ID{moveID},
		Status:     shipment.MoveStatusInTransit,
	})
	require.NoError(t, err)

	require.Len(t, plan.MovesBefore, 1)
	require.Len(t, plan.MovesAfter, 1)
	assert.Equal(t, shipment.MoveStatusAssigned, plan.MovesBefore[0].Status)
	assert.Equal(t, shipment.MoveStatusInTransit, plan.MovesAfter[0].Status)
	assert.Equal(t, int64(4), plan.MovesBefore[0].Version)
	require.Len(t, plan.ShipmentsAfter, 1)
	assert.Equal(t, shipment.StatusAssigned, plan.ShipmentsBefore[0].Status)
	assert.Equal(t, shipment.StatusInTransit, plan.ShipmentsAfter[0].Status)
}

// A move is never moved backwards: the preview refuses the transition the
// write refuses, with the same words, from the one check both run.
func TestPreviewUpdateStatus_RefusesWhatTheWriteRefuses(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	moveRepo := mocks.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetMoveByIDRequest) bool {
			return req.MoveID == moveID
		})).
		Return(&shipment.ShipmentMove{ID: moveID, Status: shipment.MoveStatusCompleted}, nil).
		Twice()

	svc := &service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         moveRepo,
		shipmentRepo: mocks.NewMockShipmentRepository(t),
		holdRepo:     mocks.NewMockShipmentHoldRepository(t),
	}

	_, previewErr := svc.PreviewUpdateStatus(t.Context(), &repositories.BulkUpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveIDs:    []pulid.ID{moveID},
		Status:     shipment.MoveStatusInTransit,
	})
	_, writeErr := svc.UpdateStatus(t.Context(), &repositories.UpdateMoveStatusRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Status:     shipment.MoveStatusInTransit,
	})

	require.Error(t, previewErr)
	assert.True(t, errortypes.IsBusinessError(previewErr))
	require.Error(t, writeErr)
	assert.Equal(t, writeErr.Error(), previewErr.Error())
}

func TestPreviewUpdateStatus_RefusesARepeatedMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	svc := &service{l: zap.NewNop()}

	_, err := svc.PreviewUpdateStatus(t.Context(), &repositories.BulkUpdateMoveStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		MoveIDs: []pulid.ID{moveID, moveID},
		Status:  shipment.MoveStatusCompleted,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}
