package tariff

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/effectiveversioncache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTenant = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

type fakeLibraryVersions struct {
	approved   map[string]*formulalibrary.FormulaLibraryVersion
	pinned     []*formulalibrary.FormulaLibraryVersion
	nameCalls  [][]string
	pinCalls   int
	lastTenant pagination.TenantInfo
}

func (f *fakeLibraryVersions) LatestApprovedByNames(
	_ context.Context,
	req *formulalibraryports.LatestApprovedByNamesRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	f.nameCalls = append(f.nameCalls, req.Names)
	f.lastTenant = req.TenantInfo
	out := make([]*formulalibrary.FormulaLibraryVersion, 0, len(req.Names))
	for _, name := range req.Names {
		if version, ok := f.approved[strings.ToLower(name)]; ok {
			out = append(out, version)
		}
	}
	return out, nil
}

func (f *fakeLibraryVersions) GetByPins(
	_ context.Context,
	req *formulalibraryports.GetVersionsByPinsRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	f.pinCalls++
	out := make([]*formulalibrary.FormulaLibraryVersion, 0, len(req.Pins))
	for _, pin := range req.Pins {
		for _, version := range f.pinned {
			if version.LibraryID == pin.LibraryID && version.VersionNumber == pin.VersionNumber {
				out = append(out, version)
			}
		}
	}
	return out, nil
}

func libraryVersion(name string, number int64, source string) *formulalibrary.FormulaLibraryVersion {
	return &formulalibrary.FormulaLibraryVersion{
		ID:            pulid.MustNew("flv_"),
		LibraryID:     pulid.MustNew("fl_"),
		Name:          name,
		VersionNumber: number,
		Source:        source,
		Status:        formulatemplate.StatusActive,
	}
}

func TestImportedLibraryNames_ReadsOrgUsesOnly(t *testing.T) {
	t.Parallel()

	source := `tariff "Dry van"

use std.fuel.{ stepPercent }
use org.accessorials.{ hazmat }
  use org.zones
// use org.commented
use org.accessorials.{ residential }
let note = "use org.notAtLineStart"
`
	assert.Equal(t,
		[]string{"accessorials", "zones"},
		ImportedLibraryNames(source),
	)
}

func TestLatestLibraries_ResolvesTransitivelyInBatches(t *testing.T) {
	t.Parallel()

	accessorials := libraryVersion("accessorials", 3,
		"library \"org.accessorials\"\n\nuse org.money\n")
	money := libraryVersion("money", 1, "library \"org.money\"\n\nuse org.accessorials\n")
	zones := libraryVersion("zones", 2, "library \"org.zones\"\n")
	fake := &fakeLibraryVersions{approved: map[string]*formulalibrary.FormulaLibraryVersion{
		"accessorials": accessorials,
		"money":        money,
		"zones":        zones,
	}}
	loader := libraryLoader{versions: fake}

	set, err := loader.latest(t.Context(), testTenant,
		"tariff \"x\"\nuse org.accessorials\nuse org.missing\n")
	require.NoError(t, err)

	assert.Equal(t, [][]string{{"accessorials", "missing"}, {"money"}}, fake.nameCalls,
		"one query per round, a cycle is not asked twice")
	require.Len(t, set, 2)
	assert.Equal(t, "accessorials", set[0].Name)
	assert.Equal(t, "money", set[1].Name)
	assert.Equal(t, testTenant, fake.lastTenant)

	context := set.Context()
	assert.Equal(t, "org.accessorials", context[0].Path)
	assert.Equal(t, accessorials.ID.String(), context[0].Version)
	assert.Equal(t, accessorials.Source, context[0].Source)
}

func TestLatestLibraries_IsBounded(t *testing.T) {
	t.Parallel()

	approved := make(map[string]*formulalibrary.FormulaLibraryVersion, 64)
	var source strings.Builder
	source.WriteString("tariff \"many\"\n")
	for i := range 64 {
		name := "lib" + strings.Repeat("x", i)
		approved[strings.ToLower(name)] = libraryVersion(name, 1, "library \"x\"\n")
		source.WriteString("use org." + name + "\n")
	}
	fake := &fakeLibraryVersions{approved: approved}

	set, err := libraryLoader{versions: fake}.latest(t.Context(), testTenant, source.String())
	require.NoError(t, err)
	assert.Len(t, set, MaxLibraries)
}

func TestLibrarySet_PinsWhatTheCheckImportedAndWhatThoseImport(t *testing.T) {
	t.Parallel()

	accessorials := libraryVersion("accessorials", 3, "library \"a\"\nuse org.money\n")
	money := libraryVersion("money", 1, "library \"m\"\n")
	unused := libraryVersion("unused", 4, "library \"u\"\n")
	set := LibrarySet{accessorials, money, unused}

	pins := set.Pins([]rules.TariffImport{
		{Path: "std.fuel", Version: "1.0.0"},
		{Path: "org.accessorials", Version: accessorials.ID.String()},
	})

	assert.Equal(t, []tarifftypes.Import{
		{Path: "org.accessorials", LibraryID: accessorials.LibraryID, VersionNumber: 3},
		{Path: "org.money", LibraryID: money.LibraryID, VersionNumber: 1},
	}, pins)
}

type fakeTemplates struct {
	template *formulatemplate.FormulaTemplate
	calls    int
}

func (f *fakeTemplates) GetByID(
	_ context.Context,
	_ formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	f.calls++
	return f.template, nil
}

type fakeApproved struct {
	versions []*formulatemplate.FormulaTemplateVersion
	calls    int
}

func (f *fakeApproved) ListApproved(
	_ context.Context,
	_ *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	f.calls++
	return f.versions, nil
}

func TestResolveBilling_UsesTheVersionInForceWithItsPinnedLibraries(t *testing.T) {
	t.Parallel()

	accessorials := libraryVersion("accessorials", 2, "library \"org.accessorials\"\n")
	template := &formulatemplate.FormulaTemplate{
		ID:                   pulid.MustNew("ft_"),
		Name:                 "Dry van",
		Status:               formulatemplate.StatusDraft,
		Source:               "tariff \"draft edit\"\n",
		CurrentVersionNumber: 5,
	}
	from, later := int64(1_000), int64(5_000)
	inForce := &formulatemplate.FormulaTemplateVersion{
		VersionNumber: 3,
		Status:        formulatemplate.StatusActive,
		Source:        "tariff \"approved\"\n",
		Inputs:        []tarifftypes.InputDecl{{Name: "rate", Required: true}},
		Imports:       []tarifftypes.Import{accessorials.Pin()},
		EffectiveFrom: &from,
	}
	future := &formulatemplate.FormulaTemplateVersion{
		VersionNumber: 4,
		Status:        formulatemplate.StatusActive,
		Source:        "tariff \"next\"\n",
		EffectiveFrom: &later,
	}
	templates := &fakeTemplates{template: template}
	approved := &fakeApproved{versions: []*formulatemplate.FormulaTemplateVersion{inForce, future}}
	libraries := &fakeLibraryVersions{pinned: []*formulalibrary.FormulaLibraryVersion{accessorials}}
	resolver := &TemplateResolver{
		templates: templates,
		versions:  approved,
		libraries: libraryLoader{versions: libraries},
		now:       func() int64 { return 2_000 },
	}

	ctx := effectiveversioncache.With(t.Context())
	req := &tariffports.ResolveRequest{TenantInfo: testTenant, TemplateID: template.ID, AsOf: 2_000}
	resolved, err := resolver.ResolveBilling(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, template.ID, resolved.TemplateID)
	assert.EqualValues(t, 3, resolved.VersionNumber)
	assert.Equal(t, inForce.Source, resolved.Source)
	assert.Equal(t, inForce.Inputs, resolved.Inputs)
	assert.Equal(t, inForce.Imports, resolved.Imports)
	require.Len(t, resolved.Libraries, 1)
	assert.Equal(t, "org.accessorials", resolved.Libraries[0].Path)
	assert.Equal(t, accessorials.Source, resolved.Libraries[0].Source)

	_, err = resolver.ResolveBilling(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, templates.calls, "the template is read once per request")
	assert.Equal(t, 1, approved.calls, "the approved versions are read once per request")
	assert.Equal(t, 1, libraries.pinCalls, "the pinned libraries are read in one batch, once")
}

func TestBillingContent_RefusesWhatCannotRate(t *testing.T) {
	t.Parallel()

	later := int64(9_000)
	tests := []struct {
		name    string
		status  formulatemplate.Status
		billing BillingResolution
		wantErr string
	}{
		{name: "archived", status: formulatemplate.StatusInactive, wantErr: "archived"},
		{
			name:   "approved for later",
			status: formulatemplate.StatusActive,
			billing: BillingResolution{Scheduled: &formulatemplate.FormulaTemplateVersion{
				EffectiveFrom: &later,
			}},
			wantErr: "approved to bill from",
		},
		{name: "never approved", status: formulatemplate.StatusDraft, wantErr: "never been approved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := BillingContent(&formulatemplate.FormulaTemplate{
				Name:   "Dry van",
				Status: tt.status,
			}, tt.billing)
			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, FieldFormulaTemplateID, validation.Field)
			assert.Contains(t, validation.Error(), tt.wantErr)
		})
	}

	active := &formulatemplate.FormulaTemplate{Status: formulatemplate.StatusActive, Source: "x"}
	content, err := BillingContent(active, BillingResolution{})
	require.NoError(t, err)
	assert.Same(t, active, content, "an active template with no snapshot rates its row")
}

type fakeContext struct {
	requests []*tariffports.ContextRequest
}

func (f *fakeContext) Context(
	_ context.Context,
	req *tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	f.requests = append(f.requests, req)
	codes := []rules.TariffChargeCode{{Code: "FSC", Active: true}}
	return &rules.TariffContext{
		Libraries: req.Libraries,
		Codes:     &codes,
		Settings:  rules.TariffSettings{DistanceUnit: "mi"},
	}, nil
}

func (*fakeContext) TableStamp(context.Context, pagination.TenantInfo) (string, error) {
	return "", nil
}

func (f *fakeContext) LoadTables(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]string,
) error {
	return nil
}

func (f *fakeContext) LoadCells(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]rules.TariffNeed,
) error {
	return nil
}

type fakeModule struct {
	checks   []*rules.TariffCheckRequest
	check    *rules.TariffCheck
	tests    []*rules.TariffTestRequest
	run      *rules.TariffTestRun
	catalogs int
	version  string
}

func (f *fakeModule) CheckTariff(
	_ context.Context,
	req *rules.TariffCheckRequest,
) (*rules.TariffCheck, error) {
	f.checks = append(f.checks, req)
	return f.check, nil
}

func (f *fakeModule) TestTariff(
	_ context.Context,
	req *rules.TariffTestRequest,
) (*rules.TariffTestRun, error) {
	f.tests = append(f.tests, req)
	return f.run, nil
}

func (f *fakeModule) ScaffoldTariffTest(
	_ context.Context,
	req *rules.TariffScaffoldRequest,
) (*rules.TariffScaffold, error) {
	block := "test \"" + req.Name + "\" {\n}\n"
	return &rules.TariffScaffold{Test: &block}, nil
}

func (f *fakeModule) SetTariffName(
	ctx context.Context,
	source, name string,
) (*rules.TariffNamed, error) {
	return rules.SetTariffName(ctx, source, name)
}

func (f *fakeModule) TariffCatalog(context.Context) (rules.RawJSON, error) {
	f.catalogs++
	return rules.RawJSON(`{"modules":[]}`), nil
}

func (f *fakeModule) Version(context.Context) (rules.VersionInfo, error) {
	return rules.VersionInfo{Version: f.version}, nil
}

func TestCheckSource_ChecksWithTheLatestApprovedLibrariesAndPinsThem(t *testing.T) {
	t.Parallel()

	accessorials := libraryVersion("accessorials", 2, "library \"org.accessorials\"\n")
	libraries := &fakeLibraryVersions{approved: map[string]*formulalibrary.FormulaLibraryVersion{
		"accessorials": accessorials,
	}}
	module := &fakeModule{check: &rules.TariffCheck{
		OK:      true,
		Imports: []rules.TariffImport{{Path: "org.accessorials", Version: accessorials.ID.String()}},
	}}
	tariffCtx := &fakeContext{}
	checker := &Checker{
		resolver: &TemplateResolver{libraries: libraryLoader{versions: libraries}},
		context:  tariffCtx,
		module:   module,
	}

	source := "tariff \"x\"\nuse org.accessorials.{ hazmat }\n"
	checked, err := checker.CheckSource(t.Context(), &SourceCheckRequest{
		TenantInfo: testTenant,
		Source:     source,
		Kind:       rules.TariffKindTariff,
	})
	require.NoError(t, err)

	require.Len(t, module.checks, 1)
	sent := module.checks[0]
	assert.Equal(t, source, sent.Source)
	assert.Equal(t, rules.TariffKindTariff, sent.Kind)
	require.Len(t, sent.Context.Libraries, 1)
	assert.Equal(t, "org.accessorials", sent.Context.Libraries[0].Path)
	assert.Equal(t, testTenant, tariffCtx.requests[0].TenantInfo)
	assert.True(t, checked.OK())
	assert.Equal(t, []tarifftypes.Import{accessorials.Pin()}, checked.Imports)
}

func TestCheckSource_PinnedModeChecksAgainstThePinnedVersions(t *testing.T) {
	t.Parallel()

	pinned := libraryVersion("accessorials", 1, "library \"org.accessorials\" // v1\n")
	libraries := &fakeLibraryVersions{pinned: []*formulalibrary.FormulaLibraryVersion{pinned}}
	module := &fakeModule{check: &rules.TariffCheck{OK: true}}
	checker := &Checker{
		resolver: &TemplateResolver{libraries: libraryLoader{versions: libraries}},
		context:  &fakeContext{},
		module:   module,
	}

	_, err := checker.CheckSource(t.Context(), &SourceCheckRequest{
		TenantInfo: testTenant,
		Source:     "tariff \"x\"\nuse org.accessorials\n",
		Kind:       rules.TariffKindTariff,
		Mode:       LibrariesPinned,
		Pins:       []tarifftypes.Import{pinned.Pin()},
	})
	require.NoError(t, err)
	assert.Empty(t, libraries.nameCalls, "a pinned check never asks for the latest version")
	assert.Equal(t, pinned.Source, module.checks[0].Context.Libraries[0].Source)
}

func TestCheck_UsesTheLibrariesTheCallerPinned(t *testing.T) {
	t.Parallel()

	libraries := &fakeLibraryVersions{}
	module := &fakeModule{check: &rules.TariffCheck{OK: true}}
	checker := &Checker{
		resolver: &TemplateResolver{libraries: libraryLoader{versions: libraries}},
		context:  &fakeContext{},
		module:   module,
	}
	values := &rules.TariffInputValues{Rule: map[string]rules.RawJSON{
		"rate": rules.QuantityValue(mustDecimal(t, "2.35"), "$/mi"),
	}}

	_, err := checker.Check(t.Context(), &tariffports.CheckRequest{
		TenantInfo: testTenant,
		Source:     "tariff \"x\"\nuse org.accessorials\n",
		Kind:       rules.TariffKindTariff,
		Libraries:  []rules.TariffLibrary{{Path: "org.accessorials", Source: "library \"a\"\n"}},
		Values:     values,
	})
	require.NoError(t, err)
	assert.Empty(t, libraries.nameCalls)
	assert.Same(t, values, module.checks[0].Values)
	assert.Equal(t, "library \"a\"\n", module.checks[0].Context.Libraries[0].Source)
}

func TestTestChecked_SkipsTestsOfASourceThatDoesNotCheck(t *testing.T) {
	t.Parallel()

	module := &fakeModule{run: &rules.TariffTestRun{}}
	checker := &Checker{module: module}

	run, err := checker.TestChecked(t.Context(), &SourceCheck{
		Check: &rules.TariffCheck{OK: false},
	}, "tariff \"x\"\n")
	require.NoError(t, err)
	assert.Empty(t, run.Tests)
	assert.Empty(t, module.tests)

	run, err = checker.TestChecked(t.Context(), &SourceCheck{
		Check:   &rules.TariffCheck{OK: true},
		Context: &rules.TariffContext{},
	}, "tariff \"x\"\n")
	require.NoError(t, err)
	assert.NotNil(t, run.Tests, "an empty run is [] not null")
	require.Len(t, module.tests, 1)
}

func TestCatalog_IsCachedPerModuleVersion(t *testing.T) {
	t.Parallel()

	module := &fakeModule{version: "1.0.0"}
	cache := newCatalogCache(module)

	first, err := cache.get(t.Context())
	require.NoError(t, err)
	_, err = cache.get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, module.catalogs)
	assert.Equal(t, "1.0.0", first.ModuleVersion)

	module.version = "1.1.0"
	second, err := cache.get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, module.catalogs, "a new module version reads the catalog again")
	assert.Equal(t, "1.1.0", second.ModuleVersion)
}

func mustDecimal(t *testing.T, text string) decimal.Decimal {
	t.Helper()
	value, err := decimal.NewFromString(text)
	require.NoError(t, err)
	return value
}
