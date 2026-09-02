package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newImportPayload(name string) *ImportTemplatePayload {
	return &ImportTemplatePayload{
		Name:        name,
		Description: "Imported template",
		Type:        formulatemplate.TemplateTypeFreightCharge,
		Expression:  "totalDistance * 2.5",
		SchemaID:    "shipment",
	}
}

func importErrorFields(err error) []string {
	multiErr, ok := err.(*errortypes.MultiError)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	return fields
}

func TestImport_CreatesDraftWithSnapshot(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)
	deps.repo.On("Create", mock.Anything, mock.MatchedBy(
		func(entity *formulatemplate.FormulaTemplate) bool {
			return entity.Status == formulatemplate.StatusDraft &&
				entity.CurrentVersionNumber == 1
		},
	)).Return(newTestTemplate(), nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.1",
		Template:      newImportPayload("Imported Per Mile"),
		TenantInfo:    newTenantInfo(),
	})

	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	assert.Empty(t, result.Renamed)
	deps.repo.AssertExpectations(t)
	deps.versionRepo.AssertExpectations(t)
}

func TestImport_RejectsUnsupportedExportVersion(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "9.9",
		Template:      newImportPayload("Anything"),
		TenantInfo:    newTenantInfo(),
	})

	require.Error(t, err)
	assert.Contains(t, importErrorFields(err), "exportVersion")
}

func TestImport_RejectsNameConflictByDefault(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	existing := newTestTemplate()
	existing.Name = "Imported Per Mile"

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{existing}, nil)

	_, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.0",
		Template:      newImportPayload("Imported Per Mile"),
		TenantInfo:    newTenantInfo(),
	})

	require.Error(t, err)
	assert.Contains(t, importErrorFields(err), "templates[0].name")
	deps.repo.AssertNotCalled(t, "Create")
}

func TestImport_RenamesOnConflictWhenAsked(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	existing := newTestTemplate()
	existing.Name = "Imported Per Mile"

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{existing}, nil)
	deps.repo.On("Create", mock.Anything, mock.MatchedBy(
		func(entity *formulatemplate.FormulaTemplate) bool {
			return entity.Name == "Imported Per Mile (Imported)"
		},
	)).Return(newTestTemplate(), nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.1",
		Template:      newImportPayload("Imported Per Mile"),
		OnConflict:    ImportConflictRename,
		TenantInfo:    newTenantInfo(),
	})

	require.NoError(t, err)
	assert.Equal(t, "Imported Per Mile (Imported)", result.Renamed["Imported Per Mile"])
}

func TestImport_ReportsInvalidExpressionWithIndexedField(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)

	payload := newImportPayload("Broken Import")
	payload.Expression = "totalDistance +* 2"

	_, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.0",
		Templates:     []*ImportTemplatePayload{payload},
		TenantInfo:    newTenantInfo(),
	})

	require.Error(t, err)
	assert.Contains(t, importErrorFields(err), "templates[0].expression")
	deps.repo.AssertNotCalled(t, "Create")
}

func TestImport_CreatesTestScenariosWithTemplate(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	createdTemplate := newTestTemplate()
	payload := newImportPayload("Imported With Scenarios")
	payload.TestCases = []*TestCaseInput{
		{
			Name:           "500 mile haul",
			Variables:      map[string]any{"totalDistance": 500},
			ExpectedAmount: decimal.NewFromInt(1250),
			Tolerance:      decimal.NewFromFloat(0.05),
		},
		{
			Name:           "Zero-distance move",
			ExpectedAmount: decimal.Zero,
		},
	}

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(createdTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	tenant := newTenantInfo()
	result, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.2",
		Template:      payload,
		TenantInfo:    tenant,
	})

	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	require.Len(t, deps.testCaseRepo.created, 2)

	first := deps.testCaseRepo.created[0]
	assert.Equal(t, createdTemplate.ID, first.TemplateID)
	assert.Equal(t, tenant.OrgID, first.OrganizationID)
	assert.Equal(t, tenant.UserID, first.CreatedByID)
	assert.Equal(t, "500 mile haul", first.Name)
	assert.True(t, decimal.NewFromFloat(0.05).Equal(first.Tolerance))

	second := deps.testCaseRepo.created[1]
	assert.Equal(t, "Zero-distance move", second.Name)
	assert.True(t, defaultTestCaseTolerance.Equal(second.Tolerance))
	assert.NotNil(t, second.Variables)
}

func TestImport_RejectsInvalidTestScenarioWithIndexedField(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	payload := newImportPayload("Broken Scenarios")
	payload.TestCases = []*TestCaseInput{
		{ExpectedAmount: decimal.NewFromInt(100)},
		{Name: "Negative expectation", ExpectedAmount: decimal.NewFromInt(-5)},
	}

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)

	_, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.2",
		Template:      payload,
		TenantInfo:    newTenantInfo(),
	})

	require.Error(t, err)
	fields := importErrorFields(err)
	assert.Contains(t, fields, "templates[0].testCases[0].name")
	assert.Contains(t, fields, "templates[0].testCases[1].expectedAmount")
	deps.repo.AssertNotCalled(t, "Create")
	assert.Empty(t, deps.testCaseRepo.created)
}

func TestNormalizeImportRequest_RequiresTemplates(t *testing.T) {
	t.Parallel()

	_, err := normalizeImportRequest(&ImportTemplatesRequest{ExportVersion: "1.0"})

	require.Error(t, err)
	assert.Contains(t, importErrorFields(err), "templates")
}

func TestImport_IgnoresUnresolvableSourceTemplate(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sourceID := newTestTemplate().ID
	payload := newImportPayload("Imported With Lineage")
	payload.SourceTemplateID = &sourceID

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)
	deps.repo.On("GetByIDs", mock.Anything, mock.MatchedBy(
		func(req repositories.GetFormulaTemplatesByIDsRequest) bool {
			return len(req.TemplateIDs) == 1 && req.TemplateIDs[0] == sourceID
		},
	)).Return([]*formulatemplate.FormulaTemplate{}, nil)
	deps.repo.On("Create", mock.Anything, mock.MatchedBy(
		func(entity *formulatemplate.FormulaTemplate) bool {
			return entity.SourceTemplateID == nil &&
				entity.Metadata["importedSource"] == sourceID.String()
		},
	)).Return(newTestTemplate(), nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.1",
		Template:      payload,
		TenantInfo:    newTenantInfo(),
	})

	require.NoError(t, err)
	deps.repo.AssertExpectations(t)
}
