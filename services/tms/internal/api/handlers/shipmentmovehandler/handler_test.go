package shipmentmovehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/shipmentmovehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupShipmentMoveHandler(
	t *testing.T,
	service *mocks.MockShipmentMoveService,
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
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestShipmentMoveHandler_UpdateStatus_Success(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	service := mocks.NewMockShipmentMoveService(t)
	service.EXPECT().
		UpdateStatus(mock.Anything, mock.MatchedBy(func(req *repositories.UpdateMoveStatusRequest) bool {
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

	handler := setupShipmentMoveHandler(t, mocks.NewMockShipmentMoveService(t))
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipment-moves/bulk-update-status/").
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
	service := mocks.NewMockShipmentMoveService(t)
	service.EXPECT().
		SplitMove(mock.Anything, mock.MatchedBy(func(req *repositories.SplitMoveRequest) bool {
			return req.MoveID == moveID &&
				req.NewDeliveryLocationID == newDeliveryLocationID &&
				req.TenantInfo.OrgID == testutil.TestOrgID
		})).
		RunAndReturn(func(_ context.Context, req *repositories.SplitMoveRequest) (*repositories.SplitMoveResponse, error) {
			return &repositories.SplitMoveResponse{
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

	var resp repositories.SplitMoveResponse
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotNil(t, resp.OriginalMove)
	require.NotNil(t, resp.NewMove)
}
