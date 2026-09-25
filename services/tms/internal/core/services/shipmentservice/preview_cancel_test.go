package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The repository has no Cancel expectation while the preview runs, so a
// cancellation it made would fail the test.
func TestPreviewCancel_IsTheCancellationCancelMakes(t *testing.T) {
	t.Parallel()

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	original := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ProNumber:      "SHP-3001",
		Status:         shipment.StatusAssigned,
	}
	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *repositories.GetShipmentByIDRequest) (*shipment.Shipment, error) {
			copied := *original
			return &copied, nil
		})
	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    NewTestValidator(t),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}
	request := func() *repositories.CancelShipmentRequest {
		return &repositories.CancelShipmentRequest{
			TenantInfo:   pagination.TenantInfo{OrgID: orgID, BuID: buID},
			ShipmentID:   original.ID,
			CancelReason: "customer request",
		}
	}
	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}

	req := request()
	preview, err := svc.PreviewCancel(t.Context(), req, actor)
	require.NoError(t, err)
	assert.True(t, req.CanceledByID.IsNil(), "a preview must not stamp the caller's request")

	var cancelled *repositories.CancelShipmentRequest
	repo.EXPECT().Cancel(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.CancelShipmentRequest,
		) (*shipment.Shipment, error) {
			cancelled = req
			saved := *original
			saved.ApplyCancel(req.CanceledByID, req.CanceledAt, req.CancelReason)
			return &saved, nil
		}).
		Once()
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).
		Return(nil).Maybe()
	svc.auditService = audit
	svc.realtime = realtime

	saved, err := svc.Cancel(t.Context(), request(), actor)
	require.NoError(t, err)
	require.NotNil(t, cancelled)

	assert.Equal(t, shipment.StatusAssigned, preview.Before.Status)
	assert.Equal(t, saved.Status, preview.After.Status)
	assert.Equal(t, saved.CanceledByID, preview.After.CanceledByID)
	assert.Equal(t, saved.CancelReason, preview.After.CancelReason)
	assert.NotNil(t, preview.After.CanceledAt)
}
