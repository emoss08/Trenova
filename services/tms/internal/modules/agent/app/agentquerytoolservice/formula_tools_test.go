package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorkbench struct {
	referenced string
	tenant     pagination.TenantInfo
	tested     *formulaassistantservice.TestRequest
	proposed   *formulaassistantservice.ProposeRequest
	proposal   *pagedraft.FormulaProposal
}

func (f *fakeWorkbench) Reference(
	_ context.Context,
	tenant pagination.TenantInfo,
	schemaID string,
) (*formulaassistantservice.Reference, error) {
	f.tenant = tenant
	f.referenced = schemaID

	return &formulaassistantservice.Reference{
		SchemaID: formulaassistantservice.SchemaIDOrDefault(schemaID),
	}, nil
}

func (f *fakeWorkbench) Test(
	_ context.Context,
	req *formulaassistantservice.TestRequest,
) (*formulaassistantservice.TestOutcome, error) {
	f.tested = req

	return &formulaassistantservice.TestOutcome{
		SchemaID: formulaassistantservice.SchemaIDOrDefault(req.SchemaID),
		Check:    pagedraft.FormulaCheck{Valid: true, Result: "1250"},
	}, nil
}

func (f *fakeWorkbench) Propose(
	_ context.Context,
	req *formulaassistantservice.ProposeRequest,
) (*pagedraft.FormulaProposal, error) {
	f.proposed = req
	if f.proposal != nil {
		return f.proposal, nil
	}

	return &pagedraft.FormulaProposal{
		SchemaID:    formulaassistantservice.SchemaIDOrDefault(req.SchemaID),
		Expression:  req.Expression,
		Variables:   req.Variables,
		Explanation: req.Explanation,
		Check:       pagedraft.FormulaCheck{Valid: true, Result: "875"},
	}, nil
}

type authorPermissions struct {
	permissionports.PermissionEngine

	operations []permission.Operation
}

func (p *authorPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	for _, operation := range p.operations {
		if req.Resource == permission.ResourceFormulaTemplate.String() &&
			req.Operation == operation {
			return &permissionports.PermissionCheckResult{Allowed: true}, nil
		}
	}

	return &permissionports.PermissionCheckResult{Allowed: false, Reason: "not permitted"}, nil
}

func TestFormulaTools_Policies(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	describe := newDescribeFormulaSchemaTool(workbench).Policy()
	assert.Equal(t, permission.ResourceFormulaTemplate, describe.Resource)
	assert.Equal(t, agent.ToolEffectLookup, describe.Effect)

	test := newTestFormulaExpressionTool(workbench, &fakePermissions{}).Policy()
	assert.Equal(t, permission.ResourceFormulaTemplate, test.Resource)
	assert.Equal(t, agent.ToolKindQuery, test.Kind)

	propose := newProposeFormulaTool(workbench, &fakePermissions{}).Policy()
	assert.Equal(t, agent.ToolScopeSelf, propose.Scope)
	assert.Equal(t, agent.ToolEffectPresent, propose.Effect)
	assert.True(t, pagedraft.IsEditTool(newProposeFormulaTool(workbench, nil).Name()))
}

func TestDescribeFormulaSchema_ReadsTheCallersTenant(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	params := chatParams(map[string]any{"schemaId": "shipment"}, "")

	result, err := newDescribeFormulaSchemaTool(workbench).Query(t.Context(), params)
	require.NoError(t, err)

	view, ok := result.(*formulaSchemaView)
	require.True(t, ok)
	assert.Equal(t, "shipment", view.SchemaID)
	assert.Equal(t, params.OrganizationID, workbench.tenant.OrgID)
	assert.Equal(t, params.BusinessUnitID, workbench.tenant.BuID)
}

func TestTestFormulaExpression_PricesScenariosWithTheEngine(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	tool := newTestFormulaExpressionTool(workbench, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), chatParams(map[string]any{
		"expression": "max(totalDistance * ratePerMile, 350)",
		"variables":  map[string]any{"ratePerMile": 2.5},
		"scenarios": []any{
			map[string]any{
				"name":        "Long haul",
				"description": "A 500 mile dry van load.",
				"variables":   map[string]any{"totalDistance": 500},
			},
		},
	}, ""))
	require.NoError(t, err)

	tested := result.(testedFormula)
	assert.True(t, tested.Check.Valid)
	assert.Contains(t, tested.Note, "formula engine")
	assert.Empty(t, tested.ShipmentID)
	require.Len(t, workbench.tested.Scenarios, 1)
	assert.Equal(t, "Long haul", workbench.tested.Scenarios[0].Name)
	assert.Equal(t, 2.5, workbench.tested.Variables["ratePerMile"])
	assert.True(t, workbench.tested.ShipmentID.IsNil())
	assert.Nil(t, workbench.tested.Policy, "no policy given means the engine's default")
}

func TestTestFormulaExpression_PricesWithThePageDraftsPolicy(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	tool := newTestFormulaExpressionTool(workbench, &fakePermissions{allowed: true})

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"expression": "totalDistance * ratePerMile",
		"policy": map[string]any{
			"minCharge":         350.5,
			"maxCharge":         nil,
			"roundingMode":      "Down",
			"roundingPrecision": 0,
		},
	}, ""))
	require.NoError(t, err)

	policy := workbench.tested.Policy
	require.NotNil(t, policy)
	require.NotNil(t, policy.MinCharge)
	assert.Equal(t, "350.5", *policy.MinCharge)
	assert.Nil(t, policy.MaxCharge, "null removes the maximum")
	assert.Equal(t, "Down", policy.RoundingMode)
	require.NotNil(t, policy.RoundingPrecision)
	assert.Equal(t, 0, *policy.RoundingPrecision)

	properties := tool.ParamSchema()["properties"].(map[string]any)
	assert.Contains(t, properties, "policy")

	_, err = tool.Query(t.Context(), chatParams(map[string]any{
		"expression": "totalDistance * ratePerMile",
		"policy":     map[string]any{"minCharge": "two hundred"},
	}, ""))
	require.Error(t, err, "a policy that is not the schema's shape is refused")
}

func TestTestFormulaExpression_PricesAShipmentOnlyForAPersonWhoMayReadIt(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	workbench := &fakeWorkbench{}

	allowed := newTestFormulaExpressionTool(workbench, &fakePermissions{allowed: true})
	result, err := allowed.Query(t.Context(), chatParams(map[string]any{
		"expression": "totalDistance * 2",
		"shipmentId": shipmentID.String(),
	}, ""))
	require.NoError(t, err)
	assert.Equal(t, shipmentID.String(), result.(testedFormula).ShipmentID)
	assert.Equal(t, shipmentID, workbench.tested.ShipmentID)

	denied := newTestFormulaExpressionTool(&fakeWorkbench{}, &fakePermissions{allowed: false})
	_, err = denied.Query(t.Context(), chatParams(map[string]any{
		"expression": "totalDistance * 2",
		"shipmentId": shipmentID.String(),
	}, ""))
	require.ErrorContains(t, err, "may not read shipments")

	_, err = allowed.Query(t.Context(), chatParams(map[string]any{
		"expression": "totalDistance * 2",
		"shipmentId": "not-an-id",
	}, ""))
	require.Error(t, err)

	_, err = allowed.Query(t.Context(), chatParams(map[string]any{}, ""))
	require.Error(t, err)
}

func TestProposeFormula_HandsTheEnginesProposalToThePage(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	tool := newProposeFormulaTool(workbench,
		&authorPermissions{operations: []permission.Operation{permission.OpUpdate}})

	result, err := tool.Query(t.Context(), chatParams(map[string]any{
		"expression":  "max(totalDistance * ratePerMile, minimumCharge)",
		"explanation": "Charges per mile, never less than the minimum.",
		"variables": []any{
			map[string]any{"name": "ratePerMile", "type": "Number", "defaultValue": 2.5},
			map[string]any{"name": "minimumCharge", "type": "Number", "defaultValue": 350},
		},
		"scenarios": []any{
			map[string]any{"name": "Short", "variables": map[string]any{"totalDistance": 40}},
		},
	}, ""))
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	assert.Equal(t, pagedraft.SurfaceFormula, edit.Surface)
	assert.Equal(t, pagedraft.ActionProposeFormula, edit.Action)
	require.NotNil(t, edit.Formula)
	assert.Equal(t, "875", edit.Formula.Check.Result)
	require.Len(t, workbench.proposed.Variables, 2)
	assert.Equal(t, pagedraft.VariableNumber, workbench.proposed.Variables[0].Type)
	require.Len(t, workbench.proposed.Scenarios, 1)
	assert.Nil(t, workbench.proposed.Details.Policy, "no policy given keeps the editor's")

	edited := result.(pagedraft.EditResult)
	assert.Equal(t, "The page applied this to the editor; the person can undo it. It shows "+
		"in the page draft on your next turn. "+pricedByEngineNote, edited.Note)
	assert.Contains(t, tool.Description(), "Never describe a change without making it.")
}

func TestProposeFormula_CarriesTheChargePolicyToTheProposal(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	tool := newProposeFormulaTool(workbench,
		&authorPermissions{operations: []permission.Operation{permission.OpUpdate}})

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"expression":  "totalDistance * ratePerMile",
		"explanation": "Per mile, never less than the minimum.",
		"policy": map[string]any{
			"minCharge":         250,
			"maxCharge":         1999.99,
			"roundingMode":      "HalfEven",
			"roundingPrecision": 2,
		},
	}, ""))
	require.NoError(t, err)

	policy := workbench.proposed.Details.Policy
	require.NotNil(t, policy)
	require.NotNil(t, policy.MinCharge)
	assert.Equal(t, "250", *policy.MinCharge)
	require.NotNil(t, policy.MaxCharge)
	assert.Equal(t, "1999.99", *policy.MaxCharge)
	assert.Equal(t, "HalfEven", policy.RoundingMode)
	require.NotNil(t, policy.RoundingPrecision)
	assert.Equal(t, 2, *policy.RoundingPrecision)

	schema := tool.ParamSchema()["properties"].(map[string]any)["policy"].(map[string]any)
	assert.Equal(t, false, schema["additionalProperties"])
	fields := schema["properties"].(map[string]any)
	for _, name := range []string{"minCharge", "maxCharge", "roundingMode", "roundingPrecision"} {
		assert.Contains(t, fields, name)
	}
	assert.Equal(t, []string{"number", "null"}, fields["minCharge"].(map[string]any)["type"])
	assert.Equal(t, 4, fields["roundingPrecision"].(map[string]any)["maximum"])
}

func TestProposeFormula_PassesTheTemplateDetailsAndBreakdown(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	tool := newProposeFormulaTool(workbench,
		&authorPermissions{operations: []permission.Operation{permission.OpCreate}})

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"expression":   "totalDistance * ratePerMile + handling",
		"explanation":  "Per mile plus handling.",
		"name":         "Per mile with handling",
		"description":  "Linehaul by the mile plus a handling fee.",
		"templateType": "FreightCharge",
		"breakdown": []any{
			map[string]any{
				"name":       "linehaul",
				"label":      "Linehaul",
				"expression": "totalDistance * ratePerMile",
			},
			map[string]any{"name": "handlingFee", "label": "Handling", "expression": "handling"},
		},
	}, ""))
	require.NoError(t, err)

	details := workbench.proposed.Details
	assert.Equal(t, "Per mile with handling", details.Name)
	assert.Equal(t, "Linehaul by the mile plus a handling fee.", details.Description)
	assert.Equal(t, "FreightCharge", details.TemplateType)
	require.Len(t, details.Breakdown, 2)
	assert.Equal(t, "handlingFee", details.Breakdown[1].Name)

	properties := tool.ParamSchema()["properties"].(map[string]any)
	for _, name := range []string{"name", "description", "templateType", "breakdown"} {
		assert.Contains(t, properties, name)
	}
}

func TestProposeFormula_TellsTheModelWhenTheEngineCouldNotRunIt(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{proposal: &pagedraft.FormulaProposal{
		Expression: "totalDistance *",
		Check:      pagedraft.FormulaCheck{Valid: false, Error: "unexpected end of expression"},
	}}
	tool := newProposeFormulaTool(workbench,
		&authorPermissions{operations: []permission.Operation{permission.OpCreate}})

	result, err := tool.Query(t.Context(), chatParams(map[string]any{
		"expression":  "totalDistance *",
		"explanation": "Per mile.",
	}, ""))
	require.NoError(t, err)

	edited := result.(pagedraft.EditResult)
	assert.Contains(t, edited.Note, "unexpected end of expression")
	assert.Contains(t, edited.Note, "propose it again")
}

func TestProposeFormula_RefusesAPersonWhoMayOnlyReadFormulas(t *testing.T) {
	t.Parallel()

	workbench := &fakeWorkbench{}
	tool := newProposeFormulaTool(workbench,
		&authorPermissions{operations: []permission.Operation{permission.OpRead}})

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"expression":  "totalDistance * 2",
		"explanation": "Two dollars a mile.",
	}, ""))
	require.ErrorContains(t, err, "may read formulas but not write them")
	assert.Nil(t, workbench.proposed)

	_, err = tool.Query(t.Context(), chatParams(map[string]any{
		"expression": "totalDistance * 2",
	}, ""))
	require.Error(t, err)
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

func TestGetFormulaTemplate_IsAReadOfFormulaTemplates(t *testing.T) {
	t.Parallel()

	tool := newGetFormulaTemplateTool(&stubFormulaTemplates{}, &fakePermissions{})
	policy := tool.Policy()

	assert.Equal(t, "get_formula_template", tool.Name())
	assert.Equal(t, permission.ResourceFormulaTemplate, policy.Resource)
	assert.Equal(t, agent.ToolKindQuery, policy.Kind)
	assert.Contains(t, tool.Description(), "list_formula_templates")
}

/*
A formula is explained or changed from the template whole: its expression,
its own variables with their defaults and the limits its price is held to.
The list gives only the name and status, so the get returns the record.
*/
func TestGetFormulaTemplate_ReturnsTheTemplateWholeFromTheCallersTenant(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ft_")
	repo := &stubFormulaTemplates{entity: &formulatemplate.FormulaTemplate{
		ID:         id,
		Name:       "Reefer per mile",
		Expression: "baseRate * totalDistance + reeferFee",
		Status:     formulatemplate.StatusActive,
		SchemaID:   "shipment",
	}}
	tool := newGetFormulaTemplateTool(repo, &fakePermissions{allowed: true})
	params := testParams(map[string]any{"formulaTemplateId": id.String()})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, id, repo.last.TemplateID)
	assert.Equal(t, params.OrganizationID, repo.last.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, repo.last.TenantInfo.BuID)
	document := encodedDocument(t, result)
	assert.Equal(t, "Reefer per mile", document["name"])
	assert.Equal(t, "baseRate * totalDistance + reeferFee", document["expression"])
}

func TestGetFormulaTemplate_RefusesSomethingThatIsNotAnID(t *testing.T) {
	t.Parallel()

	repo := &stubFormulaTemplates{}
	tool := newGetFormulaTemplateTool(repo, &fakePermissions{allowed: true})

	_, err := tool.Query(t.Context(), testParams(map[string]any{
		"formulaTemplateId": "Reefer per mile",
	}))
	require.Error(t, err)
	assert.True(t, repo.last.TemplateID.IsNil(), "nothing is read for a name passed as an id")
}
