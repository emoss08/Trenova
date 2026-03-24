package formulatemplatehandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/formulatemplatehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("formula template not found")

type mockFormulaTemplateRepo struct {
	listFunc             func(ctx context.Context, req *repositories.ListFormulaTemplatesRequest) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error)
	getByIDFunc          func(ctx context.Context, req repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error)
	getByIDsFunc         func(ctx context.Context, req repositories.GetFormulaTemplatesByIDsRequest) ([]*formulatemplate.FormulaTemplate, error)
	createFunc           func(ctx context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error)
	updateFunc           func(ctx context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error)
	bulkUpdateStatusFunc func(ctx context.Context, req *repositories.BulkUpdateFormulaTemplateStatusRequest) ([]*formulatemplate.FormulaTemplate, error)
	bulkDuplicateFunc    func(ctx context.Context, req *repositories.BulkDuplicateFormulaTemplateRequest) ([]*formulatemplate.FormulaTemplate, error)
	countUsagesFunc      func(ctx context.Context, req *repositories.GetTemplateUsageRequest) (*repositories.GetTemplateUsageResponse, error)
	selectOptionsFunc    func(ctx context.Context, req *repositories.FormulaTemplateSelectOptionsRequest) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error)
}

func (m *mockFormulaTemplateRepo) List(
	ctx context.Context,
	req *repositories.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, req)
	}
	return &pagination.ListResult[*formulatemplate.FormulaTemplate]{
		Items: []*formulatemplate.FormulaTemplate{},
		Total: 0,
	}, nil
}

func (m *mockFormulaTemplateRepo) GetByID(
	ctx context.Context,
	req repositories.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, req)
	}
	return nil, errNotFound
}

func (m *mockFormulaTemplateRepo) GetByIDs(
	ctx context.Context,
	req repositories.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	if m.getByIDsFunc != nil {
		return m.getByIDsFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockFormulaTemplateRepo) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, entity)
	}
	return entity, nil
}

func (m *mockFormulaTemplateRepo) Update(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, entity)
	}
	return entity, nil
}

func (m *mockFormulaTemplateRepo) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	if m.bulkUpdateStatusFunc != nil {
		return m.bulkUpdateStatusFunc(ctx, req)
	}
	return []*formulatemplate.FormulaTemplate{}, nil
}

func (m *mockFormulaTemplateRepo) BulkDuplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	if m.bulkDuplicateFunc != nil {
		return m.bulkDuplicateFunc(ctx, req)
	}
	return []*formulatemplate.FormulaTemplate{}, nil
}

func (m *mockFormulaTemplateRepo) CountUsages(
	ctx context.Context,
	req *repositories.GetTemplateUsageRequest,
) (*repositories.GetTemplateUsageResponse, error) {
	if m.countUsagesFunc != nil {
		return m.countUsagesFunc(ctx, req)
	}
	return &repositories.GetTemplateUsageResponse{
		InUse:  false,
		Usages: []repositories.TemplateUsageCount{},
	}, nil
}

func (m *mockFormulaTemplateRepo) SelectOptions(
	ctx context.Context,
	req *repositories.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	if m.selectOptionsFunc != nil {
		return m.selectOptionsFunc(ctx, req)
	}
	return &pagination.ListResult[*formulatemplate.FormulaTemplate]{
		Items: []*formulatemplate.FormulaTemplate{},
		Total: 0,
	}, nil
}

type mockVersionRepo struct {
	createFunc                func(ctx context.Context, version *formulatemplate.FormulaTemplateVersion) (*formulatemplate.FormulaTemplateVersion, error)
	getByTemplateAndVersionFn func(ctx context.Context, req *repositories.GetVersionRequest) (*formulatemplate.FormulaTemplateVersion, error)
	listFunc                  func(ctx context.Context, req *repositories.ListVersionsRequest) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error)
	getVersionRangeFunc       func(ctx context.Context, req *repositories.GetVersionRangeRequest) ([]*formulatemplate.FormulaTemplateVersion, error)
	getLatestVersionFunc      func(ctx context.Context, templateID pulid.ID, tenantInfo pagination.TenantInfo) (*formulatemplate.FormulaTemplateVersion, error)
	getForkedTemplatesFunc    func(ctx context.Context, req *repositories.GetForkedTemplatesRequest) ([]*formulatemplate.FormulaTemplate, error)
	updateTagsFunc            func(ctx context.Context, req *repositories.UpdateVersionTagsRequest) (*formulatemplate.FormulaTemplateVersion, error)
}

func (m *mockVersionRepo) Create(
	ctx context.Context,
	version *formulatemplate.FormulaTemplateVersion,
) (*formulatemplate.FormulaTemplateVersion, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, version)
	}
	version.ID = pulid.MustNew("ftv_")
	return version, nil
}

func (m *mockVersionRepo) GetByTemplateAndVersion(
	ctx context.Context,
	req *repositories.GetVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	if m.getByTemplateAndVersionFn != nil {
		return m.getByTemplateAndVersionFn(ctx, req)
	}
	return nil, errNotFound
}

func (m *mockVersionRepo) List(
	ctx context.Context,
	req *repositories.ListVersionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, req)
	}
	return &pagination.ListResult[*formulatemplate.FormulaTemplateVersion]{
		Items: []*formulatemplate.FormulaTemplateVersion{},
		Total: 0,
	}, nil
}

func (m *mockVersionRepo) GetVersionRange(
	ctx context.Context,
	req *repositories.GetVersionRangeRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	if m.getVersionRangeFunc != nil {
		return m.getVersionRangeFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockVersionRepo) GetLatestVersion(
	ctx context.Context,
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.FormulaTemplateVersion, error) {
	if m.getLatestVersionFunc != nil {
		return m.getLatestVersionFunc(ctx, templateID, tenantInfo)
	}
	return nil, errNotFound
}

func (m *mockVersionRepo) GetForkedTemplates(
	ctx context.Context,
	req *repositories.GetForkedTemplatesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	if m.getForkedTemplatesFunc != nil {
		return m.getForkedTemplatesFunc(ctx, req)
	}
	return []*formulatemplate.FormulaTemplate{}, nil
}

func (m *mockVersionRepo) UpdateTags(
	ctx context.Context,
	req *repositories.UpdateVersionTagsRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	if m.updateTagsFunc != nil {
		return m.updateTagsFunc(ctx, req)
	}
	return &formulatemplate.FormulaTemplateVersion{}, nil
}

func newTestFormulaService() *formula.Service {
	registry := schema.NewRegistry()
	registerShipmentSchema(registry)
	res := resolver.NewResolver()
	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})
	eng := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	return formula.NewService(formula.ServiceParams{
		Logger:   zap.NewNop(),
		Registry: registry,
		Engine:   eng,
		Resolver: res,
	})
}

func registerShipmentSchema(registry *schema.Registry) {
	const shipmentSchema = `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "shipment-test-schema",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"customer": {
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"code": { "type": "string" }
				}
			},
			"weight": { "type": "number" },
			"pieces": { "type": "integer" },
			"ratingUnit": { "type": "integer" },
			"freightChargeAmount": { "type": "number" },
			"otherChargeAmount": { "type": "number" },
			"currentTotalCharge": { "type": "number" },
			"totalDistance": { "type": "number" },
			"totalStops": { "type": "integer" },
			"totalWeight": { "type": "number" },
			"totalPieces": { "type": "integer" },
			"totalLinearFeet": { "type": "number" },
			"hasHazmat": { "type": "boolean" },
			"requiresTemperatureControl": { "type": "boolean" },
			"temperatureDifferential": { "type": "number" }
		}
	}`

	if err := registry.Register("shipment", []byte(shipmentSchema)); err != nil {
		panic(err)
	}
}

func setupHandler(
	t *testing.T,
	repo *mockFormulaTemplateRepo,
	versionRepo *mockVersionRepo,
) *formulatemplatehandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := formulatemplateservice.New(formulatemplateservice.Params{
		Logger:         logger,
		Repo:           repo,
		VersionRepo:    versionRepo,
		FormulaService: newTestFormulaService(),
		AuditService:   &mocks.NoopAuditService{},
	})

	cfg := &config.Config{
		App: config.AppConfig{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	return formulatemplatehandler.New(formulatemplatehandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
	})
}

func TestFormulaTemplateHandler_List_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		listFunc: func(_ context.Context, _ *repositories.ListFormulaTemplatesRequest) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			return &pagination.ListResult[*formulatemplate.FormulaTemplate]{
				Items: []*formulatemplate.FormulaTemplate{
					{
						ID:             ftID,
						OrganizationID: testutil.TestOrgID,
						BusinessUnitID: testutil.TestBuID,
						Name:           "Test Template",
						Type:           formulatemplate.TemplateTypeFreightCharge,
						Expression:     "totalDistance * 2.5",
						Status:         formulatemplate.StatusActive,
						SchemaID:       "shipment",
					},
				},
				Total: 1,
			}, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestFormulaTemplateHandler_List_Error(t *testing.T) {
	t.Parallel()

	repo := &mockFormulaTemplateRepo{
		listFunc: func(_ context.Context, _ *repositories.ListFormulaTemplatesRequest) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			return nil, errors.New("database error")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Get_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, req repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:             req.TemplateID,
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "Test Template",
				Type:           formulatemplate.TemplateTypeFreightCharge,
				Expression:     "totalDistance * 2.5",
				Status:         formulatemplate.StatusActive,
				SchemaID:       "shipment",
			}, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Test Template", resp["name"])
}

func TestFormulaTemplateHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/invalid-id/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return nil, errNotFound
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_GetUsage_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		countUsagesFunc: func(_ context.Context, _ *repositories.GetTemplateUsageRequest) (*repositories.GetTemplateUsageResponse, error) {
			return &repositories.GetTemplateUsageResponse{
				InUse: true,
				Usages: []repositories.TemplateUsageCount{
					{Type: "accessorial_charge", Count: 3},
				},
			}, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/usage").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["inUse"])
}

func TestFormulaTemplateHandler_GetUsage_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/invalid-id/usage").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_GetUsage_Error(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		countUsagesFunc: func(_ context.Context, _ *repositories.GetTemplateUsageRequest) (*repositories.GetTemplateUsageResponse, error) {
			return nil, errors.New("database error")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/usage").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := &mockFormulaTemplateRepo{
		createFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			entity.ID = pulid.MustNew("ft_")
			return entity, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "New Template",
			"type":       "FreightCharge",
			"expression": "totalDistance * 3.0",
			"status":     "Active",
			"schemaId":   "shipment",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "New Template", resp["name"])
}

func TestFormulaTemplateHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := &mockFormulaTemplateRepo{
		createFunc: func(_ context.Context, _ *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("create failed")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "New Template",
			"type":       "FreightCharge",
			"expression": "totalDistance * 3.0",
			"status":     "Active",
			"schemaId":   "shipment",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Update_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, req repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   req.TemplateID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Old Template",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2.5",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 1,
			}, nil
		},
		updateFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			return entity, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Updated Template",
			"type":       "FreightCharge",
			"expression": "totalDistance * 5.0",
			"status":     "Active",
			"schemaId":   "shipment",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated Template", resp["name"])
}

func TestFormulaTemplateHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/formula-templates/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Updated Template",
			"type":       "FreightCharge",
			"expression": "totalDistance * 5.0",
			"status":     "Active",
			"schemaId":   "shipment",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, req repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   req.TemplateID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Old Template",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2.5",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 1,
			}, nil
		},
		updateFunc: func(_ context.Context, _ *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("update failed")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Updated Template",
			"type":       "FreightCharge",
			"expression": "totalDistance * 5.0",
			"status":     "Active",
			"schemaId":   "shipment",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Duplicate_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		bulkDuplicateFunc: func(_ context.Context, _ *repositories.BulkDuplicateFormulaTemplateRequest) ([]*formulatemplate.FormulaTemplate, error) {
			return []*formulatemplate.FormulaTemplate{
				{
					ID:             pulid.MustNew("ft_"),
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Name:           "Test Template (Copy)",
					Type:           formulatemplate.TemplateTypeFreightCharge,
					Expression:     "totalDistance * 2.5",
					Status:         formulatemplate.StatusDraft,
					SchemaID:       "shipment",
				},
			}, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/duplicate").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"templateIds": []string{ftID.String()},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "Test Template (Copy)", resp[0]["name"])
}

func TestFormulaTemplateHandler_Duplicate_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/duplicate").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Duplicate_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		bulkDuplicateFunc: func(_ context.Context, _ *repositories.BulkDuplicateFormulaTemplateRequest) ([]*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("duplicate failed")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/duplicate").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"templateIds": []string{ftID.String()},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_BulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()

	ftID1 := pulid.MustNew("ft_")
	ftID2 := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		bulkUpdateStatusFunc: func(_ context.Context, _ *repositories.BulkUpdateFormulaTemplateStatusRequest) ([]*formulatemplate.FormulaTemplate, error) {
			return []*formulatemplate.FormulaTemplate{
				{
					ID:             ftID1,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Name:           "Template 1",
					Status:         formulatemplate.StatusInactive,
				},
				{
					ID:             ftID2,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					Name:           "Template 2",
					Status:         formulatemplate.StatusInactive,
				},
			}, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/bulk-update-status").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"templateIds": []string{ftID1.String(), ftID2.String()},
			"status":      "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 2)
}

func TestFormulaTemplateHandler_BulkUpdateStatus_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/bulk-update-status").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_BulkUpdateStatus_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		bulkUpdateStatusFunc: func(_ context.Context, _ *repositories.BulkUpdateFormulaTemplateStatusRequest) ([]*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("bulk update failed")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/bulk-update-status").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"templateIds": []string{ftID.String()},
			"status":      "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Patch_UpdatesOnlyProvidedFields(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")

	existingTemplate := &formulatemplate.FormulaTemplate{
		ID:                   templateID,
		OrganizationID:       testutil.TestOrgID,
		BusinessUnitID:       testutil.TestBuID,
		Name:                 "Original Name",
		Description:          "Original Description",
		Type:                 formulatemplate.TemplateTypeFreightCharge,
		Expression:           "totalDistance * 2",
		Status:               formulatemplate.StatusActive,
		SchemaID:             "shipment",
		Version:              1,
		CurrentVersionNumber: 1,
	}

	var updatedEntity *formulatemplate.FormulaTemplate

	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			copied := *existingTemplate
			return &copied, nil
		},
		updateFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			updatedEntity = entity
			return entity, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + templateID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]string{
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	require.NotNil(t, updatedEntity)
	assert.Equal(t, "Original Name", updatedEntity.Name)
	assert.Equal(t, "Original Description", updatedEntity.Description)
	assert.Equal(t, formulatemplate.TemplateTypeFreightCharge, updatedEntity.Type)
	assert.Equal(t, "totalDistance * 2", updatedEntity.Expression)
	assert.Equal(t, formulatemplate.StatusInactive, updatedEntity.Status)
}

func TestFormulaTemplateHandler_Patch_ReturnsNotFoundForMissingEntity(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")

	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return nil, errNotFound
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + templateID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]string{
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]string{
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_Patch_BadJSON(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")

	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   templateID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Original Name",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 1,
			}, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + templateID.String() + "/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Patch_ServiceError(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")

	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   templateID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Original Name",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 1,
			}, nil
		},
		updateFunc: func(_ context.Context, _ *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("update failed")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + templateID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]string{
			"status": "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Patch_PreservesAllFields(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")

	existingTemplate := &formulatemplate.FormulaTemplate{
		ID:                   templateID,
		OrganizationID:       testutil.TestOrgID,
		BusinessUnitID:       testutil.TestBuID,
		Name:                 "Test Template",
		Description:          "A test formula template",
		Type:                 formulatemplate.TemplateTypeAccessorialCharge,
		Expression:           "freightChargeAmount * 0.15",
		Status:               formulatemplate.StatusDraft,
		SchemaID:             "shipment",
		Version:              5,
		CurrentVersionNumber: 1,
	}

	var updatedEntity *formulatemplate.FormulaTemplate

	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			copied := *existingTemplate
			return &copied, nil
		},
		updateFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			updatedEntity = entity
			return entity, nil
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + templateID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]string{
			"status": "Active",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	require.NotNil(t, updatedEntity)
	assert.Equal(t, templateID, updatedEntity.ID)
	assert.Equal(t, testutil.TestOrgID, updatedEntity.OrganizationID)
	assert.Equal(t, testutil.TestBuID, updatedEntity.BusinessUnitID)
	assert.Equal(t, "Test Template", updatedEntity.Name)
	assert.Equal(t, "A test formula template", updatedEntity.Description)
	assert.Equal(t, formulatemplate.TemplateTypeAccessorialCharge, updatedEntity.Type)
	assert.Equal(t, "freightChargeAmount * 0.15", updatedEntity.Expression)
	assert.Equal(t, formulatemplate.StatusActive, updatedEntity.Status)
	assert.Equal(t, "shipment", updatedEntity.SchemaID)
	assert.Equal(t, int64(5), updatedEntity.Version)
}

func TestFormulaTemplateHandler_TestExpression_Success(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/test").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"expression": "totalDistance * 2.5",
			"schemaId":   "shipment",
			"variables": map[string]any{
				"totalDistance": 100.0,
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["valid"])
	assert.Equal(t, "250", resp["result"])
}

func TestFormulaTemplateHandler_TestExpression_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/test").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_TestExpression_DefaultSchemaID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/test").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"expression": "totalDistance * 2.5",
			"variables": map[string]any{
				"totalDistance": 100.0,
			},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["valid"])
}

func TestFormulaTemplateHandler_TestExpression_InvalidExpression(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/test").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"expression": "invalid @@@ expression !!!",
			"schemaId":   "shipment",
			"variables":  map[string]any{},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, false, resp["valid"])
	assert.NotEmpty(t, resp["error"])
}

func TestFormulaTemplateHandler_ListVersions_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		listFunc: func(_ context.Context, _ *repositories.ListVersionsRequest) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
			return &pagination.ListResult[*formulatemplate.FormulaTemplateVersion]{
				Items: []*formulatemplate.FormulaTemplateVersion{
					{
						ID:            pulid.MustNew("ftv_"),
						TemplateID:    ftID,
						VersionNumber: 1,
						Name:          "Test Template",
						Expression:    "totalDistance * 2.5",
						Type:          formulatemplate.TemplateTypeFreightCharge,
						Status:        formulatemplate.StatusActive,
						SchemaID:      "shipment",
						ChangeMessage: "Initial version",
					},
				},
				Total: 1,
			}, nil
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestFormulaTemplateHandler_ListVersions_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/invalid-id/versions").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_GetVersion_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		getByTemplateAndVersionFn: func(_ context.Context, req *repositories.GetVersionRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			return &formulatemplate.FormulaTemplateVersion{
				ID:            pulid.MustNew("ftv_"),
				TemplateID:    req.TemplateID,
				VersionNumber: req.VersionNumber,
				Name:          "Test Template",
				Expression:    "totalDistance * 2.5",
				Type:          formulatemplate.TemplateTypeFreightCharge,
				Status:        formulatemplate.StatusActive,
				SchemaID:      "shipment",
				ChangeMessage: "Initial version",
			}, nil
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/1").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Test Template", resp["name"])
}

func TestFormulaTemplateHandler_GetVersion_InvalidTemplateID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/invalid-id/versions/1").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_GetVersion_InvalidVersionNumber(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/abc").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_GetVersion_NotFound(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		getByTemplateAndVersionFn: func(_ context.Context, _ *repositories.GetVersionRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			return nil, errNotFound
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/99").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_CreateVersion_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   ftID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Test Template",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2.5",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 1,
			}, nil
		},
		updateFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			return entity, nil
		},
	}

	versionRepo := &mockVersionRepo{
		getByTemplateAndVersionFn: func(_ context.Context, _ *repositories.GetVersionRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			return &formulatemplate.FormulaTemplateVersion{
				ID:            pulid.MustNew("ftv_"),
				TemplateID:    ftID,
				VersionNumber: 1,
				Name:          "Test Template",
				Expression:    "totalDistance * 2.5",
				Type:          formulatemplate.TemplateTypeFreightCharge,
				Status:        formulatemplate.StatusActive,
				SchemaID:      "shipment",
			}, nil
		},
		createFunc: func(_ context.Context, version *formulatemplate.FormulaTemplateVersion) (*formulatemplate.FormulaTemplateVersion, error) {
			version.ID = pulid.MustNew("ftv_")
			return version, nil
		},
	}

	handler := setupHandler(t, repo, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"changeMessage": "Added new logic",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.NotEmpty(t, resp["id"])
}

func TestFormulaTemplateHandler_CreateVersion_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/invalid-id/versions").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"changeMessage": "test",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_CreateVersion_BadJSON(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_CreateVersion_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("not found")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"changeMessage": "test",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Rollback_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   ftID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Test Template",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 5.0",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 3,
			}, nil
		},
		updateFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			return entity, nil
		},
	}

	versionRepo := &mockVersionRepo{
		getByTemplateAndVersionFn: func(_ context.Context, req *repositories.GetVersionRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			return &formulatemplate.FormulaTemplateVersion{
				ID:            pulid.MustNew("ftv_"),
				TemplateID:    ftID,
				VersionNumber: req.VersionNumber,
				Name:          "Test Template v1",
				Expression:    "totalDistance * 2.5",
				Type:          formulatemplate.TemplateTypeFreightCharge,
				Status:        formulatemplate.StatusActive,
				SchemaID:      "shipment",
			}, nil
		},
		createFunc: func(_ context.Context, version *formulatemplate.FormulaTemplateVersion) (*formulatemplate.FormulaTemplateVersion, error) {
			version.ID = pulid.MustNew("ftv_")
			return version, nil
		},
	}

	handler := setupHandler(t, repo, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/rollback").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"targetVersion": 1,
			"changeMessage": "Rolling back to v1",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Test Template v1", resp["name"])
}

func TestFormulaTemplateHandler_Rollback_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/invalid-id/rollback").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"targetVersion": 1,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_Rollback_BadJSON(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/rollback").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Rollback_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		getByTemplateAndVersionFn: func(_ context.Context, _ *repositories.GetVersionRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			return nil, errors.New("version not found")
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/rollback").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"targetVersion": 99,
			"changeMessage": "Rolling back",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Fork_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   ftID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Source Template",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2.5",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 2,
			}, nil
		},
		createFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			entity.ID = pulid.MustNew("ft_")
			return entity, nil
		},
	}

	versionRepo := &mockVersionRepo{
		getLatestVersionFunc: func(_ context.Context, _ pulid.ID, _ pagination.TenantInfo) (*formulatemplate.FormulaTemplateVersion, error) {
			return &formulatemplate.FormulaTemplateVersion{
				ID:            pulid.MustNew("ftv_"),
				TemplateID:    ftID,
				VersionNumber: 2,
				Name:          "Source Template",
				Description:   "Source description",
				Expression:    "totalDistance * 2.5",
				Type:          formulatemplate.TemplateTypeFreightCharge,
				Status:        formulatemplate.StatusActive,
				SchemaID:      "shipment",
			}, nil
		},
		createFunc: func(_ context.Context, version *formulatemplate.FormulaTemplateVersion) (*formulatemplate.FormulaTemplateVersion, error) {
			version.ID = pulid.MustNew("ftv_")
			return version, nil
		},
	}

	handler := setupHandler(t, repo, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/fork").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"newName":       "Forked Template",
			"changeMessage": "Forked for testing",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Forked Template", resp["name"])
}

func TestFormulaTemplateHandler_Fork_WithSourceVersion(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	sourceVersion := int64(1)
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   ftID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Source Template",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2.5",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 2,
			}, nil
		},
		createFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			entity.ID = pulid.MustNew("ft_")
			return entity, nil
		},
	}

	versionRepo := &mockVersionRepo{
		getByTemplateAndVersionFn: func(_ context.Context, req *repositories.GetVersionRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			if req.VersionNumber == sourceVersion {
				return &formulatemplate.FormulaTemplateVersion{
					ID:            pulid.MustNew("ftv_"),
					TemplateID:    ftID,
					VersionNumber: sourceVersion,
					Name:          "Source Template v1",
					Description:   "v1 description",
					Expression:    "totalDistance * 1.0",
					Type:          formulatemplate.TemplateTypeFreightCharge,
					Status:        formulatemplate.StatusActive,
					SchemaID:      "shipment",
				}, nil
			}
			return nil, errNotFound
		},
		createFunc: func(_ context.Context, version *formulatemplate.FormulaTemplateVersion) (*formulatemplate.FormulaTemplateVersion, error) {
			version.ID = pulid.MustNew("ftv_")
			return version, nil
		},
	}

	handler := setupHandler(t, repo, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/fork").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"newName":       "Forked From V1",
			"sourceVersion": sourceVersion,
			"changeMessage": "Fork from version 1",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Forked From V1", resp["name"])
}

func TestFormulaTemplateHandler_Fork_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/invalid-id/fork").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"newName": "Forked Template",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_Fork_BadJSON(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/fork").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_Fork_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("not found")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/fork").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"newName":       "Forked Template",
			"changeMessage": "test",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_CompareVersions_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		getVersionRangeFunc: func(_ context.Context, _ *repositories.GetVersionRangeRequest) ([]*formulatemplate.FormulaTemplateVersion, error) {
			return []*formulatemplate.FormulaTemplateVersion{
				{
					ID:            pulid.MustNew("ftv_"),
					TemplateID:    ftID,
					VersionNumber: 1,
					Name:          "Template v1",
					Expression:    "totalDistance * 2.5",
					Type:          formulatemplate.TemplateTypeFreightCharge,
					Status:        formulatemplate.StatusActive,
					SchemaID:      "shipment",
				},
				{
					ID:            pulid.MustNew("ftv_"),
					TemplateID:    ftID,
					VersionNumber: 2,
					Name:          "Template v2",
					Expression:    "totalDistance * 5.0",
					Type:          formulatemplate.TemplateTypeFreightCharge,
					Status:        formulatemplate.StatusActive,
					SchemaID:      "shipment",
				},
			}, nil
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/compare").
		WithQuery(map[string]string{"from": "1", "to": "2"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, float64(1), resp["fromVersion"])
	assert.Equal(t, float64(2), resp["toVersion"])
}

func TestFormulaTemplateHandler_CompareVersions_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/invalid-id/compare").
		WithQuery(map[string]string{"from": "1", "to": "2"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_CompareVersions_MissingFromParam(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/compare").
		WithQuery(map[string]string{"to": "2"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_CompareVersions_MissingToParam(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/compare").
		WithQuery(map[string]string{"from": "1"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_CompareVersions_SameVersions(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/compare").
		WithQuery(map[string]string{"from": "1", "to": "1"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Contains(t, resp["error"], "different")
}

func TestFormulaTemplateHandler_CompareVersions_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		getVersionRangeFunc: func(_ context.Context, _ *repositories.GetVersionRangeRequest) ([]*formulatemplate.FormulaTemplateVersion, error) {
			return nil, errors.New("database error")
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/compare").
		WithQuery(map[string]string{"from": "1", "to": "2"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_GetLineage_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	forkedID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:                   ftID,
				OrganizationID:       testutil.TestOrgID,
				BusinessUnitID:       testutil.TestBuID,
				Name:                 "Root Template",
				Type:                 formulatemplate.TemplateTypeFreightCharge,
				Expression:           "totalDistance * 2.5",
				Status:               formulatemplate.StatusActive,
				SchemaID:             "shipment",
				CurrentVersionNumber: 1,
			}, nil
		},
	}

	sourceVersion := int64(1)
	versionRepo := &mockVersionRepo{
		getForkedTemplatesFunc: func(_ context.Context, _ *repositories.GetForkedTemplatesRequest) ([]*formulatemplate.FormulaTemplate, error) {
			return []*formulatemplate.FormulaTemplate{
				{
					ID:                  forkedID,
					OrganizationID:      testutil.TestOrgID,
					BusinessUnitID:      testutil.TestBuID,
					Name:                "Forked Template",
					SourceTemplateID:    &ftID,
					SourceVersionNumber: &sourceVersion,
				},
			}, nil
		},
	}

	handler := setupHandler(t, repo, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/lineage").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Root Template", resp["templateName"])
}

func TestFormulaTemplateHandler_GetLineage_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/invalid-id/lineage").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_GetLineage_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("not found")
		},
	}

	handler := setupHandler(t, repo, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/lineage").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_UpdateVersionTags_Success(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		updateTagsFunc: func(_ context.Context, req *repositories.UpdateVersionTagsRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			return &formulatemplate.FormulaTemplateVersion{
				ID:            pulid.MustNew("ftv_"),
				TemplateID:    req.TemplateID,
				VersionNumber: req.VersionNumber,
				Name:          "Test Template",
				Expression:    "totalDistance * 2.5",
				Type:          formulatemplate.TemplateTypeFreightCharge,
				Status:        formulatemplate.StatusActive,
				SchemaID:      "shipment",
				Tags: []formulatemplate.VersionTag{
					formulatemplate.VersionTagStable,
					formulatemplate.VersionTagProduction,
				},
			}, nil
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/1/tags").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"tags": []string{"Stable", "Production"},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.NotNil(t, resp["tags"])
}

func TestFormulaTemplateHandler_UpdateVersionTags_InvalidTemplateID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/invalid-id/versions/1/tags").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"tags": []string{"Stable"},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestFormulaTemplateHandler_UpdateVersionTags_InvalidVersionNumber(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/abc/tags").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"tags": []string{"Stable"},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_UpdateVersionTags_BadJSON(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/1/tags").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_UpdateVersionTags_InvalidTag(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	handler := setupHandler(t, &mockFormulaTemplateRepo{}, &mockVersionRepo{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/1/tags").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"tags": []string{"InvalidTag"},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestFormulaTemplateHandler_UpdateVersionTags_ServiceError(t *testing.T) {
	t.Parallel()

	ftID := pulid.MustNew("ft_")
	versionRepo := &mockVersionRepo{
		updateTagsFunc: func(_ context.Context, _ *repositories.UpdateVersionTagsRequest) (*formulatemplate.FormulaTemplateVersion, error) {
			return nil, errors.New("update tags failed")
		},
	}

	handler := setupHandler(t, &mockFormulaTemplateRepo{}, versionRepo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/formula-templates/" + ftID.String() + "/versions/1/tags").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"tags": []string{"Stable"},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}
