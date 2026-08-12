package permithandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/permithandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type serviceStub struct {
	listPermitsFn      func(pulid.ID, pagination.TenantInfo) ([]*permit.Permit, error)
	listRequirementsFn func(pulid.ID, pagination.TenantInfo) ([]*permit.Requirement, error)
	createPermitFn     func(*permit.Permit) (*permit.Permit, error)
	updatePermitFn     func(*permit.Permit) (*permit.Permit, error)
	waiveFn            func(*services.WaiveRequirementRequest) (*permit.Requirement, error)
}

func (s *serviceStub) Assess(
	context.Context,
	*shipment.Shipment,
) (*services.PermitAssessment, error) {
	panic("unexpected Assess call")
}

func (s *serviceStub) Sync(
	context.Context,
	*shipment.Shipment,
	*services.RequestActor,
) (*services.PermitAssessment, error) {
	panic("unexpected Sync call")
}

func (s *serviceStub) ListPermits(
	_ context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*permit.Permit, error) {
	return s.listPermitsFn(shipmentID, tenantInfo)
}

func (s *serviceStub) ListRequirements(
	_ context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*permit.Requirement, error) {
	return s.listRequirementsFn(shipmentID, tenantInfo)
}

func (s *serviceStub) CreatePermit(
	_ context.Context,
	entity *permit.Permit,
	_ *services.RequestActor,
) (*permit.Permit, error) {
	return s.createPermitFn(entity)
}

func (s *serviceStub) UpdatePermit(
	_ context.Context,
	entity *permit.Permit,
	_ *services.RequestActor,
) (*permit.Permit, error) {
	return s.updatePermitFn(entity)
}

func (s *serviceStub) WaiveRequirement(
	_ context.Context,
	req *services.WaiveRequirementRequest,
) (*permit.Requirement, error) {
	return s.waiveFn(req)
}

func setupHandler(t *testing.T, service services.PermitService) *permithandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})

	return permithandler.New(permithandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
		Logger: logger,
	})
}

func TestPermitHandler_ListPermits_ScopesToTheSessionTenant(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	var gotShipment pulid.ID
	var gotTenant pagination.TenantInfo

	handler := setupHandler(t, &serviceStub{
		listPermitsFn: func(id pulid.ID, tenant pagination.TenantInfo) ([]*permit.Permit, error) {
			gotShipment, gotTenant = id, tenant
			return []*permit.Permit{{PermitNumber: "GA-1234"}}, nil
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/permits/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	assert.Equal(t, shipmentID, gotShipment)
	assert.Equal(t, testutil.TestOrgID, gotTenant.OrgID)
	assert.Equal(t, testutil.TestBuID, gotTenant.BuID)
}

// The shipment and the tenant come from the path and the session. A payload
// naming a different shipment or organization must not be able to attach a
// permit to someone else's load.
func TestPermitHandler_CreatePermit_IgnoresTenantAndShipmentInTheBody(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	var captured *permit.Permit

	handler := setupHandler(t, &serviceStub{
		createPermitFn: func(entity *permit.Permit) (*permit.Permit, error) {
			captured = entity
			return entity, nil
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/permits/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"permitNumber":   "GA-1234",
			"stateId":        "us_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"shipmentId":     "shp_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"organizationId": "org_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"businessUnitId": "bu_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
	require.NotNil(t, captured)
	assert.Equal(t, shipmentID, captured.ShipmentID)
	assert.Equal(t, testutil.TestOrgID, captured.OrganizationID)
	assert.Equal(t, testutil.TestBuID, captured.BusinessUnitID)
	assert.Equal(t, "GA-1234", captured.PermitNumber)
}

// The waiving user is the record of who accepted the compliance risk, so it has
// to come from the session rather than from anything the caller can set.
func TestPermitHandler_WaiveRequirement_TakesTheWaivingUserFromTheSession(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	requirementID := pulid.MustNew("pmr_")
	var captured *services.WaiveRequirementRequest

	handler := setupHandler(t, &serviceStub{
		waiveFn: func(req *services.WaiveRequirementRequest) (*permit.Requirement, error) {
			captured = req
			return &permit.Requirement{ID: req.RequirementID}, nil
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath(
			"/api/v1/shipments/" + shipmentID.String() +
				"/permit-requirements/" + requirementID.String() + "/waive/",
		).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"reason":     "Escort booked and route surveyed with the state",
			"waivedById": "usr_01ARZ3NDEKTSV4RRFFQ69G5FAV",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.NotNil(t, captured)
	assert.Equal(t, requirementID, captured.RequirementID)
	assert.Equal(t, testutil.TestUserID, captured.WaivedByID)
	assert.Equal(t, "Escort booked and route surveyed with the state", captured.Reason)
	assert.Equal(t, testutil.TestOrgID, captured.TenantInfo.OrgID)
}

func TestPermitHandler_ListRequirements_Success(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	handler := setupHandler(t, &serviceStub{
		listRequirementsFn: func(
			pulid.ID, pagination.TenantInfo,
		) ([]*permit.Requirement, error) {
			return []*permit.Requirement{
				{ID: pulid.MustNew("pmr_"), Status: permit.RequirementOpen},
			}, nil
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/shipments/" + shipmentID.String() + "/permit-requirements/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []*permit.Requirement
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, permit.RequirementOpen, resp[0].Status)
}

func TestPermitHandler_InvalidRequirementID(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	handler := setupHandler(t, &serviceStub{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath(
			"/api/v1/shipments/" + shipmentID.String() +
				"/permit-requirements/not-a-pulid/waive/",
		).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"reason": "Escort booked and route surveyed"})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
