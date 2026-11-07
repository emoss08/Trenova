package fleetcodeservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fleetcodeportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
		validator: tenantedvalidation.NewBuilder[*fleetcode.FleetCode]().
			WithModelName("FleetCode").
			Build(),
	}
}

type testDeps struct {
	repo        *fleetcodeportsmock.MockFleetCodeRepository
	audit       *auditportsmock.MockAuditService
	transformer *accessorialchargeportsmock.MockDataTransformer
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := fleetcodeportsmock.NewMockFleetCodeRepository(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	transformer := accessorialchargeportsmock.NewMockDataTransformer(t)
	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    newTestValidator(),
		auditService: auditSvc,
		transformer:  transformer,
	}
	return &testDeps{repo: repo, audit: auditSvc, transformer: transformer, svc: svc}
}

func newTestEntity() *fleetcode.FleetCode {
	return &fleetcode.FleetCode{
		ID:             pulid.MustNew("fc_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		ManagerID:      pulid.MustNew("usr_"),
		Status:         domaintypes.StatusActive,
		Code:           "FC001",
		Description:    "Test fleet code",
		Version:        1,
	}
}

func newCreateEntity() *fleetcode.FleetCode {
	return &fleetcode.FleetCode{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		ManagerID:      pulid.MustNew("usr_"),
		Status:         domaintypes.StatusActive,
		Code:           "FC001",
		Description:    "Test fleet code",
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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(transformErr)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, transformErr, err)
	deps.repo.AssertNotCalled(t, "Create")
	deps.transformer.AssertExpectations(t)
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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(ctx, entity, userID)

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

	expected := &pagination.ListResult[*fleetcode.FleetCode]{
		Items:      []*fleetcode.FleetCode{newTestEntity()},
		Total:      1,
		TotalExact: true,
	}
	req := &fleetcodeports.ListFleetCodesRequest{
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

	req := fleetcodeports.GetFleetCodeByIDRequest{
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

	expected := &pagination.ListResult[*fleetcode.FleetCode]{
		Items:      []*fleetcode.FleetCode{newTestEntity()},
		Total:      1,
		TotalExact: true,
	}
	req := &pagination.SelectQueryRequest{
		Query: "FC001",
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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(ctx, entity, userID)

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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(nil, repoErr)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_TransformError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	transformErr := errors.New("transform failed")

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(transformErr)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, transformErr, err)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_GetByIDError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	notFoundErr := errors.New("not found")

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	result, err := deps.svc.Update(ctx, entity, userID)

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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)
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
	req := &fleetcodeports.ListFleetCodesRequest{
		Filter: &pagination.QueryOptions{},
	}
	dbErr := errors.New("db error")

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

	req := fleetcodeports.GetFleetCodeByIDRequest{
		ID: pulid.MustNew("fc_"),
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
	repo := fleetcodeportsmock.NewMockFleetCodeRepository(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	transformer := accessorialchargeportsmock.NewMockDataTransformer(t)
	logger := zap.NewNop()

	svc := New(Params{
		Logger:       logger,
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: auditSvc,
		Transformer:  transformer,
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()
	require.NotNil(t, v)
	require.NotNil(t, v.validator)
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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("audit error"))

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

	entity := &fleetcode.FleetCode{
		ID:             pulid.MustNew("fc_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Code:           "FC001",
		Version:        1,
	}

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestNewValidator(t *testing.T) {
	t.Parallel()

	v := NewValidator(ValidatorParams{})
	require.NotNil(t, v)
	require.NotNil(t, v.validator)
}

func TestValidator_ValidateCreate(t *testing.T) {
	t.Parallel()

	v := newTestValidator()
	ctx := t.Context()

	entity := &fleetcode.FleetCode{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		ManagerID:      pulid.MustNew("usr_"),
		Status:         domaintypes.StatusActive,
		Code:           "FC001",
		Description:    "Test",
	}

	err := v.ValidateCreate(ctx, entity)
	require.Nil(t, err)
}

func TestValidator_ValidateUpdate(t *testing.T) {
	t.Parallel()

	v := newTestValidator()
	ctx := t.Context()

	entity := &fleetcode.FleetCode{
		ID:             pulid.MustNew("fc_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		ManagerID:      pulid.MustNew("usr_"),
		Status:         domaintypes.StatusActive,
		Code:           "FC001",
		Description:    "Test",
		Version:        1,
	}

	err := v.ValidateUpdate(ctx, entity)
	require.Nil(t, err)
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

	deps.transformer.On("TransformFleetCode", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("audit error"))

	result, err := deps.svc.Update(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	deps.transformer.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func newBulkStatusRequest(ids ...pulid.ID) *fleetcodeports.BulkUpdateFleetCodeStatusRequest {
	return &fleetcodeports.BulkUpdateFleetCodeStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		FleetCodeIDs: ids,
		Status:       domaintypes.StatusInactive,
	}
}

func TestBulkUpdateStatus_UpdatesAndAuditsEveryFleetCode(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	original1, original2 := newTestEntity(), newTestEntity()
	updated1, updated2 := *original1, *original2
	updated1.Status = domaintypes.StatusInactive
	updated2.Status = domaintypes.StatusInactive
	req := newBulkStatusRequest(original1.ID, original2.ID)

	deps.repo.EXPECT().
		GetByIDs(mock.Anything, fleetcodeports.GetFleetCodesByIDsRequest{
			TenantInfo:   req.TenantInfo,
			FleetCodeIDs: req.FleetCodeIDs,
		}).
		Return([]*fleetcode.FleetCode{original1, original2}, nil).
		Once()
	deps.repo.EXPECT().
		BulkUpdateStatus(mock.Anything, req).
		Return([]*fleetcode.FleetCode{&updated1, &updated2}, nil).
		Once()
	deps.audit.EXPECT().LogActions(mock.Anything, mock.Anything).Return(nil).Once()

	result, err := deps.svc.BulkUpdateStatus(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, domaintypes.StatusInactive, result[0].Status)
	assert.Equal(t, domaintypes.StatusInactive, result[1].Status)
}

func TestBulkUpdateStatus_StopsWhenTheOriginalsCannotBeRead(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	readErr := errors.New("read failed")
	req := newBulkStatusRequest(pulid.MustNew("fc_"))

	deps.repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(nil, readErr).Once()

	result, err := deps.svc.BulkUpdateStatus(t.Context(), req)

	require.ErrorIs(t, err, readErr)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "BulkUpdateStatus", mock.Anything, mock.Anything)
}

func TestBulkUpdateStatus_ReturnsTheRepositoryError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	bulkErr := errors.New("bulk update failed")
	original := newTestEntity()
	req := newBulkStatusRequest(original.ID)

	deps.repo.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return([]*fleetcode.FleetCode{original}, nil).
		Once()
	deps.repo.EXPECT().BulkUpdateStatus(mock.Anything, req).Return(nil, bulkErr).Once()

	result, err := deps.svc.BulkUpdateStatus(t.Context(), req)

	require.ErrorIs(t, err, bulkErr)
	assert.Nil(t, result)
}
