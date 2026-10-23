package formulaassistantservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const perMileSource = "tariff \"Per Mile\"\n\ninput rate: Money/mi\n\n" +
	"charge \"Per Mile\" (quantity: shipment.distance, rate: rate)\n"

type fakeTemplates struct {
	result   *formulatemplateservice.TestSourceResult
	preview  *formulatemplateports.PreviewResult
	tested   []*formulatemplateservice.TestSourceRequest
	previews []*formulatemplateservice.PreviewRequest
}

func (f *fakeTemplates) StudioContext(
	context.Context,
	pagination.TenantInfo,
) (*formulatemplateservice.StudioContext, error) {
	return &formulatemplateservice.StudioContext{
		Context: &rules.TariffContext{
			Libraries: []rules.TariffLibrary{{Path: "org.accessorials", Source: "library \"org.accessorials\"\n"}},
		},
		Libraries: []formulatemplateservice.StudioLibrary{{Path: "org.accessorials", Name: "accessorials"}},
	}, nil
}

func (f *fakeTemplates) TestSource(
	_ context.Context,
	req *formulatemplateservice.TestSourceRequest,
) (*formulatemplateservice.TestSourceResult, error) {
	f.tested = append(f.tested, req)
	return f.result, nil
}

func (f *fakeTemplates) Preview(
	_ context.Context,
	req *formulatemplateservice.PreviewRequest,
) (*formulatemplateports.PreviewResult, error) {
	f.previews = append(f.previews, req)
	return f.preview, nil
}

type fakeLibraries struct {
	result *formulalibraryservice.TestSourceResult
	tested []*formulalibraryservice.TestSourceRequest
}

func (f *fakeLibraries) TestSource(
	_ context.Context,
	req *formulalibraryservice.TestSourceRequest,
) (*formulalibraryservice.TestSourceResult, error) {
	f.tested = append(f.tested, req)
	return f.result, nil
}

type fakeCatalog struct{}

func (fakeCatalog) Catalog(context.Context) (*tariff.CatalogEntry, error) {
	return &tariff.CatalogEntry{ModuleVersion: "1.2.3", Catalog: rules.RawJSON(`{"types":[]}`)}, nil
}

func passing(names ...string) *rules.TariffTestRun {
	run := &rules.TariffTestRun{Tests: []rules.TariffTestResult{}}
	for _, name := range names {
		run.Tests = append(run.Tests, rules.TariffTestResult{
			Name:   name,
			Status: rules.TariffTestPassed,
			Range:  rules.TariffRange{From: 0, To: 4},
		})
	}
	return run
}

func cleanCheck() *rules.TariffCheck {
	return &rules.TariffCheck{
		OK:     true,
		Header: &rules.TariffHeader{Kind: rules.TariffKindTariff, Name: "Per Mile"},
		Inputs: []rules.TariffInput{{Name: "rate", Type: "Money/mi", Required: true}},
		Diagnostics: []rules.TariffDiagnostic{{
			Severity: rules.SeverityWarning,
			Code:     "unused",
			Message:  rules.Message{Template: "`miles` is never used"},
			Range:    &rules.TariffRange{From: 19, To: 24},
		}},
		Charges: []rules.TariffChargeRef{{Label: "Per Mile"}},
	}
}

type fixture struct {
	svc       *Service
	templates *fakeTemplates
	libraries *fakeLibraries
	tenant    pagination.TenantInfo
}

func newFixture() *fixture {
	templates := &fakeTemplates{result: &formulatemplateservice.TestSourceResult{
		Check:   cleanCheck(),
		Imports: []tarifftypes.Import{{Path: "org.accessorials", VersionNumber: 2}},
		Tests:   passing("a short haul"),
	}}
	libraries := &fakeLibraries{result: &formulalibraryservice.TestSourceResult{
		Check: &rules.TariffCheck{
			OK:     true,
			Header: &rules.TariffHeader{Kind: rules.TariffKindLibrary, Name: "org.accessorials"},
		},
		Imports: []tarifftypes.Import{},
		Tests:   passing("hazmat adds the per-mile surcharge"),
	}}
	return &fixture{
		svc:       &Service{templates: templates, libraries: libraries, catalog: fakeCatalog{}},
		templates: templates,
		libraries: libraries,
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
}

func TestLanguage_JoinsTheCatalogAndTheOrganisationsContext(t *testing.T) {
	t.Parallel()

	f := newFixture()
	language, err := f.svc.Language(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", language.ModuleVersion)
	assert.JSONEq(t, `{"types":[]}`, string(language.Catalog))
	require.Len(t, language.Libraries, 1)
	assert.Equal(t, "org.accessorials", language.Context.Libraries[0].Path)
}

func TestCheck_ReadsTemplatesAndLibrariesWithTheirOwnKind(t *testing.T) {
	t.Parallel()

	f := newFixture()
	checked, err := f.svc.Check(t.Context(), &CheckRequest{
		TenantInfo: f.tenant,
		Kind:       KindTemplate,
		Source:     perMileSource,
	})
	require.NoError(t, err)
	assert.True(t, checked.OK)
	assert.True(t, checked.Passing())
	assert.Equal(t, "tariff", checked.Kind)
	assert.Equal(t, []string{"org.accessorials"}, checked.Imports)
	assert.Equal(t, []string{"Per Mile"}, checked.Charges)
	require.Len(t, checked.Inputs, 1)
	assert.Equal(t, "Money/mi", checked.Inputs[0].Type)
	require.Len(t, checked.Findings, 1)
	assert.Equal(t, &tariff.SourcePosition{Line: 3, Column: 1}, checked.Findings[0].At)
	assert.Equal(t, "input", checked.Findings[0].Excerpt)
	assert.Equal(t, 1, checked.Tests.Passed)
	require.Len(t, f.templates.tested, 1)
	assert.Empty(t, f.libraries.tested)

	library, err := f.svc.Check(t.Context(), &CheckRequest{
		TenantInfo: f.tenant,
		Kind:       KindLibrary,
		Source:     "library \"org.accessorials\"\n",
	})
	require.NoError(t, err)
	assert.Equal(t, "library", library.Kind)
	require.Len(t, f.libraries.tested, 1)
	assert.Equal(t, f.tenant, f.libraries.tested[0].TenantInfo)
}

func TestCheck_RefusesAnUnknownKindOrAnEmptySource(t *testing.T) {
	t.Parallel()

	f := newFixture()
	_, err := f.svc.Check(t.Context(), &CheckRequest{Kind: "condition", Source: perMileSource})
	assert.True(t, errortypes.IsError(err))
	_, err = f.svc.Check(t.Context(), &CheckRequest{Kind: KindTemplate, Source: "  "})
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, f.templates.tested)
}

func TestPreview_ReportsTheBillOrWhereItStopped(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shipmentID := pulid.MustNew("shp_")
	f.templates.preview = &formulatemplateports.PreviewResult{
		Valid:  true,
		Inputs: []tarifftypes.InputDecl{{Name: "rate", Type: "Money/mi", Required: true}},
		Bill:   &tarifftypes.Bill{Total: decimal.NewFromInt(865)},
		Tests:  passing("a short haul").Tests,
	}

	preview, err := f.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: f.tenant,
		Source:     perMileSource,
		ShipmentID: shipmentID,
	})
	require.NoError(t, err)
	assert.True(t, preview.OK)
	require.NotNil(t, preview.Bill)
	assert.Equal(t, "865", preview.Bill.Total.String())
	require.Len(t, f.templates.previews, 1)
	assert.Equal(t, shipmentID, *f.templates.previews[0].ShipmentID)

	f.templates.preview = &formulatemplateports.PreviewResult{
		Valid: true,
		Refusal: &rules.TariffRefusal{
			Text:  "This rate needs the trip's miles",
			Range: &rules.TariffRange{From: 19, To: 30},
		},
		Tests: []rules.TariffTestResult{},
	}
	preview, err = f.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: f.tenant,
		Source:     perMileSource,
	})
	require.NoError(t, err)
	assert.Equal(t, "This rate needs the trip's miles", preview.Refusal)
	assert.Equal(t, &tariff.SourcePosition{Line: 3, Column: 1}, preview.RefusalAt)
	assert.Nil(t, f.templates.previews[1].ShipmentID, "no shipment rates the sample record")
}

func TestPropose_ShowsOnlySourceThatChecksAndPassesItsTests(t *testing.T) {
	t.Parallel()

	f := newFixture()
	proposed, err := f.svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo:  f.tenant,
		Kind:        KindTemplate,
		Name:        " Per Mile ",
		Source:      perMileSource,
		Explanation: " Bills the miles at the contract rate. ",
	})
	require.NoError(t, err)
	require.NotNil(t, proposed.Proposal)
	assert.Equal(t, pagedraft.SourceProposal{
		Name:        "Per Mile",
		Source:      perMileSource,
		Explanation: "Bills the miles at the contract rate.",
		Inputs:      []string{"rate"},
		Imports:     []string{"org.accessorials"},
		Charges:     []string{"Per Mile"},
		Warnings:    []string{"`miles` is never used"},
		Tests:       pagedraft.ProposalTests{Total: 1, Passed: 1},
	}, *proposed.Proposal)

	f.templates.result.Tests = &rules.TariffTestRun{Tests: []rules.TariffTestResult{{
		Name:   "a short haul",
		Status: rules.TariffTestFailed,
	}}}
	proposed, err = f.svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo:  f.tenant,
		Kind:        KindTemplate,
		Source:      perMileSource,
		Explanation: "Bills the miles.",
	})
	require.NoError(t, err)
	assert.Nil(t, proposed.Proposal, "a failing test keeps the proposal off the page")
	assert.False(t, proposed.Checked.Passing())

	f.templates.result.Tests = passing()
	proposed, err = f.svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo:  f.tenant,
		Kind:        KindTemplate,
		Source:      perMileSource,
		Explanation: "Bills the miles.",
	})
	require.NoError(t, err)
	assert.Nil(t, proposed.Proposal, "a proposal without a test is not shown")

	f.templates.result.Check = &rules.TariffCheck{OK: false, Diagnostics: []rules.TariffDiagnostic{{
		Severity: rules.SeverityError,
		Code:     "unknown-name",
		Message:  rules.Message{Template: "There is no `ratee`"},
	}}}
	f.templates.result.Tests = passing("a short haul")
	proposed, err = f.svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo:  f.tenant,
		Kind:        KindTemplate,
		Source:      perMileSource,
		Explanation: "Bills the miles.",
	})
	require.NoError(t, err)
	assert.Nil(t, proposed.Proposal)
	require.Len(t, proposed.Checked.Findings, 1)
	assert.Equal(t, "There is no `ratee`", proposed.Checked.Findings[0].Message)
}

func TestPropose_BoundsWhatTheModelWrites(t *testing.T) {
	t.Parallel()

	f := newFixture()
	_, err := f.svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo: f.tenant,
		Kind:       KindTemplate,
		Source:     perMileSource,
	})
	require.Error(t, err, "an explanation is required")

	_, err = f.svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo:  f.tenant,
		Kind:        KindLibrary,
		Name:        strings.Repeat("n", pagedraft.MaxLibraryNameLength+1),
		Description: strings.Repeat("d", pagedraft.MaxDescriptionLength+1),
		Source:      "library \"org.accessorials\"\n",
		Explanation: strings.Repeat("e", MaxExplanationRunes+1),
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Len(t, multiErr.Errors, 3)
	assert.Empty(t, f.libraries.tested, "nothing runs for a request out of bounds")
}

func TestPreview_ReportsEachChoiceAndTheInputsBounds(t *testing.T) {
	t.Parallel()

	f := newFixture()
	chosen := "Flat"
	bounds := &rules.TariffConstraint{Range: &rules.TariffRangeConstraint{
		Min:          rules.QuantityValue(decimal.RequireFromString("0.5"), "$/mi"),
		Max:          rules.QuantityValue(decimal.NewFromInt(15), "$/mi"),
		MaxInclusive: true,
	}}
	f.templates.preview = &formulatemplateports.PreviewResult{
		Valid: true,
		Inputs: []tarifftypes.InputDecl{
			{Name: "rate", Type: "Money/mi", Required: true, Constraint: bounds},
		},
		Bill: &tarifftypes.Bill{Currency: "USD", Total: decimal.NewFromInt(900)},
		Choices: []rules.TariffChoice{{
			Label:  "Linehaul",
			Kind:   rules.TariffChoiceBest,
			Chosen: &chosen,
			Options: []rules.TariffChoiceOption{
				{
					Label:    "Per mile",
					Status:   rules.TariffOptionLost,
					Subtotal: rules.MoneyValue(decimal.NewFromInt(1175)),
				},
				{
					Label:    "Flat",
					Status:   rules.TariffOptionChosen,
					Subtotal: rules.MoneyValue(decimal.NewFromInt(900)),
				},
			},
		}},
		Tests: passing("a short haul").Tests,
	}

	preview, err := f.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: f.tenant,
		Source:     perMileSource,
	})
	require.NoError(t, err)
	require.Len(t, preview.Inputs, 1)
	assert.Same(t, bounds, preview.Inputs[0].Constraint)
	require.Len(t, preview.Choices, 1)
	assert.Equal(t, "USD", preview.Choices[0].Currency)
	require.NotNil(t, preview.Choices[0].Chosen)
	assert.Equal(t, "Flat", *preview.Choices[0].Chosen)
	assert.Equal(t, "1175", preview.Choices[0].Options[0].Subtotal.String())
}

func TestCheck_ReportsAnInputsConstraint(t *testing.T) {
	t.Parallel()

	f := newFixture()
	bounds := &rules.TariffConstraint{OneOf: []rules.RawJSON{rules.TextValue("Standard")}}
	f.templates.result.Check.Inputs[0].Constraint = bounds

	checked, err := f.svc.Check(t.Context(), &CheckRequest{
		TenantInfo: f.tenant,
		Kind:       KindTemplate,
		Source:     perMileSource,
	})
	require.NoError(t, err)
	require.Len(t, checked.Inputs, 1)
	assert.Same(t, bounds, checked.Inputs[0].Constraint)
}
