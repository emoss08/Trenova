package shipmenthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/shipmenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type importAssistantStub struct {
	completeHistoryFn func(context.Context, string, pagination.TenantInfo) error
}

func (s *importAssistantStub) Chat(
	context.Context,
	*servicesport.ShipmentImportChatRequest,
) (*servicesport.ShipmentImportChatResponse, error) {
	panic("unexpected Chat call")
}

func (s *importAssistantStub) ChatStream(
	context.Context,
	*servicesport.ShipmentImportChatRequest,
	func(servicesport.StreamEvent),
) error {
	panic("unexpected ChatStream call")
}

func (s *importAssistantStub) GetHistory(
	context.Context,
	string,
	pagination.TenantInfo,
) (*servicesport.ShipmentImportChatHistoryResponse, error) {
	panic("unexpected GetHistory call")
}

func (s *importAssistantStub) ArchiveHistory(
	context.Context,
	string,
	pagination.TenantInfo,
) error {
	panic("unexpected ArchiveHistory call")
}

func (s *importAssistantStub) CompleteHistory(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
) error {
	if s.completeHistoryFn != nil {
		return s.completeHistoryFn(ctx, documentID, tenantInfo)
	}

	return nil
}

func setupShipmentHandler(
	t *testing.T,
	service *mocks.MockShipmentService,
) *shipmenthandler.Handler {
	t.Helper()

	commentService := mocks.NewMockShipmentCommentService(t)
	holdService := mocks.NewMockShipmentHoldService(t)
	return setupShipmentHandlerWithSubresources(t, service, commentService, holdService, nil)
}

func setupShipmentHandlerWithComments(
	t *testing.T,
	service *mocks.MockShipmentService,
	commentService *mocks.MockShipmentCommentService,
) *shipmenthandler.Handler {
	t.Helper()
	holdService := mocks.NewMockShipmentHoldService(t)
	return setupShipmentHandlerWithSubresources(t, service, commentService, holdService, nil)
}

func setupShipmentHandlerWithSubresources(
	t *testing.T,
	service *mocks.MockShipmentService,
	commentService *mocks.MockShipmentCommentService,
	holdService *mocks.MockShipmentHoldService,
	importAssistant ...servicesport.ShipmentImportAssistantService,
) *shipmenthandler.Handler {
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

	var assistant servicesport.ShipmentImportAssistantService
	if len(importAssistant) > 0 {
		assistant = importAssistant[0]
	}

	return shipmenthandler.New(shipmenthandler.Params{
		Service:              service,
		CommentService:       commentService,
		HoldService:          holdService,
		ImportAssistant:      assistant,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
		Logger:               logger,
	})
}

func TestShipmentHandler_CalculateTotals_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		CalculateTotals(mock.Anything, mock.Anything, testutil.TestUserID).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment, _ pulid.ID) (*repositories.ShipmentTotalsResponse, error) {
			assert.Equal(t, testutil.TestOrgID, entity.OrganizationID)
			assert.Equal(t, testutil.TestBuID, entity.BusinessUnitID)
			assert.Equal(t, "BOL-100", entity.BOL)
			return &repositories.ShipmentTotalsResponse{
				FreightChargeAmount: decimal.NewFromInt(250),
				OtherChargeAmount:   decimal.NewFromInt(10),
				TotalChargeAmount:   decimal.NewFromInt(260),
			}, nil
		})

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/calculate-totals/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"formulaTemplateId": "fmt_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"serviceTypeId":     "svc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"customerId":        "cus_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"bol":               "BOL-100",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp repositories.ShipmentTotalsResponse
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.True(t, decimal.NewFromInt(250).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(10).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromInt(260).Equal(resp.TotalChargeAmount))
}

func TestShipmentHandler_CalculateTotals_BadJSON(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/calculate-totals/").
		WithDefaultAuthContext().
		WithBody("{invalid")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_CalculateTotals_MissingFormulaTemplateID(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		CalculateTotals(mock.Anything, mock.Anything, testutil.TestUserID).
		Return(nil, missingFormulaTemplateError())

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/calculate-totals/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"serviceTypeId": "svc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"customerId":    "cus_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"bol":           "BOL-100",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "https://api.trenova.app/problems/validation-error", resp["type"])
}

func TestShipmentHandler_Create_CompletesImportHistoryWhenSourceDocumentPresent(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	sourceDocumentID := pulid.MustNew("doc_")
	shipmentID := pulid.MustNew("shp_")
	completed := false

	service.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *shipment.Shipment) bool {
			return entity.OrganizationID == testutil.TestOrgID &&
				entity.BusinessUnitID == testutil.TestBuID &&
				entity.SourceDocumentID == sourceDocumentID.String()
		}), mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	handler := setupShipmentHandlerWithSubresources(
		t,
		service,
		mocks.NewMockShipmentCommentService(t),
		mocks.NewMockShipmentHoldService(t),
		&importAssistantStub{
			completeHistoryFn: func(_ context.Context, documentID string, tenantInfo pagination.TenantInfo) error {
				assert.Equal(t, sourceDocumentID.String(), documentID)
				assert.Equal(t, testutil.TestOrgID, tenantInfo.OrgID)
				assert.Equal(t, testutil.TestBuID, tenantInfo.BuID)
				assert.Equal(t, testutil.TestUserID, tenantInfo.UserID)
				completed = true
				return nil
			},
		},
	)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"formulaTemplateId": "fmt_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"serviceTypeId":     "svc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"customerId":        "cus_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"sourceDocumentId":  sourceDocumentID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
	assert.True(t, completed)
}

func TestShipmentHandler_Create_InvalidSourceDocumentID(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"formulaTemplateId": "fmt_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"serviceTypeId":     "svc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"customerId":        "cus_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"sourceDocumentId":  "not-a-document-id",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_Create_StillSucceedsWhenHistoryCompletionFails(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	sourceDocumentID := pulid.MustNew("doc_")
	shipmentID := pulid.MustNew("shp_")

	service.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*shipment.Shipment"), mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	handler := setupShipmentHandlerWithSubresources(
		t,
		service,
		mocks.NewMockShipmentCommentService(t),
		mocks.NewMockShipmentHoldService(t),
		&importAssistantStub{
			completeHistoryFn: func(context.Context, string, pagination.TenantInfo) error {
				return errortypes.NewBusinessError("completion failed")
			},
		},
	)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"formulaTemplateId": "fmt_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"serviceTypeId":     "svc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"customerId":        "cus_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"sourceDocumentId":  sourceDocumentID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
}

func TestShipmentHandler_List_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		List(mock.Anything, mock.MatchedBy(func(req *repositories.ListShipmentsRequest) bool {
			return req.Filter.TenantInfo.OrgID == testutil.TestOrgID &&
				req.Filter.TenantInfo.BuID == testutil.TestBuID &&
				req.ShipmentOptions.ExpandShipmentDetails &&
				req.ShipmentOptions.Status == string(shipment.StatusAssigned)
		})).
		Return(&pagination.ListResult[*shipment.Shipment]{
			Items: []*shipment.Shipment{{ID: pulid.MustNew("shp_")}},
			Total: 1,
		}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/").
		WithQuery(map[string]string{
			"expandShipmentDetails": "true",
			"status":                "Assigned",
		}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_Get_Success(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID &&
				req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				req.ExpandShipmentDetails
		})).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String()).
		WithQuery(map[string]string{
			"expandShipmentDetails": "true",
		}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_GetUIPolicy_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		GetUIPolicy(mock.Anything, pagination.TenantInfo{
			OrgID: testutil.TestOrgID,
			BuID:  testutil.TestBuID,
		}).
		Return(&servicesport.ShipmentUIPolicy{
			AllowMoveRemovals:      false,
			CheckForDuplicateBOLs:  true,
			MaxShipmentWeightLimit: 80000,
		}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/ui-policy/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp servicesport.ShipmentUIPolicy
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.False(t, resp.AllowMoveRemovals)
	assert.True(t, resp.CheckForDuplicateBOLs)
	assert.Equal(t, int32(80000), resp.MaxShipmentWeightLimit)
}

func TestShipmentHandler_GetBillingReadiness_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	shipmentID := pulid.MustNew("shp_")
	service.EXPECT().
		GetBillingReadiness(mock.Anything, shipmentID, pagination.TenantInfo{
			OrgID: testutil.TestOrgID,
			BuID:  testutil.TestBuID,
		}).
		Return(&servicesport.ShipmentBillingReadiness{
			ShipmentID:            shipmentID.String(),
			ShipmentStatus:        shipment.StatusCompleted,
			CanMarkReadyToInvoice: true,
		}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/billing-readiness/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp servicesport.ShipmentBillingReadiness
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, shipmentID.String(), resp.ShipmentID)
	assert.True(t, resp.CanMarkReadyToInvoice)
}

func TestShipmentHandler_Duplicate_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		Duplicate(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.BulkDuplicateShipmentRequest) (*repositories.ShipmentDuplicateWorkflowResponse, error) {
			assert.Equal(t, testutil.TestOrgID, req.TenantInfo.OrgID)
			assert.Equal(t, testutil.TestBuID, req.TenantInfo.BuID)
			assert.Equal(t, testutil.TestUserID, req.TenantInfo.UserID)
			assert.Equal(t, 2, req.Count)
			assert.True(t, req.OverrideDates)

			return &repositories.ShipmentDuplicateWorkflowResponse{
				WorkflowID:  "workflow-1",
				RunID:       "run-1",
				TaskQueue:   "system-queue",
				Status:      "RUNNING",
				SubmittedAt: 1710000000,
			}, nil
		})

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/duplicate/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"shipmentId":    "shp_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"count":         2,
			"overrideDates": true,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusAccepted, ginCtx.ResponseCode())

	var resp repositories.ShipmentDuplicateWorkflowResponse
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "workflow-1", resp.WorkflowID)
	assert.Equal(t, "run-1", resp.RunID)
}

func TestShipmentHandler_GetCommentCount_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	commentService := mocks.NewMockShipmentCommentService(t)
	commentService.EXPECT().
		GetCountByShipmentID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentCommentCountRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				req.ShipmentID.IsNotNil()
		})).
		Return(3, nil).
		Once()

	handler := setupShipmentHandlerWithComments(t, service, commentService)
	shipmentID := pulid.MustNew("shp_")

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/comments/count/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp map[string]int
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 3, resp["count"])
}

func TestShipmentHandler_CreateComment_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	commentService := mocks.NewMockShipmentCommentService(t)
	shipmentID := pulid.MustNew("shp_")

	commentService.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *shipment.ShipmentComment) bool {
			return entity.ShipmentID == shipmentID &&
				entity.OrganizationID == testutil.TestOrgID &&
				entity.BusinessUnitID == testutil.TestBuID &&
				entity.Comment == "hello"
		}), mock.Anything).
		Return(&shipment.ShipmentComment{ID: pulid.MustNew("shc_"), ShipmentID: shipmentID, Comment: "hello"}, nil).
		Once()

	handler := setupShipmentHandlerWithComments(t, service, commentService)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/comments/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"comment":          "hello",
			"mentionedUserIds": []string{pulid.MustNew("usr_").String()},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
}

func TestShipmentHandler_DeleteComment_InvalidCommentID(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	commentService := mocks.NewMockShipmentCommentService(t)
	handler := setupShipmentHandlerWithComments(t, service, commentService)
	shipmentID := pulid.MustNew("shp_")

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/comments/not-a-pulid/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_Duplicate_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupShipmentHandler(t, mocks.NewMockShipmentService(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/duplicate/").
		WithDefaultAuthContext().
		WithBody("{invalid")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_CheckForDuplicateBOLs_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.MatchedBy(func(req *repositories.DuplicateBOLCheckRequest) bool {
			return req.BOL == "BOL-123" &&
				req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		})).
		Return(nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/check-for-duplicate-bols/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"bol": "BOL-123",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]bool
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.True(t, resp["valid"])
}

func TestShipmentHandler_CheckForDuplicateBOLs_Duplicate(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.Anything).
		Return(func() error {
			multiErr := errortypes.NewMultiError()
			multiErr.Add("bol", errortypes.ErrInvalid, "duplicate")
			return multiErr
		}()).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/check-for-duplicate-bols/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"bol": "BOL-123",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_GetPreviousRates_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		GetPreviousRates(mock.Anything, mock.MatchedBy(func(req *repositories.GetPreviousRatesRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				req.OriginLocationID.IsNotNil() &&
				req.DestinationLocationID.IsNotNil()
		})).
		Return(&pagination.ListResult[*repositories.PreviousRateSummary]{
			Items: []*repositories.PreviousRateSummary{{
				ShipmentID: pulid.MustNew("shp_"),
				ProNumber:  "PRO-100",
				CreatedAt:  1710000000,
			}},
			Total: 1,
		}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/previous-rates/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"originLocationId":      "loc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"destinationLocationId": "loc_01ARZ3NDEKTSV4RRFFQ69G5FAA",
			"shipmentTypeId":        "sht_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"serviceTypeId":         "svc_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.ListResult[*repositories.PreviousRateSummary]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "PRO-100", resp.Items[0].ProNumber)
}

func TestShipmentHandler_GetPreviousRates_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupShipmentHandler(t, mocks.NewMockShipmentService(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/previous-rates/").
		WithDefaultAuthContext().
		WithBody("{invalid")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_GetDelayedShipments_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		GetDelayedShipments(mock.Anything, mock.MatchedBy(func(req *repositories.GetDelayedShipmentsRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		})).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/delayed/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_DelayShipments_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		DelayShipments(mock.Anything, mock.MatchedBy(func(req *repositories.DelayShipmentsRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		}), mock.Anything).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/delay/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_DelayShipments_ServiceError(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		DelayShipments(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, missingFormulaTemplateError()).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/delay/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_GetAutoCancelableShipments_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		GetAutoCancelableShipments(mock.Anything, mock.MatchedBy(func(req *repositories.GetAutoCancelableShipmentsRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		})).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/auto-cancel/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_AutoCancelShipments_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		AutoCancelShipments(mock.Anything, mock.MatchedBy(func(req *repositories.AutoCancelShipmentsRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID
		}), mock.Anything).
		Return([]*shipment.Shipment{{ID: pulid.MustNew("shp_")}}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/auto-cancel/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_AutoCancelShipments_ServiceError(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		AutoCancelShipments(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, missingFormulaTemplateError()).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/auto-cancel/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_Cancel_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		Cancel(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.CancelShipmentRequest, actor *servicesport.RequestActor) (*shipment.Shipment, error) {
			assert.Equal(t, testutil.TestOrgID, req.TenantInfo.OrgID)
			assert.Equal(t, testutil.TestBuID, req.TenantInfo.BuID)
			assert.Equal(t, "customer request", req.CancelReason)
			assert.Equal(t, "shp_01ARZ3NDEKTSV4RRFFQ69G5FAV", req.ShipmentID.String())
			assert.NotNil(t, actor)
			return &shipment.Shipment{ID: req.ShipmentID, Status: shipment.StatusCanceled}, nil
		})

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/shp_01ARZ3NDEKTSV4RRFFQ69G5FAV/cancel/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"cancelReason": "customer request",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_TransferOwnership_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		TransferOwnership(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.TransferOwnershipRequest, actor *servicesport.RequestActor) (*shipment.Shipment, error) {
			assert.Equal(t, testutil.TestOrgID, req.TenantInfo.OrgID)
			assert.Equal(t, testutil.TestBuID, req.TenantInfo.BuID)
			assert.Equal(t, "shp_01ARZ3NDEKTSV4RRFFQ69G5FAV", req.ShipmentID.String())
			assert.Equal(t, "usr_01ARZ3NDEKTSV4RRFFQ69G5FAV", req.OwnerID.String())
			assert.NotNil(t, actor)
			return &shipment.Shipment{ID: req.ShipmentID, OwnerID: req.OwnerID}, nil
		})

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/shp_01ARZ3NDEKTSV4RRFFQ69G5FAV/transfer-ownership/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"ownerId": "usr_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_Uncancel_Success(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockShipmentService(t)
	service.EXPECT().
		Uncancel(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.UncancelShipmentRequest, actor *servicesport.RequestActor) (*shipment.Shipment, error) {
			assert.Equal(t, testutil.TestOrgID, req.TenantInfo.OrgID)
			assert.Equal(t, testutil.TestBuID, req.TenantInfo.BuID)
			assert.Equal(t, "shp_01ARZ3NDEKTSV4RRFFQ69G5FAV", req.ShipmentID.String())
			assert.NotNil(t, actor)
			return &shipment.Shipment{ID: req.ShipmentID, Status: shipment.StatusNew}, nil
		})

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/shp_01ARZ3NDEKTSV4RRFFQ69G5FAV/uncancel/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_Cancel_BadShipmentID(t *testing.T) {
	t.Parallel()

	handler := setupShipmentHandler(t, mocks.NewMockShipmentService(t))

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/not-a-real-id/cancel/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func missingFormulaTemplateError() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("formulaTemplateId", errortypes.ErrRequired, "Formula template is required")
	return multiErr
}
