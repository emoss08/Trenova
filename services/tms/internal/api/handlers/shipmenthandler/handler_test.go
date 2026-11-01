package shipmenthandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/shipmenthandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	service *shipmentServiceMocks,
) *shipmenthandler.Handler {
	t.Helper()

	commentService := shipmentcommentportsmock.NewMockShipmentCommentService(t)
	holdService := shipmentholdportsmock.NewMockShipmentHoldService(t)
	return setupShipmentHandlerWithSubresources(t, service, commentService, holdService, nil)
}

func setupShipmentHandlerWithSubresources(
	t *testing.T,
	service *shipmentServiceMocks,
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

type shipmentServiceMocks struct {
	reader    *shipmentserviceportsmock.MockShipmentReader
	writer    *shipmentserviceportsmock.MockShipmentWriter
	lifecycle *shipmentserviceportsmock.MockShipmentLifecycleService
	readiness *shipmentserviceportsmock.MockShipmentBillingReadinessReader
	transfer  *shipmentserviceportsmock.MockShipmentBillingTransferService
	autorater *shipmentserviceportsmock.MockShipmentAutoRater
	loading   *shipmentserviceportsmock.MockShipmentLoadingPlanner
}

func newShipmentServiceMocks(t *testing.T) *shipmentServiceMocks {
	t.Helper()

	return &shipmentServiceMocks{
		reader:    shipmentserviceportsmock.NewMockShipmentReader(t),
		writer:    shipmentserviceportsmock.NewMockShipmentWriter(t),
		lifecycle: shipmentserviceportsmock.NewMockShipmentLifecycleService(t),
		readiness: shipmentserviceportsmock.NewMockShipmentBillingReadinessReader(t),
		transfer:  shipmentserviceportsmock.NewMockShipmentBillingTransferService(t),
		autorater: shipmentserviceportsmock.NewMockShipmentAutoRater(t),
		loading:   shipmentserviceportsmock.NewMockShipmentLoadingPlanner(t),
	}
}

type handlerDeps struct {
	service         *shipmentServiceMocks
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
		App: appconfig.Config{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &cfg.App,
	})

	permissions := deps.permissions
	if permissions == nil {
		permissions = &mocks.AllowAllPermissionEngine{}
	}

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: permissions,
		ErrorHandler:     errorHandler,
	})

	services := deps.service
	if services == nil {
		services = newShipmentServiceMocks(t)
	}

	return shipmenthandler.New(shipmenthandler.Params{
		Reader:               services.reader,
		Writer:               services.writer,
		Lifecycle:            services.lifecycle,
		Readiness:            services.readiness,
		Transfer:             services.transfer,
		AutoRater:            services.autorater,
		Loading:              services.loading,
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

func TestShipmentHandler_Create_CompletesImportHistoryWhenSourceDocumentPresent(t *testing.T) {
	t.Parallel()

	service := newShipmentServiceMocks(t)
	sourceDocumentID := pulid.MustNew("doc_")
	shipmentID := pulid.MustNew("shp_")
	completed := false

	service.writer.EXPECT().
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

	service := newShipmentServiceMocks(t)
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

	service := newShipmentServiceMocks(t)
	sourceDocumentID := pulid.MustNew("doc_")
	shipmentID := pulid.MustNew("shp_")

	service.writer.EXPECT().
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

func TestShipmentHandler_GetPermitAssessment_InvalidShipmentID(t *testing.T) {
	t.Parallel()

	handler := buildShipmentHandler(t, handlerDeps{
		service:        newShipmentServiceMocks(t),
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

func TestShipmentHandler_DelayShipments_Success(t *testing.T) {
	t.Parallel()

	service := newShipmentServiceMocks(t)
	service.lifecycle.EXPECT().
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

	service := newShipmentServiceMocks(t)
	service.lifecycle.EXPECT().
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

func missingFormulaTemplateError() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("formulaTemplateId", errortypes.ErrRequired, "Formula template is required")
	return multiErr
}
