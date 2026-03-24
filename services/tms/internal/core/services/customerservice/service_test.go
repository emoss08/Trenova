package customerservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
		validator: validationframework.NewTenantedValidatorBuilder[*customer.Customer]().
			WithModelName("Customer").
			Build(),
	}
}

type testDeps struct {
	repo        *mocks.MockCustomerRepository
	cacheRepo   *mocks.MockCustomerCacheRepository
	audit       *mocks.MockAuditService
	transformer *mocks.MockDataTransformer
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockCustomerRepository(t)
	cacheRepo := mocks.NewMockCustomerCacheRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	transformer := mocks.NewMockDataTransformer(t)
	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		cacheRepo:    cacheRepo,
		validator:    newTestValidator(),
		auditService: auditSvc,
		realtime:     &mocks.NoopRealtimeService{},
		transformer:  transformer,
	}
	return &testDeps{repo: repo, audit: auditSvc, transformer: transformer, cacheRepo: cacheRepo, svc: svc}
}

func newTestEntity() *customer.Customer {
	return &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		StateID:        pulid.MustNew("us_"),
		Status:         domaintypes.StatusActive,
		Code:           "CUST001",
		Name:           "Test Customer",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "62701",
		Version:        1,
	}
}

func newCreateEntity() *customer.Customer {
	return &customer.Customer{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		StateID:        pulid.MustNew("us_"),
		Status:         domaintypes.StatusActive,
		Code:           "CUST001",
		Name:           "Test Customer",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "62701",
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockCustomerRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	transformer := mocks.NewMockDataTransformer(t)
	validator := newTestValidator()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		CacheRepo:    mocks.NewMockCustomerCacheRepository(t),
		Validator:    validator,
		AuditService: auditSvc,
		Realtime:     &mocks.NoopRealtimeService{},
		Transformer:  transformer,
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}

func TestGet_UsesCacheBeforeDatabase(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	req := repositories.GetCustomerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}

	deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(entity, nil).Once()

	result, err := deps.svc.Get(t.Context(), req)

	require.NoError(t, err)
	require.Equal(t, entity.ID, result.ID)
	deps.repo.AssertNotCalled(t, "GetByID")
}

func TestGet_FallsBackToDatabaseOnCacheMiss(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	req := repositories.GetCustomerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}

	deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)

	result, err := deps.svc.Get(t.Context(), req)

	require.NoError(t, err)
	require.Equal(t, entity.ID, result.ID)
	deps.repo.AssertExpectations(t)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")

	created := newTestEntity()
	created.BusinessUnitID = entity.BusinessUnitID
	created.OrganizationID = entity.OrganizationID

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Equal(t, created.ID, result.ID)
	assert.Equal(t, created.Code, result.Code)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestCreate_TransformError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")
	transformErr := errors.New("transform failed")

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).
		Return(transformErr)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, transformErr, err)
	deps.repo.AssertNotCalled(t, "Create")
	deps.transformer.AssertExpectations(t)
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")
	repoErr := errors.New("database error")

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(nil, repoErr)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertNotCalled(t, "LogAction")
}

func TestCreate_AuditLogError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")

	created := newTestEntity()
	created.BusinessUnitID = entity.BusinessUnitID
	created.OrganizationID = entity.OrganizationID

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(errors.New("audit error"))

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Equal(t, created.ID, result.ID)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.Code = "OLDCODE"

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	assert.Equal(t, entity.Code, result.Code)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestUpdate_TransformError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	transformErr := errors.New("transform failed")

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).
		Return(transformErr)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, transformErr, err)
	deps.repo.AssertNotCalled(t, "GetByID")
}

func TestUpdate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		StateID:        pulid.MustNew("us_"),
		Status:         domaintypes.StatusActive,
		Code:           "",
		Name:           "Test Customer",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "62701",
		Version:        1,
	}

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_RepoGetError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	getErr := errors.New("not found")

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, getErr)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, getErr, err)
	deps.repo.AssertNotCalled(t, "Update")
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_RepoUpdateError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	updateErr := errors.New("update failed")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(nil, updateErr)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, updateErr, err)
}

func TestUpdate_AuditLogError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.Code = "OLDCODE"

	deps.transformer.On("TransformCustomer", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("audit error"))

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*customer.Customer]{
		Items: []*customer.Customer{newTestEntity()},
		Total: 1,
	}
	req := &repositories.ListCustomerRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	deps.repo.AssertExpectations(t)
}

func TestList_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	dbErr := errors.New("db error")
	req := &repositories.ListCustomerRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.List(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestGet_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()

	req := repositories.GetCustomerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}

	deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)

	result, err := deps.svc.Get(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	deps.repo.AssertExpectations(t)
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	notFoundErr := errors.New("not found")

	req := repositories.GetCustomerByIDRequest{
		ID: pulid.MustNew("cus_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	deps.cacheRepo.EXPECT().GetByID(mock.Anything, req).Return(nil, repositories.ErrCacheMiss).Once()
	deps.repo.On("GetByID", mock.Anything, req).Return(nil, notFoundErr)

	result, err := deps.svc.Get(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
}

func TestSelectOptions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*customer.Customer]{
		Items: []*customer.Customer{newTestEntity()},
		Total: 1,
	}
	req := &repositories.CustomerSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{
			Query: "CUST001",
		},
	}

	deps.repo.On("SelectOptions", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	deps.repo.AssertExpectations(t)
}

func TestSelectOptions_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	dbErr := errors.New("db error")
	req := &repositories.CustomerSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{Query: "test"},
	}

	deps.repo.On("SelectOptions", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}
