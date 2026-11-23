package workerservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptopolicyports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workercredentialports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customfieldportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dataentrycontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*worker.Worker]().
			WithModelName("Worker").
			Build(),
	}
}

type testDeps struct {
	repo      *workerportsmock.MockWorkerRepository
	audit     *auditportsmock.MockAuditService
	valueRepo *customfieldportsmock.MockCustomFieldValueRepository
	defRepo   *customfieldportsmock.MockCustomFieldDefinitionRepository
	cacheRepo *customerportsmock.MockWorkerCacheRepository
	formatter *dataentrycontrolportsmock.MockTextFormatter
	svc       *Service
}

type noDefaultPTOPolicy struct {
	ptopolicyports.PTOPolicyRepository
}

func (noDefaultPTOPolicy) GetDefault(
	context.Context,
	pagination.TenantInfo,
) (*pto.PTOPolicy, error) {
	return nil, errortypes.NewNotFoundError("no default PTO policy")
}

type noCredentialTypes struct {
	workercredentialports.WorkerCredentialRepository
}

func (noCredentialTypes) EnsureSystemTypes(
	context.Context,
	pagination.TenantInfo,
	[]*workercredential.WorkerCredentialType,
) (int, error) {
	return 0, errortypes.NewBusinessError("credential types are not seeded here")
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := workerportsmock.NewMockWorkerRepository(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	valueRepo := customfieldportsmock.NewMockCustomFieldValueRepository(t)
	defRepo := customfieldportsmock.NewMockCustomFieldDefinitionRepository(t)
	cacheRepo := customerportsmock.NewMockWorkerCacheRepository(t)
	formatter := dataentrycontrolportsmock.NewMockTextFormatter(t)
	formatter.EXPECT().Format(mock.Anything, mock.Anything).Return(nil).Maybe()

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
		db:                        dbtest.NopConnection{},
		repo:                      repo,
		cacheRepo:                 cacheRepo,
		validator:                 newTestValidator(),
		auditService:              auditSvc,
		realtime:                  &mocks.NoopRealtimeService{},
		customFieldsValuesService: cfService,
		ptoPolicyService: ptopolicyservice.New(ptopolicyservice.Params{
			Logger: logger,
			Repo:   noDefaultPTOPolicy{},
		}),
		credentialService: workercredentialservice.New(workercredentialservice.Params{
			Logger: logger,
			Repo:   noCredentialTypes{},
		}),
		planFields: plantest.Keeper(nil),
		formatter:  formatter,
	}
	return &testDeps{
		repo:      repo,
		audit:     auditSvc,
		valueRepo: valueRepo,
		defRepo:   defRepo,
		cacheRepo: cacheRepo,
		formatter: formatter,
		svc:       svc,
	}
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

		total := 2
		expected := &pagination.CursorListResult[*worker.Worker]{
			Items:      []*worker.Worker{newTestWorker(), newTestWorker()},
			TotalCount: &total,
		}
		req := &workerports.ListWorkersRequest{
			Filter: &pagination.QueryOptions{},
		}

		deps.repo.On("List", mock.Anything, req).Return(expected, nil)
		deps.valueRepo.On("GetByResources", mock.Anything, mock.Anything).
			Return(make(map[string][]*customfield.CustomFieldValue), nil)

		result, err := deps.svc.List(t.Context(), req)

		require.NoError(t, err)
		require.NotNil(t, result.TotalCount)
		assert.Equal(t, 2, *result.TotalCount)
		assert.Len(t, result.Items, 2)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns empty list", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		total := 0
		expected := &pagination.CursorListResult[*worker.Worker]{
			Items:      []*worker.Worker{},
			TotalCount: &total,
		}
		req := &workerports.ListWorkersRequest{
			Filter: &pagination.QueryOptions{},
		}

		deps.repo.On("List", mock.Anything, req).Return(expected, nil)

		result, err := deps.svc.List(t.Context(), req)

		require.NoError(t, err)
		require.NotNil(t, result.TotalCount)
		assert.Equal(t, 0, *result.TotalCount)
		assert.Empty(t, result.Items)
		deps.repo.AssertExpectations(t)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)

		req := &workerports.ListWorkersRequest{
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

		req := &workerports.ListWorkersRequest{
			Filter: nil,
		}
		total := 0
		expected := &pagination.CursorListResult[*worker.Worker]{
			Items:      []*worker.Worker{},
			TotalCount: &total,
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

		req := workerports.GetWorkerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}

		deps.cacheRepo.EXPECT().
			GetByID(mock.Anything, req).
			Return(nil, customerports.ErrCacheMiss).
			Once()
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

		req := workerports.GetWorkerByIDRequest{
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

		req := workerports.GetWorkerByIDRequest{
			ID: pulid.MustNew("wrk_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		}

		notFoundErr := errors.New("worker not found")
		deps.cacheRepo.EXPECT().
			GetByID(mock.Anything, req).
			Return(nil, customerports.ErrCacheMiss).
			Once()
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

		req := workerports.GetWorkerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			IncludeProfile: true,
		}

		deps.cacheRepo.EXPECT().
			GetByID(mock.Anything, req).
			Return(nil, customerports.ErrCacheMiss).
			Once()
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

		req := workerports.GetWorkerByIDRequest{
			ID: pulid.MustNew("wrk_"),
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		}

		dbErr := errors.New("connection timeout")
		deps.cacheRepo.EXPECT().
			GetByID(mock.Anything, req).
			Return(nil, customerports.ErrCacheMiss).
			Once()
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

		req := workerports.GetWorkerByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}

		deps.cacheRepo.EXPECT().
			GetByID(mock.Anything, req).
			Return(nil, customerports.ErrCacheMiss).
			Once()
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

	t.Run("never links a portal user at creation", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newCreateWorker()
		entity.UserID = userID

		created := newTestWorker()
		deps.repo.On("Create", mock.Anything, mock.MatchedBy(func(w *worker.Worker) bool {
			return w.UserID.IsNil()
		})).Return(created, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		_, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		assert.True(t, entity.UserID.IsNil())
		deps.repo.AssertExpectations(t)
	})

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
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

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
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

type fixedFieldAccess protectedfield.GroupSet

func (f fixedFieldAccess) Visible(
	context.Context,
	*permissionports.RequestActor,
) protectedfield.GroupSet {
	return protectedfield.GroupSet(f)
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("keeps stored personal details for an editor without the group", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		deps.svc.fields = fixedFieldAccess(0)
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()
		entity.FirstName = "Jonathan"
		entity.Profile.DOB = 0
		entity.Profile.LicenseNumber = ""
		entity.Profile.TWICCardNumber = ""

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID
		original.Profile.TWICCardNumber = "TWIC-1"

		deps.repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req workerports.GetWorkerByIDRequest) bool {
			return req.IncludeProfile
		})).
			Return(original, nil)
		deps.repo.On("Update", mock.Anything, mock.MatchedBy(func(w *worker.Worker) bool {
			return w.FirstName == "Jonathan" &&
				w.Profile.DOB == 946684800 &&
				w.Profile.LicenseNumber == "DL123456" &&
				w.Profile.TWICCardNumber == "TWIC-1"
		})).Return(entity, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		_, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		deps.repo.AssertExpectations(t)
	})

	t.Run("takes personal details from an editor with the group", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		deps.svc.fields = fixedFieldAccess(protectedfield.AllGroups())
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()
		entity.Profile.LicenseNumber = "DL999999"

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID

		deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
		deps.repo.On("Update", mock.Anything, mock.MatchedBy(func(w *worker.Worker) bool {
			return w.Profile.LicenseNumber == "DL999999"
		})).Return(entity, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		_, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		deps.repo.AssertExpectations(t)
	})

	t.Run("keeps the stored portal user link on the edit path", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()
		entity.UserID = userID

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID
		original.UserID = pulid.MustNew("usr_")

		deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
		deps.repo.On("Update", mock.Anything, mock.MatchedBy(func(w *worker.Worker) bool {
			return w.UserID == original.UserID
		})).Return(original, nil)
		deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		_, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.NoError(t, err)
		assert.Equal(t, original.UserID, entity.UserID)
		deps.repo.AssertExpectations(t)
	})

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

	t.Run("refuses a bare status change on the edit path", func(t *testing.T) {
		t.Parallel()
		deps := setupTest(t)
		userID := pulid.MustNew("usr_")
		entity := newTestWorker()
		entity.Status = domaintypes.StatusInactive

		original := newTestWorker()
		original.ID = entity.ID
		original.OrganizationID = entity.OrganizationID
		original.BusinessUnitID = entity.BusinessUnitID

		deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)

		result, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
		)

		require.Error(t, err)
		assert.Nil(t, result)
		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
		assert.Equal(t, "status", multiErr.Errors[0].Field())
		deps.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("updates worker status to inactive through the employment path", func(t *testing.T) {
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

		result, err := deps.svc.UpdateFromEmployment(
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

func upperTextFormat(_ context.Context, target dataentrycontrol.TextFormattable) {
	target.ApplyTextFormat(dataentrycontrol.TextFormat{
		NameCase:  dataentrycontrol.CaseFormatUpper,
		EmailCase: dataentrycontrol.CaseFormatLower,
		CityCase:  dataentrycontrol.CaseFormatUpper,
	})
}

func TestCreateFormatsTheWorkerBeforeSaving(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	formatter := dataentrycontrolportsmock.NewMockTextFormatter(t)
	deps.svc.formatter = formatter
	entity := newCreateWorker()
	entity.FirstName = " jane "
	entity.LastName = "smith"
	entity.City = "portland"
	entity.Email = "Jane.Smith@Example.COM"
	entity.EmergencyContactName = "john  smith"
	formatter.EXPECT().Format(mock.Anything, entity).Run(upperTextFormat).Return(nil).Once()

	var saved worker.Worker
	deps.repo.EXPECT().
		Create(mock.Anything, entity).
		RunAndReturn(func(_ context.Context, w *worker.Worker) (*worker.Worker, error) {
			saved = *w
			w.ID = pulid.MustNew("wrk_")
			return w, nil
		}).
		Once()
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Create(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(
			pulid.MustNew("usr_"),
			entity.OrganizationID,
			entity.BusinessUnitID,
		),
	)

	require.NoError(t, err)
	assert.Equal(t, "JANE", saved.FirstName)
	assert.Equal(t, "SMITH", saved.LastName)
	assert.Equal(t, "PORTLAND", saved.City)
	assert.Equal(t, "jane.smith@example.com", saved.Email)
	assert.Equal(t, "JOHN SMITH", saved.EmergencyContactName)
}

func TestUpdateFormatsTheWorkerBeforeSaving(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.fields = fixedFieldAccess(protectedfield.AllGroups())
	formatter := dataentrycontrolportsmock.NewMockTextFormatter(t)
	deps.svc.formatter = formatter
	entity := newTestWorker()
	entity.FirstName = "jonathan"
	entity.City = "salem"
	entity.Email = "Jon@Example.com"

	original := newTestWorker()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID

	formatter.EXPECT().Format(mock.Anything, entity).Run(upperTextFormat).Return(nil).Once()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.MatchedBy(func(w *worker.Worker) bool {
		return w.FirstName == "JONATHAN" &&
			w.City == "SALEM" &&
			w.Email == "jon@example.com"
	})).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(
			pulid.MustNew("usr_"),
			entity.OrganizationID,
			entity.BusinessUnitID,
		),
	)

	require.NoError(t, err)
	deps.repo.AssertExpectations(t)
}

func TestCreateStopsWhenFormattingFails(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	formatter := dataentrycontrolportsmock.NewMockTextFormatter(t)
	deps.svc.formatter = formatter
	entity := newCreateWorker()
	formatErr := errors.New("format failed")
	formatter.EXPECT().Format(mock.Anything, entity).Return(formatErr).Once()

	result, err := deps.svc.Create(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(
			pulid.MustNew("usr_"),
			entity.OrganizationID,
			entity.BusinessUnitID,
		),
	)

	require.ErrorIs(t, err, formatErr)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	validator := newTestValidator()

	svc := New(Params{
		Logger:     zap.NewNop(),
		Repo:       repo,
		CacheRepo:  customerportsmock.NewMockWorkerCacheRepository(t),
		Validator:  validator,
		Realtime:   &mocks.NoopRealtimeService{},
		PlanFields: plantest.Keeper(nil),
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}

type capturedHires struct {
	workers []*worker.Worker
	userIDs []pulid.ID
}

func (c *capturedHires) RecordHired(
	_ context.Context,
	wrk *worker.Worker,
	userID pulid.ID,
) error {
	c.workers = append(c.workers, wrk)
	c.userIDs = append(c.userIDs, userID)
	return nil
}

// Creating a worker opens their timeline through the employment recorder,
// which records the hire and assigns the required courses, so the New worker
// page's promise that creation starts training holds.
func TestCreate_HandsTheNewWorkerToTheHireRecorder(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	hires := &capturedHires{}
	deps.svc.SetEmploymentRecorder(hires)
	entity := newCreateWorker()
	userID := pulid.MustNew("usr_")
	created := newTestWorker()

	deps.repo.On("Create", mock.Anything, entity).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Create(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	require.Len(t, hires.workers, 1)
	assert.Same(t, created, hires.workers[0])
	assert.Equal(t, []pulid.ID{userID}, hires.userIDs)
}

func storedRollups(p *worker.WorkerProfile) {
	credentialExpiry := int64(1_900_000_000)
	trainingDue := int64(1_850_000_000)
	lastQuery := int64(1_790_000_000)
	nextQuery := int64(1_820_000_000)
	p.ComplianceStatus = worker.ComplianceStatusNonCompliant
	p.NextCredentialExpiry = &credentialExpiry
	p.TrainingHealth = worker.TrainingHealthOverdue
	p.NextTrainingDue = &trainingDue
	p.SafetyRating = worker.SafetyRatingAtRisk
	p.SafetyScore = 42
	p.DrugAlcoholStatus = worker.DrugAlcoholProhibited
	p.ReturnToDutyStatus = worker.ReturnToDutyFollowUpTesting
	p.LastClearinghouseQueryAt = &lastQuery
	p.NextClearinghouseQueryDue = &nextQuery
}

// The profile form sends the whole record back, and GraphQL does not expose
// every rollup, so a save carries zero values for fields only the server
// writes. They keep what the server stored.
func TestUpdate_KeepsTheProfileFieldsTheServerOwns(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	userID := pulid.MustNew("usr_")
	entity := newTestWorker()
	entity.Profile.ComplianceStatus = worker.ComplianceStatusCompliant
	entity.Profile.SafetyScore = 0
	entity.Profile.MedicalExaminerName = "Dr. Patel"

	original := newTestWorker()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	storedRollups(original.Profile)
	want := *original.Profile

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	var saved *worker.WorkerProfile
	deps.repo.On("Update", mock.Anything, mock.MatchedBy(func(w *worker.Worker) bool {
		saved = w.Profile
		return true
	})).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)
	require.NoError(t, err)

	require.NotNil(t, saved)
	assert.Equal(t, want.ComplianceStatus, saved.ComplianceStatus)
	assert.Equal(t, want.NextCredentialExpiry, saved.NextCredentialExpiry)
	assert.Equal(t, want.TrainingHealth, saved.TrainingHealth)
	assert.Equal(t, want.NextTrainingDue, saved.NextTrainingDue)
	assert.Equal(t, want.SafetyRating, saved.SafetyRating)
	assert.Equal(t, want.SafetyScore, saved.SafetyScore)
	assert.Equal(t, want.DrugAlcoholStatus, saved.DrugAlcoholStatus)
	assert.Equal(t, want.ReturnToDutyStatus, saved.ReturnToDutyStatus)
	assert.Equal(t, want.LastClearinghouseQueryAt, saved.LastClearinghouseQueryAt)
	assert.Equal(t, want.NextClearinghouseQueryDue, saved.NextClearinghouseQueryDue)
	assert.Equal(t, "Dr. Patel", saved.MedicalExaminerName, "fields a person edits still save")
}

func TestCreate_TakesNoServerOwnedProfileFieldsFromTheRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	userID := pulid.MustNew("usr_")
	entity := newCreateWorker()
	storedRollups(entity.Profile)

	deps.repo.On("Create", mock.Anything, mock.MatchedBy(func(w *worker.Worker) bool {
		p := w.Profile
		return p.ComplianceStatus == worker.ComplianceStatusPending &&
			p.NextCredentialExpiry == nil &&
			p.TrainingHealth == worker.TrainingHealthCurrent &&
			p.NextTrainingDue == nil &&
			p.SafetyRating == worker.SafetyRatingExcellent &&
			p.SafetyScore == worker.DefaultSafetyScore &&
			p.DrugAlcoholStatus == worker.DrugAlcoholUnknown &&
			p.ReturnToDutyStatus == worker.ReturnToDutyNotRequired &&
			p.LastClearinghouseQueryAt == nil &&
			p.NextClearinghouseQueryDue == nil
	})).Return(newTestWorker(), nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Create(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	deps.repo.AssertExpectations(t)
}
