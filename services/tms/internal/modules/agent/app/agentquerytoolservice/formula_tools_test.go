package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const toolPerMileSource = "tariff \"Per Mile\"\n\ninput rate: Money/mi\n\n" +
	"charge \"Per Mile\" (quantity: shipment.distance, rate: rate)\n"

type fakeWorkbench struct {
	tenant   pagination.TenantInfo
	checked  *formulaassistantservice.CheckRequest
	previews *formulaassistantservice.PreviewRequest
	proposed *formulaassistantservice.ProposeRequest
	outcome  *formulaassistantservice.Checked
	proposal *pagedraft.SourceProposal
}

func (f *fakeWorkbench) Language(
	_ context.Context,
	tenant pagination.TenantInfo,
) (*formulaassistantservice.Language, error) {
	f.tenant = tenant
	return &formulaassistantservice.Language{
		Catalog: rules.RawJSON(testCatalog),
		Context: &rules.TariffContext{
			Tables: []rules.TariffTable{{
				Code:      "ZONES",
				Name:      "Zone rates",
				ValueType: "Money/cwt",
				Axes:      []rules.TariffAxis{{Match: "exact", Kind: "zip3", Label: "Origin"}},
			}},
			Libraries: []rules.TariffLibrary{{
				Path:   "org.accessorials",
				Source: "library \"org.accessorials\"\n\nfn hazmat(miles: Distance) -> Money = $1\n",
			}},
			Codes: &[]rules.TariffChargeCode{
				{Code: "FSC", Description: "Fuel surcharge", Active: true},
				{Code: "OLD", Description: "Retired", Active: false},
			},
			Settings: rules.TariffSettings{DistanceUnit: "mi"},
		},
		Libraries: []formulatemplateservice.StudioLibrary{{
			Path:          "org.accessorials",
			Name:          "accessorials",
			Description:   "Hazmat and residential fees",
			VersionNumber: 2,
		}},
	}, nil
}

func (f *fakeWorkbench) outcomeOrDefault() *formulaassistantservice.Checked {
	if f.outcome != nil {
		return f.outcome
	}
	return &formulaassistantservice.Checked{
		OK:       true,
		Kind:     "tariff",
		Name:     "Per Mile",
		Findings: []tariff.Finding{},
		Inputs:   []formulaassistantservice.Input{{Name: "rate", Type: "Money/mi", Required: true}},
		Imports:  []string{},
		Charges:  []string{"Per Mile"},
		Tests:    tariff.TestSummary{Total: 1, Passed: 1, Tests: []tariff.TestOutcome{}},
	}
}

func (f *fakeWorkbench) Check(
	_ context.Context,
	req *formulaassistantservice.CheckRequest,
) (*formulaassistantservice.Checked, error) {
	f.checked = req
	return f.outcomeOrDefault(), nil
}

func (f *fakeWorkbench) Preview(
	_ context.Context,
	req *formulaassistantservice.PreviewRequest,
) (*formulaassistantservice.Preview, error) {
	f.previews = req
	return &formulaassistantservice.Preview{
		Checked: f.outcomeOrDefault(),
		Bill:    &tarifftypes.Bill{Total: decimal.NewFromInt(865)},
	}, nil
}

func (f *fakeWorkbench) Propose(
	_ context.Context,
	req *formulaassistantservice.ProposeRequest,
) (*formulaassistantservice.Proposed, error) {
	f.proposed = req
	return &formulaassistantservice.Proposed{Checked: f.outcomeOrDefault(), Proposal: f.proposal}, nil
}

type authorPermissions struct {
	permissionports.PermissionEngine

	resource   permission.Resource
	operations []permission.Operation
}

func (p *authorPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	for _, operation := range p.operations {
		if req.Resource == p.resource.String() && req.Operation == operation {
			return &permissionports.PermissionCheckResult{Allowed: true}, nil
		}
	}

	return &permissionports.PermissionCheckResult{Allowed: false, Reason: "not permitted"}, nil
}

func TestFormulaTools_Policies(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	describe := (&describeFormulaLanguageTool{workbench: workbench}).Policy()
	assert.Equal(t, permission.ResourceFormulaTemplate, describe.Resource)
	assert.Equal(t, agent.ToolEffectLookup, describe.Effect)

	for _, policy := range []string{
		(&checkFormulaTool{workbench: workbench}).Policy().Name,
		(&runFormulaTestsTool{workbench: workbench}).Policy().Name,
	} {
		assert.NotEmpty(t, policy)
	}
	preview := (&previewFormulaTool{workbench: workbench}).Policy()
	assert.Equal(t, agent.ToolKindQuery, preview.Kind)

	formula := newProposeSourceTool(proposeFormulaSpec, workbench, &fakePermissions{})
	assert.Equal(t, agent.ToolScopeSelf, formula.Policy().Scope)
	assert.Equal(t, agent.ToolEffectPresent, formula.Policy().Effect)
	assert.Equal(t, permission.ResourceFormulaTemplate, formula.Policy().Resource)
	assert.True(t, pagedraft.IsEditTool(formula.Name()))

	library := newProposeSourceTool(proposeLibrarySpec, workbench, &fakePermissions{})
	assert.Equal(t, "formula_library_propose", library.Name())
	assert.Equal(t, permission.ResourceFormulaLibrary, library.Policy().Resource)
	assert.True(t, pagedraft.IsEditTool(library.Name()))
}

func TestDescribeFormulaLanguage_ListsEverythingShortenedFromTheCallersTenant(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	params := chatParams(map[string]any{}, "")

	result, err := (&describeFormulaLanguageTool{workbench: workbench}).Query(t.Context(), params)
	require.NoError(t, err)

	view, ok := result.(*formulaLanguageView)
	require.True(t, ok)
	assert.Equal(t, params.OrganizationID, workbench.tenant.OrgID)
	assert.Contains(t, lineNames(view.Record), "shipment.distance")
	assert.Contains(t, lineNames(view.Record), "shipment.temperature.min")
	assert.Contains(t, lineNames(view.Record), "Stop.dwell")
	assert.Equal(t, "Distance", lineOf(view.Record, "shipment.distance").Type)
	assert.Equal(t, "The trip's distance.", lineOf(view.Record, "shipment.distance").Doc,
		"the full list keeps the first sentence")
	assert.Equal(t, []string{"mi", "km"}, view.Units["Distance"])
	require.Len(t, view.Tables, 1)
	assert.Equal(t, []string{"Origin: exact (zip3)"}, view.Tables[0].Axes)
	assert.Equal(t, []string{"FSC: Fuel surcharge"}, view.Codes)
	require.Len(t, view.Libraries, 1)
	assert.Empty(t, view.Libraries[0].Source, "the full list leaves sources out")
	require.Len(t, view.Modules, 1)
	assert.Equal(t, "fn stepPercent(price: Money/gal) -> Percent",
		view.Modules[0].Declarations[0].Name)
	assert.Equal(t, "mi", view.DistanceUnit)
}

func TestDescribeFormulaLanguage_FindsWhatPeopleCallIt(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	result, err := (&describeFormulaLanguageTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{"query": "reefer"}, ""))
	require.NoError(t, err)
	view := result.(*formulaLanguageView)
	assert.Contains(t, lineNames(view.Record), "shipment.temperature.min")
	assert.NotContains(t, lineNames(view.Record), "shipment.distance")
	assert.Equal(t, "The coldest the load may ride. Set on reefer loads only.",
		lineOf(view.Record, "shipment.temperature.min").Doc, "a match is described in full")

	result, err = (&describeFormulaLanguageTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{"query": "accessorials"}, ""))
	require.NoError(t, err)
	view = result.(*formulaLanguageView)
	require.Len(t, view.Libraries, 1)
	assert.Contains(t, view.Libraries[0].Source, "fn hazmat")

	result, err = (&describeFormulaLanguageTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{"query": "xyzzy"}, ""))
	require.NoError(t, err)
	assert.Contains(t, result.(*formulaLanguageView).Note, "an input the formula declares")
}

func TestCheckAndRunTests_PassTheWholeSourceAndItsKind(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	source := "  " + toolPerMileSource
	result, err := (&checkFormulaTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{"source": source}, ""))
	require.NoError(t, err)
	assert.True(t, result.(checkedFormula).OK)
	assert.Equal(t, source, workbench.checked.Source, "the source is passed as written")
	assert.Equal(t, formulaassistantservice.KindTemplate, workbench.checked.Kind)

	_, err = (&runFormulaTestsTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{"source": "library \"org.x\"\n", "kind": "library"}, ""))
	require.NoError(t, err)
	assert.Equal(t, formulaassistantservice.KindLibrary, workbench.checked.Kind)

	_, err = (&checkFormulaTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{"source": toolPerMileSource, "kind": "condition"}, ""))
	require.Error(t, err)
	_, err = (&checkFormulaTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{}, ""))
	require.Error(t, err)

	workbench.outcome = &formulaassistantservice.Checked{
		Findings: []tariff.Finding{{Severity: "error", Message: "There is no `ratee`"}},
	}
	tested, err := (&runFormulaTestsTool{workbench: workbench}).Query(t.Context(),
		chatParams(map[string]any{"source": toolPerMileSource}, ""))
	require.NoError(t, err)
	assert.Contains(t, tested.(testedFormula).Note, "no test ran")
	assert.NotNil(t, tested.(testedFormula).Findings)
}

func TestPreviewFormula_RatesAShipmentOnlyForAPersonWhoMayReadIt(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	workbench := &fakeWorkbench{}

	allowed := &previewFormulaTool{
		workbench: workbench,
		access:    newFieldAccess(&fakePermissions{allowed: true}),
	}
	result, err := allowed.Query(t.Context(), chatParams(map[string]any{
		"source":     toolPerMileSource,
		"shipmentId": shipmentID.String(),
		"inputs":     map[string]any{"rate": map[string]any{"$q": "2.10", "u": "$/mi"}},
	}, ""))
	require.NoError(t, err)
	previewed := result.(previewedFormula)
	assert.Equal(t, shipmentID.String(), previewed.ShipmentID)
	assert.Equal(t, "865", previewed.Bill.Total.String())
	assert.Equal(t, shipmentID, workbench.previews.ShipmentID)
	assert.JSONEq(t, `{"$q":"2.10","u":"$/mi"}`, string(workbench.previews.Inputs["rate"]))

	denied := &previewFormulaTool{
		workbench: &fakeWorkbench{},
		access:    newFieldAccess(&fakePermissions{allowed: false}),
	}
	_, err = denied.Query(t.Context(), chatParams(map[string]any{
		"source":     toolPerMileSource,
		"shipmentId": shipmentID.String(),
	}, ""))
	require.ErrorContains(t, err, "may not read shipments")

	_, err = allowed.Query(t.Context(), chatParams(map[string]any{
		"source":     toolPerMileSource,
		"shipmentId": "not-an-id",
	}, ""))
	require.Error(t, err)
}

func TestProposeFormula_HandsACheckedProposalToThePage(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{proposal: &pagedraft.SourceProposal{
		Source:      toolPerMileSource,
		Explanation: "Bills the miles.",
		Tests:       pagedraft.ProposalTests{Total: 1, Passed: 1},
	}}
	tool := newProposeSourceTool(proposeFormulaSpec, workbench, &authorPermissions{
		resource:   permission.ResourceFormulaTemplate,
		operations: []permission.Operation{permission.OpUpdate},
	})

	result, err := tool.Query(t.Context(), chatParams(map[string]any{
		"source":      toolPerMileSource,
		"explanation": "Bills the miles.",
		"name":        "Per Mile",
	}, ""))
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	assert.Equal(t, pagedraft.SurfaceFormula, edit.Surface)
	assert.Equal(t, pagedraft.ActionProposeFormula, edit.Action)
	require.NotNil(t, edit.Formula)
	assert.Nil(t, edit.Library)
	assert.Equal(t, formulaassistantservice.KindTemplate, workbench.proposed.Kind)
	assert.Equal(t, "Per Mile", workbench.proposed.Name)
	assert.Contains(t, result.(pagedraft.EditResult).Note, pricedByEngineNote)
	assert.Contains(t, tool.Description(), "Never describe a change without making it.")
}

func TestProposeLibrary_LandsInTheLibraryEditor(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{proposal: &pagedraft.SourceProposal{
		Source: "library \"org.accessorials\"\n",
		Tests:  pagedraft.ProposalTests{Total: 2, Passed: 2},
	}}
	tool := newProposeSourceTool(proposeLibrarySpec, workbench, &authorPermissions{
		resource:   permission.ResourceFormulaLibrary,
		operations: []permission.Operation{permission.OpCreate},
	})

	result, err := tool.Query(t.Context(), chatParams(map[string]any{
		"source":      "library \"org.accessorials\"\n",
		"explanation": "Adds a liftgate fee.",
	}, ""))
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	assert.Equal(t, pagedraft.SurfaceFormulaLibrary, edit.Surface)
	assert.Equal(t, pagedraft.ActionProposeLibrary, edit.Action)
	require.NotNil(t, edit.Library)
	assert.Nil(t, edit.Formula)
	assert.Equal(t, formulaassistantservice.KindLibrary, workbench.proposed.Kind)
}

func TestProposeFormula_TellsTheModelWhyNothingWasShown(t *testing.T) {
	t.Parallel()

	author := &authorPermissions{
		resource:   permission.ResourceFormulaTemplate,
		operations: []permission.Operation{permission.OpCreate},
	}
	params := chatParams(map[string]any{
		"source":      toolPerMileSource,
		"explanation": "Bills the miles.",
	}, "")

	failing := &fakeWorkbench{outcome: &formulaassistantservice.Checked{
		Findings: []tariff.Finding{{Severity: "error", Message: "There is no `ratee`"}},
	}}
	result, err := newProposeSourceTool(proposeFormulaSpec, failing, author).Query(t.Context(), params)
	require.NoError(t, err)
	refused := result.(refusedProposal)
	assert.False(t, refused.Applied)
	assert.Contains(t, refused.Note, "does not check")
	assert.NotNil(t, refused.Findings)

	untested := &fakeWorkbench{outcome: &formulaassistantservice.Checked{OK: true}}
	result, err = newProposeSourceTool(proposeFormulaSpec, untested, author).Query(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, result.(refusedProposal).Note, "no tests")

	red := &fakeWorkbench{outcome: &formulaassistantservice.Checked{
		OK:    true,
		Tests: tariff.TestSummary{Total: 2, Passed: 1},
	}}
	result, err = newProposeSourceTool(proposeFormulaSpec, red, author).Query(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, result.(refusedProposal).Note, "a test fails")
}

func TestProposeFormula_RefusesAPersonWhoMayOnlyRead(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	tool := newProposeSourceTool(proposeFormulaSpec, workbench, &authorPermissions{
		resource:   permission.ResourceFormulaTemplate,
		operations: []permission.Operation{permission.OpRead},
	})

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"source":      toolPerMileSource,
		"explanation": "Two dollars a mile.",
	}, ""))
	require.ErrorContains(t, err, "may read but not write a formula template")
	assert.Nil(t, workbench.proposed)

	library := newProposeSourceTool(proposeLibrarySpec, workbench, &authorPermissions{
		resource:   permission.ResourceFormulaTemplate,
		operations: []permission.Operation{permission.OpCreate},
	})
	_, err = library.Query(t.Context(), chatParams(map[string]any{
		"source":      "library \"org.x\"\n",
		"explanation": "A library.",
	}, ""))
	require.ErrorContains(t, err, "formula library", "a template author is not a library author")
}

type stubFormulaTemplates struct {
	formulatemplateports.FormulaTemplateRepository

	entity *formulatemplate.FormulaTemplate
	last   formulatemplateports.GetFormulaTemplateByIDRequest
}

func (s *stubFormulaTemplates) GetByID(
	_ context.Context,
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	s.last = req

	return s.entity, nil
}

func TestGetFormulaTemplate_ReturnsTheSourceWholeFromTheCallersTenant(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ft_")
	repo := &stubFormulaTemplates{entity: &formulatemplate.FormulaTemplate{
		ID:     id,
		Name:   "Per Mile",
		Source: toolPerMileSource,
		Status: formulatemplate.StatusActive,
	}}
	tool := newGetFormulaTemplateTool(repo, &fakePermissions{allowed: true})
	params := testParams(map[string]any{"formulaTemplateId": id.String()})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, id, repo.last.TemplateID)
	assert.Equal(t, params.OrganizationID, repo.last.TenantInfo.OrgID)
	document := encodedDocument(t, result)
	assert.Equal(t, "Per Mile", document["name"])
	assert.Equal(t, toolPerMileSource, document["source"])

	_, err = tool.Query(t.Context(), testParams(map[string]any{"formulaTemplateId": "Per Mile"}))
	require.Error(t, err)
}

type stubFormulaLibraries struct {
	formulalibraryports.FormulaLibraryRepository

	entity *formulalibrary.FormulaLibrary
	last   *formulalibraryports.GetLibraryByIDRequest
}

func (s *stubFormulaLibraries) GetByID(
	_ context.Context,
	req *formulalibraryports.GetLibraryByIDRequest,
) (*formulalibrary.FormulaLibrary, error) {
	s.last = req
	return s.entity, nil
}

func TestGetFormulaLibrary_IsAReadOfFormulaLibraries(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("fl_")
	repo := &stubFormulaLibraries{entity: &formulalibrary.FormulaLibrary{
		ID:     id,
		Name:   "accessorials",
		Source: "library \"org.accessorials\"\n",
	}}
	tool := newGetFormulaLibraryTool(repo, &fakePermissions{allowed: true})
	assert.Equal(t, permission.ResourceFormulaLibrary, tool.Policy().Resource)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"formulaLibraryId": id.String()}))
	require.NoError(t, err)
	assert.Equal(t, id, repo.last.LibraryID)
	assert.Equal(t, "library \"org.accessorials\"\n", encodedDocument(t, result)["source"])
}

func lineNames(lines []languageLine) []string {
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		names = append(names, line.Name)
	}
	return names
}

func lineOf(lines []languageLine, name string) languageLine {
	for _, line := range lines {
		if line.Name == name {
			return line
		}
	}
	return languageLine{}
}

const testCatalog = `{
  "schema": {
    "roots": [
      {"name": "shipment", "doc": "The shipment being rated.", "fields": [
        {"name": "distance", "type": "Distance", "doc": "The trip's distance. Loaded miles only.", "fields": []},
        {"name": "temperature", "type": "{ min: Temperature? }", "doc": "The reefer setting.", "fields": [
          {"name": "min", "type": "Temperature?", "doc": "The coldest the load may ride. Set on reefer loads only.", "fields": []}
        ]}
      ]},
      {"name": "tables", "type": "tables", "doc": "The organisation's rate matrices.", "fields": []}
    ],
    "records": [
      {"name": "Stop", "fields": [{"name": "dwell", "type": "Duration?", "doc": "Time at the stop.", "fields": []}]}
    ]
  },
  "types": [{"name": "Money", "doc": {"message": "An amount of money."}}],
  "units": [{"dimension": "Distance", "unit": "mi"}, {"dimension": "Distance", "unit": "km"}],
  "methods": [{"name": "round", "receiver": "Number", "signature": "round(places: Int = 0) -> Number", "doc": {"message": "Rounds half away from zero."}}],
  "tableMethods": [{"name": "cell", "doc": {"message": "The matching cell."}}],
  "modules": [{"path": "std.fuel", "doc": "Fuel surcharges.", "declarations": [
    {"kind": "fn", "name": "stepPercent", "signature": "fn stepPercent(price: Money/gal) -> Percent", "doc": "A stepped fuel surcharge."}
  ]}]
}`
