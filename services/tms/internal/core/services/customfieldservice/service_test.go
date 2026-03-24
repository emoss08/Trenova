package customfieldservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newNoOpValidator() *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*customfield.CustomFieldDefinition]().
			WithModelName("Custom Field Definition").
			Build(),
	}
}

func newServiceDefinition() *customfield.CustomFieldDefinition {
	return &customfield.CustomFieldDefinition{
		ID:             pulid.MustNew("cfd_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		ResourceType:   "trailer",
		Name:           "test_field",
		Label:          "Test Field",
		FieldType:      customfield.FieldTypeText,
		IsActive:       true,
		Options:        []customfield.SelectOption{},
	}
}

func newService(
	repo *mockCustomFieldRepository,
	valueRepo *mocks.MockCustomFieldValueRepository,
	auditSvc *mocks.MockAuditService,
	validator *Validator,
) *Service {
	return &Service{
		l:            zap.NewNop(),
		repo:         repo,
		valueRepo:    valueRepo,
		validator:    validator,
		auditService: auditSvc,
	}
}

func TestService_List_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	def := newServiceDefinition()
	expected := &pagination.ListResult[*customfield.CustomFieldDefinition]{
		Items: []*customfield.CustomFieldDefinition{def},
		Total: 1,
	}

	req := &repositories.ListCustomFieldDefinitionsRequest{
		Filter: &pagination.QueryOptions{},
	}

	repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := svc.List(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, def.ID, result.Items[0].ID)
	repo.AssertExpectations(t)
}

func TestService_List_Error(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	req := &repositories.ListCustomFieldDefinitionsRequest{
		Filter: &pagination.QueryOptions{},
	}

	repo.On("List", mock.Anything, req).Return(nil, errors.New("database error"))

	result, err := svc.List(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "database error", err.Error())
	repo.AssertExpectations(t)
}

func TestService_Get_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	def := newServiceDefinition()
	req := repositories.GetCustomFieldDefinitionByIDRequest{
		ID: def.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: def.OrganizationID,
			BuID:  def.BusinessUnitID,
		},
	}

	repo.On("GetByID", mock.Anything, req).Return(def, nil)

	result, err := svc.Get(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, def.ID, result.ID)
	assert.Equal(t, def.Name, result.Name)
	repo.AssertExpectations(t)
}

func TestService_Get_NotFound(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	req := repositories.GetCustomFieldDefinitionByIDRequest{
		ID: pulid.MustNew("cfd_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	repo.On("GetByID", mock.Anything, req).Return(nil, errors.New("not found"))

	result, err := svc.Get(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "not found", err.Error())
	repo.AssertExpectations(t)
}

func TestService_Create_Success(t *testing.T) {
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
	}).Return(5, nil)

	createdEntity := newServiceDefinition()
	createdEntity.OrganizationID = entity.OrganizationID
	createdEntity.BusinessUnitID = entity.BusinessUnitID

	repo.On("Create", mock.Anything, entity).Return(createdEntity, nil)
	auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := svc.Create(t.Context(), entity, userID)

	require.NoError(t, err)
	assert.Equal(t, createdEntity.ID, result.ID)
	repo.AssertExpectations(t)
	auditSvc.AssertExpectations(t)
}

func TestService_Create_QuotaExceeded(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	userID := pulid.MustNew("usr_")

	repo.On("CountByResourceType", mock.Anything, repositories.CountByResourceTypeRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceType: entity.ResourceType,
	}).Return(MaxCustomFieldsPerResourceType, nil)

	result, err := svc.Create(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)

	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))
	assert.True(t, multiErr.HasErrors())
	assert.Equal(t, "resourceType", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
	repo.AssertExpectations(t)
}

func TestService_Create_ValidationFailure(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.ID = ""
	entity.Name = "INVALID NAME!"
	userID := pulid.MustNew("usr_")

	repo.On("CountByResourceType", mock.Anything, repositories.CountByResourceTypeRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceType: entity.ResourceType,
	}).Return(5, nil)

	result, err := svc.Create(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)

	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))
	assert.True(t, multiErr.HasErrors())
	repo.AssertExpectations(t)
}

func TestService_Update_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.Label = "Updated Label"
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

	repo.On("Update", mock.Anything, entity).Return(entity, nil)
	auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.Update(t.Context(), entity, userID)

	require.NoError(t, err)
	assert.Equal(t, "Updated Label", result.Label)
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
	auditSvc.AssertExpectations(t)
}

func TestService_Update_BreakingChange_FieldTypeChange(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.FieldType = customfield.FieldTypeNumber
	userID := pulid.MustNew("usr_")

	original := newServiceDefinition()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.FieldType = customfield.FieldTypeText

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
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(10, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).Return(5, nil)

	result, err := svc.Update(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)

	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))
	assert.True(t, multiErr.HasErrors())
	assert.Equal(t, "fieldType", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrBreakingChange, multiErr.Errors[0].Code)
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestService_Update_BreakingChange_OptionInUseRemoved(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := newService(repo, valueRepo, auditSvc, newNoOpValidator())

	entity := newServiceDefinition()
	entity.FieldType = customfield.FieldTypeSelect
	entity.Options = []customfield.SelectOption{
		{Value: "option_a", Label: "Option A"},
	}
	userID := pulid.MustNew("usr_")

	original := newServiceDefinition()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.FieldType = customfield.FieldTypeSelect
	original.Options = []customfield.SelectOption{
		{Value: "option_a", Label: "Option A"},
		{Value: "option_b", Label: "Option B"},
	}

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
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(3, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).Return(2, nil)

	optionUsageReq := &repositories.GetOptionUsageRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: entity.ID,
	}
	valueRepo.On("GetOptionUsageCounts", mock.Anything, optionUsageReq).Return(map[string]int{
		"option_a": 1,
		"option_b": 2,
	}, nil)

	result, err := svc.Update(t.Context(), entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)

	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))
	assert.True(t, multiErr.HasErrors())

	foundOptionErr := false
	for _, e := range multiErr.Errors {
		if e.Field == "options" && e.Code == errortypes.ErrBreakingChange {
			foundOptionErr = true
			assert.Contains(t, e.Message, "Option B")
		}
	}
	assert.True(t, foundOptionErr)
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestService_Delete_Success(t *testing.T) {
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

	repo.On("Delete", mock.Anything, req).Return(nil)
	auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	err := svc.Delete(t.Context(), req, userID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
	auditSvc.AssertExpectations(t)
}

func TestService_Delete_ValuesExistConflict(t *testing.T) {
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
	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(5, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).Return(3, nil)

	err := svc.Delete(t.Context(), req, userID)

	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Contains(t, err.Error(), "5 values exist")
	assert.Contains(t, err.Error(), "3 resources")
	repo.AssertExpectations(t)
	valueRepo.AssertExpectations(t)
}

func TestService_GetUsageStats_Success(t *testing.T) {
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

	valueRepo.On("CountByDefinition", mock.Anything, valueReq).Return(10, nil)
	valueRepo.On("CountResourcesByDefinition", mock.Anything, valueReq).Return(4, nil)

	def := &customfield.CustomFieldDefinition{
		ID:             defID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		FieldType:      customfield.FieldTypeSelect,
		Options: []customfield.SelectOption{
			{Value: "opt1", Label: "Option 1"},
			{Value: "opt2", Label: "Option 2"},
		},
	}

	repo.On("GetByID", mock.Anything, repositories.GetCustomFieldDefinitionByIDRequest{
		ID:         defID,
		TenantInfo: tenantInfo,
	}).Return(def, nil)

	valueRepo.On("GetOptionUsageCounts", mock.Anything, &repositories.GetOptionUsageRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: defID,
	}).Return(map[string]int{
		"opt1": 6,
		"opt2": 4,
	}, nil)

	stats, err := svc.GetUsageStats(t.Context(), defID, tenantInfo)

	require.NoError(t, err)
	assert.Equal(t, defID, stats.DefinitionID)
	assert.Equal(t, 10, stats.TotalValueCount)
	assert.Equal(t, 4, stats.ResourceCount)
	assert.Len(t, stats.OptionUsage, 2)
	valueRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	valueRepo := mocks.NewMockCustomFieldValueRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	validator := newNoOpValidator()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		ValueRepo:    valueRepo,
		Validator:    validator,
		AuditService: auditSvc,
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}
