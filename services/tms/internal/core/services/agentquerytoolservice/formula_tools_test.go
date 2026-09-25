package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formulaassistantservice"
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
	serviceports.PermissionEngine

	operations []permission.Operation
}

func (p *authorPermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	for _, operation := range p.operations {
		if req.Resource == permission.ResourceFormulaTemplate.String() &&
			req.Operation == operation {
			return &serviceports.PermissionCheckResult{Allowed: true}, nil
		}
	}

	return &serviceports.PermissionCheckResult{Allowed: false, Reason: "not permitted"}, nil
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

	reference := result.(*formulaassistantservice.Reference)
	assert.Equal(t, "shipment", reference.SchemaID)
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
