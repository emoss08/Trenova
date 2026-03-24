package accessorialchargeservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*accessorialcharge.AccessorialCharge]().
			WithModelName("AccessorialCharge").
			Build(),
	}
}

type testDeps struct {
	repo        *mocks.MockAccessorialChargeRepository
	audit       *mocks.MockAuditService
	transformer *mocks.MockDataTransformer
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockAccessorialChargeRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	transformer := mocks.NewMockDataTransformer(t)
	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    newTestValidator(),
		auditService: auditSvc,
		transformer:  transformer,
	}
	return &testDeps{repo: repo, audit: auditSvc, transformer: transformer, svc: svc}
}

func newTestEntity() *accessorialcharge.AccessorialCharge {
	return &accessorialcharge.AccessorialCharge{
		ID:             pulid.MustNew("acc_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "FUEL",
		Description:    "Fuel surcharge",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromFloat(75.00),
		Version:        1,
	}
}

func newCreateEntity() *accessorialcharge.AccessorialCharge {
	return &accessorialcharge.AccessorialCharge{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "FUEL",
		Description:    "Fuel surcharge",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromFloat(75.00),
	}
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

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(ctx, entity, userID)

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

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).
		Return(transformErr)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, transformErr, err)
	deps.repo.AssertNotCalled(t, "Create")
	deps.transformer.AssertExpectations(t)
}

func TestCreate_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")
	repoErr := errors.New("database error")

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(nil, repoErr)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertNotCalled(t, "LogAction")
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
	original.Code = "OLD"

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	assert.Equal(t, entity.Code, result.Code)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestUpdate_RepoGetError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	getErr := errors.New("not found")

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, getErr)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, getErr, err)
	deps.repo.AssertNotCalled(t, "Update")
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*accessorialcharge.AccessorialCharge]{
		Items: []*accessorialcharge.AccessorialCharge{newTestEntity()},
		Total: 1,
	}
	req := &repositories.ListAccessorialChargeRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	deps.repo.AssertExpectations(t)
}

func TestGet_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()

	req := repositories.GetAccessorialChargeByIDRequest{
		ID: entity.ID,
		TenantInfo: &pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)

	result, err := deps.svc.Get(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	deps.repo.AssertExpectations(t)
}

func TestSelectOptions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*accessorialcharge.AccessorialCharge]{
		Items: []*accessorialcharge.AccessorialCharge{newTestEntity()},
		Total: 1,
	}
	req := &pagination.SelectQueryRequest{
		Query: "FUEL",
	}

	deps.repo.On("SelectOptions", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	deps.repo.AssertExpectations(t)
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestUpdate_TransformError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	transformErr := errors.New("transform failed")

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).
		Return(transformErr)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, transformErr, err)
	deps.repo.AssertNotCalled(t, "GetByID")
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

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(nil, updateErr)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, updateErr, err)
}

func TestList_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	dbErr := errors.New("db error")
	req := &repositories.ListAccessorialChargeRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.List(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	notFoundErr := errors.New("not found")

	req := repositories.GetAccessorialChargeByIDRequest{
		ID: pulid.MustNew("acc_"),
		TenantInfo: &pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(nil, notFoundErr)

	result, err := deps.svc.Get(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
}

func TestSelectOptions_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	dbErr := errors.New("db error")
	req := &pagination.SelectQueryRequest{Query: "test"}

	deps.repo.On("SelectOptions", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccessorialChargeRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	transformer := mocks.NewMockDataTransformer(t)
	validator := newTestValidator()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    validator,
		AuditService: auditSvc,
		Transformer:  transformer,
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
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

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(errors.New("audit error"))

	result, err := deps.svc.Create(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, created.ID, result.ID)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestUpdate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := &accessorialcharge.AccessorialCharge{
		ID:             pulid.MustNew("acc_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "",
		Description:    "Fuel surcharge",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromFloat(75.00),
		Version:        1,
	}

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "Update")
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
	original.Code = "OLD"

	deps.transformer.On("TransformAccessorialCharge", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("audit error"))

	result, err := deps.svc.Update(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}
