package shipmenthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/shipmenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentcommentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentholdportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
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

func (s *importAssistantStub) GetHistory(
	context.Context,
	string,
	pagination.TenantInfo,
) (*shipmentimportassistantports.ShipmentImportChatHistoryResponse, error) {
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

type permitServiceStub struct {
	assessFn func(context.Context, *shipment.Shipment) (*permitports.PermitAssessment, error)
}

func (s *permitServiceStub) Assess(
	ctx context.Context,
	entity *shipment.Shipment,
) (*permitports.PermitAssessment, error) {
	if s.assessFn != nil {
		return s.assessFn(ctx, entity)
	}

	panic("unexpected Assess call")
}

func (s *permitServiceStub) Sync(
	context.Context,
	*shipment.Shipment,
	*permissionports.RequestActor,
) (*permitports.PermitAssessment, error) {
	panic("unexpected Sync call")
}

func (s *permitServiceStub) ListPermits(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*permit.Permit, error) {
	panic("unexpected ListPermits call")
}

func (s *permitServiceStub) ListRequirements(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*permit.Requirement, error) {
	panic("unexpected ListRequirements call")
}

func (s *permitServiceStub) CreatePermit(
	context.Context,
	*permit.Permit,
	*permissionports.RequestActor,
) (*permit.Permit, error) {
	panic("unexpected CreatePermit call")
}

func (s *permitServiceStub) UpdatePermit(
	context.Context,
	*permit.Permit,
	*permissionports.RequestActor,
) (*permit.Permit, error) {
	panic("unexpected UpdatePermit call")
}

func (s *permitServiceStub) PlanCreatePermit(
	context.Context,
	*permit.Permit,
) (*permit.Permit, error) {
	panic("unexpected PlanCreatePermit call")
}

func (s *permitServiceStub) PlanUpdatePermit(
	context.Context,
	*permit.Permit,
) (*kernelports.RecordChange[permit.Permit], error) {
	panic("unexpected PlanUpdatePermit call")
}

func (s *permitServiceStub) WaiveRequirement(
	context.Context,
	*permitports.WaiveRequirementRequest,
) (*permit.Requirement, error) {
	panic("unexpected WaiveRequirement call")
}

func setupShipmentHandler(
	t *testing.T,
	service *shipmentserviceportsmock.MockShipmentService,
) *shipmenthandler.Handler {
	t.Helper()

	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
	holdService := shipmentholdportsmock.NewMockShipmentHoldService(t)
	return setupShipmentHandlerWithSubresources(t, service, commentService, holdService, nil)
}

func setupShipmentHandlerWithComments(
	t *testing.T,
	service *shipmentserviceportsmock.MockShipmentService,
	commentService *shipmentcommentportsmock.MockShipmentCommentService,
) *shipmenthandler.Handler {
	t.Helper()
	holdService := shipmentholdportsmock.NewMockShipmentHoldService(t)
	return setupShipmentHandlerWithSubresources(t, service, commentService, holdService, nil)
}

func setupShipmentHandlerWithSubresources(
	t *testing.T,
	service *shipmentserviceportsmock.MockShipmentService,
	commentService *shipmentcommentportsmock.MockShipmentCommentService,
	holdService *shipmentholdportsmock.MockShipmentHoldService,
	importAssistant ...shipmentimportassistantports.ShipmentImportAssistantService,
) *shipmenthandler.Handler {
	t.Helper()

	var assistant shipmentimportassistantports.ShipmentImportAssistantService
	if len(importAssistant) > 0 {
		assistant = importAssistant[0]
	}

	return buildShipmentHandler(t, handlerDeps{
		service:         service,
		commentService:  commentService,
		holdService:     holdService,
		importAssistant: assistant,
	})
}

type handlerDeps struct {
	service         shipmentserviceports.ShipmentService
	commentService  shipmentcommentports.ShipmentCommentService
	holdService     shipmentholdports.ShipmentHoldService
	importAssistant shipmentimportassistantports.ShipmentImportAssistantService
	permitService   permitports.PermitService
	etas            shipmenttrackingports.ShipmentEtaReader
	permissions     permissionports.PermissionEngine
	tracking        customertrackingports.CustomerTrackingService
}

func buildShipmentHandler(t *testing.T, deps handlerDeps) *shipmenthandler.Handler {
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

	permissions := deps.permissions
	if permissions == nil {
		permissions = &mocks.AllowAllPermissionEngine{}
	}

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: permissions,
		ErrorHandler:     errorHandler,
	})

	return shipmenthandler.New(shipmenthandler.Params{
		Service:              deps.service,
		CommentService:       deps.commentService,
		HoldService:          deps.holdService,
		ImportAssistant:      deps.importAssistant,
		PermitService:        deps.permitService,
		Etas:                 deps.etas,
		CustomerTracking:     deps.tracking,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
		Logger:               logger,
	})
}

func TestShipmentHandler_CalculateTotals_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		CalculateTotals(mock.Anything, mock.Anything, testutil.TestUserID).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment, _ pulid.ID) (*shipmentports.ShipmentTotalsResponse, error) {
			assert.Equal(t, testutil.TestOrgID, entity.OrganizationID)
			assert.Equal(t, testutil.TestBuID, entity.BusinessUnitID)
			assert.Equal(t, "BOL-100", entity.BOL)
			return &shipmentports.ShipmentTotalsResponse{
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

	var resp shipmentports.ShipmentTotalsResponse
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.True(t, decimal.NewFromInt(250).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(10).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromInt(260).Equal(resp.TotalChargeAmount))
}

func TestShipmentHandler_CalculateTotals_BadJSON(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
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
		shipmentcommentportsmock.NewMockShipmentCommentService(t),
		shipmentholdportsmock.NewMockShipmentHoldService(t),
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	sourceDocumentID := pulid.MustNew("doc_")
	shipmentID := pulid.MustNew("shp_")

	service.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*shipment.Shipment"), mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID}, nil).
		Once()

	handler := setupShipmentHandlerWithSubresources(
		t,
		service,
		shipmentcommentportsmock.NewMockShipmentCommentService(t),
		shipmentholdportsmock.NewMockShipmentHoldService(t),
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		List(mock.Anything, mock.MatchedBy(func(req *shipmentports.ListShipmentsRequest) bool {
			return req.Filter.TenantInfo.OrgID == testutil.TestOrgID &&
				req.Filter.TenantInfo.BuID == testutil.TestBuID &&
				req.ShipmentOptions.ExpandShipmentDetails &&
				req.ShipmentOptions.Status == string(shipment.StatusAssigned)
		})).
		Return(&pagination.CursorListResult[*shipment.Shipment]{
			Items:       []*shipment.Shipment{{ID: pulid.MustNew("shp_")}},
			HasNextPage: false,
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

func TestShipmentHandler_GetUnassigned_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		GetUnassigned(
			mock.Anything,
			mock.MatchedBy(func(req *shipmentports.GetUnassignedShipmentsRequest) bool {
				return req.Filter.TenantInfo.OrgID == testutil.TestOrgID &&
					req.Filter.TenantInfo.BuID == testutil.TestBuID &&
					req.Filter.Pagination.SafeLimit() == 10 &&
					req.Cursor.Limit == 10 &&
					req.Cursor.After == "" &&
					req.ShipmentOptions.ExpandShipmentDetails
			}),
		).
		Return(&pagination.CursorListResult[*shipment.Shipment]{
			Items:       []*shipment.Shipment{{ID: pulid.MustNew("shp_")}},
			HasNextPage: false,
		}, nil).
		Once()

	handler := setupShipmentHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/unassigned/").
		WithQuery(map[string]string{
			"limit":                 "10",
			"expandShipmentDetails": "true",
		}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestShipmentHandler_GetPermitAssessment_Success(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	entity := &shipment.Shipment{ID: shipmentID}

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID &&
				req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				req.ExpandShipmentDetails
		})).
		Return(entity, nil).
		Once()

	var assessed *shipment.Shipment
	permitService := &permitServiceStub{
		assessFn: func(
			_ context.Context,
			passed *shipment.Shipment,
		) (*permitports.PermitAssessment, error) {
			assessed = passed
			return &permitports.PermitAssessment{
				Measurements: permitports.AssessedMeasurements{
					WidthFeet:    12.5,
					HeightFeet:   13,
					LengthFeet:   48,
					WeightPounds: 74000,
				},
				Jurisdictions: []permitports.AssessedJurisdiction{
					{
						StateCode:         "GA",
						MaxWidthFeet:      8.5,
						WidthHeadroomFeet: -4,
						RequiresPermit:    true,
					},
				},
				TotalEscorts:    2,
				MaxLeadTimeDays: 5,
				RouteResolved:   true,
				FeeIsBaseOnly:   true,
			}, nil
		},
	}

	handler := buildShipmentHandler(t, handlerDeps{
		service:        service,
		commentService: shipmentcommentportsmock.NewMockShipmentCommentService(t),
		holdService:    shipmentholdportsmock.NewMockShipmentHoldService(t),
		permitService:  permitService,
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/permit-assessment/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	assert.Same(t, entity, assessed)

	var resp permitports.PermitAssessment
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.True(t, resp.RouteResolved)
	assert.True(t, resp.FeeIsBaseOnly)
	assert.Equal(t, 2, resp.TotalEscorts)
	assert.Equal(t, int16(5), resp.MaxLeadTimeDays)
	assert.InDelta(t, 12.5, resp.Measurements.WidthFeet, 0.0001)
	require.Len(t, resp.Jurisdictions, 1)
	assert.Equal(t, "GA", resp.Jurisdictions[0].StateCode)
	assert.InDelta(t, -4, resp.Jurisdictions[0].WidthHeadroomFeet, 0.0001)
	assert.True(t, resp.Jurisdictions[0].RequiresPermit)
}

func TestShipmentHandler_GetPermitAssessment_InvalidShipmentID(t *testing.T) {
	t.Parallel()

	handler := buildShipmentHandler(t, handlerDeps{
		service:        shipmentserviceportsmock.NewMockShipmentService(t),
		commentService: shipmentcommentportsmock.NewMockShipmentCommentService(t),
		holdService:    shipmentholdportsmock.NewMockShipmentHoldService(t),
		permitService:  &permitServiceStub{},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/not-a-pulid/permit-assessment/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestShipmentHandler_Get_Success(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		GetUIPolicy(mock.Anything, pagination.TenantInfo{
			OrgID: testutil.TestOrgID,
			BuID:  testutil.TestBuID,
		}).
		Return(&shipmentserviceports.ShipmentUIPolicy{
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

	var resp shipmentserviceports.ShipmentUIPolicy
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.False(t, resp.AllowMoveRemovals)
	assert.True(t, resp.CheckForDuplicateBOLs)
	assert.Equal(t, int32(80000), resp.MaxShipmentWeightLimit)
}

func TestShipmentHandler_GetBillingReadiness_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	shipmentID := pulid.MustNew("shp_")
	service.EXPECT().
		GetBillingReadiness(mock.Anything, shipmentID, pagination.TenantInfo{
			OrgID: testutil.TestOrgID,
			BuID:  testutil.TestBuID,
		}).
		Return(&shipmentbillingports.ShipmentBillingReadiness{
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

	var resp shipmentbillingports.ShipmentBillingReadiness
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, shipmentID.String(), resp.ShipmentID)
	assert.True(t, resp.CanMarkReadyToInvoice)
}

func TestShipmentHandler_Duplicate_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		Duplicate(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *shipmentports.BulkDuplicateShipmentRequest) (*shipmentports.ShipmentDuplicateWorkflowResponse, error) {
			assert.Equal(t, testutil.TestOrgID, req.TenantInfo.OrgID)
			assert.Equal(t, testutil.TestBuID, req.TenantInfo.BuID)
			assert.Equal(t, testutil.TestUserID, req.TenantInfo.UserID)
			assert.Equal(t, 2, req.Count)
			assert.True(t, req.OverrideDates)

			return &shipmentports.ShipmentDuplicateWorkflowResponse{
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

	var resp shipmentports.ShipmentDuplicateWorkflowResponse
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "workflow-1", resp.WorkflowID)
	assert.Equal(t, "run-1", resp.RunID)
}

func TestShipmentHandler_GetCommentCount_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
	commentService.EXPECT().
		GetCountByShipmentID(mock.Anything, mock.MatchedBy(func(req *shipmentcommentports.GetShipmentCommentCountRequest) bool {
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

func TestShipmentHandler_ListComments_CursorSortValidationError(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
	shipmentID := pulid.MustNew("shp_")
	commentID := pulid.MustNew("shc_")
	encodedCursor, err := pagination.EncodeKeysetCursor([]pagination.CursorSortField{
		{Field: "createdAt", Direction: "asc"},
		{Field: "id", Direction: "desc"},
	}, commentID.String())
	require.NoError(t, err)

	commentService.EXPECT().
		ListByShipmentID(mock.Anything, mock.MatchedBy(func(req *shipmentcommentports.ListShipmentCommentsRequest) bool {
			return req.ShipmentID == shipmentID &&
				req.Filter.TenantInfo.OrgID == testutil.TestOrgID &&
				req.Filter.TenantInfo.BuID == testutil.TestBuID &&
				req.Cursor.After == encodedCursor &&
				req.Cursor.Cursor.Keyset == commentID.String()
		})).
		Return(nil, errortypes.NewValidationError(
			"after",
			errortypes.ErrInvalid,
			"Cursor sort does not match request sort",
		)).
		Once()

	handler := setupShipmentHandlerWithComments(t, service, commentService)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/comments/").
		WithQuery(map[string]string{"after": encodedCursor}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())

	var problem helpers.ProblemDetail
	require.NoError(t, ginCtx.ResponseJSON(&problem))
	assert.Equal(t, "https://api.trenova.app/problems/validation-error", problem.Type)
	assert.Equal(t, "Validation Failed", problem.Title)
	assert.Equal(t, http.StatusBadRequest, problem.Status)
	require.Len(t, problem.Errors, 1)
	assert.Equal(t, "after", problem.Errors[0].Field)
	assert.Equal(t, string(errortypes.ErrInvalid), problem.Errors[0].Code)
	assert.Equal(t, "Cursor sort does not match request sort", problem.Errors[0].Message)
}

func TestShipmentHandler_CreateComment_Success(t *testing.T) {
	t.Parallel()

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
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

	handler := setupShipmentHandler(t, shipmentserviceportsmock.NewMockShipmentService(t))

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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.MatchedBy(func(req *shipmentports.DuplicateBOLCheckRequest) bool {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		GetPreviousRates(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetPreviousRatesRequest) bool {
			return req.TenantInfo.OrgID == testutil.TestOrgID &&
				req.TenantInfo.BuID == testutil.TestBuID &&
				req.OriginLocationID.IsNotNil() &&
				req.DestinationLocationID.IsNotNil()
		})).
		Return(&pagination.ListResult[*shipmentports.PreviousRateSummary]{
			Items: []*shipmentports.PreviousRateSummary{{
				ShipmentID: pulid.MustNew("shp_"),
				ProNumber:  "PRO-100",
				CreatedAt:  1710000000,
			}},
			Total:      1,
			TotalExact: true,
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

	var resp pagination.ListResult[*shipmentports.PreviousRateSummary]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "PRO-100", resp.Items[0].ProNumber)
}

func TestShipmentHandler_GetPreviousRates_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupShipmentHandler(t, shipmentserviceportsmock.NewMockShipmentService(t))

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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		GetDelayedShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetDelayedShipmentsRequest) bool {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		DelayShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.DelayShipmentsRequest) bool {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		GetAutoCancelableShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetAutoCancelableShipmentsRequest) bool {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		AutoCancelShipments(mock.Anything, mock.MatchedBy(func(req *shipmentports.AutoCancelShipmentsRequest) bool {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		Cancel(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *shipmentports.CancelShipmentRequest, actor *permissionports.RequestActor) (*shipment.Shipment, error) {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		TransferOwnership(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *shipmentports.TransferOwnershipRequest, actor *permissionports.RequestActor) (*shipment.Shipment, error) {
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

	service := shipmentserviceportsmock.NewMockShipmentService(t)
	service.EXPECT().
		Uncancel(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *shipmentports.UncancelShipmentRequest, actor *permissionports.RequestActor) (*shipment.Shipment, error) {
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

	handler := setupShipmentHandler(t, shipmentserviceportsmock.NewMockShipmentService(t))

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
