package workerhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/workerhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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
	valueRepo *mocks.MockCustomFieldValueRepository
	defRepo   *mocks.MockCustomFieldDefinitionRepository
	auditSvc  *mocks.MockAuditService
}

func setupWorkerHandler(
	t *testing.T,
	repo *mocks.MockWorkerRepository,
) *handlerDeps {
	t.Helper()

	logger := zap.NewNop()
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	defRepo := mocks.NewMockCustomFieldDefinitionRepository(t)
	auditSvc := mocks.NewMockAuditService(t)

	valuesValidator := customfieldservice.NewValuesValidator(
		customfieldservice.ValuesValidatorParams{
			Logger: logger,
			Repo:   defRepo,
		},
	)

	cfService := customfieldservice.NewValuesService(customfieldservice.ValuesServiceParams{
		Logger:         logger,
		ValueRepo:      valueRepo,
		DefinitionRepo: defRepo,
		Validator:      valuesValidator,
	})

	service := workerservice.New(workerservice.Params{
		Logger:                    logger,
		Repo:                      repo,
		Validator:                 workerservice.NewTestValidator(),
		AuditService:              auditSvc,
		Realtime:                  &mocks.NoopRealtimeService{},
		CustomFieldsValuesService: cfService,
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

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	handler := workerhandler.New(workerhandler.Params{
		Service:              service,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})

	return &handlerDeps{
		handler:   handler,
		valueRepo: valueRepo,
		defRepo:   defRepo,
		auditSvc:  auditSvc,
	}
}

func TestWorkerHandler_List_Success(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	repo := mocks.NewMockWorkerRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*worker.Worker]{
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
		Total: 1,
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

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestWorkerHandler_Get_Success(t *testing.T) {
	t.Parallel()

	wkrID := pulid.MustNew("wrk_")
	repo := mocks.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		FirstName:      "John",
		LastName:       "Doe",
		Status:         domaintypes.StatusActive,
	}, nil)

	deps := setupWorkerHandler(t, repo)
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

func TestWorkerHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockWorkerRepository(t)
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
	repo := mocks.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	deps := setupWorkerHandler(t, repo)

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
	repo := mocks.NewMockWorkerRepository(t)
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

	repo := mocks.NewMockWorkerRepository(t)
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
	repo := mocks.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
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

	repo := mocks.NewMockWorkerRepository(t)
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
	repo := mocks.NewMockWorkerRepository(t)
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

	repo := mocks.NewMockWorkerRepository(t)
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
	repo := mocks.NewMockWorkerRepository(t)
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
	repo := mocks.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&worker.Worker{
		ID:             wkrID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
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
	repo := mocks.NewMockWorkerRepository(t)
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
	repo := mocks.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	deps := setupWorkerHandler(t, repo)

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
