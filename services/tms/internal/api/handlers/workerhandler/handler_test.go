package workerhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/workerhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customfieldportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("worker not found")

type handlerDeps struct {
	handler   *workerhandler.Handler
	valueRepo *customfieldportsmock.MockCustomFieldValueRepository
	defRepo   *customfieldportsmock.MockCustomFieldDefinitionRepository
	auditSvc  *auditportsmock.MockAuditService
	cacheRepo *customerportsmock.MockWorkerCacheRepository
}

func setupWorkerHandler(
	t *testing.T,
	repo *workerportsmock.MockWorkerRepository,
) *handlerDeps {
	t.Helper()
	return setupWorkerHandlerWithEngine(t, repo, &mocks.AllowAllPermissionEngine{})
}

func setupWorkerHandlerWithEngine(
	t *testing.T,
	repo *workerportsmock.MockWorkerRepository,
	engine permissionports.PermissionEngine,
) *handlerDeps {
	t.Helper()
	return setupWorkerHandlerWithEngines(t, repo, &mocks.AllowAllPermissionEngine{}, engine)
}

func setupWorkerHandlerWithEngines(
	t *testing.T,
	repo *workerportsmock.MockWorkerRepository,
	routes permissionports.PermissionEngine,
	engine permissionports.PermissionEngine,
) *handlerDeps {
	t.Helper()

	logger := zap.NewNop()
	valueRepo := customfieldportsmock.NewMockCustomFieldValueRepository(t)
	defRepo := customfieldportsmock.NewMockCustomFieldDefinitionRepository(t)
	auditSvc := auditportsmock.NewMockAuditService(t)

	valuesValidator := customfieldservice.NewValuesValidator(
		customfieldservice.ValuesValidatorParams{
			Logger: logger,
			Repo:   defRepo,
		},
	)
	cacheRepo := customerportsmock.NewMockWorkerCacheRepository(t)

	cfService := customfieldservice.NewValuesService(customfieldservice.ValuesServiceParams{
		Logger:         logger,
		ValueRepo:      valueRepo,
		DefinitionRepo: defRepo,
		Validator:      valuesValidator,
	})

	service := workerservice.New(workerservice.Params{
		Logger:                    logger,
		Repo:                      repo,
		CacheRepo:                 cacheRepo,
		Validator:                 workerservice.NewTestValidator(),
		AuditService:              auditSvc,
		Realtime:                  &mocks.NoopRealtimeService{},
		CustomFieldsValuesService: cfService,
	})

	cfg := &config.Config{
		App: appconfig.Config{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &cfg.App,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: routes,
		ErrorHandler:     errorHandler,
	})

	handler := workerhandler.New(workerhandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
		PermissionEngine:     engine,
	})

	return &handlerDeps{
		handler:   handler,
		valueRepo: valueRepo,
		defRepo:   defRepo,
		auditSvc:  auditSvc,
		cacheRepo: cacheRepo,
	}
}

func TestWorkerHandler_List_Success(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	total := 42
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.CursorListResult[*worker.Worker]{
			Items: []*worker.Worker{
				{
					ID:             wkrID,
					OrganizationID: testutil.TestOrgID,
					BusinessUnitID: testutil.TestBuID,
					FirstName:      "John",
					LastName:       "Doe",
					Status:         domaintypes.StatusActive,
				},
			},
			TotalCount: &total,
		}, nil)

	deps := setupWorkerHandler(t, repo)
	deps.valueRepo.On("GetByResources", mock.Anything, mock.Anything).
		Return(make(map[string][]*customfield.CustomFieldValue), nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/workers/").
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.CursorResponse[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, total, resp.Count)
	require.NotNil(t, resp.TotalCount)
	assert.Equal(t, total, *resp.TotalCount)
	assert.Len(t, resp.Results, 1)
}

// Row-level security shows every worker to a person who cannot read workers,
// because a record that links to one shows its reference view. A root read of
// workers is therefore refused by the route before it reaches the database.
func TestWorkerHandler_RefusesARootReadWithoutWorkerRead(t *testing.T) {
	t.Parallel()

	noWorkerRead := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	}
	for _, path := range []string{
		"/api/v1/workers/",
		"/api/v1/workers/" + pulid.MustNew("wrk_").String() + "/",
	} {
		repo := workerportsmock.NewMockWorkerRepository(t)
		deps := setupWorkerHandlerWithEngines(t, repo, noWorkerRead, noWorkerRead)

		ginCtx := testutil.NewGinTestContext().
			WithMethod(http.MethodGet).
			WithPath(path).
			WithDefaultAuthContext()
		deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
		ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

		assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode(), path)
		repo.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
		repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	}
}

func TestWorkerHandler_Get_Success(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		FirstName:      "John",
		LastName:       "Doe",
		Status:         domaintypes.StatusActive,
	}, nil)

	deps := setupWorkerHandler(t, repo)
	deps.cacheRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, customerports.ErrCacheMiss)
	deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldValue{}, nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "John", resp["firstName"])
}

func getWorkerAs(t *testing.T, engine permissionports.PermissionEngine) map[string]any {
	t.Helper()

	wkrID := pulid.MustNew("wrk_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		FirstName:      "John",
		Status:         domaintypes.StatusActive,
		Profile: &worker.WorkerProfile{
			DOB:            946684800,
			LicenseNumber:  "DL123456",
			TWICCardNumber: "TWIC-1",
			LastDrugTest:   1767225600,
			LicenseExpiry:  1893456000,
		},
	}, nil)

	deps := setupWorkerHandlerWithEngine(t, repo, engine)
	deps.cacheRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, customerports.ErrCacheMiss)
	deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldValue{}, nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	profile, ok := resp["profile"].(map[string]any)
	require.True(t, ok)
	return profile
}

func TestWorkerHandler_Get_WithholdsPersonalDetailsFromAReaderWithoutTheGroup(t *testing.T) {
	t.Parallel()

	profile := getWorkerAs(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceProtectedPersonal.String()},
	})

	assert.Equal(t, "", profile["licenseNumber"])
	assert.Equal(t, "", profile["twicCardNumber"])
	assert.InDelta(t, 0, profile["dob"], 0)
	assert.InDelta(t, 0, profile["lastDrugTest"], 0)
	assert.InDelta(t, 1893456000, profile["licenseExpiry"], 0)
}

func TestWorkerHandler_Get_ShowsPersonalDetailsToAReaderWithTheGroup(t *testing.T) {
	t.Parallel()

	profile := getWorkerAs(t, &mocks.AllowAllPermissionEngine{})

	assert.Equal(t, "DL123456", profile["licenseNumber"])
	assert.Equal(t, "TWIC-1", profile["twicCardNumber"])
	assert.InDelta(t, 946684800, profile["dob"], 0)
}

func TestWorkerHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/workers/invalid-id/").
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestWorkerHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	deps := setupWorkerHandler(t, repo)
	deps.cacheRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, customerports.ErrCacheMiss)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestWorkerHandler_Create_Success(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("uss_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *worker.Worker) *worker.Worker {
			entity.ID = pulid.MustNew("wrk_")
			return entity
		}, nil)

	deps := setupWorkerHandler(t, repo)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/workers/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName":    "John",
			"lastName":     "Doe",
			"status":       "Active",
			"type":         "Employee",
			"driverType":   "OTR",
			"gender":       "Male",
			"addressLine1": "123 Main St",
			"city":         "Springfield",
			"postalCode":   "12345",
			"stateId":      stateID.String(),
			"profile": map[string]any{
				"dob":              946684800,
				"licenseNumber":    "DL123456",
				"endorsement":      "O",
				"licenseExpiry":    1893456000,
				"hireDate":         1609459200,
				"complianceStatus": "Pending",
			},
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "John", resp["firstName"])
}

func TestWorkerHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/workers/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestWorkerHandler_Update_Success(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	stateID := pulid.MustNew("uss_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	// A persisted worker always carries a status, and the edit form refuses to
	// move it — employment state only changes through a timeline event.
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *worker.Worker) *worker.Worker {
			return entity
		}, nil)

	deps := setupWorkerHandler(t, repo)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName":    "Jane",
			"lastName":     "Smith",
			"status":       "Active",
			"type":         "Employee",
			"driverType":   "OTR",
			"gender":       "Female",
			"addressLine1": "456 Oak Ave",
			"city":         "Portland",
			"postalCode":   "97201",
			"stateId":      stateID.String(),
			"profile": map[string]any{
				"dob":              946684800,
				"licenseNumber":    "DL789012",
				"endorsement":      "O",
				"licenseExpiry":    1893456000,
				"hireDate":         1609459200,
				"complianceStatus": "Pending",
			},
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Jane", resp["firstName"])
}

func TestWorkerHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/workers/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName": "Jane",
			"lastName":  "Smith",
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestWorkerHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	stateID := pulid.MustNew("uss_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		FirstName:      "John",
		LastName:       "Doe",
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		Gender:         worker.GenderMale,
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "12345",
		StateID:        stateID,
		Profile: &worker.WorkerProfile{
			DOB:              946684800,
			LicenseNumber:    "DL123456",
			Endorsement:      worker.EndorsementTypeNone,
			LicenseExpiry:    1893456000,
			HireDate:         1609459200,
			ComplianceStatus: worker.ComplianceStatusPending,
		},
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *worker.Worker) *worker.Worker {
			return entity
		}, nil)

	deps := setupWorkerHandler(t, repo)
	deps.cacheRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, customerports.ErrCacheMiss)
	deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldValue{}, nil)
	deps.defRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldDefinition{}, nil)
	deps.valueRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName": "Updated",
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated", resp["firstName"])
}

func TestWorkerHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/workers/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName": "Updated",
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestWorkerHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("uss_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	deps := setupWorkerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/workers/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName":    "John",
			"lastName":     "Doe",
			"status":       "Active",
			"type":         "Employee",
			"driverType":   "OTR",
			"gender":       "Male",
			"addressLine1": "123 Main St",
			"city":         "Springfield",
			"postalCode":   "12345",
			"stateId":      stateID.String(),
			"profile": map[string]any{
				"dob":              946684800,
				"licenseNumber":    "DL123456",
				"endorsement":      "O",
				"licenseExpiry":    1893456000,
				"hireDate":         1609459200,
				"complianceStatus": "Pending",
			},
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestWorkerHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	stateID := pulid.MustNew("uss_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	// A persisted worker always carries a status, and the edit form refuses to
	// move it — employment state only changes through a timeline event.
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		Status:         domaintypes.StatusActive,
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	deps := setupWorkerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName":    "Jane",
			"lastName":     "Smith",
			"status":       "Active",
			"type":         "Employee",
			"driverType":   "OTR",
			"gender":       "Female",
			"addressLine1": "456 Oak Ave",
			"city":         "Portland",
			"postalCode":   "97201",
			"stateId":      stateID.String(),
			"profile": map[string]any{
				"dob":              946684800,
				"licenseNumber":    "DL789012",
				"endorsement":      "O",
				"licenseExpiry":    1893456000,
				"hireDate":         1609459200,
				"complianceStatus": "Pending",
			},
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestWorkerHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestWorkerHandler_Patch_NotFound(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	deps := setupWorkerHandler(t, repo)
	deps.cacheRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, customerports.ErrCacheMiss)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/workers/" + wkrID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"firstName": "Updated",
		})

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}
