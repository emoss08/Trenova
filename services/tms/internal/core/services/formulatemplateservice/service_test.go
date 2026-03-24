package formulatemplateservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testDeps struct {
	repo        *mocks.MockFormulaTemplateRepository
	versionRepo *mocks.MockFormulaTemplateVersionRepository
	auditSvc    *mocks.MockAuditService
	svc         *Service
}

func newFormulaService() *formula.Service {
	registry := schema.NewRegistry()
	registerShipmentSchema(registry)
	res := resolver.NewResolver()
	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})
	eng := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	return formula.NewService(formula.ServiceParams{
		Logger:   zap.NewNop(),
		Registry: registry,
		Engine:   eng,
		Resolver: res,
	})
}

func registerShipmentSchema(registry *schema.Registry) {
	const shipmentSchema = `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "shipment-test-schema",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"customer": {
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"code": { "type": "string" }
				}
			},
			"weight": { "type": "number" },
			"pieces": { "type": "integer" },
			"ratingUnit": { "type": "integer" },
			"freightChargeAmount": { "type": "number" },
			"otherChargeAmount": { "type": "number" },
			"currentTotalCharge": { "type": "number" },
			"totalDistance": { "type": "number" },
			"totalStops": { "type": "integer" },
			"totalWeight": { "type": "number" },
			"totalPieces": { "type": "integer" },
			"totalLinearFeet": { "type": "number" },
			"hasHazmat": { "type": "boolean" },
			"requiresTemperatureControl": { "type": "boolean" },
			"temperatureDifferential": { "type": "number" }
		}
	}`

	if err := registry.Register("shipment", []byte(shipmentSchema)); err != nil {
		panic(err)
	}
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockFormulaTemplateRepository(t)
	versionRepo := mocks.NewMockFormulaTemplateVersionRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := &Service{
		l:              zap.NewNop(),
		repo:           repo,
		versionRepo:    versionRepo,
		formulaService: newFormulaService(),
		auditService:   auditSvc,
	}
	return &testDeps{repo: repo, versionRepo: versionRepo, auditSvc: auditSvc, svc: svc}
}

func newTestTemplate() *formulatemplate.FormulaTemplate {
	return &formulatemplate.FormulaTemplate{
		ID:                   pulid.MustNew("ft_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Test Template",
		Description:          "A test formula template",
		Type:                 formulatemplate.TemplateTypeFreightCharge,
		Expression:           "totalDistance * 2.5",
		Status:               formulatemplate.StatusActive,
		SchemaID:             "shipment",
		CurrentVersionNumber: 1,
		Version:              1,
	}
}

func newTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*formulatemplate.FormulaTemplate]{
		Items: []*formulatemplate.FormulaTemplate{newTestTemplate()},
		Total: 1,
	}
	req := &repositories.ListFormulaTemplatesRequest{
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

	req := &repositories.ListFormulaTemplatesRequest{
		Filter: &pagination.QueryOptions{},
	}
	repoErr := errors.New("database error")

	deps.repo.On("List", mock.Anything, req).Return(nil, repoErr)

	result, err := deps.svc.List(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.repo.AssertExpectations(t)
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity := newTestTemplate()
	req := repositories.GetFormulaTemplateByIDRequest{
		TemplateID: entity.ID,
		TenantInfo: newTenantInfo(),
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)

	result, err := deps.svc.GetByID(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	assert.Equal(t, entity.Name, result.Name)
	deps.repo.AssertExpectations(t)
}

func TestGetByID_NotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := repositories.GetFormulaTemplateByIDRequest{
		TemplateID: pulid.MustNew("ft_"),
		TenantInfo: newTenantInfo(),
	}
	notFoundErr := errors.New("not found")

	deps.repo.On("GetByID", mock.Anything, req).Return(nil, notFoundErr)

	result, err := deps.svc.GetByID(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
	deps.repo.AssertExpectations(t)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := newTestTemplate()
	entity.ID = ""

	created := newTestTemplate()
	created.CurrentVersionNumber = 1

	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, created.ID, result.ID)
	assert.Equal(t, created.Name, result.Name)
	assert.Equal(t, int64(1), result.CurrentVersionNumber)
	deps.repo.AssertExpectations(t)
}

func TestCreate_ValidationFailure(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := &formulatemplate.FormulaTemplate{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "",
		Expression:     "",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_InvalidExpression(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := &formulatemplate.FormulaTemplate{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Bad Expression Template",
		Expression:     "unknownVariable + !!!",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := newTestTemplate()
	entity.ID = ""
	repoErr := errors.New("database error")

	deps.repo.On("Create", mock.Anything, mock.Anything).Return(nil, repoErr)

	result, err := deps.svc.Create(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := newTestTemplate()
	entity.Name = "Updated Name"

	original := newTestTemplate()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.CurrentVersionNumber = 1

	updated := newTestTemplate()
	updated.ID = entity.ID
	updated.Name = "Updated Name"
	updated.CurrentVersionNumber = 2

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(updated, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	assert.Equal(t, "Updated Name", result.Name)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_NotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := newTestTemplate()
	notFoundErr := errors.New("not found")

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_ValidationFailure(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "",
		Expression:     "",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}

	result, err := deps.svc.Update(ctx, entity, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestDuplicate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	req := &repositories.BulkDuplicateFormulaTemplateRequest{
		TenantInfo:  tenant,
		TemplateIDs: []pulid.ID{pulid.MustNew("ft_")},
	}
	duplicated := []*formulatemplate.FormulaTemplate{newTestTemplate()}

	deps.repo.On("BulkDuplicate", mock.Anything, req).Return(duplicated, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Duplicate(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	deps.repo.AssertExpectations(t)
}

func TestDuplicate_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.BulkDuplicateFormulaTemplateRequest{
		TenantInfo:  newTenantInfo(),
		TemplateIDs: []pulid.ID{pulid.MustNew("ft_")},
	}
	repoErr := errors.New("duplicate error")

	deps.repo.On("BulkDuplicate", mock.Anything, req).Return(nil, repoErr)

	result, err := deps.svc.Duplicate(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.repo.AssertExpectations(t)
}

func TestBulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entities := []*formulatemplate.FormulaTemplate{newTestTemplate()}
	req := &repositories.BulkUpdateFormulaTemplateStatusRequest{
		TenantInfo:  newTenantInfo(),
		TemplateIDs: []pulid.ID{entities[0].ID},
		Status:      formulatemplate.StatusInactive,
	}

	deps.repo.On("BulkUpdateStatus", mock.Anything, req).Return(entities, nil)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	deps.repo.AssertExpectations(t)
}

func TestGetUsage_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.GetTemplateUsageRequest{
		TemplateID: pulid.MustNew("ft_"),
		TenantInfo: newTenantInfo(),
	}
	expected := &repositories.GetTemplateUsageResponse{
		InUse: true,
		Usages: []repositories.TemplateUsageCount{
			{Type: "accessorial_charges", Count: 3},
		},
	}

	deps.repo.On("CountUsages", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.GetUsage(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.InUse)
	assert.Len(t, result.Usages, 1)
	assert.Equal(t, 3, result.Usages[0].Count)
	deps.repo.AssertExpectations(t)
}

func TestCreateVersion_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	templateID := pulid.MustNew("ft_")

	template := newTestTemplate()
	template.ID = templateID
	template.OrganizationID = tenant.OrgID
	template.BusinessUnitID = tenant.BuID
	template.CurrentVersionNumber = 2

	req := &repositories.CreateVersionRequest{
		TenantInfo:    tenant,
		TemplateID:    templateID,
		ChangeMessage: "New version",
	}

	prevVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    templateID,
		VersionNumber: 2,
		Name:          template.Name,
		Expression:    template.Expression,
		Type:          template.Type,
		Status:        template.Status,
		SchemaID:      template.SchemaID,
	}

	createdVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    templateID,
		VersionNumber: 3,
		Name:          template.Name,
		Expression:    template.Expression,
		ChangeMessage: "New version",
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(prevVersion, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).Return(createdVersion, nil)

	result, err := deps.svc.CreateVersion(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, int64(3), result.VersionNumber)
	deps.repo.AssertExpectations(t)
	deps.versionRepo.AssertExpectations(t)
}

func TestCreateVersion_TemplateNotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.CreateVersionRequest{
		TenantInfo:    newTenantInfo(),
		TemplateID:    pulid.MustNew("ft_"),
		ChangeMessage: "Test",
	}
	notFoundErr := errors.New("not found")

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	result, err := deps.svc.CreateVersion(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
}

func TestListVersions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.ListVersionsRequest{
		TemplateID: pulid.MustNew("ft_"),
		Filter:     &pagination.QueryOptions{},
	}
	expected := &pagination.ListResult[*formulatemplate.FormulaTemplateVersion]{
		Items: []*formulatemplate.FormulaTemplateVersion{
			{ID: pulid.MustNew("ftv_"), VersionNumber: 1},
		},
		Total: 1,
	}

	deps.versionRepo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.ListVersions(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	deps.versionRepo.AssertExpectations(t)
}

func TestGetVersion_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.GetVersionRequest{
		TenantInfo:    newTenantInfo(),
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 2,
	}
	expected := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		VersionNumber: 2,
		Name:          "Template v2",
	}

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.GetVersion(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, int64(2), result.VersionNumber)
	deps.versionRepo.AssertExpectations(t)
}

func TestRollback_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	templateID := pulid.MustNew("ft_")

	currentTemplate := newTestTemplate()
	currentTemplate.ID = templateID
	currentTemplate.OrganizationID = tenant.OrgID
	currentTemplate.BusinessUnitID = tenant.BuID
	currentTemplate.CurrentVersionNumber = 3
	currentTemplate.Name = "Current Name"

	targetVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    templateID,
		VersionNumber: 1,
		Name:          "Original Name",
		Description:   "Original desc",
		Type:          formulatemplate.TemplateTypeFreightCharge,
		Expression:    "totalWeight * 1.5",
		Status:        formulatemplate.StatusActive,
		SchemaID:      "shipment",
	}

	updatedTemplate := newTestTemplate()
	updatedTemplate.ID = templateID
	updatedTemplate.Name = "Original Name"
	updatedTemplate.Expression = "totalWeight * 1.5"
	updatedTemplate.CurrentVersionNumber = 4

	req := &repositories.RollbackRequest{
		TenantInfo:    tenant,
		TemplateID:    templateID,
		TargetVersion: 1,
		ChangeMessage: "Rolling back",
	}

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(targetVersion, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(currentTemplate, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(updatedTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Rollback(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "Original Name", result.Name)
	assert.Equal(t, int64(4), result.CurrentVersionNumber)
	deps.repo.AssertExpectations(t)
	deps.versionRepo.AssertExpectations(t)
}

func TestRollback_TargetVersionNotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.RollbackRequest{
		TenantInfo:    newTenantInfo(),
		TemplateID:    pulid.MustNew("ft_"),
		TargetVersion: 99,
	}
	notFoundErr := errors.New("version not found")

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(nil, notFoundErr)

	result, err := deps.svc.Rollback(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
}

func TestRollback_DefaultChangeMessage(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	templateID := pulid.MustNew("ft_")

	currentTemplate := newTestTemplate()
	currentTemplate.ID = templateID
	currentTemplate.OrganizationID = tenant.OrgID
	currentTemplate.BusinessUnitID = tenant.BuID
	currentTemplate.CurrentVersionNumber = 2

	targetVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    templateID,
		VersionNumber: 1,
		Name:          "V1 Name",
		Type:          formulatemplate.TemplateTypeFreightCharge,
		Expression:    "totalDistance * 1.0",
		Status:        formulatemplate.StatusActive,
		SchemaID:      "shipment",
	}

	updatedTemplate := newTestTemplate()
	updatedTemplate.ID = templateID
	updatedTemplate.CurrentVersionNumber = 3

	req := &repositories.RollbackRequest{
		TenantInfo:    tenant,
		TemplateID:    templateID,
		TargetVersion: 1,
		ChangeMessage: "",
	}

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(targetVersion, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(currentTemplate, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(updatedTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Rollback(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestFork_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	sourceTemplateID := pulid.MustNew("ft_")

	sourceTemplate := newTestTemplate()
	sourceTemplate.ID = sourceTemplateID
	sourceTemplate.OrganizationID = tenant.OrgID
	sourceTemplate.BusinessUnitID = tenant.BuID
	sourceTemplate.CurrentVersionNumber = 2

	latestVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    sourceTemplateID,
		VersionNumber: 2,
		Name:          sourceTemplate.Name,
		Description:   sourceTemplate.Description,
		Type:          sourceTemplate.Type,
		Expression:    sourceTemplate.Expression,
		Status:        sourceTemplate.Status,
		SchemaID:      sourceTemplate.SchemaID,
	}

	forkedTemplate := newTestTemplate()
	forkedTemplate.Name = "Forked Template"
	forkedTemplate.SourceTemplateID = &sourceTemplateID

	req := &repositories.ForkTemplateRequest{
		TenantInfo:       tenant,
		SourceTemplateID: sourceTemplateID,
		NewName:          "Forked Template",
		ChangeMessage:    "Forked for testing",
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(sourceTemplate, nil)
	deps.versionRepo.On("GetLatestVersion", mock.Anything, sourceTemplateID, tenant).
		Return(latestVersion, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(forkedTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Fork(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "Forked Template", result.Name)
	deps.repo.AssertExpectations(t)
	deps.versionRepo.AssertExpectations(t)
}

func TestFork_SourceNotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.ForkTemplateRequest{
		TenantInfo:       newTenantInfo(),
		SourceTemplateID: pulid.MustNew("ft_"),
		NewName:          "Fork",
	}
	notFoundErr := errors.New("source not found")

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	result, err := deps.svc.Fork(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
}

func TestFork_WithSpecificVersion(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	sourceTemplateID := pulid.MustNew("ft_")
	sourceVersion := int64(1)

	sourceTemplate := newTestTemplate()
	sourceTemplate.ID = sourceTemplateID
	sourceTemplate.OrganizationID = tenant.OrgID
	sourceTemplate.BusinessUnitID = tenant.BuID
	sourceTemplate.CurrentVersionNumber = 3

	requestedVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    sourceTemplateID,
		VersionNumber: 1,
		Name:          "V1 Name",
		Description:   "V1 Desc",
		Type:          formulatemplate.TemplateTypeFreightCharge,
		Expression:    "totalWeight * 1.0",
		Status:        formulatemplate.StatusActive,
		SchemaID:      "shipment",
	}

	forkedTemplate := newTestTemplate()
	forkedTemplate.Name = "Forked From V1"

	req := &repositories.ForkTemplateRequest{
		TenantInfo:       tenant,
		SourceTemplateID: sourceTemplateID,
		SourceVersion:    &sourceVersion,
		NewName:          "Forked From V1",
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(sourceTemplate, nil)
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(requestedVersion, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(forkedTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Fork(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "Forked From V1", result.Name)
	deps.versionRepo.AssertExpectations(t)
}

func TestCompareVersions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	templateID := pulid.MustNew("ft_")

	fromVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    templateID,
		VersionNumber: 1,
		Name:          "Version 1",
		Expression:    "totalDistance * 1.0",
		Type:          formulatemplate.TemplateTypeFreightCharge,
		Status:        formulatemplate.StatusActive,
		SchemaID:      "shipment",
	}

	toVersion := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    templateID,
		VersionNumber: 2,
		Name:          "Version 2",
		Expression:    "totalDistance * 2.0",
		Type:          formulatemplate.TemplateTypeFreightCharge,
		Status:        formulatemplate.StatusActive,
		SchemaID:      "shipment",
	}

	req := &repositories.CompareVersionsRequest{
		TenantInfo:  tenant,
		TemplateID:  templateID,
		FromVersion: 1,
		ToVersion:   2,
	}

	deps.versionRepo.On("GetVersionRange", mock.Anything, mock.Anything).Return(
		[]*formulatemplate.FormulaTemplateVersion{fromVersion, toVersion}, nil,
	)

	result, err := deps.svc.CompareVersions(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.FromVersion)
	assert.Equal(t, int64(2), result.ToVersion)
	assert.GreaterOrEqual(t, result.ChangeCount, 0)
	deps.versionRepo.AssertExpectations(t)
}

func TestCompareVersions_NotEnoughVersions(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.CompareVersionsRequest{
		TenantInfo:  newTenantInfo(),
		TemplateID:  pulid.MustNew("ft_"),
		FromVersion: 1,
		ToVersion:   2,
	}

	deps.versionRepo.On("GetVersionRange", mock.Anything, mock.Anything).Return(
		[]*formulatemplate.FormulaTemplateVersion{
			{ID: pulid.MustNew("ftv_"), VersionNumber: 1},
		}, nil,
	)

	result, err := deps.svc.CompareVersions(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestCompareVersions_VersionRangeError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.CompareVersionsRequest{
		TenantInfo:  newTenantInfo(),
		TemplateID:  pulid.MustNew("ft_"),
		FromVersion: 1,
		ToVersion:   5,
	}
	repoErr := errors.New("version range error")

	deps.versionRepo.On("GetVersionRange", mock.Anything, mock.Anything).Return(nil, repoErr)

	result, err := deps.svc.CompareVersions(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
}

func TestCompareVersions_VersionNotFoundInRange(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.CompareVersionsRequest{
		TenantInfo:  newTenantInfo(),
		TemplateID:  pulid.MustNew("ft_"),
		FromVersion: 1,
		ToVersion:   2,
	}

	deps.versionRepo.On("GetVersionRange", mock.Anything, mock.Anything).Return(
		[]*formulatemplate.FormulaTemplateVersion{
			{ID: pulid.MustNew("ftv_"), VersionNumber: 3},
			{ID: pulid.MustNew("ftv_"), VersionNumber: 4},
		}, nil,
	)

	result, err := deps.svc.CompareVersions(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestGetLineage_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenant := newTenantInfo()
	templateID := pulid.MustNew("ft_")

	template := newTestTemplate()
	template.ID = templateID
	template.OrganizationID = tenant.OrgID
	template.BusinessUnitID = tenant.BuID

	forkedChild := newTestTemplate()
	forkedChild.SourceTemplateID = &templateID

	req := &repositories.GetLineageRequest{
		TenantInfo: tenant,
		TemplateID: templateID,
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetForkedTemplates", mock.Anything, mock.Anything).Return(
		[]*formulatemplate.FormulaTemplate{forkedChild}, nil,
	)

	result, err := deps.svc.GetLineage(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, templateID, result.TemplateID)
	assert.Len(t, result.ForkedTemplates, 1)
	deps.repo.AssertExpectations(t)
	deps.versionRepo.AssertExpectations(t)
}

func TestGetLineage_TemplateNotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.GetLineageRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: pulid.MustNew("ft_"),
	}
	notFoundErr := errors.New("not found")

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, notFoundErr)

	result, err := deps.svc.GetLineage(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
}

func TestUpdateVersionTags_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.UpdateVersionTagsRequest{
		TenantInfo:    newTenantInfo(),
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 1,
		Tags:          []string{"Stable", "Production"},
	}

	expected := &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		VersionNumber: 1,
		Tags: []formulatemplate.VersionTag{
			formulatemplate.VersionTagStable,
			formulatemplate.VersionTagProduction,
		},
	}

	deps.versionRepo.On("UpdateTags", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.UpdateVersionTags(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result.Tags, 2)
	deps.versionRepo.AssertExpectations(t)
}

func TestUpdateVersionTags_InvalidTag(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := &repositories.UpdateVersionTagsRequest{
		TenantInfo:    newTenantInfo(),
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 1,
		Tags:          []string{"InvalidTag"},
	}

	result, err := deps.svc.UpdateVersionTags(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.versionRepo.AssertNotCalled(t, "UpdateTags")
}

func TestTestExpression_ValidExpression(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	req := &TestExpressionRequest{
		Expression: "totalDistance * 2.5",
		SchemaID:   "shipment",
		Variables:  map[string]any{},
	}

	result := deps.svc.TestExpression(req)

	require.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, "Expression is valid", result.Message)
	assert.Empty(t, result.Error)
}

func TestTestExpression_InvalidExpression(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	req := &TestExpressionRequest{
		Expression: "unknownVar + !!!invalid",
		SchemaID:   "shipment",
		Variables:  map[string]any{},
	}

	result := deps.svc.TestExpression(req)

	require.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Error)
}

func TestTestExpression_WithCustomVariables(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	req := &TestExpressionRequest{
		Expression: "customRate * totalDistance",
		SchemaID:   "shipment",
		Variables: map[string]any{
			"customRate": 3.0,
		},
	}

	result := deps.svc.TestExpression(req)

	require.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, "Expression is valid", result.Message)
}

func TestTestExpression_UndefinedVariable(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	req := &TestExpressionRequest{
		Expression: "nonExistentVar * 2",
		SchemaID:   "shipment",
		Variables:  map[string]any{},
	}

	result := deps.svc.TestExpression(req)

	require.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Error)
}

func TestTestExpression_ComplexExpression(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	req := &TestExpressionRequest{
		Expression: "totalDistance * 2.5 + (hasHazmat ? 150.0 : 0.0) + freightChargeAmount",
		SchemaID:   "shipment",
		Variables:  map[string]any{},
	}

	result := deps.svc.TestExpression(req)

	require.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, "Expression is valid", result.Message)
}

func TestTestExpression_NestedShipmentFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	req := &TestExpressionRequest{
		Expression: `customer.name == "Acme" ? 100.0 : 0.0`,
		SchemaID:   "shipment",
		Variables: map[string]any{
			"customer": map[string]any{
				"name": "Acme",
			},
		},
	}

	result := deps.svc.TestExpression(req)

	require.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, "Expression is valid", result.Message)
}

func TestValidateTemplate_ValidEntity(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestTemplate()

	err := deps.svc.validateTemplate(entity)

	assert.NoError(t, err)
}

func TestValidateTemplate_NestedShipmentFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestTemplate()
	entity.Expression = `customer.code == "ACME" ? totalDistance : 0`

	err := deps.svc.validateTemplate(entity)

	assert.NoError(t, err)
}

func TestValidateTemplate_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := &formulatemplate.FormulaTemplate{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "",
		Expression:     "",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}

	err := deps.svc.validateTemplate(entity)

	require.Error(t, err)
}

func TestValidateTemplate_InvalidExpression(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := &formulatemplate.FormulaTemplate{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Valid Name",
		Expression:     "completelyBogusVar + !!!",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}

	err := deps.svc.validateTemplate(entity)

	require.Error(t, err)
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockFormulaTemplateRepository(t)
	versionRepo := mocks.NewMockFormulaTemplateVersionRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	formulaSvc := newFormulaService()

	svc := New(Params{
		Logger:         zap.NewNop(),
		Repo:           repo,
		VersionRepo:    versionRepo,
		FormulaService: formulaSvc,
		AuditService:   auditSvc,
	})

	require.NotNil(t, svc)
}
