package workerservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*worker.Worker]().
			WithModelName("Worker").
			Build(),
	}
}

type testDeps struct {
	repo      *mocks.MockWorkerRepository
	audit     *mocks.MockAuditService
	valueRepo *mocks.MockCustomFieldValueRepository
	defRepo   *mocks.MockCustomFieldDefinitionRepository
	cacheRepo *mocks.MockWorkerCacheRepository
	svc       *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockWorkerRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	defRepo := mocks.NewMockCustomFieldDefinitionRepository(t)
	cacheRepo := mocks.NewMockWorkerCacheRepository(t)

	logger := zap.NewNop()

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

	svc := &Service{
		l:                         logger,
		repo:                      repo,
		cacheRepo:                 cacheRepo,
		validator:                 newTestValidator(),
		auditService:              auditSvc,
		realtime:                  &mocks.NoopRealtimeService{},
		customFieldsValuesService: cfService,
	}
	return &testDeps{repo: repo, audit: auditSvc, valueRepo: valueRepo, defRepo: defRepo, cacheRepo: cacheRepo, svc: svc}
}

func newTestWorker() *worker.Worker {
	return &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		StateID:        pulid.MustNew("st_"),
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      "John",
		LastName:       "Doe",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "12345",
		Gender:         worker.GenderMale,
		Version:        1,
		Profile: &worker.WorkerProfile{
			DOB:              946684800,
			LicenseNumber:    "DL123456",
			Endorsement:      worker.EndorsementTypeNone,
			LicenseExpiry:    1893456000,
			HireDate:         1609459200,
			ComplianceStatus: worker.ComplianceStatusPending,
		},
	}
}

func newCreateWorker() *worker.Worker {
	return &worker.Worker{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		StateID:        pulid.MustNew("st_"),
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      "Jane",
		LastName:       "Smith",
		AddressLine1:   "456 Oak Ave",
		City:           "Portland",
		PostalCode:     "97201",
		Gender:         worker.GenderFemale,
		Profile: &worker.WorkerProfile{
			DOB:              946684800,
			LicenseNumber:    "DL789012",
			Endorsement:      worker.EndorsementTypeNone,
			LicenseExpiry:    1893456000,
			HireDate:         1609459200,
			ComplianceStatus: worker.ComplianceStatusPending,
		},
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("returns workers successfully", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		expected := &pagination.ListResult[*worker.Worker]{
			Items: []*worker.Worker{newTestWorker(), newTestWorker()},
			Total: 2,
		}
		req := &repositories.ListWorkersRequest{
			Filter: &pagination.QueryOptions{},
		}

		deps.repo.On("List", mock.Anything, req).Return(expected, nil)
		deps.valueRepo.On("GetByResources", mock.Anything, mock.Anything).
			Return(make(map[string][]*customfield.CustomFieldValue), nil)

		result, err := deps.svc.List(t.Context(), req)

		require.NoError(t, err)
		assert.Equal(t, 2, result.Total)
		assert.Len(t, result.Items, 2)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns empty list", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		expected := &pagination.ListResult[*worker.Worker]{
			Items: []*worker.Worker{},
			Total: 0,
		}
		req := &repositories.ListWorkersRequest{
			Filter: &pagination.QueryOptions{},
		}

		deps.repo.On("List", mock.Anything, req).Return(expected, nil)

		result, err := deps.svc.List(t.Context(), req)

		require.NoError(t, err)
		assert.Equal(t, 0, result.Total)
		assert.Empty(t, result.Items)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		req := &repositories.ListWorkersRequest{
			Filter: &pagination.QueryOptions{},
		}
		repoErr := errors.New("database connection failed")
		deps.repo.On("List", mock.Anything, req).Return(nil, repoErr)

		result, err := deps.svc.List(t.Context(), req)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, repoErr, err)
		deps.repo.AssertExpectations(t)
	})

	t.Run("passes nil filter", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		req := &repositories.ListWorkersRequest{
			Filter: nil,
		}
		expected := &pagination.ListResult[*worker.Worker]{
			Items: []*worker.Worker{},
			Total: 0,
		}
		deps.repo.On("List", mock.Anything, req).Return(expected, nil)

		result, err := deps.svc.List(t.Context(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		deps.repo.AssertExpectations(t)
	})
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("returns worker by ID", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		entity := newTestWorker()

		req := repositories.GetWorkerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}

		deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
		deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)
		deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
			Return([]*customfield.CustomFieldValue{}, nil)

		result, err := deps.svc.Get(t.Context(), req)

		require.NoError(t, err)
		assert.Equal(t, entity.ID, result.ID)
		assert.Equal(t, entity.FirstName, result.FirstName)
		assert.Equal(t, entity.LastName, result.LastName)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns worker from cache when available", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		entity := newTestWorker()

		req := repositories.GetWorkerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}

		deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(entity, nil).Once()
		deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
			Return([]*customfield.CustomFieldValue{}, nil)

		result, err := deps.svc.Get(t.Context(), req)

		require.NoError(t, err)
		assert.Equal(t, entity.ID, result.ID)
		deps.repo.AssertNotCalled(t, "GetByID")
	})

	t.Run("returns error when not found", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		req := repositories.GetWorkerByIDRequest{
			ID: pulid.MustNew("wrk_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		}

		notFoundErr := errors.New("worker not found")
		deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
		deps.repo.On("GetByID", mock.Anything, req).Return(nil, notFoundErr)

		result, err := deps.svc.Get(t.Context(), req)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, notFoundErr, err)
		deps.repo.AssertExpectations(t)
	})

	t.Run("includes profile when requested", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		entity := newTestWorker()

		req := repositories.GetWorkerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			IncludeProfile: true,
		}

		deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
		deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)
		deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
			Return([]*customfield.CustomFieldValue{}, nil)

		result, err := deps.svc.Get(t.Context(), req)

		require.NoError(t, err)
		assert.NotNil(t, result.Profile)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		req := repositories.GetWorkerByIDRequest{
			ID: pulid.MustNew("wrk_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		}

		dbErr := errors.New("connection timeout")
		deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
		deps.repo.On("GetByID", mock.Anything, req).Return(nil, dbErr)

		result, err := deps.svc.Get(t.Context(), req)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertExpectations(t)
	})

	t.Run("falls back to database on cache miss", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		entity := newTestWorker()

		req := repositories.GetWorkerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}

		deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
		deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)
		deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
			Return([]*customfield.CustomFieldValue{}, nil)

		result, err := deps.svc.Get(t.Context(), req)

		require.NoError(t, err)
		assert.Equal(t, entity.ID, result.ID)
		deps.repo.AssertExpectations(t)
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("creates worker successfully", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		entity := newCreateWorker()
		userID := pulid.MustNew("usr_")

		created := newTestWorker()
		created.BusinessUnitID = entity.BusinessUnitID
		created.OrganizationID = entity.OrganizationID
		created.FirstName = entity.FirstName
		created.LastName = entity.LastName

		deps.repo.On("Create", mock.Anything, entity).Return(created, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.ID)
		assert.Equal(t, entity.FirstName, result.FirstName)
		assert.Equal(t, entity.LastName, result.LastName)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns validation error for missing required fields", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := &worker.Worker{
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
			FirstName:      "",
			LastName:       "",
			AddressLine1:   "",
			City:           "",
			PostalCode:     "",
		}

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Create")
	})

	t.Run("returns validation error for missing profile", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := &worker.Worker{
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
			StateID:        pulid.MustNew("st_"),
			Status:         domaintypes.StatusActive,
			Type:           worker.WorkerTypeEmployee,
			FirstName:      "Test",
			LastName:       "User",
			AddressLine1:   "123 Main St",
			City:           "TestCity",
			PostalCode:     "12345",
			Gender:         worker.GenderMale,
		}

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Create")
	})

	t.Run("returns validation error for invalid postal code", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := newCreateWorker()
		entity.PostalCode = "ABCDE"

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Create")
	})

	t.Run("returns repository error", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newCreateWorker()

		repoErr := errors.New("database error")
		deps.repo.On("Create", mock.Anything, entity).Return(nil, repoErr)

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, repoErr, err)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns validation error for invalid status", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := newCreateWorker()
		entity.Status = "InvalidStatus"

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Create")
	})

	t.Run("returns validation error for invalid worker type", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := newCreateWorker()
		entity.Type = "InvalidType"

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Create")
	})

	t.Run("creates contractor worker", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := newCreateWorker()
		entity.Type = worker.WorkerTypeContractor

		created := newTestWorker()
		created.Type = worker.WorkerTypeContractor

		deps.repo.On("Create", mock.Anything, entity).Return(created, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		assert.Equal(t, worker.WorkerTypeContractor, result.Type)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns validation error for missing state ID", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := newCreateWorker()
		entity.StateID = ""

		result, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Create")
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("updates worker successfully", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID

		updated := newTestWorker()
		updated.ID = entity.ID
		updated.FirstName = "Updated"
		updated.Version = 2

		deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
		deps.repo.On("Update", mock.Anything, entity).Return(updated, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		result, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		assert.Equal(t, entity.ID, result.ID)
		assert.Equal(t, "Updated", result.FirstName)
		assert.Equal(t, int64(2), result.Version)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns validation error for empty required fields", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := &worker.Worker{
			ID:             pulid.MustNew("wrk_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
			FirstName:      "",
			LastName:       "",
			AddressLine1:   "",
			City:           "",
			PostalCode:     "",
		}

		result, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Update")
	})

	t.Run("returns repository error", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID

		repoErr := errors.New("update conflict")
		deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
		deps.repo.On("Update", mock.Anything, entity).Return(nil, repoErr)

		result, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, repoErr, err)
		deps.repo.AssertExpectations(t)
	})

	t.Run("updates worker status to inactive", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()
		entity.Status = domaintypes.StatusInactive

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID

		updated := newTestWorker()
		updated.ID = entity.ID
		updated.Status = domaintypes.StatusInactive

		deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
		deps.repo.On("Update", mock.Anything, entity).Return(updated, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		result, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		assert.Equal(t, domaintypes.StatusInactive, result.Status)
		deps.repo.AssertExpectations(t)
	})

	t.Run("updates driver type", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()
		entity.DriverType = worker.DriverTypeLocal

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID

		updated := newTestWorker()
		updated.ID = entity.ID
		updated.DriverType = worker.DriverTypeLocal

		deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
		deps.repo.On("Update", mock.Anything, entity).Return(updated, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		result, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		assert.Equal(t, worker.DriverTypeLocal, result.DriverType)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns validation error for invalid gender", func(t *testing.T) {
		t.Skip("Skipping until validation rules are added to test validator")
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")

		entity := newTestWorker()
		entity.Gender = "Other"

		result, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		deps.repo.AssertNotCalled(t, "Update")
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockWorkerRepository(t)
	validator := newTestValidator()

	svc := New(Params{
		Logger:    zap.NewNop(),
		Repo:      repo,
		CacheRepo: mocks.NewMockWorkerCacheRepository(t),
		Validator: validator,
		Realtime:  &mocks.NoopRealtimeService{},
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}
