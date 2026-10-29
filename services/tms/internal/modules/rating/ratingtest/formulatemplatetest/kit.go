package formulatemplatetest

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulalibraryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const PerMileSource = `tariff "Per Mile"

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

const PerMileFailingSource = `tariff "Per Mile"

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

const PerMileRetypedSource = `tariff "Per Mile"

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

const BrokenSource = `tariff "Per Mile"

charge "Per Mile" (quantity: shipment.distance, rate: ratee)
`

type EmptyContext struct{}

func (EmptyContext) Context(
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

func (EmptyContext) TableStamp(context.Context, pagination.TenantInfo) (string, error) {
	return TablesStamp, nil
}

func (EmptyContext) LoadTables(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]string,
) error {
	return nil
}

func (EmptyContext) LoadCells(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]rules.TariffNeed,
) error {
	return nil
}

type FakeDependents struct {
	Sets     []*rateagreementports.TemplateInputs
	Requests []*rateagreementports.ListInputsByTemplateRequest
}

func (f *FakeDependents) ListInputsByTemplate(
	_ context.Context,
	req *rateagreementports.ListInputsByTemplateRequest,
) ([]*rateagreementports.TemplateInputs, error) {
	f.Requests = append(f.Requests, req)
	return f.Sets, nil
}

type FakeRater struct {
	Result   *tariffports.RateResult
	Err      error
	Requests []*tariffports.RateRequest
}

func (f *FakeRater) Rate(
	_ context.Context,
	req *tariffports.RateRequest,
) (*tariffports.RateResult, error) {
	f.Requests = append(f.Requests, req)
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Result, nil
}

type FakeRecords struct {
	Record   rules.RawJSON
	Err      error
	Requests []*tariffports.ShipmentRecordRequest
}

func (f *FakeRecords) ShipmentRecord(
	_ context.Context,
	req *tariffports.ShipmentRecordRequest,
) (rules.RawJSON, error) {
	f.Requests = append(f.Requests, req)
	return f.Record, f.Err
}

type NoApprovedLibraries struct{}

func (NoApprovedLibraries) ListLatestApproved(
	context.Context,
	*formulalibraryports.ListApprovedLibrariesRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return []*formulalibrary.FormulaLibraryVersion{}, nil
}

func CheckedInputs(t *testing.T, source string) []tarifftypes.InputDecl {
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

func NewTemplate() *formulatemplate.FormulaTemplate {
	return &formulatemplate.FormulaTemplate{
		ID:                   pulid.MustNew("ft_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Per Mile",
		Description:          "Linehaul per loaded mile with a minimum",
		Source:               PerMileSource,
		Inputs:               []tarifftypes.InputDecl{},
		Imports:              []tarifftypes.Import{},
		TestCount:            1,
		Status:               formulatemplate.StatusActive,
		CurrentVersionNumber: 1,
		Version:              1,
	}
}

func NewTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func ApprovedVersionOf(
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
		Inputs:         CheckedInputs(t, source),
		Imports:        []tarifftypes.Import{},
		TestCount:      1,
		Status:         formulatemplate.StatusActive,
	}
}

func ReturnsTemplate(
	_ context.Context,
	entity *formulatemplate.FormulaTemplate,
) *formulatemplate.FormulaTemplate {
	return entity
}

func ReturnsVersion(
	_ context.Context,
	version *formulatemplate.FormulaTemplateVersion,
) *formulatemplate.FormulaTemplateVersion {
	return version
}

func ValidationFields(t *testing.T, err error) map[string]*errortypes.Error {
	t.Helper()
	fields := map[string]*errortypes.Error{}
	var multi *errortypes.MultiError
	if errors.As(err, &multi) {
		for _, fieldErr := range multi.Errors {
			fields[fieldErr.Field] = fieldErr
		}
		return fields
	}
	var single *errortypes.Error
	require.ErrorAs(t, err, &single)
	fields[single.Field] = single
	return fields
}

func ReturnsTemplates(
	_ context.Context,
	entities []*formulatemplate.FormulaTemplate,
) []*formulatemplate.FormulaTemplate {
	return entities
}

func NewTenantInfoFor(template *formulatemplate.FormulaTemplate) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: template.OrganizationID,
		BuID:  template.BusinessUnitID,
	}
}

type AuditLog struct {
	mu      sync.Mutex
	entries []*audit.Entry
}

func (l *AuditLog) Last(t *testing.T) *audit.Entry {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	require.NotEmpty(t, l.entries)
	return l.entries[len(l.entries)-1]
}

func CaptureAudit(audits *auditportsmock.MockAuditService) *AuditLog {
	log := &AuditLog{}
	audits.On("LogAction", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			params := args.Get(0).(*auditports.LogActionParams)
			entry := &audit.Entry{
				ResourceID: params.ResourceID,
				Operation:  params.Operation,
			}
			if opts, ok := args.Get(1).([]auditports.LogOption); ok {
				for _, opt := range opts {
					_ = opt(entry)
				}
			}
			log.mu.Lock()
			log.entries = append(log.entries, entry)
			log.mu.Unlock()
		}).
		Return(nil)
	return log
}

func RequestContext(t *testing.T) context.Context {
	t.Helper()
	return requestmeta.With(t.Context(), requestmeta.New("req-42", "10.0.0.7", "studio"))
}

const TablesStamp = "2:4:1767225600"

type Deps struct {
	Repo            *formulatemplateportsmock.MockFormulaTemplateRepository
	VersionRepo     *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	LibraryVersions *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	AuditSvc        *auditportsmock.MockAuditService
	Resolver        *tariff.TemplateResolver
	Checker         *tariff.Checker
}

func NewDeps(t *testing.T) *Deps {
	t.Helper()

	repo := formulatemplateportsmock.NewMockFormulaTemplateRepository(t)
	versionRepo := formulatemplateportsmock.NewMockFormulaTemplateVersionRepository(t)
	libraryVersions := formulalibraryportsmock.NewMockFormulaLibraryVersionRepository(t)
	resolver := tariff.NewTemplateResolver(tariff.TemplateResolverParams{
		Templates:       repo,
		Versions:        versionRepo,
		LibraryVersions: libraryVersions,
	})
	return &Deps{
		Repo:            repo,
		VersionRepo:     versionRepo,
		LibraryVersions: libraryVersions,
		AuditSvc:        auditportsmock.NewMockAuditService(t),
		Resolver:        resolver,
		Checker: tariff.NewChecker(
			tariff.CheckerParams{Resolver: resolver, Context: EmptyContext{}},
		),
	}
}
