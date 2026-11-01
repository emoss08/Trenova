package formulatemplateresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatereviewservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulalibraryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const perMileSource = `tariff "Per Mile"

/// What the contract pays per loaded mile.
input rate: Money/mi
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)
minimum minimumCharge

test "a short haul bills its miles" {
  given input.rate = $2/mi
  given shipment.distance = 100 mi
  expect total == $200
}
`

const brokenSource = `tariff "Per Mile"

charge "Per Mile" (quantity: shipment.distance, rate: ratee)
`

type emptyContext struct{}

func (emptyContext) Context(
	_ context.Context,
	req *tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	return &rules.TariffContext{
		Libraries: req.Libraries,
		Settings:  rules.TariffSettings{DistanceUnit: "mi"},
	}, nil
}

func (emptyContext) TableStamp(context.Context, pagination.TenantInfo) (string, error) {
	return "12:40:1767225600", nil
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

type noDependents struct{}

func (noDependents) ListInputsByTemplate(
	context.Context,
	*rateagreementports.ListInputsByTemplateRequest,
) ([]*rateagreementports.TemplateInputs, error) {
	return nil, nil
}

type fakeRater struct {
	result   *tariffports.RateResult
	requests []*tariffports.RateRequest
}

func (f *fakeRater) Rate(
	_ context.Context,
	req *tariffports.RateRequest,
) (*tariffports.RateResult, error) {
	f.requests = append(f.requests, req)
	return f.result, nil
}

type fakeRecords struct {
	requests []*tariffports.ShipmentRecordRequest
}

func (f *fakeRecords) ShipmentRecord(
	_ context.Context,
	req *tariffports.ShipmentRecordRequest,
) (rules.RawJSON, error) {
	f.requests = append(f.requests, req)
	return rules.RawJSON(`{"shipment":{}}`), nil
}

type recordingStudio struct {
	explained []*formulastudioports.ExplainFormulaRequest
	described []*formulastudioports.DescribeFormulaInputsRequest
	suggested []*formulastudioports.SuggestFormulaTestsRequest
	tested    []*formulastudioports.ExplainFormulaTestRequest
	notes     []*formulastudioports.DraftFormulaReviewNoteRequest
	actors    []*permissionports.RequestActor
}

func (s *recordingStudio) ExplainFormula(
	_ context.Context,
	req *formulastudioports.ExplainFormulaRequest,
	actor *permissionports.RequestActor,
) (*formulastudioports.ExplainFormulaResponse, error) {
	s.explained = append(s.explained, req)
	s.actors = append(s.actors, actor)
	return &formulastudioports.ExplainFormulaResponse{
		Items: []formulastudioports.FormulaTraceNote{{
			Index: 0,
			Range: &rules.TariffRange{From: 3, To: 9},
			Says:  "It charges by the mile.",
		}},
	}, nil
}

func (s *recordingStudio) DescribeInputs(
	_ context.Context,
	req *formulastudioports.DescribeFormulaInputsRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.DescribeFormulaInputsResponse, error) {
	s.described = append(s.described, req)
	return &formulastudioports.DescribeFormulaInputsResponse{
		Descriptions: map[string]string{"rate": "What the contract pays per mile"},
	}, nil
}

func (s *recordingStudio) SuggestTests(
	_ context.Context,
	req *formulastudioports.SuggestFormulaTestsRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.SuggestFormulaTestsResponse, error) {
	s.suggested = append(s.suggested, req)
	return &formulastudioports.SuggestFormulaTestsResponse{
		Tests: []formulastudioports.SuggestedFormulaTest{{
			Name:   "Minimum reached",
			Block:  "test \"Minimum reached\" {\n  expect total >= $0\n}",
			Status: rules.TariffTestPassed,
		}},
	}, nil
}

func (s *recordingStudio) ExplainTest(
	_ context.Context,
	req *formulastudioports.ExplainFormulaTestRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.ExplainFormulaTestResponse, error) {
	s.tested = append(s.tested, req)
	return &formulastudioports.ExplainFormulaTestResponse{
		Verdict: formulastudioports.FormulaTestVerdictUnclear,
		Why:     "The expectation reads the wrong total.",
	}, nil
}

func (s *recordingStudio) DraftReviewNote(
	_ context.Context,
	req *formulastudioports.DraftFormulaReviewNoteRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.DraftFormulaReviewNoteResponse, error) {
	s.notes = append(s.notes, req)
	return &formulastudioports.DraftFormulaReviewNoteResponse{
		Note:      "Raised the rate.",
		Generated: true,
	}, nil
}

type recordingAssistant struct {
	requests []*assistantports.OpenPageThreadRequest
	actors   []*permissionports.RequestActor
	opened   *assistantports.PageThread
}

func (a *recordingAssistant) OpenPageThread(
	_ context.Context,
	req *assistantports.OpenPageThreadRequest,
	actor *permissionports.RequestActor,
) (*assistantports.PageThread, error) {
	a.requests = append(a.requests, req)
	a.actors = append(a.actors, actor)
	return a.opened, nil
}

type fixtureOptions struct {
	granted   map[string]bool
	studio    formulastudioports.FormulaStudioAssist
	assistant assistantports.PageAssistant
}

type fixture struct {
	deps            *Deps
	engine          *resolvertest.GrantingPermissionEngine
	authCtx         *authctx.AuthContext
	repo            *formulatemplateportsmock.MockFormulaTemplateRepository
	versions        *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	reviews         *formulatemplateportsmock.MockFormulaTemplateReviewRepository
	libraryVersions *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	rater           *fakeRater
	records         *fakeRecords
}

func newFixture(t *testing.T, opts fixtureOptions) *fixture {
	t.Helper()

	f := &fixture{
		engine:          &resolvertest.GrantingPermissionEngine{Granted: opts.granted},
		authCtx:         resolvertest.UserAuthContext(),
		repo:            formulatemplateportsmock.NewMockFormulaTemplateRepository(t),
		versions:        formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t),
		reviews:         formulatemplateportsmock.NewMockFormulaTemplateReviewRepository(t),
		libraryVersions: formulalibraryportsmock.NewMockFormulaLibraryVersionRepository(t),
		rater:           &fakeRater{},
		records:         &fakeRecords{},
	}
	f.libraryVersions.On("ListLatestApproved", mock.Anything, mock.Anything).
		Return([]*formulalibrary.FormulaLibraryVersion{}, nil).Maybe()
	f.versions.On("ListApproved", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplateVersion{}, nil).Maybe()

	logger := zap.NewNop()
	resolver := tariff.NewTemplateResolver(tariff.TemplateResolverParams{
		Templates:       f.repo,
		Versions:        f.versions,
		LibraryVersions: f.libraryVersions,
	})
	checker := tariff.NewChecker(
		tariff.CheckerParams{Resolver: resolver, Context: emptyContext{}},
	)
	templates := formulatemplateservice.New(formulatemplateservice.Params{
		Logger:       logger,
		DB:           dbtest.NopConnection{},
		Repo:         f.repo,
		VersionRepo:  f.versions,
		Checker:      checker,
		Resolver:     resolver,
		AuditService: &mocks.NoopAuditService{},
	})
	f.deps = &Deps{
		Core:                   &resolvercore.Core{L: logger, PermissionEngine: f.engine},
		FormulaTemplateService: templates,
		FormulaTemplateReviewService: formulatemplatereviewservice.New(
			formulatemplatereviewservice.Params{
				Logger:      logger,
				DB:          dbtest.NopConnection{},
				Templates:   templates,
				Repo:        f.repo,
				VersionRepo: f.versions,
				ReviewRepo:  f.reviews,
				Checker:     checker,
				Resolver:    resolver,
				Dependents:  noDependents{},
			},
		),
		FormulaTemplateStudioService: formulatemplatestudioservice.New(
			formulatemplatestudioservice.Params{
				Logger:          logger,
				Templates:       templates,
				VersionRepo:     f.versions,
				LibraryVersions: f.libraryVersions,
				Checker:         checker,
				Resolver:        resolver,
				Rater:           f.rater,
				Records:         f.records,
			},
		),
		FormulaTemplateImportService: formulatemplateimportservice.New(
			formulatemplateimportservice.Params{
				Logger:      logger,
				DB:          dbtest.NopConnection{},
				Templates:   templates,
				Repo:        f.repo,
				VersionRepo: f.versions,
			},
		),
		FormulaStudioAssist: opts.studio,
		PageAssistant:       opts.assistant,
	}
	return f
}

func (f *fixture) ctx(t *testing.T) context.Context {
	t.Helper()
	return resolvertest.AuthedContext(t, f.authCtx)
}

func (f *fixture) query() *QueryResolver { return &QueryResolver{f.deps} }

func (f *fixture) mutation() *MutationResolver { return &MutationResolver{f.deps} }

func authorGrants() map[string]bool {
	return map[string]bool{
		"formula_template|read":   true,
		"formula_template|create": true,
		"formula_template|update": true,
		"formula_template|submit": true,
		"shipment|read":           true,
		"assistant|create":        true,
	}
}

func returnsTemplate(
	_ context.Context,
	entity *formulatemplate.FormulaTemplate,
) *formulatemplate.FormulaTemplate {
	return entity
}

func returnsVersion(
	_ context.Context,
	version *formulatemplate.FormulaTemplateVersion,
) *formulatemplate.FormulaTemplateVersion {
	return version
}
