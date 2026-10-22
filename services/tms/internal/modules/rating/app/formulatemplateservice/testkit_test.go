package formulatemplateservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulalibraryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const perMileSource = `tariff "Per Mile"

/// What the contract pays per loaded mile.
input rate: Money/mi
/// The least a load bills.
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)
minimum minimumCharge

test "a short haul bills its miles" {
  given input.rate = $2/mi
  given shipment.distance = 100 mi
  expect total == $200
}
`

const perMileFailingSource = `tariff "Per Mile"

input rate: Money/mi
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)
minimum minimumCharge

test "a short haul bills its miles" {
  given input.rate = $2/mi
  given shipment.distance = 100 mi
  expect total == $999
}
`

const perMileRetypedSource = `tariff "Per Mile"

input rate: Money/cwt
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.weight ?? 0 lb, rate: rate)
minimum minimumCharge

test "a light load bills its hundredweight" {
  given input.rate = $10/cwt
  given shipment.weight = 1_000 lb
  expect total == $100
}
`

const brokenSource = `tariff "Per Mile"

charge "Per Mile" (quantity: shipment.distance, rate: ratee)
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
	codes := []rules.TariffChargeCode{{Code: "FSC", Description: "Fuel surcharge", Active: true}}
	return &rules.TariffContext{
		Libraries: req.Libraries,
		Codes:     &codes,
		Settings:  rules.TariffSettings{DistanceUnit: "mi"},
	}, nil
}

func (emptyContext) TableStamp(context.Context, pagination.TenantInfo) (string, error) {
	return testTablesStamp, nil
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

type fakeDependents struct {
	sets     []*rateagreementports.TemplateInputs
	requests []*rateagreementports.ListInputsByTemplateRequest
}

func (f *fakeDependents) ListInputsByTemplate(
	_ context.Context,
	req *rateagreementports.ListInputsByTemplateRequest,
) ([]*rateagreementports.TemplateInputs, error) {
	f.requests = append(f.requests, req)
	return f.sets, nil
}

type fakeRater struct {
	result   *tariffports.RateResult
	err      error
	requests []*tariffports.RateRequest
}

func (f *fakeRater) Rate(
	_ context.Context,
	req *tariffports.RateRequest,
) (*tariffports.RateResult, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeRecords struct {
	record   rules.RawJSON
	err      error
	requests []*tariffports.ShipmentRecordRequest
}

func (f *fakeRecords) ShipmentRecord(
	_ context.Context,
	req *tariffports.ShipmentRecordRequest,
) (rules.RawJSON, error) {
	f.requests = append(f.requests, req)
	return f.record, f.err
}

type noApprovedLibraries struct{}

func (noApprovedLibraries) ListLatestApproved(
	context.Context,
	*formulalibraryports.ListApprovedLibrariesRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return []*formulalibrary.FormulaLibraryVersion{}, nil
}

type testDeps struct {
	repo            *formulatemplateportsmock.MockFormulaTemplateRepository
	versionRepo     *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	libraryVersions *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	auditSvc        *auditportsmock.MockAuditService
	dependents      *fakeDependents
	rater           *fakeRater
	records         *fakeRecords
	resolver        *tariff.TemplateResolver
	svc             *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()

	repo := formulatemplateportsmock.NewMockFormulaTemplateRepository(t)
	versionRepo := formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t)
	libraryVersions := formulalibraryportsmock.NewMockFormulaLibraryVersionRepository(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	resolver := tariff.NewTemplateResolver(tariff.TemplateResolverParams{
		Templates:       repo,
		Versions:        versionRepo,
		LibraryVersions: libraryVersions,
	})
	checker := tariff.NewChecker(tariff.CheckerParams{Resolver: resolver, Context: emptyContext{}})
	deps := &testDeps{
		repo:            repo,
		versionRepo:     versionRepo,
		libraryVersions: libraryVersions,
		auditSvc:        auditSvc,
		dependents:      &fakeDependents{},
		rater:           &fakeRater{},
		records:         &fakeRecords{},
		resolver:        resolver,
	}
	deps.svc = &Service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         repo,
		versionRepo:  versionRepo,
		reviewRepo:   &stubReviewRepo{},
		libraries:    noApprovedLibraries{},
		checker:      checker,
		resolver:     resolver,
		rater:        deps.rater,
		dependents:   deps.dependents,
		records:      deps.records,
		auditService: auditSvc,
	}
	return deps
}

func checkedInputs(t *testing.T, source string) []tarifftypes.InputDecl {
	t.Helper()
	check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source:  source,
		Kind:    rules.TariffKindTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.True(t, check.OK, "%v", check.Diagnostics)
	return tarifftypes.InputDeclsFrom(check.Inputs)
}

func newTestTemplate() *formulatemplate.FormulaTemplate {
	return &formulatemplate.FormulaTemplate{
		ID:                   pulid.MustNew("ft_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Per Mile",
		Description:          "Linehaul per loaded mile with a minimum",
		Source:               perMileSource,
		Inputs:               []tarifftypes.InputDecl{},
		Imports:              []tarifftypes.Import{},
		TestCount:            1,
		Status:               formulatemplate.StatusActive,
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

func approvedVersionOf(
	t *testing.T,
	template *formulatemplate.FormulaTemplate,
	number int64,
	source string,
) *formulatemplate.FormulaTemplateVersion {
	t.Helper()
	return &formulatemplate.FormulaTemplateVersion{
		ID:             pulid.MustNew("ftv_"),
		TemplateID:     template.ID,
		OrganizationID: template.OrganizationID,
		BusinessUnitID: template.BusinessUnitID,
		VersionNumber:  number,
		Name:           template.Name,
		Description:    template.Description,
		Source:         source,
		Inputs:         checkedInputs(t, source),
		Imports:        []tarifftypes.Import{},
		TestCount:      1,
		Status:         formulatemplate.StatusActive,
	}
}
