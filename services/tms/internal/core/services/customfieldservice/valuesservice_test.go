package customfieldservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestValuesService(
	defRepo *mockCustomFieldRepository,
	valueRepo *mocks.MockCustomFieldValueRepository,
) *ValuesService {
	validator := &ValuesValidator{
		l:    zap.NewNop(),
		repo: defRepo,
	}
	return &ValuesService{
		l:              zap.NewNop(),
		valueRepo:      valueRepo,
		definitionRepo: defRepo,
		validator:      validator,
	}
}

func TestValuesService_GetForResource_Success(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()
	defID1 := pulid.MustNew("cfd_")
	defID2 := pulid.MustNew("cfd_")

	values := []*customfield.CustomFieldValue{
		{DefinitionID: defID1, Value: "hello"},
		{DefinitionID: defID2, Value: float64(42)},
	}

	valueRepo.On("GetByResource", mock.Anything, &repositories.GetCustomFieldValuesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceID:   "res_123",
	}).
		Return(values, nil)

	result, err := svc.GetForResource(t.Context(), tenantInfo, "trailer", "res_123")

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "hello", result[defID1.String()])
	assert.Equal(t, float64(42), result[defID2.String()])
	valueRepo.AssertExpectations(t)
}

func TestValuesService_GetForResource_Empty(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()

	valueRepo.On("GetByResource", mock.Anything, &repositories.GetCustomFieldValuesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceID:   "res_123",
	}).
		Return([]*customfield.CustomFieldValue{}, nil)

	result, err := svc.GetForResource(t.Context(), tenantInfo, "trailer", "res_123")

	require.NoError(t, err)
	assert.Empty(t, result)
	valueRepo.AssertExpectations(t)
}

func TestValuesService_GetForResource_Error(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()

	valueRepo.On("GetByResource", mock.Anything, &repositories.GetCustomFieldValuesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceID:   "res_123",
	}).
		Return(nil, errors.New("db error"))

	result, err := svc.GetForResource(t.Context(), tenantInfo, "trailer", "res_123")

	require.Error(t, err)
	assert.Nil(t, result)
	valueRepo.AssertExpectations(t)
}

func TestValuesService_GetForResources_Success(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()
	defID := pulid.MustNew("cfd_")

	valuesMap := map[string][]*customfield.CustomFieldValue{
		"res_1": {
			{DefinitionID: defID, Value: "val1"},
		},
		"res_2": {
			{DefinitionID: defID, Value: "val2"},
		},
	}

	valueRepo.On("GetByResources", mock.Anything, &repositories.GetCustomFieldValuesByResourcesRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceIDs:  []string{"res_1", "res_2"},
	}).
		Return(valuesMap, nil)

	result, err := svc.GetForResources(
		t.Context(),
		tenantInfo,
		"trailer",
		[]string{"res_1", "res_2"},
	)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "val1", result["res_1"][defID.String()])
	assert.Equal(t, "val2", result["res_2"][defID.String()])
	valueRepo.AssertExpectations(t)
}

func TestValuesService_GetForResources_EmptyIDs(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()

	result, err := svc.GetForResources(t.Context(), tenantInfo, "trailer", []string{})

	require.NoError(t, err)
	assert.Empty(t, result)
	valueRepo.AssertNotCalled(t, "GetByResources")
}

func TestValuesService_GetForResources_Error(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()

	valueRepo.On("GetByResources", mock.Anything, &repositories.GetCustomFieldValuesByResourcesRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceIDs:  []string{"res_1"},
	}).
		Return(nil, errors.New("db error"))

	result, err := svc.GetForResources(
		t.Context(),
		tenantInfo,
		"trailer",
		[]string{"res_1"},
	)

	require.Error(t, err)
	assert.Nil(t, result)
	valueRepo.AssertExpectations(t)
}

func TestValuesService_ValidateAndSave_Success(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()
	defID := pulid.MustNew("cfd_")

	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "text_field",
			Label:     "Text Field",
			FieldType: customfield.FieldTypeText,
			IsActive:  true,
		},
	}

	defRepo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).
		Return(definitions, nil)

	values := map[string]any{
		defID.String(): "valid value",
	}

	valueRepo.On("Upsert", mock.Anything, &repositories.UpsertCustomFieldValuesRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceID:   "res_123",
		Values:       values,
	}).Return(nil)

	multiErr := svc.ValidateAndSave(t.Context(), tenantInfo, "trailer", "res_123", values)

	assert.Nil(t, multiErr)
	defRepo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestValuesService_ValidateAndSave_ValidationFailure(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()
	defID := pulid.MustNew("cfd_")

	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:         defID,
			Name:       "required_field",
			Label:      "Required Field",
			FieldType:  customfield.FieldTypeText,
			IsRequired: true,
			IsActive:   true,
		},
	}

	defRepo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).
		Return(definitions, nil)

	values := map[string]any{}

	multiErr := svc.ValidateAndSave(t.Context(), tenantInfo, "trailer", "res_123", values)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	valueRepo.AssertNotCalled(t, "Upsert")
}

func TestValuesService_ValidateAndSave_UpsertError(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()

	defRepo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).
		Return([]*customfield.CustomFieldDefinition{}, nil)

	values := map[string]any{}

	valueRepo.On("Upsert", mock.Anything, &repositories.UpsertCustomFieldValuesRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceID:   "res_123",
		Values:       values,
	}).Return(errors.New("upsert failed"))

	multiErr := svc.ValidateAndSave(t.Context(), tenantInfo, "trailer", "res_123", values)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	defRepo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestValuesService_Delete_Success(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()

	valueRepo.On("DeleteByResource", mock.Anything, &repositories.GetCustomFieldValuesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceID:   "res_123",
	}).
		Return(nil)

	err := svc.Delete(t.Context(), tenantInfo, "trailer", "res_123")

	require.NoError(t, err)
	valueRepo.AssertExpectations(t)
}

func TestValuesService_Delete_Error(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)

	tenantInfo := newTenantInfo()

	valueRepo.On("DeleteByResource", mock.Anything, &repositories.GetCustomFieldValuesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
		ResourceID:   "res_123",
	}).
		Return(errors.New("delete failed"))

	err := svc.Delete(t.Context(), tenantInfo, "trailer", "res_123")

	require.Error(t, err)
	assert.Equal(t, "delete failed", err.Error())
	valueRepo.AssertExpectations(t)
}

func TestService_GetActiveByResourceType_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	tenantInfo := newTenantInfo()
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        pulid.MustNew("cfd_"),
			Name:      "field1",
			FieldType: customfield.FieldTypeText,
			IsActive:  true,
		},
		{
			ID:        pulid.MustNew("cfd_"),
			Name:      "field2",
			FieldType: customfield.FieldTypeNumber,
			IsActive:  true,
		},
	}

	req := repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}

	repo.On("GetActiveByResourceType", mock.Anything, req).Return(definitions, nil)

	result, err := svc.GetActiveByResourceType(t.Context(), req)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestService_GetActiveByResourceType_Error(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	tenantInfo := newTenantInfo()
	req := repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}

	repo.On("GetActiveByResourceType", mock.Anything, req).Return(nil, errors.New("db error"))

	result, err := svc.GetActiveByResourceType(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetSupportedResourceTypes(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	result := svc.GetSupportedResourceTypes()

	assert.NotEmpty(t, result)
}

func TestService_Create_CountByResourceTypeError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.ID = ""
	userID := pulid.MustNew("usr_")

	repo.On("CountByResourceType", mock.Anything, repositories.CountByResourceTypeRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceType: entity.ResourceType,
	}).Return(0, errors.New("count error"))

	result, err := svc.Create(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "count error", err.Error())
	repo.AssertExpectations(t)
}

func TestService_Create_RepoError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.ID = ""
	userID := pulid.MustNew("usr_")

	repo.On("CountByResourceType", mock.Anything, repositories.CountByResourceTypeRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceType: entity.ResourceType,
	}).Return(0, nil)

	repo.On("Create", mock.Anything, entity).Return(nil, errors.New("create failed"))

	result, err := svc.Create(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "create failed", err.Error())
	repo.AssertExpectations(t)
}

func TestService_Update_ValidationFailure(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.Name = "INVALID NAME!"
	userID := pulid.MustNew("usr_")

	result, err := svc.Update(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestService_Update_GetByIDError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.Label = "Updated"
	userID := pulid.MustNew("usr_")

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	repo.On("GetByID", mock.Anything, repositories.GetCustomFieldDefinitionByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo,
	}).Return(nil, errors.New("not found"))

	result, err := svc.Update(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_Update_UsageStatsError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.Label = "Updated"
	userID := pulid.MustNew("usr_")

	original := newServiceDefinition()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	repo.On("GetByID", mock.Anything, repositories.GetCustomFieldDefinitionByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo,
	}).Return(original, nil)

	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: entity.ID,
	}
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(0, errors.New("count error"))

	result, err := svc.Update(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestService_Update_RepoError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.Label = "Updated"
	userID := pulid.MustNew("usr_")

	original := newServiceDefinition()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	repo.On("GetByID", mock.Anything, repositories.GetCustomFieldDefinitionByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo,
	}).Return(original, nil)

	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: entity.ID,
	}
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).Return(0, nil)

	repo.On("Update", mock.Anything, entity).Return(nil, errors.New("update failed"))

	result, err := svc.Update(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "update failed", err.Error())
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestService_Delete_GetByIDError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	userID := pulid.MustNew("usr_")
	req := repositories.GetCustomFieldDefinitionByIDRequest{
		ID: pulid.MustNew("cfd_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	repo.On("GetByID", mock.Anything, req).Return(nil, errors.New("not found"))

	err := svc.Delete(t.Context(), req, userID)

	require.Error(t, err)
	assert.Equal(t, "not found", err.Error())
	repo.AssertExpectations(t)
}

func TestService_Delete_UsageStatsError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	def := newServiceDefinition()
	userID := pulid.MustNew("usr_")
	req := repositories.GetCustomFieldDefinitionByIDRequest{
		ID: def.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: def.OrganizationID,
			BuID:  def.BusinessUnitID,
		},
	}

	repo.On("GetByID", mock.Anything, req).Return(def, nil)

	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: def.ID,
	}
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(0, errors.New("count error"))

	err := svc.Delete(t.Context(), req, userID)

	require.Error(t, err)
	assert.Equal(t, "count error", err.Error())
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestService_Delete_RepoError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	def := newServiceDefinition()
	userID := pulid.MustNew("usr_")
	req := repositories.GetCustomFieldDefinitionByIDRequest{
		ID: def.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: def.OrganizationID,
			BuID:  def.BusinessUnitID,
		},
	}

	repo.On("GetByID", mock.Anything, req).Return(def, nil)

	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: def.ID,
	}
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).Return(0, nil)

	repo.On("Delete", mock.Anything, req).Return(errors.New("delete failed"))

	err := svc.Delete(t.Context(), req, userID)

	require.Error(t, err)
	assert.Equal(t, "delete failed", err.Error())
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestService_GetUsageStats_CountByDefinitionError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	defID := pulid.MustNew("cfd_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: defID,
	}
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(0, errors.New("count error"))

	stats, err := svc.GetUsageStats(t.Context(), defID, tenantInfo)

	require.Error(t, err)
	assert.Nil(t, stats)
	valueRepo.AssertExpectations(t)
}

func TestService_GetUsageStats_CountResourcesError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	defID := pulid.MustNew("cfd_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: defID,
	}
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(5, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).
		Return(0, errors.New("resource count error"))

	stats, err := svc.GetUsageStats(t.Context(), defID, tenantInfo)

	require.Error(t, err)
	assert.Nil(t, stats)
	valueRepo.AssertExpectations(t)
}

func TestService_GetUsageStats_NoValues(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	defID := pulid.MustNew("cfd_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: defID,
	}
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(0, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).Return(0, nil)

	stats, err := svc.GetUsageStats(t.Context(), defID, tenantInfo)

	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalValueCount)
	assert.Equal(t, 0, stats.ResourceCount)
	assert.Nil(t, stats.OptionUsage)
	valueRepo.AssertExpectations(t)
}

func TestService_DetectBreakingChanges_NoExistingValues(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	original := newServiceDefinition()
	updated := newServiceDefinition()
	updated.FieldType = customfield.FieldTypeNumber

	stats := &customfield.DefinitionUsageStats{
		TotalValueCount: 0,
	}

	result := svc.detectBreakingChanges(original, updated, stats)

	assert.False(t, result.HasBlockingChanges)
	assert.Empty(t, result.Changes)
}

func TestService_DetectBreakingChanges_RequiredFlagWarning(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	original := newServiceDefinition()
	original.IsRequired = false

	updated := newServiceDefinition()
	updated.IsRequired = true

	stats := &customfield.DefinitionUsageStats{
		TotalValueCount: 5,
		ResourceCount:   3,
	}

	result := svc.detectBreakingChanges(original, updated, stats)

	assert.False(t, result.HasBlockingChanges)
	assert.Len(t, result.Changes, 1)
	assert.Equal(t, customfield.BreakingChangeTypeWarning, result.Changes[0].ChangeType)
	assert.Equal(t, "isRequired", result.Changes[0].Field)
}
