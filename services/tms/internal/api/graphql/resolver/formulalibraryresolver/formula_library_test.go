package formulalibraryresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulalibraryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const accessorialsSource = `library "org.accessorials"

/// The hazmat handling fee: a flat fee plus a surcharge for every mile.
fn hazmat(miles: Distance, fee: Money = $175, perMile: Money/mi = $0.10/mi) -> Money = fee + perMile * miles

test "hazmat adds the per-mile surcharge to the fee" {
  expect hazmat(100 mi) == $185
}
`

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

type pageAssistant struct {
	opened []*assistantports.OpenPageThreadRequest
	actors []*permissionports.RequestActor
}

func (p *pageAssistant) OpenPageThread(
	_ context.Context,
	req *assistantports.OpenPageThreadRequest,
	actor *permissionports.RequestActor,
) (*assistantports.PageThread, error) {
	p.opened = append(p.opened, req)
	p.actors = append(p.actors, actor)
	return &assistantports.PageThread{
		Thread: &conversation.Thread{ID: pulid.MustNew("athr_"), Origin: req.Origin},
	}, nil
}

type fixture struct {
	deps      *Deps
	engine    *resolvertest.GrantingPermissionEngine
	authCtx   *authctx.AuthContext
	repo      *formulalibraryportsmock.MockFormulaLibraryRepository
	versions  *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	usages    *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	assistant *pageAssistant
}

func newFixture(t *testing.T, granted map[string]bool) *fixture {
	t.Helper()

	f := &fixture{
		engine:    &resolvertest.GrantingPermissionEngine{Granted: granted},
		authCtx:   resolvertest.UserAuthContext(),
		repo:      formulalibraryportsmock.NewMockFormulaLibraryRepository(t),
		versions:  formulalibraryportsmock.NewMockFormulaLibraryVersionRepository(t),
		usages:    formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t),
		assistant: &pageAssistant{},
	}
	logger := zap.NewNop()
	resolver := tariff.NewTemplateResolver(tariff.TemplateResolverParams{
		Templates:       formulatemplateportsmock.NewMockFormulaTemplateRepository(t),
		Versions:        f.usages,
		LibraryVersions: f.versions,
	})
	f.deps = &Deps{
		Core: &resolvercore.Core{L: logger, PermissionEngine: f.engine},
		FormulaLibraryService: formulalibraryservice.New(formulalibraryservice.Params{
			Logger:           logger,
			DB:               dbtest.NopConnection{},
			Repo:             f.repo,
			VersionRepo:      f.versions,
			ReviewRepo:       formulalibraryportsmock.NewMockFormulaLibraryReviewRepository(t),
			TemplateVersions: f.usages,
			Checker: tariff.NewChecker(
				tariff.CheckerParams{Resolver: resolver, Context: emptyContext{}},
			),
			AuditService: &mocks.NoopAuditService{},
		}),
		PageAssistant: f.assistant,
	}
	return f
}

func (f *fixture) ctx(t *testing.T) context.Context {
	t.Helper()
	return resolvertest.AuthedContext(t, f.authCtx)
}

func authorGrants() map[string]bool {
	return map[string]bool{
		"formula_library|read":      true,
		"formula_library|create":    true,
		"formula_library|update":    true,
		"formula_library|duplicate": true,
		"formula_library|archive":   true,
		"assistant|create":          true,
	}
}

func returnsLibrary(
	_ context.Context,
	entity *formulalibrary.FormulaLibrary,
) *formulalibrary.FormulaLibrary {
	return entity
}

func returnsVersion(
	_ context.Context,
	version *formulalibrary.FormulaLibraryVersion,
) *formulalibrary.FormulaLibraryVersion {
	return version
}

func requireValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var multi *errortypes.MultiError
	if errors.As(err, &multi) {
		require.NotEmpty(t, multi.Errors)
		fields := make([]string, 0, len(multi.Errors))
		for _, item := range multi.Errors {
			fields = append(fields, item.Field)
		}
		assert.Contains(t, fields, field)
		return
	}
	var single *errortypes.Error
	require.True(t, errors.As(err, &single), "a validation error on %s: %v", field, err)
	assert.Equal(t, field, single.Field)
}

func TestFormulaLibraryOperations_CheckTheirPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("fl_").String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"formulaLibraryVersionDiff",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaLibraryVersionDiff(ctx, id, 1, 2)
				return err
			},
		},
		{"formulaLibraryReadiness", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).FormulaLibraryReadiness(ctx, id)
			return err
		}},
		{
			"formulaLibraryReviewDiff",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FormulaLibraryReviewDiff(ctx, id)
				return err
			},
		},
		{"formulaLibraryReviewers", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).FormulaLibraryReviewers(ctx, id)
			return err
		}},
		{"formulaLibraryUsages", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).FormulaLibraryUsages(ctx, id, nil)
			return err
		}},
		{"createFormulaLibrary", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateFormulaLibrary(
				ctx,
				gqlmodel.CreateFormulaLibraryInput{Name: "fees", Source: accessorialsSource},
			)
			return err
		}},
		{"updateFormulaLibrary", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateFormulaLibrary(
				ctx,
				id,
				gqlmodel.UpdateFormulaLibraryInput{Name: "fees", Source: accessorialsSource},
			)
			return err
		}},
		{"duplicateFormulaLibraries", permission.OpDuplicate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DuplicateFormulaLibraries(ctx, []string{id})
				return err
			}},
		{
			"rollbackFormulaLibrary",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RollbackFormulaLibrary(
					ctx,
					id,
					gqlmodel.RollbackFormulaVersionInput{TargetVersion: 1},
				)
				return err
			},
		},
		{
			"archiveFormulaLibrary",
			permission.OpArchive,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ArchiveFormulaLibrary(ctx, id, false)
				return err
			},
		},
		{"submitFormulaLibrary", permission.OpSubmit, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SubmitFormulaLibrary(
				ctx,
				id,
				gqlmodel.SubmitFormulaReviewInput{},
			)
			return err
		}},
		{
			"approveFormulaLibrary",
			permission.OpApprove,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ApproveFormulaLibrary(ctx, id, nil)
				return err
			},
		},
		{"rejectFormulaLibrary", permission.OpReject, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RejectFormulaLibrary(ctx, id, nil)
			return err
		}},
		{"requestFormulaLibraryChanges", permission.OpReject,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RequestFormulaLibraryChanges(ctx, id, nil)
				return err
			}},
		{"openFormulaLibraryAssistantThread", permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).OpenFormulaLibraryAssistantThread(ctx, &id)
				return err
			}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceFormulaLibrary,
				call.operation,
			)
		})
	}
}

func TestCreateFormulaLibrary_StoresACheckedLibraryAsADraft(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	f.repo.On("Create", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	f.versions.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)
	description := "Hazmat and residential fees"

	created, err := (&MutationResolver{f.deps}).CreateFormulaLibrary(
		f.ctx(t),
		gqlmodel.CreateFormulaLibraryInput{
			Name:        "accessorials",
			Description: &description,
			Source:      accessorialsSource,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusDraft, created.Status)
	assert.EqualValues(t, 1, created.TestCount)
	assert.Equal(t, description, created.Description)
	assert.Equal(t, f.authCtx.OrganizationID, created.OrganizationID)
	assert.Equal(t, f.authCtx.BusinessUnitID, created.BusinessUnitID)
}

func TestCreateFormulaLibrary_RefusesASourceThatDoesNotCheck(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())

	_, err := (&MutationResolver{f.deps}).CreateFormulaLibrary(
		f.ctx(t),
		gqlmodel.CreateFormulaLibraryInput{
			Name:   "accessorials",
			Source: "library \"org.accessorials\"\n\nfn hazmat() -> Money = $1 +\n",
		},
	)

	requireValidationField(t, err, "source")
	f.repo.AssertNotCalled(t, "Create")
}

func TestArchiveFormulaLibrary_AsksForConfirmationWhileTemplatesImportIt(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	libraryID := pulid.MustNew("fl_")
	f.repo.On("GetByID", mock.Anything, mock.Anything).Return(&formulalibrary.FormulaLibrary{
		ID:     libraryID,
		Name:   "accessorials",
		Source: accessorialsSource,
		Status: formulatemplate.StatusActive,
	}, nil)
	f.usages.On("ListLibraryUsages", mock.Anything, mock.Anything).
		Return([]*formulatemplateports.LibraryUsage{{TemplateName: "Hazmat lane"}}, nil)

	_, err := (&MutationResolver{f.deps}).ArchiveFormulaLibrary(
		f.ctx(t),
		libraryID.String(),
		false,
	)

	requireValidationField(t, err, "confirmInUse")
}

func TestFormulaLibraryVersionDiff_NeedsTwoVersionNumbers(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())

	_, err := (&QueryResolver{f.deps}).FormulaLibraryVersionDiff(
		f.ctx(t),
		pulid.MustNew("fl_").String(),
		1,
		0,
	)

	requireValidationField(t, err, "versions")
}

func TestFormulaLibraryUsages_ListsTheTemplateVersionsPinningAVersion(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	libraryID := pulid.MustNew("fl_")
	f.repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&formulalibrary.FormulaLibrary{ID: libraryID, Name: "accessorials"}, nil)
	f.usages.On("ListLibraryUsages", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.ListLibraryUsagesRequest) bool {
			return req.LibraryID == libraryID &&
				req.VersionNumber != nil && *req.VersionNumber == 2 &&
				req.TenantInfo.OrgID == f.authCtx.OrganizationID
		},
	)).Return([]*formulatemplateports.LibraryUsage{{TemplateName: "Hazmat lane"}}, nil)
	version := 2

	usages, err := (&QueryResolver{f.deps}).FormulaLibraryUsages(
		f.ctx(t),
		libraryID.String(),
		&version,
	)
	require.NoError(t, err)
	require.Len(t, usages, 1)
	assert.Equal(t, "Hazmat lane", usages[0].TemplateName)

	zero := 0
	_, err = (&QueryResolver{f.deps}).FormulaLibraryUsages(f.ctx(t), libraryID.String(), &zero)
	requireValidationField(t, err, "versionNumber")
}

func TestOpenFormulaLibraryAssistantThread_OpensTheLibrarysConversation(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	libraryID := pulid.MustNew("fl_")
	f.repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&formulalibrary.FormulaLibrary{ID: libraryID, Name: "accessorials"}, nil)
	raw := libraryID.String()

	opened, err := (&MutationResolver{f.deps}).OpenFormulaLibraryAssistantThread(f.ctx(t), &raw)

	require.NoError(t, err)
	require.Len(t, f.assistant.opened, 1)
	assert.Equal(t, conversation.ThreadOriginFormula, f.assistant.opened[0].Origin)
	assert.Equal(t, agent.SubjectFormulaLibrary, f.assistant.opened[0].SubjectType)
	assert.Equal(t, libraryID, f.assistant.opened[0].SubjectID)
	assert.Equal(t, f.authCtx.UserID, f.assistant.actors[0].UserID)
	assert.Equal(t, "Formula", opened.Thread.Origin)

	reader := newFixture(t, map[string]bool{"formula_library|read": true})
	_, err = (&MutationResolver{reader.deps}).OpenFormulaLibraryAssistantThread(
		reader.ctx(t),
		nil,
	)
	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
	assert.Empty(t, reader.assistant.opened)
}
