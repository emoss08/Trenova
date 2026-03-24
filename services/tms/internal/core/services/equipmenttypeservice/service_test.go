package equipmenttypeservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
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
		validator: validationframework.NewTenantedValidatorBuilder[*equipmenttype.EquipmentType]().
			WithModelName("Equipment Type").
			Build(),
	}
}

type testDeps struct {
	repo        *mocks.MockEquipmentTypeRepository
	audit       *mocks.MockAuditService
	transformer *mocks.MockDataTransformer
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockEquipmentTypeRepository(t)
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

func newTestEntity() *equipmenttype.EquipmentType {
	return &equipmenttype.EquipmentType{
		ID:             pulid.MustNew("et_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "TRLR",
		Description:    "Trailer type",
		Class:          equipmenttype.ClassTrailer,
		Version:        1,
	}
}

func newCreateEntity() *equipmenttype.EquipmentType {
	return &equipmenttype.EquipmentType{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "TRLR",
		Description:    "Trailer type",
		Class:          equipmenttype.ClassTrailer,
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

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(nil)
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

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(nil)
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

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*equipmenttype.EquipmentType]{
		Items: []*equipmenttype.EquipmentType{newTestEntity()},
		Total: 1,
	}
	req := &repositories.ListEquipmentTypesRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	deps.repo.AssertExpectations(t)
}

func TestBulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity1 := newTestEntity()
	entity2 := newTestEntity()

	original1 := newTestEntity()
	original1.ID = entity1.ID
	original1.Status = domaintypes.StatusActive

	original2 := newTestEntity()
	original2.ID = entity2.ID
	original2.Status = domaintypes.StatusActive

	entity1.Status = domaintypes.StatusInactive
	entity2.Status = domaintypes.StatusInactive

	req := &repositories.BulkUpdateEquipmentTypeStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity1.OrganizationID,
			BuID:   entity1.BusinessUnitID,
			UserID: pulid.MustNew("usr_"),
		},
		EquipmentTypeIDs: []pulid.ID{entity1.ID, entity2.ID},
		Status:           domaintypes.StatusInactive,
	}

	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*equipmenttype.EquipmentType{original1, original2}, nil)
	deps.repo.On("BulkUpdateStatus", mock.Anything, req).
		Return([]*equipmenttype.EquipmentType{entity1, entity2}, nil)
	deps.audit.On("LogActions", mock.Anything).Return(nil)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, domaintypes.StatusInactive, result[0].Status)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestBulkUpdateStatus_GetByIDsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	getErr := errors.New("not found")

	req := &repositories.BulkUpdateEquipmentTypeStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		EquipmentTypeIDs: []pulid.ID{pulid.MustNew("et_")},
		Status:           domaintypes.StatusInactive,
	}

	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).Return(nil, getErr)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, getErr, err)
	deps.repo.AssertNotCalled(t, "BulkUpdateStatus")
	deps.repo.AssertExpectations(t)
}

func TestBulkUpdateStatus_BulkUpdateError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	bulkErr := errors.New("bulk update failed")

	req := &repositories.BulkUpdateEquipmentTypeStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		EquipmentTypeIDs: []pulid.ID{pulid.MustNew("et_")},
		Status:           domaintypes.StatusInactive,
	}

	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*equipmenttype.EquipmentType{newTestEntity()}, nil)
	deps.repo.On("BulkUpdateStatus", mock.Anything, req).Return(nil, bulkErr)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, bulkErr, err)
}

func TestCreate_TransformError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")
	transformErr := errors.New("transform failed")

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(transformErr)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, transformErr, err)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(nil)

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
	repoErr := errors.New("create failed")

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(nil, repoErr)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
}

func TestUpdate_TransformError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	transformErr := errors.New("transform failed")

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(transformErr)

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

func TestUpdate_GetByIDError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	notFoundErr := errors.New("not found")

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_RepoError(t *testing.T) {
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

	deps.transformer.On("TransformEquipmentType", mock.Anything, mock.Anything).Return(nil)
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

func TestGet_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()

	req := repositories.GetEquipmentTypeByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
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

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	notFoundErr := errors.New("not found")

	req := repositories.GetEquipmentTypeByIDRequest{
		ID: pulid.MustNew("et_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(nil, notFoundErr)

	result, err := deps.svc.Get(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestList_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	dbErr := errors.New("db error")
	req := &repositories.ListEquipmentTypesRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.List(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestSelectOptions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*equipmenttype.EquipmentType]{
		Items: []*equipmenttype.EquipmentType{newTestEntity()},
		Total: 1,
	}
	req := &repositories.EquipmentTypeSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{Query: "TRLR"},
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
	req := &repositories.EquipmentTypeSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{Query: "test"},
	}

	deps.repo.On("SelectOptions", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockEquipmentTypeRepository(t)
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
