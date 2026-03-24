package tableconfigurationservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	tc "github.com/emoss08/trenova/internal/core/services/tableconfigurationservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Create(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tableconfiguration.TableConfiguration), args.Error(1)
}

func (m *mockRepository) Update(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tableconfiguration.TableConfiguration), args.Error(1)
}

func (m *mockRepository) GetByID(
	ctx context.Context,
	req repositories.GetTableConfigurationByIDRequest,
) (*tableconfiguration.TableConfiguration, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tableconfiguration.TableConfiguration), args.Error(1)
}

func (m *mockRepository) List(
	ctx context.Context,
	req *repositories.ListTableConfigurationsRequest,
) (*pagination.ListResult[*tableconfiguration.TableConfiguration], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.ListResult[*tableconfiguration.TableConfiguration]), args.Error(
		1,
	)
}

func (m *mockRepository) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	args := m.Called(ctx, id, tenantInfo)
	return args.Error(0)
}

func (m *mockRepository) GetDefaultForResource(
	ctx context.Context,
	req repositories.GetDefaultTableConfigurationRequest,
) (*tableconfiguration.TableConfiguration, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tableconfiguration.TableConfiguration), args.Error(1)
}

func (m *mockRepository) ClearDefaultForResource(
	ctx context.Context,
	userID pulid.ID,
	resource string,
	tenantInfo pagination.TenantInfo,
) error {
	args := m.Called(ctx, userID, resource, tenantInfo)
	return args.Error(0)
}

func setupTestService(repo repositories.TableConfigurationRepository) *tc.Service {
	return tc.New(tc.Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})
}

func createValidEntity() *tableconfiguration.TableConfiguration {
	return &tableconfiguration.TableConfiguration{
		ID:             pulid.MustNew("tc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Name:           "Test Config",
		Description:    "Test description",
		Resource:       "shipments",
		TableConfig: &tableconfiguration.TableConfig{
			FilterGroups:     []domaintypes.FilterGroup{},
			FieldFilters:     []domaintypes.FieldFilter{},
			JoinOperator:     "and",
			Sort:             []domaintypes.SortField{},
			PageSize:         10,
			ColumnVisibility: map[string]bool{},
			ColumnOrder:      []string{},
		},
		Visibility: tableconfiguration.VisibilityPrivate,
		IsDefault:  false,
	}
}

func TestCreate_Success(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	repo.On("Create", ctx, entity).Return(entity, nil)

	result, err := service.Create(ctx, entity)

	require.NoError(t, err)
	assert.Equal(t, entity.Name, result.Name)
	repo.AssertExpectations(t)
}

func TestCreate_ValidationError(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.Name = ""

	result, err := service.Create(ctx, entity)

	assert.Nil(t, result)
	assert.Error(t, err)

	var multiErr *errortypes.MultiError
	assert.True(t, errors.As(err, &multiErr))
	assert.True(t, multiErr.HasErrors())
}

func TestCreate_WithIsDefault_ClearsExisting(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.IsDefault = true

	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}

	repo.On("ClearDefaultForResource", ctx, entity.UserID, entity.Resource, tenantInfo).Return(nil)
	repo.On("Create", ctx, entity).Return(entity, nil)

	result, err := service.Create(ctx, entity)

	require.NoError(t, err)
	assert.True(t, result.IsDefault)
	repo.AssertExpectations(t)
}

func TestCreate_RepositoryError(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	expectedErr := errors.New("database error")
	repo.On("Create", ctx, entity).Return(nil, expectedErr)

	result, err := service.Create(ctx, entity)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	repo.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.Name = "Updated Config"
	repo.On("Update", ctx, entity).Return(entity, nil)

	result, err := service.Update(ctx, entity)

	require.NoError(t, err)
	assert.Equal(t, "Updated Config", result.Name)
	repo.AssertExpectations(t)
}

func TestUpdate_WithIsDefault_ClearsExisting(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.IsDefault = true

	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}

	repo.On("ClearDefaultForResource", ctx, entity.UserID, entity.Resource, tenantInfo).Return(nil)
	repo.On("Update", ctx, entity).Return(entity, nil)

	result, err := service.Update(ctx, entity)

	require.NoError(t, err)
	assert.True(t, result.IsDefault)
	repo.AssertExpectations(t)
}

func TestGetByID_Success(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	req := repositories.GetTableConfigurationByIDRequest{
		ConfigurationID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: entity.UserID,
		},
	}

	repo.On("GetByID", ctx, req).Return(entity, nil)

	result, err := service.GetByID(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	repo.AssertExpectations(t)
}

func TestList_Success(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entities := []*tableconfiguration.TableConfiguration{createValidEntity()}
	listResult := &pagination.ListResult[*tableconfiguration.TableConfiguration]{
		Items: entities,
		Total: 1,
	}

	req := &repositories.ListTableConfigurationsRequest{
		Resource: "shipments",
	}

	repo.On("List", ctx, req).Return(listResult, nil)

	result, err := service.List(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, len(result.Items))
	repo.AssertExpectations(t)
}

func TestDelete_Success(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}

	repo.On("Delete", ctx, entity.ID, tenantInfo).Return(nil)

	err := service.Delete(ctx, entity.ID, tenantInfo)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSetDefault_Success(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.IsDefault = false

	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}

	req := repositories.GetTableConfigurationByIDRequest{
		ConfigurationID: entity.ID,
		TenantInfo:      tenantInfo,
	}

	updatedEntity := *entity
	updatedEntity.IsDefault = true

	repo.On("GetByID", ctx, req).Return(entity, nil)
	repo.On("ClearDefaultForResource", ctx, entity.UserID, entity.Resource, tenantInfo).Return(nil)
	repo.On("Update", ctx, mock.AnythingOfType("*tableconfiguration.TableConfiguration")).
		Return(&updatedEntity, nil)

	result, err := service.SetDefault(ctx, entity.ID, tenantInfo)

	require.NoError(t, err)
	assert.True(t, result.IsDefault)
	repo.AssertExpectations(t)
}

func TestSetDefault_GetByIDError(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}

	req := repositories.GetTableConfigurationByIDRequest{
		ConfigurationID: entity.ID,
		TenantInfo:      tenantInfo,
	}

	expectedErr := errors.New("not found")
	repo.On("GetByID", ctx, req).Return(nil, expectedErr)

	result, err := service.SetDefault(ctx, entity.ID, tenantInfo)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	repo.AssertExpectations(t)
}

func TestGetDefaultForResource_Success(t *testing.T) {
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.IsDefault = true

	req := repositories.GetDefaultTableConfigurationRequest{
		Resource: "shipments",
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: entity.UserID,
		},
	}

	repo.On("GetDefaultForResource", ctx, req).Return(entity, nil)

	result, err := service.GetDefaultForResource(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.IsDefault)
	assert.Equal(t, "shipments", result.Resource)
	repo.AssertExpectations(t)
}

func TestDelete_Error(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}
	deleteErr := errors.New("delete failed")

	repo.On("Delete", ctx, entity.ID, tenantInfo).Return(deleteErr)

	err := service.Delete(ctx, entity.ID, tenantInfo)

	require.Error(t, err)
	assert.Equal(t, deleteErr, err)
	repo.AssertExpectations(t)
}

func TestUpdate_ValidationError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.Name = ""

	result, err := service.Update(ctx, entity)

	assert.Nil(t, result)
	assert.Error(t, err)

	var multiErr *errortypes.MultiError
	assert.True(t, errors.As(err, &multiErr))
	assert.True(t, multiErr.HasErrors())
	repo.AssertNotCalled(t, "Update")
}

func TestUpdate_RepositoryError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.Name = "Updated Config"
	expectedErr := errors.New("database error")

	repo.On("Update", ctx, entity).Return(nil, expectedErr)

	result, err := service.Update(ctx, entity)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	repo.AssertExpectations(t)
}

func TestCreate_ClearDefaultError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.IsDefault = true

	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}
	clearErr := errors.New("clear default failed")

	repo.On("ClearDefaultForResource", ctx, entity.UserID, entity.Resource, tenantInfo).
		Return(clearErr)

	result, err := service.Create(ctx, entity)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, clearErr, err)
	repo.AssertNotCalled(t, "Create")
	repo.AssertExpectations(t)
}

func TestUpdate_ClearDefaultError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	entity.IsDefault = true

	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}
	clearErr := errors.New("clear default failed")

	repo.On("ClearDefaultForResource", ctx, entity.UserID, entity.Resource, tenantInfo).
		Return(clearErr)

	result, err := service.Update(ctx, entity)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, clearErr, err)
	repo.AssertNotCalled(t, "Update")
	repo.AssertExpectations(t)
}

func TestSetDefault_ClearDefaultError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}

	req := repositories.GetTableConfigurationByIDRequest{
		ConfigurationID: entity.ID,
		TenantInfo:      tenantInfo,
	}
	clearErr := errors.New("clear default failed")

	repo.On("GetByID", ctx, req).Return(entity, nil)
	repo.On("ClearDefaultForResource", ctx, entity.UserID, entity.Resource, tenantInfo).
		Return(clearErr)

	result, err := service.SetDefault(ctx, entity.ID, tenantInfo)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, clearErr, err)
	repo.AssertNotCalled(t, "Update")
	repo.AssertExpectations(t)
}

func TestSetDefault_UpdateError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	tenantInfo := pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	}

	req := repositories.GetTableConfigurationByIDRequest{
		ConfigurationID: entity.ID,
		TenantInfo:      tenantInfo,
	}
	updateErr := errors.New("update failed")

	repo.On("GetByID", ctx, req).Return(entity, nil)
	repo.On("ClearDefaultForResource", ctx, entity.UserID, entity.Resource, tenantInfo).Return(nil)
	repo.On("Update", ctx, mock.AnythingOfType("*tableconfiguration.TableConfiguration")).
		Return(nil, updateErr)

	result, err := service.SetDefault(ctx, entity.ID, tenantInfo)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, updateErr, err)
	repo.AssertExpectations(t)
}

func TestGetByID_Error(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	entity := createValidEntity()
	req := repositories.GetTableConfigurationByIDRequest{
		ConfigurationID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: entity.UserID,
		},
	}
	notFoundErr := errors.New("not found")

	repo.On("GetByID", ctx, req).Return(nil, notFoundErr)

	result, err := service.GetByID(ctx, req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, notFoundErr, err)
	repo.AssertExpectations(t)
}

func TestList_Error(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	req := &repositories.ListTableConfigurationsRequest{
		Resource: "shipments",
	}
	dbErr := errors.New("db error")

	repo.On("List", ctx, req).Return(nil, dbErr)

	result, err := service.List(ctx, req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, dbErr, err)
	repo.AssertExpectations(t)
}

func TestGetDefaultForResource_Error(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)
	service := setupTestService(repo)
	ctx := t.Context()

	req := repositories.GetDefaultTableConfigurationRequest{
		Resource: "shipments",
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
	notFoundErr := errors.New("not found")

	repo.On("GetDefaultForResource", ctx, req).Return(nil, notFoundErr)

	result, err := service.GetDefaultForResource(ctx, req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, notFoundErr, err)
	repo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()
	repo := new(mockRepository)

	svc := tc.New(tc.Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})

	require.NotNil(t, svc)
}
