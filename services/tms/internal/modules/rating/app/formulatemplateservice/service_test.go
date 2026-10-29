package formulatemplateservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatetest"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreate_StoresTheSourceWithWhatTheCheckerDerived(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	template := formulatemplatetest.NewTemplate()
	template.Status = formulatemplate.StatusActive
	approver := pulid.MustNew("usr_")
	template.ApprovedByID = &approver
	created, err := deps.svc.Create(
		t.Context(),
		template,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)

	assert.Equal(t, formulatemplate.StatusDraft, created.Status, "a new template starts as a draft")
	assert.Nil(t, created.ApprovedByID)
	assert.EqualValues(t, 1, created.TestCount)
	assert.Empty(t, created.Imports)
	require.Len(t, created.Inputs, 2)
	assert.Equal(t, "rate", created.Inputs[0].Name)
	assert.Equal(t, "Money/mi", created.Inputs[0].Type)
	assert.True(t, created.Inputs[0].Required)
	assert.Equal(t, "What the contract pays per loaded mile.", created.Inputs[0].Doc)
	assert.Equal(t, "minimumCharge", created.Inputs[1].Name)
	assert.False(t, created.Inputs[1].Required)
	assert.NotEmpty(t, created.Inputs[1].Default)

	require.NotNil(t, snapshot)
	assert.EqualValues(t, 1, snapshot.VersionNumber)
	assert.Equal(t, formulatemplatetest.PerMileSource, snapshot.Source)
	assert.Len(t, snapshot.Inputs, 2)
}

func TestCreate_RefusesSourceThatDoesNotCheckAtItsRange(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := formulatemplatetest.NewTemplate()
	template.Source = formulatemplatetest.BrokenSource
	_, err := deps.svc.Create(t.Context(), template, formulatemplatetest.NewTenantInfo().UserID)
	require.Error(t, err)

	fields := formulatemplatetest.ValidationFields(t, err)
	refusal, ok := fields[FieldSource]
	require.True(t, ok, "the refusal names the source: %v", err)
	require.NotNil(t, refusal.Range, "the editor can underline it")
	assert.Greater(t, refusal.Range.To, refusal.Range.From)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestUpdate_ASourceChangeRevertsApprovalAndSnapshotsAVersion(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	approver := pulid.MustNew("usr_")
	approvedAt := timeutils.NowUnix() - 3600
	original := formulatemplatetest.NewTemplate()
	original.ApprovedByID = &approver
	original.ApprovedAt = &approvedAt
	original.Inputs = formulatemplatetest.CheckedInputs(t, formulatemplatetest.PerMileSource)
	edited := *original
	edited.Source = formulatemplatetest.PerMileRetypedSource

	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.versionRepo.On("ClearScheduled", mock.Anything, mock.Anything).Return(int64(0), nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.Update(
		t.Context(),
		&edited,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)

	assert.Equal(
		t,
		formulatemplate.StatusDraft,
		updated.Status,
		"a changed source needs a new review",
	)
	assert.Nil(t, updated.ApprovedByID)
	require.Len(t, updated.Inputs, 2)
	assert.Equal(t, "Money/cwt", updated.Inputs[0].Type, "the declarations follow the source")
	require.NotNil(t, snapshot)
	assert.Equal(t, formulatemplatetest.PerMileRetypedSource, snapshot.Source)
}

func TestUpdate_AContentSaveKeepsTheServerHeldFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	approver := pulid.MustNew("usr_")
	approvedAt := timeutils.NowUnix() - 3600
	forkedFrom := pulid.MustNew("ft_")
	forkedAt := int64(3)
	original := formulatemplatetest.NewTemplate()
	original.ApprovedByID = &approver
	original.ApprovedAt = &approvedAt
	original.SourceTemplateID = &forkedFrom
	original.SourceVersionNumber = &forkedAt
	original.Metadata = map[string]any{"importedSource": forkedFrom.String()}
	payload := &formulatemplate.FormulaTemplate{
		ID:             original.ID,
		OrganizationID: original.OrganizationID,
		BusinessUnitID: original.BusinessUnitID,
		Name:           original.Name,
		Description:    "Linehaul per loaded mile, minimum per load",
		Source:         original.Source,
		Version:        original.Version,
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.Update(
		t.Context(),
		payload,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)

	assert.Equal(t, payload.Description, updated.Description)
	assert.Equal(t, formulatemplate.StatusActive, updated.Status, "a save never moves the status")
	assert.Equal(t, &approver, updated.ApprovedByID)
	assert.Equal(t, &forkedFrom, updated.SourceTemplateID, "the fork lineage survives a save")
	assert.Equal(t, &forkedAt, updated.SourceVersionNumber)
	assert.Equal(t, original.Metadata, updated.Metadata)
	assert.EqualValues(
		t,
		1,
		updated.CurrentVersionNumber,
		"a description edit is not a new version",
	)
	deps.versionRepo.AssertNotCalled(t, "Create")
}

func TestCreate_TakesTheNameFromTheSourceHeader(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	deps.repo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	template := formulatemplatetest.NewTemplate()
	template.Name = "Something the header does not say"
	created, err := deps.svc.Create(
		t.Context(),
		template,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)

	assert.Equal(t, "Per Mile", created.Name)
}

func TestCreate_RefusesAHeaderNameLongerThanANameMayBe(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := formulatemplatetest.NewTemplate()
	template.Source = strings.Replace(
		formulatemplatetest.PerMileSource,
		`tariff "Per Mile"`,
		`tariff "`+strings.Repeat("a", 101)+`"`,
		1,
	)
	_, err := deps.svc.Create(t.Context(), template, formulatemplatetest.NewTenantInfo().UserID)

	assert.Contains(t, formulatemplatetest.ValidationFields(t, err), "name")
	deps.repo.AssertNotCalled(t, "Create")
}

func TestUpdate_RenamingTheHeaderOfAnActiveTemplateKeepsItActive(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	original := formulatemplatetest.NewTemplate()
	payload := &formulatemplate.FormulaTemplate{
		ID:             original.ID,
		OrganizationID: original.OrganizationID,
		BusinessUnitID: original.BusinessUnitID,
		Name:           original.Name,
		Description:    original.Description,
		Source: strings.Replace(
			original.Source,
			`tariff "Per Mile"`,
			`tariff "Per Mile, regional"`,
			1,
		),
		Version: original.Version,
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.Update(
		t.Context(),
		payload,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)

	assert.Equal(t, "Per Mile, regional", updated.Name)
	assert.Equal(t, formulatemplate.StatusActive, updated.Status, "a rename bills nothing new")
	assert.EqualValues(t, 1, updated.CurrentVersionNumber)
	deps.versionRepo.AssertNotCalled(t, "Create")
}

func TestUpdate_RenamingAndChangingWhatItBillsReturnsItToDraft(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	original := formulatemplatetest.NewTemplate()
	payload := &formulatemplate.FormulaTemplate{
		ID:             original.ID,
		OrganizationID: original.OrganizationID,
		BusinessUnitID: original.BusinessUnitID,
		Description:    original.Description,
		Source: strings.Replace(
			original.Source,
			`tariff "Per Mile"`,
			"tariff \"Per Mile, regional\"\n\n// billed per loaded mile",
			1,
		),
		Version: original.Version,
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.versionRepo.On("ClearScheduled", mock.Anything, mock.Anything).Return(int64(0), nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.Update(
		t.Context(),
		payload,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)

	assert.Equal(t, "Per Mile, regional", updated.Name)
	assert.Equal(t, formulatemplate.StatusDraft, updated.Status)
	assert.EqualValues(t, 2, updated.CurrentVersionNumber)
}

func TestDuplicate_NamesEachCopyInItsHeader(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	source := formulatemplatetest.NewTemplate()
	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{source}, nil)
	deps.repo.On("TakenNames", mock.Anything, &formulatemplateports.ListTakenNamesRequest{
		TenantInfo: formulatemplatetest.NewTenantInfoFor(source),
		Bases:      []string{"Per Mile (Copy)"},
	}).Return(map[string]struct{}{"Per Mile (Copy)": {}}, nil)
	deps.repo.On("BulkCreate", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplates, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	copies, err := deps.svc.Duplicate(t.Context(), duplicateRequest(source))
	require.NoError(t, err)

	require.Len(t, copies, 1)
	assert.Equal(t, "Per Mile (Copy) 2", copies[0].Name)
	assert.Equal(
		t,
		strings.Replace(
			formulatemplatetest.PerMileSource,
			`tariff "Per Mile"`,
			`tariff "Per Mile (Copy) 2"`,
			1,
		),
		copies[0].Source,
	)
	assert.Equal(t, formulatemplate.StatusDraft, copies[0].Status)
	assert.Equal(t, &source.ID, copies[0].SourceTemplateID)
}

func TestDuplicate_RefusesACopyWhoseNameWouldPassTheLimit(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	source := formulatemplatetest.NewTemplate()
	source.Name = strings.Repeat("a", 96)
	source.Source = strings.Replace(
		formulatemplatetest.PerMileSource,
		`tariff "Per Mile"`,
		`tariff "`+source.Name+`"`,
		1,
	)
	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{source}, nil)
	deps.repo.On("TakenNames", mock.Anything, mock.Anything).Return(map[string]struct{}{}, nil)

	_, err := deps.svc.Duplicate(t.Context(), duplicateRequest(source))

	assert.Contains(t, formulatemplatetest.ValidationFields(t, err), "templateIds[0].name")
	deps.repo.AssertNotCalled(t, "BulkCreate")
}

func duplicateRequest(
	source *formulatemplate.FormulaTemplate,
) *formulatemplateports.BulkDuplicateFormulaTemplateRequest {
	return &formulatemplateports.BulkDuplicateFormulaTemplateRequest{
		TenantInfo:  formulatemplatetest.NewTenantInfoFor(source),
		TemplateIDs: []pulid.ID{source.ID},
	}
}

func TestFork_NamesTheForkInItsHeader(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	source := formulatemplatetest.NewTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(source, nil)
	deps.versionRepo.On("GetLatestVersion", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("no version"))
	deps.repo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	forked, err := deps.svc.Fork(t.Context(), &formulatemplateports.ForkTemplateRequest{
		TenantInfo:       formulatemplatetest.NewTenantInfoFor(source),
		SourceTemplateID: source.ID,
		NewName:          "Per Mile, West",
	})
	require.NoError(t, err)

	assert.Equal(t, "Per Mile, West", forked.Name)
	assert.Equal(
		t,
		strings.Replace(
			formulatemplatetest.PerMileSource,
			`tariff "Per Mile"`,
			`tariff "Per Mile, West"`,
			1,
		),
		forked.Source,
	)
	assert.Equal(t, &source.ID, forked.SourceTemplateID)
}

func TestNextAvailableName(t *testing.T) {
	t.Parallel()

	taken := map[string]struct{}{
		"Per Mile (Copy)":   {},
		"Per Mile (Copy) 2": {},
	}

	assert.Equal(t, "Flat (Copy)", nextAvailableName(taken, "Flat (Copy)"))
	assert.Equal(t, "Per Mile (Copy) 3", nextAvailableName(taken, "Per Mile (Copy)"))
}
