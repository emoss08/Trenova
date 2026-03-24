package assignmenthandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/assignmenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func setupAssignmentHandler(
	t *testing.T,
	service *mocks.MockAssignmentService,
) *assignmenthandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{
		App: config.AppConfig{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	return assignmenthandler.New(assignmenthandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestAssignmentHandler_Unassign_Success(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	service := mocks.NewMockAssignmentService(t)
	service.EXPECT().
		Unassign(mock.Anything, mock.MatchedBy(func(req *repositories.UnassignShipmentMoveRequest) bool {
			return req.ShipmentMoveID == moveID &&
				req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				req.TenantInfo.UserID == testutil.TestUserID
		})).
		Return(nil).
		Once()

	handler := setupAssignmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/shipment-moves/" + moveID.String() + "/assignment/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
}

func TestAssignmentHandler_Unassign_InvalidMoveID(t *testing.T) {
	t.Parallel()

	handler := setupAssignmentHandler(t, mocks.NewMockAssignmentService(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/shipment-moves/invalid-id/assignment/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestAssignmentHandler_Unassign_ServiceError(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	service := mocks.NewMockAssignmentService(t)
	service.EXPECT().
		Unassign(mock.Anything, mock.AnythingOfType("*repositories.UnassignShipmentMoveRequest")).
		Return(errortypes.NewBusinessError("Only fresh assigned shipment moves can be unassigned")).
		Once()

	handler := setupAssignmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/shipment-moves/" + moveID.String() + "/assignment/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusUnprocessableEntity, ginCtx.ResponseCode())
}
