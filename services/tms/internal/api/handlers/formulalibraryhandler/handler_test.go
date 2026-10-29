package formulalibraryhandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/formulalibraryhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
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
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
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

type grantedEngine struct {
	permissionports.PermissionEngine

	granted map[string]bool
}

func (e *grantedEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{
		Allowed: e.granted[req.Resource+":"+string(req.Operation)],
	}, nil
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

type pageAssistant struct {
	opened []*assistantports.OpenPageThreadRequest
}

func (p *pageAssistant) OpenPageThread(
	_ context.Context,
	req *assistantports.OpenPageThreadRequest,
	_ *permissionports.RequestActor,
) (*assistantports.PageThread, error) {
	p.opened = append(p.opened, req)
	return &assistantports.PageThread{
		Thread: &conversation.Thread{ID: pulid.MustNew("athr_"), Origin: req.Origin},
	}, nil
}

type fixture struct {
	handler   *formulalibraryhandler.Handler
	repo      *formulalibraryportsmock.MockFormulaLibraryRepository
	versions  *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	usages    *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	assistant *pageAssistant
}

func newFixture(t *testing.T, granted map[string]bool) *fixture {
	t.Helper()

	f := &fixture{
		repo:      formulalibraryportsmock.NewMockFormulaLibraryRepository(t),
		versions:  formulalibraryportsmock.NewMockFormulaLibraryVersionRepository(t),
		usages:    formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t),
		assistant: &pageAssistant{},
	}
	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &appconfig.Config{Debug: true},
	})
	engine := &grantedEngine{granted: granted}
	resolver := tariff.NewTemplateResolver(tariff.TemplateResolverParams{
		Templates:       formulatemplateportsmock.NewMockFormulaTemplateRepository(t),
		Versions:        f.usages,
		LibraryVersions: f.versions,
	})
	f.handler = formulalibraryhandler.New(formulalibraryhandler.Params{
		Service: formulalibraryservice.New(formulalibraryservice.Params{
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
		ErrorHandler:  errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     errorHandler,
			},
		),
	})
	return f
}

func (f *fixture) serve(method, path string, body any) *testutil.GinTestContext {
	return f.serveQuery(method, path, nil, body)
}

func (f *fixture) serveQuery(
	method, path string,
	query map[string]string,
	body any,
) *testutil.GinTestContext {
	ginCtx := testutil.NewGinTestContext().
		WithMethod(method).
		WithPath(path).
		WithDefaultAuthContext()
	if query != nil {
		ginCtx = ginCtx.WithQuery(query)
	}
	if body != nil {
		ginCtx = ginCtx.WithJSONBody(body)
	}
	f.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)
	return ginCtx
}

func authorGrants() map[string]bool {
	return map[string]bool{
		"formula_library:read":      true,
		"formula_library:create":    true,
		"formula_library:update":    true,
		"formula_library:duplicate": true,
		"formula_library:archive":   true,
		"assistant:create":          true,
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

func TestCreate_StoresACheckedLibraryAsADraft(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	f.repo.On("Create", mock.Anything, mock.Anything).Return(returnsLibrary, nil)
	f.versions.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)

	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-libraries/", map[string]any{
		"name":        "accessorials",
		"description": "Hazmat and residential fees",
		"source":      accessorialsSource,
	})
	require.Equal(t, http.StatusCreated, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var created map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&created))
	assert.Equal(t, "Draft", created["status"])
	assert.EqualValues(t, 1, created["testCount"])
	assert.Equal(t, testutil.TestOrgID.String(), created["organizationId"])
}

func TestCreate_RefusesWithoutPermissionOrAValidSource(t *testing.T) {
	t.Parallel()

	reader := newFixture(t, map[string]bool{"formula_library:read": true})
	ginCtx := reader.serve(http.MethodPost, "/api/v1/formula-libraries/", map[string]any{
		"name":   "accessorials",
		"source": accessorialsSource,
	})
	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())

	f := newFixture(t, authorGrants())
	ginCtx = f.serve(http.MethodPost, "/api/v1/formula-libraries/", map[string]any{
		"name":   "accessorials",
		"source": "library \"org.accessorials\"\n\nfn hazmat() -> Money = $1 +\n",
	})
	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
	assert.Contains(t, ginCtx.Recorder.Body.String(), `"field":"source"`)
	f.repo.AssertNotCalled(t, "Create")
}

func TestArchive_AsksForConfirmationWhileTemplatesImportIt(t *testing.T) {
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

	ginCtx := f.serve(http.MethodPost,
		"/api/v1/formula-libraries/"+libraryID.String()+"/archive", map[string]any{})
	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
	assert.Contains(t, ginCtx.Recorder.Body.String(), "confirmInUse")
}

func TestCompare_NeedsTwoVersionNumbers(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	ginCtx := f.serveQuery(http.MethodGet,
		"/api/v1/formula-libraries/"+pulid.MustNew("fl_").String()+"/compare",
		map[string]string{"from": "1"}, nil)
	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestUsages_ListsTheTemplateVersionsImportingIt(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	libraryID := pulid.MustNew("fl_")
	f.repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&formulalibrary.FormulaLibrary{ID: libraryID, Name: "accessorials"}, nil)
	f.usages.On("ListLibraryUsages", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.ListLibraryUsagesRequest) bool {
			return req.LibraryID == libraryID && req.VersionNumber != nil && *req.VersionNumber == 2
		},
	)).Return([]*formulatemplateports.LibraryUsage{{TemplateName: "Hazmat lane"}}, nil)

	ginCtx := f.serveQuery(http.MethodGet,
		"/api/v1/formula-libraries/"+libraryID.String()+"/usages",
		map[string]string{"versionNumber": "2"}, nil)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())
	assert.Contains(t, ginCtx.Recorder.Body.String(), "Hazmat lane")
}

func TestOpenAssistantThread_OpensTheLibrarysConversation(t *testing.T) {
	t.Parallel()

	f := newFixture(t, authorGrants())
	libraryID := pulid.MustNew("fl_")
	f.repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&formulalibrary.FormulaLibrary{ID: libraryID, Name: "accessorials"}, nil)

	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-libraries/ai/thread/",
		map[string]any{"libraryId": libraryID.String()})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())
	require.Len(t, f.assistant.opened, 1)
	assert.Equal(t, conversation.ThreadOriginFormula, f.assistant.opened[0].Origin)
	assert.Equal(t, agent.SubjectFormulaLibrary, f.assistant.opened[0].SubjectType)
	assert.Equal(t, libraryID, f.assistant.opened[0].SubjectID)

	reader := newFixture(t, map[string]bool{"formula_library:read": true})
	ginCtx = reader.serve(http.MethodPost, "/api/v1/formula-libraries/ai/thread/", map[string]any{})
	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
	assert.Empty(t, reader.assistant.opened)
}
