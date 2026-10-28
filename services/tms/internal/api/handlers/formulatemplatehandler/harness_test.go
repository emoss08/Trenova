package formulatemplatehandler_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/formulatemplatehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulalibraryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/testutil"
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
	return &rules.TariffContext{
		Libraries: req.Libraries,
		Settings:  rules.TariffSettings{DistanceUnit: "mi"},
	}, nil
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

type fixtureOptions struct {
	granted   map[string]bool
	studio    formulastudioports.FormulaStudioAssist
	assistant assistantports.PageAssistant
}

type fixture struct {
	handler         *formulatemplatehandler.Handler
	repo            *formulatemplateportsmock.MockFormulaTemplateRepository
	versions        *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	libraryVersions *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	rater           *fakeRater
	records         *fakeRecords
}

func newFixture(t *testing.T, opts fixtureOptions) *fixture {
	t.Helper()

	f := &fixture{
		repo:            formulatemplateportsmock.NewMockFormulaTemplateRepository(t),
		versions:        formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t),
		libraryVersions: formulalibraryportsmock.NewMockFormulaLibraryVersionRepository(t),
		rater:           &fakeRater{},
		records:         &fakeRecords{},
	}
	f.libraryVersions.On("ListLatestApproved", mock.Anything, mock.Anything).
		Return([]*formulalibrary.FormulaLibraryVersion{}, nil).Maybe()
	f.versions.On("ListApproved", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplateVersion{}, nil).Maybe()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	engine := &grantedEngine{granted: opts.granted}
	resolver := tariff.NewTemplateResolver(tariff.TemplateResolverParams{
		Templates:       f.repo,
		Versions:        f.versions,
		LibraryVersions: f.libraryVersions,
	})
	service := formulatemplateservice.New(formulatemplateservice.Params{
		Logger:          logger,
		DB:              dbtest.NopConnection{},
		Repo:            f.repo,
		VersionRepo:     f.versions,
		ReviewRepo:      formulatemplateportsmock.NewMockFormulaTemplateReviewRepository(t),
		LibraryVersions: f.libraryVersions,
		Checker: tariff.NewChecker(
			tariff.CheckerParams{Resolver: resolver, Context: emptyContext{}},
		),
		Resolver:     resolver,
		Rater:        f.rater,
		Dependents:   noDependents{},
		Records:      f.records,
		AuditService: &mocks.NoopAuditService{},
	})
	f.handler = formulatemplatehandler.New(formulatemplatehandler.Params{
		Service:       service,
		PageAssistant: opts.assistant,
		StudioAssist:  opts.studio,
		ErrorHandler:  errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     errorHandler,
			},
		),
		PermissionEngine: engine,
	})
	return f
}

func (f *fixture) serve(method, path string, body any) *testutil.GinTestContext {
	ginCtx := testutil.NewGinTestContext().
		WithMethod(method).
		WithPath(path).
		WithDefaultAuthContext()
	if body != nil {
		ginCtx = ginCtx.WithJSONBody(body)
	}
	f.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)
	return ginCtx
}

func authorGrants() map[string]bool {
	return map[string]bool{
		"formula_template:read":   true,
		"formula_template:create": true,
		"formula_template:update": true,
		"formula_template:submit": true,
		"shipment:read":           true,
		"assistant:create":        true,
	}
}
