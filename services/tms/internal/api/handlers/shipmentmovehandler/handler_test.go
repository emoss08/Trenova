package shipmentmovehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"

	"github.com/emoss08/trenova/internal/api/handlers/shipmentmovehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/trackingeventportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/idempotency"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupShipmentMoveHandler(
	t *testing.T,
	service *shipmentportsmock.MockShipmentMoveService,
) *shipmentmovehandler.Handler {
	t.Helper()
	return setupShipmentMoveHandlerWithTracking(
		t,
		service,
		trackingeventportsmock.NewMockTrackingEventService(t),
	)
}

func setupShipmentMoveHandlerWithTracking(
	t *testing.T,
	service *shipmentportsmock.MockShipmentMoveService,
	tracking *trackingeventportsmock.MockTrackingEventService,
) *shipmentmovehandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	return shipmentmovehandler.New(shipmentmovehandler.Params{
		Service:              service,
		TrackingEvents:       tracking,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestShipmentMoveHandler_UpdateStatus_Success(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	service := shipmentportsmock.NewMockShipmentMoveService(t)
	service.EXPECT().
		UpdateStatus(mock.Anything, mock.MatchedBy(func(req *shipmentports.UpdateMoveStatusRequest) bool {
			return req.MoveID == moveID &&
				req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.Status == shipment.MoveStatusInTransit
		})).
		Return(&shipment.ShipmentMove{ID: moveID, Status: shipment.MoveStatusInTransit}, nil).
		Once()

	handler := setupShipmentMoveHandler(t, service)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipment-moves/" + moveID.String() + "/update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"status": "InTransit"})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentMoveHandler_BulkUpdateStatus_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupShipmentMoveHandler(t, shipmentportsmock.NewMockShipmentMoveService(t))
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipment-moves/bulk-update-status/").
		WithDefaultAuthContext().
		WithBody("{invalid")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func recordActualPath(moveID, stopID pulid.ID) string {
	return "/api/v1/shipment-moves/" + moveID.String() +
		"/stops/" + stopID.String() +
		"/record-actual/"
}

func TestShipmentMoveHandler_RecordStopActual_Success(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	occurredAt := int64(1767787200)
	tracking := trackingeventportsmock.NewMockTrackingEventService(t)
	tracking.EXPECT().
		RecordReported(mock.Anything, mock.MatchedBy(
			func(params *trackingeventports.RecordReportedStopEventParams) bool {
				return params.MoveID == moveID &&
					params.StopID == stopID &&
					params.TenantInfo.OrgID == testutil.TestOrgID &&
					params.Action == shipmentports.StopActualActionArrive &&
					params.Source == trackingevent.SourceDispatcher &&
					params.ReportedByID == testutil.TestUserID &&
					params.SourceKey == "" &&
					params.OccurredAt != nil &&
					*params.OccurredAt == occurredAt
			},
		)).
		Return(&trackingeventports.StopEventResult{
			Move:    &shipment.ShipmentMove{ID: moveID, Status: shipment.MoveStatusInTransit},
			Outcome: trackingevent.OutcomeApplied,
		}, nil).
		Once()

	handler := setupShipmentMoveHandlerWithTracking(
		t,
		shipmentportsmock.NewMockShipmentMoveService(t),
		tracking,
	)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath(recordActualPath(moveID, stopID)).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"action": "Arrive", "occurredAt": occurredAt})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp shipment.ShipmentMove
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Equal(t, moveID, resp.ID)
	require.Equal(t, shipment.MoveStatusInTransit, resp.Status)
}

func TestShipmentMoveHandler_RecordStopActual_UsesTheIdempotencyKey(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	tracking := trackingeventportsmock.NewMockTrackingEventService(t)
	tracking.EXPECT().
		RecordReported(mock.Anything, mock.MatchedBy(
			func(params *trackingeventports.RecordReportedStopEventParams) bool {
				return params.SourceKey == "scoped-key" &&
					params.Action == shipmentports.StopActualActionDepart &&
					params.OccurredAt == nil
			},
		)).
		Return(&trackingeventports.StopEventResult{
			Move:     &shipment.ShipmentMove{ID: moveID, Status: shipment.MoveStatusCompleted},
			Outcome:  trackingevent.OutcomeApplied,
			Replayed: true,
		}, nil).
		Once()

	handler := setupShipmentMoveHandlerWithTracking(
		t,
		shipmentportsmock.NewMockShipmentMoveService(t),
		tracking,
	)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath(recordActualPath(moveID, stopID)).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"action": "Depart"})
	ginCtx.Context.Request = ginCtx.Context.Request.WithContext(
		idempotency.WithKey(ginCtx.Context.Request.Context(), "scoped-key"),
	)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentMoveHandler_RecordStopActual_BusinessErrorPassthrough(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	tracking := trackingeventportsmock.NewMockTrackingEventService(t)
	tracking.EXPECT().
		RecordReported(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewBusinessError("You've already arrived at this stop")).
		Once()

	handler := setupShipmentMoveHandlerWithTracking(
		t,
		shipmentportsmock.NewMockShipmentMoveService(t),
		tracking,
	)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath(recordActualPath(moveID, stopID)).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"action": "Arrive"})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusUnprocessableEntity, ginCtx.ResponseCode())
}

func TestShipmentMoveHandler_RecordStopActual_BadJSON(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	handler := setupShipmentMoveHandler(t, shipmentportsmock.NewMockShipmentMoveService(t))
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath(
			"/api/v1/shipment-moves/" + moveID.String() +
				"/stops/" + stopID.String() +
				"/record-actual/",
		).
		WithDefaultAuthContext().
		WithBody("{invalid")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentMoveHandler_SplitMove_Success(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	newDeliveryLocationID := pulid.MustNew("loc_")
	service := shipmentportsmock.NewMockShipmentMoveService(t)
	service.EXPECT().
		SplitMove(mock.Anything, mock.MatchedBy(func(req *shipmentports.SplitMoveRequest) bool {
			return req.MoveID == moveID &&
				req.NewDeliveryLocationID == newDeliveryLocationID &&
				req.TenantInfo.OrgID == testutil.TestOrgID
		})).
		RunAndReturn(func(_ context.Context, req *shipmentports.SplitMoveRequest) (*shipmentports.SplitMoveResponse, error) {
			return &shipmentports.SplitMoveResponse{
				OriginalMove: &shipment.ShipmentMove{ID: req.MoveID},
				NewMove:      &shipment.ShipmentMove{ID: pulid.MustNew("sm_")},
			}, nil
		}).
		Once()

	handler := setupShipmentMoveHandler(t, service)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipment-moves/" + moveID.String() + "/split/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"newDeliveryLocationId": newDeliveryLocationID.String(),
			"splitPickupTimes": map[string]any{
				"scheduledWindowStart": 5,
				"scheduledWindowEnd":   6,
			},
			"newDeliveryTimes": map[string]any{
				"scheduledWindowStart": 7,
				"scheduledWindowEnd":   8,
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp shipmentports.SplitMoveResponse
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotNil(t, resp.OriginalMove)
	require.NotNil(t, resp.NewMove)
}
