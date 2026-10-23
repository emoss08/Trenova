package formulalibraryservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulalibraryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const accessorialsSource = `library "org.accessorials"

/// The hazmat handling fee: a flat fee plus a surcharge for every mile.
fn hazmat(miles: Distance, fee: Money = $175, perMile: Money/mi = $0.10/mi) -> Money = fee + perMile * miles

test "hazmat adds the per-mile surcharge to the fee" {
  expect hazmat(100 mi) == $185
}
`

const accessorialsFailingSource = `library "org.accessorials"

fn hazmat(miles: Distance, fee: Money = $175) -> Money = fee

test "hazmat adds the per-mile surcharge to the fee" {
  expect hazmat(100 mi) == $185
}
`

const notALibrarySource = `tariff "Per Mile"

input rate: Money/mi

charge "Per Mile" (quantity: shipment.distance, rate: rate)
`

type testDBConnection struct{}

func (testDBConnection) DB() *bun.DB                          { return nil }
func (testDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (testDBConnection) HealthCheck(context.Context) error    { return nil }
func (testDBConnection) IsHealthy(context.Context) bool       { return true }
func (testDBConnection) Close() error                         { return nil }
func (testDBConnection) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type emptyContext struct{}

func (emptyContext) Context(
	_ context.Context,
	req *tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	return &rules.TariffContext{Libraries: req.Libraries}, nil
}

func (emptyContext) TableStamp(context.Context, pagination.TenantInfo) (string, error) {
	return "", nil
}

func (emptyContext) LoadTables(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]string,
) error {
	return nil
}

func (emptyContext) LoadCells(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]rules.TariffNeed,
) error {
	return nil
}

type testDeps struct {
	repo        *formulalibraryportsmock.MockFormulaLibraryRepository
	versionRepo *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	reviewRepo  *formulalibraryportsmock.MockFormulaLibraryReviewRepository
	usages      *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	auditSvc    *auditportsmock.MockAuditService
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()

	deps := &testDeps{
		repo:        formulalibraryportsmock.NewMockFormulaLibraryRepository(t),
		versionRepo: formulalibraryportsmock.NewMockFormulaLibraryVersionRepository(t),
		reviewRepo:  formulalibraryportsmock.NewMockFormulaLibraryReviewRepository(t),
		usages:      formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t),
		auditSvc:    auditportsmock.NewMockAuditService(t),
	}
	resolver := tariff.NewTemplateResolver(tariff.TemplateResolverParams{
		Templates:       formulatemplateportsmock.NewMockFormulaTemplateRepository(t),
		Versions:        deps.usages,
		LibraryVersions: deps.versionRepo,
	})
	deps.svc = &Service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         deps.repo,
		versionRepo:  deps.versionRepo,
		reviewRepo:   deps.reviewRepo,
		usages:       deps.usages,
		checker:      tariff.NewChecker(tariff.CheckerParams{Resolver: resolver, Context: emptyContext{}}),
		auditService: deps.auditSvc,
	}
	return deps
}

func newTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func newLibrary() *formulalibrary.FormulaLibrary {
	return &formulalibrary.FormulaLibrary{
		ID:                   pulid.MustNew("fl_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "accessorials",
		Description:          "Hazmat and residential fees",
		Source:               accessorialsSource,
		Imports:              []tarifftypes.Import{},
		TestCount:            1,
		Status:               formulatemplate.StatusDraft,
		CurrentVersionNumber: 1,
	}
}

func returnsLibrary(_ context.Context, entity *formulalibrary.FormulaLibrary) *formulalibrary.FormulaLibrary {
	return entity
}

func returnsVersion(
	_ context.Context,
	version *formulalibrary.FormulaLibraryVersion,
) *formulalibrary.FormulaLibraryVersion {
	return version
}

func fieldsOf(t *testing.T, err error) []string {
	t.Helper()
	var multi *errortypes.MultiError
	if errors.As(err, &multi) {
		fields := make([]string, 0, len(multi.Errors))
		for _, fieldErr := range multi.Errors {
			fields = append(fields, fieldErr.Field)
		}
		return fields
	}
	var single *errortypes.Error
	require.ErrorAs(t, err, &single)
	return []string{single.Field}
}

func TestCreate_ChecksTheSourceAsALibraryAndCountsItsTests(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	var snapshot *formulalibrary.FormulaLibraryVersion
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulalibrary.FormulaLibraryVersion)
		}).
		Return(returnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	library := newLibrary()
	library.Status = formulatemplate.StatusActive
	library.TestCount = 0
	created, err := deps.svc.Create(t.Context(), library, newTenantInfo().UserID)
	require.NoError(t, err)

	assert.Equal(t, formulatemplate.StatusDraft, created.Status)
	assert.EqualValues(t, 1, created.TestCount)
	assert.Empty(t, created.Imports)
	require.NotNil(t, snapshot)
	assert.EqualValues(t, 1, snapshot.VersionNumber)
	assert.Equal(t, accessorialsSource, snapshot.Source)
}

func TestCreate_RefusesATariffPosingAsALibrary(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	library.Source = notALibrarySource
	_, err := deps.svc.Create(t.Context(), library, newTenantInfo().UserID)
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), fieldSource)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_RefusesANameTemplatesCouldNotImport(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	library.Name = "Accessorial Fees"
	_, err := deps.svc.Create(t.Context(), library, newTenantInfo().UserID)
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "name")
}

func TestCreate_TheHeaderMustNameTheLibrary(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	library.Name = "fees"
	_, err := deps.svc.Create(t.Context(), library, newTenantInfo().UserID)
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), fieldSource)
	assert.Contains(t, err.Error(), "org.fees")
	deps.repo.AssertNotCalled(t, "Create")
}

func TestUpdate_AnApprovedLibraryCannotBeRenamed(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	original := newLibrary()
	original.Status = formulatemplate.StatusActive
	renamed := *original
	renamed.Name = "fees"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(&formulalibrary.FormulaLibraryVersion{VersionNumber: 2}, nil)

	_, err := deps.svc.Update(t.Context(), &renamed, newTenantInfo().UserID)
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "name")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_ASourceChangeRevertsApprovalAndSnapshotsAVersion(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	approver := pulid.MustNew("usr_")
	approvedAt := timeutils.NowUnix() - 60
	original := newLibrary()
	original.Status = formulatemplate.StatusActive
	original.ApprovedByID = &approver
	original.ApprovedAt = &approvedAt
	edited := *original
	edited.Source = accessorialsSource + "\n/// Same functions, a clearer doc.\nconst LIFTGATE: Money = $85\n"

	var snapshot *formulalibrary.FormulaLibraryVersion
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulalibrary.FormulaLibraryVersion)
		}).
		Return(returnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.Update(t.Context(), &edited, newTenantInfo().UserID)
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusDraft, updated.Status)
	assert.Nil(t, updated.ApprovedByID)
	assert.EqualValues(t, 2, updated.CurrentVersionNumber)
	require.NotNil(t, snapshot)
	assert.EqualValues(t, 2, snapshot.VersionNumber)
}

func TestUpdate_AContentSaveKeepsTheServerHeldFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	approver := pulid.MustNew("usr_")
	approvedAt := timeutils.NowUnix() - 60
	forkedFrom := pulid.MustNew("fl_")
	original := newLibrary()
	original.Status = formulatemplate.StatusActive
	original.ApprovedByID = &approver
	original.ApprovedAt = &approvedAt
	original.SourceLibraryID = &forkedFrom
	payload := &formulalibrary.FormulaLibrary{
		ID:             original.ID,
		OrganizationID: original.OrganizationID,
		BusinessUnitID: original.BusinessUnitID,
		Name:           original.Name,
		Description:    "Hazmat, residential and liftgate fees",
		Source:         original.Source,
		Version:        original.Version,
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.Update(t.Context(), payload, newTenantInfo().UserID)
	require.NoError(t, err)

	assert.Equal(t, payload.Description, updated.Description)
	assert.Equal(t, formulatemplate.StatusActive, updated.Status, "a save never moves the status")
	assert.Equal(t, &approver, updated.ApprovedByID)
	assert.Equal(t, &forkedFrom, updated.SourceLibraryID, "the fork lineage survives a save")
	assert.EqualValues(t, 1, updated.CurrentVersionNumber)
	deps.versionRepo.AssertNotCalled(t, "Create")
}

func TestSubmit_RefusesALibraryWhoseTestsFail(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	library.Source = accessorialsFailingSource
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(library, nil)

	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   library.ID,
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test")
	deps.repo.AssertNotCalled(t, "Update")
}

func inReview(library *formulalibrary.FormulaLibrary) {
	submitter := pulid.MustNew("usr_")
	submittedAt := timeutils.NowUnix() - 3600
	library.Status = formulatemplate.StatusInReview
	library.SubmittedByID = &submitter
	library.SubmittedAt = &submittedAt
}

func TestApprove_StoresAnApprovedVersionTemplatesCanPin(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	inReview(library)
	var snapshot *formulalibrary.FormulaLibraryVersion
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(library, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	deps.reviewRepo.On("Latest", mock.Anything, mock.Anything, mock.Anything).
		Return((*formulalibrary.FormulaLibraryReview)(nil), nil)
	deps.reviewRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulalibrary.FormulaLibraryReview{}, nil)
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).
		Return([]pulid.ID{}, nil).Maybe()
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulalibrary.FormulaLibraryVersion)
		}).
		Return(returnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	approved, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   library.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusActive, approved.Status)
	assert.EqualValues(t, 2, approved.CurrentVersionNumber)
	require.NotNil(t, snapshot)
	assert.Equal(t, formulatemplate.StatusActive, snapshot.Status)
	assert.EqualValues(t, 2, snapshot.VersionNumber)
	assert.Equal(t, "org.accessorials", snapshot.Pin().Path)
}

func TestApprove_RefusesTheSubmitter(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	inReview(library)
	tenant := newTenantInfo()
	library.SubmittedByID = &tenant.UserID
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(library, nil)

	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenant,
		EntityID:   library.ID,
	})
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "approvedById")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestDuplicate_FindsTheNextFreeCopyName(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	source := newLibrary()
	source.Status = formulatemplate.StatusActive
	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*formulalibrary.FormulaLibrary{source}, nil)
	deps.repo.On("NamesLike", mock.Anything, mock.Anything, "accessorialsCopy").
		Return([]string{"accessorialsCopy", "AccessorialsCopy2"}, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	copies, err := deps.svc.Duplicate(t.Context(), &DuplicateRequest{
		TenantInfo: newTenantInfo(),
		LibraryIDs: []pulid.ID{source.ID},
	})
	require.NoError(t, err)
	require.Len(t, copies, 1)
	assert.Equal(t, "accessorialsCopy3", copies[0].Name, "names are unique case-insensitively")
	assert.Contains(t, copies[0].Source, `library "org.accessorialsCopy3"`,
		"the copy's header names the copy")
	assert.NotContains(t, copies[0].Source, `library "org.accessorials"`)
	assert.Equal(t, formulatemplate.StatusDraft, copies[0].Status)
	require.NotNil(t, copies[0].SourceLibraryID)
	assert.Equal(t, source.ID, *copies[0].SourceLibraryID)

	_, err = deps.svc.Duplicate(t.Context(), &DuplicateRequest{TenantInfo: newTenantInfo()})
	assert.Contains(t, fieldsOf(t, err), "libraryIds")
}

func TestArchive_ALibraryInUseNeedsConfirmation(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	library.Status = formulatemplate.StatusActive
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(library, nil)
	deps.usages.On("ListLibraryUsages", mock.Anything, mock.Anything).
		Return([]*formulatemplateports.LibraryUsage{{TemplateName: "Hazmat lane"}}, nil)

	_, err := deps.svc.Archive(t.Context(), &ArchiveRequest{
		TenantInfo: newTenantInfo(),
		LibraryID:  library.ID,
	})
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "confirmInUse")

	deps.repo.On("Update", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	archived, err := deps.svc.Archive(t.Context(), &ArchiveRequest{
		TenantInfo:   newTenantInfo(),
		LibraryID:    library.ID,
		ConfirmInUse: true,
	})
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusInactive, archived.Status)
}

func TestTestSource_RunsTheLibrarysTests(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	passing, err := deps.svc.TestSource(t.Context(), &TestSourceRequest{
		TenantInfo: newTenantInfo(),
		Source:     accessorialsSource,
	})
	require.NoError(t, err)
	assert.True(t, passing.Check.OK)
	assert.True(t, passing.Tests.Passed())
	assert.NotNil(t, passing.Imports)

	failing, err := deps.svc.TestSource(t.Context(), &TestSourceRequest{
		TenantInfo: newTenantInfo(),
		Source:     accessorialsFailingSource,
	})
	require.NoError(t, err)
	assert.False(t, failing.Tests.Passed())
}
