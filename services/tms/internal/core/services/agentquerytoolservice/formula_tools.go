package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formulaassistantservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	schemaIDDescription = "The formula schema, as the page draft names it; shipment when " +
		"it names none."
	pricedByEngineNote = "Every amount here was computed by Trenova's formula engine. Quote " +
		"these amounts; never state an amount you did not get from a tool."
)

type formulaWorkbench interface {
	Reference(
		ctx context.Context,
		tenant pagination.TenantInfo,
		schemaID string,
	) (*formulaassistantservice.Reference, error)
	Test(
		ctx context.Context,
		req *formulaassistantservice.TestRequest,
	) (*formulaassistantservice.TestOutcome, error)
	Propose(
		ctx context.Context,
		req *formulaassistantservice.ProposeRequest,
	) (*pagedraft.FormulaProposal, error)
}

func formulaToolProviders() []any {
	return []any{
		newListFormulaTemplatesTool,
		provideDescribeFormulaSchemaTool,
		provideTestFormulaExpressionTool,
		provideProposeFormulaTool,
	}
}

func formulaTenant(params *serviceports.QueryToolParams) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
}

var (
	formulaTemplateTypes    = []string{"FreightCharge", "AccessorialCharge"}
	formulaTemplateStatuses = []string{"Active", "Inactive", "Draft", "InReview"}
)

type formulaTemplateRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	SchemaID    string `json:"schemaId"`
}

func newListFormulaTemplatesTool(
	repo repositories.FormulaTemplateRepository,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_formula_templates",
		entityPlural: "formula templates",
		summary: "List formula templates — the rating methods a shipment is priced with. " +
			"Use it to turn a rating method a person named into the id a shipment needs, or " +
			"to find a formula to read.",
		resource: permission.ResourceFormulaTemplate,
		config:   querybuilder.GetFieldConfiguration((*formulatemplate.FormulaTemplate)(nil)),
		fields: []listField{
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "type", Kind: filterEnum, Values: formulaTemplateTypes},
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: formulaTemplateStatuses,
				Note:   "only Active ones price shipments",
			},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListFormulaTemplatesRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *formulatemplate.FormulaTemplate) any {
				return formulaTemplateRow{
					ID:          item.ID.String(),
					Name:        item.Name,
					Description: item.Description,
					Type:        string(item.Type),
					Status:      string(item.Status),
					SchemaID:    item.SchemaID,
				}
			}), nil
		},
	})
}

type describeFormulaSchemaTool struct {
	workbench formulaWorkbench
}

func provideDescribeFormulaSchemaTool(
	workbench *formulaassistantservice.Service,
) serviceports.AgentQueryTool {
	return newDescribeFormulaSchemaTool(workbench)
}

func newDescribeFormulaSchemaTool(workbench formulaWorkbench) serviceports.AgentQueryTool {
	return &describeFormulaSchemaTool{workbench: workbench}
}

func (t *describeFormulaSchemaTool) Name() string { return "describe_formula_schema" }

func (t *describeFormulaSchemaTool) Description() string {
	return "Describe what a rating formula may use: the shipment variables, the functions " +
		"and the rate tables it can look up. Read it before writing or explaining a formula, " +
		"so every name you use is one the engine knows."
}

func (t *describeFormulaSchemaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"schemaId": map[string]any{"type": "string", "description": schemaIDDescription},
		},
		"additionalProperties": false,
	}
}

func (t *describeFormulaSchemaTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFormulaTemplate})
}

func (t *describeFormulaSchemaTool) SearchTerms() []string {
	return []string{
		"formula", "variables", "functions", "expression", "rate table", "lookup", "rating",
	}
}

func (t *describeFormulaSchemaTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	return t.workbench.Reference(ctx, formulaTenant(params),
		optionalString(params.Params, "schemaId"))
}

type scenarioParam struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Variables   map[string]any `json:"variables"`
}

func scenarioSchema() map[string]any {
	return map[string]any{
		"type":     "array",
		"maxItems": formulaassistantservice.MaxScenarios,
		"description": "Loads to price, each named and described in a sentence, with the " +
			"variable values that make it that load.",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "description": "A short name."},
				"description": map[string]any{"type": "string", "description": "One sentence."},
				"variables": map[string]any{
					"type":        "object",
					"description": "Variable values for this load, by variable name.",
				},
			},
			"required":             []string{"name", "variables"},
			"additionalProperties": false,
		},
	}
}

func decodeScenarios(params map[string]any) ([]formulaassistantservice.Scenario, error) {
	if raw, ok := params["scenarios"]; !ok || raw == nil {
		return nil, nil
	}

	var decoded []scenarioParam
	if err := decodeParam(params, "scenarios", &decoded); err != nil {
		return nil, err
	}

	scenarios := make([]formulaassistantservice.Scenario, 0, len(decoded))
	for _, scenario := range decoded {
		scenarios = append(scenarios, formulaassistantservice.Scenario{
			Name:        scenario.Name,
			Description: scenario.Description,
			Variables:   scenario.Variables,
		})
	}

	return scenarios, nil
}

type testFormulaExpressionTool struct {
	workbench formulaWorkbench
	access    fieldAccess
}

func provideTestFormulaExpressionTool(
	workbench *formulaassistantservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newTestFormulaExpressionTool(workbench, permissions)
}

func newTestFormulaExpressionTool(
	workbench formulaWorkbench,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &testFormulaExpressionTool{workbench: workbench, access: newFieldAccess(permissions)}
}

func (t *testFormulaExpressionTool) Name() string { return "test_formula_expression" }

func (t *testFormulaExpressionTool) Description() string {
	return "Price a formula expression with Trenova's formula engine, for sample loads or " +
		"against a saved shipment. Use it to check a formula runs and to learn what it " +
		"charges; say an amount only when this or propose_formula gave it."
}

func (t *testFormulaExpressionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The expression to price, such as the one in the page draft.",
			},
			"schemaId": map[string]any{"type": "string", "description": schemaIDDescription},
			"variables": map[string]any{
				"type": "object",
				"description": "Values for the formula's own variables, by name, used by " +
					"every load unless a load sets its own.",
			},
			"scenarios": scenarioSchema(),
			"shipmentId": map[string]any{
				"type": "string",
				"description": "Optional: price a saved shipment instead of sample loads, " +
					"from list_shipments or search_shipments.",
			},
		},
		"required":             []string{"expression"},
		"additionalProperties": false,
	}
}

func (t *testFormulaExpressionTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceFormulaTemplate,
		rationale: "Prices an expression with the formula engine, for sample values or a " +
			"shipment the caller may read; nothing is saved and nothing is sent.",
	})
}

func (t *testFormulaExpressionTool) SearchTerms() []string {
	return []string{
		"test", "price", "evaluate", "try", "formula", "expression", "scenario", "charge",
	}
}

type testedFormula struct {
	*formulaassistantservice.TestOutcome
	ShipmentID string `json:"shipmentId,omitempty"`
	Note       string `json:"note"`
}

func (t *testFormulaExpressionTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	expression, err := requireString(params.Params, "expression")
	if err != nil {
		return nil, err
	}
	scenarios, err := decodeScenarios(params.Params)
	if err != nil {
		return nil, err
	}

	shipmentID := pulid.Nil
	if raw := optionalString(params.Params, "shipmentId"); raw != "" {
		shipmentID, err = pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parameter \"shipmentId\" is not a valid id: %w", err)
		}
		if !t.access.mayRead(ctx, params, permission.ResourceShipment) {
			return nil, errors.New(
				"the person you are working for may not read shipments, so no shipment " +
					"was priced; price sample loads instead",
			)
		}
	}

	outcome, err := t.workbench.Test(ctx, &formulaassistantservice.TestRequest{
		TenantInfo: formulaTenant(params),
		SchemaID:   optionalString(params.Params, "schemaId"),
		Expression: expression,
		Variables:  optionalObject(params.Params, "variables"),
		ShipmentID: shipmentID,
		Scenarios:  scenarios,
	})
	if err != nil {
		return nil, err
	}

	result := testedFormula{TestOutcome: outcome, Note: pricedByEngineNote}
	if shipmentID.IsNotNil() {
		result.ShipmentID = shipmentID.String()
	}

	return result, nil
}

type proposeFormulaTool struct {
	workbench formulaWorkbench
	access    fieldAccess
}

func provideProposeFormulaTool(
	workbench *formulaassistantservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newProposeFormulaTool(workbench, permissions)
}

func newProposeFormulaTool(
	workbench formulaWorkbench,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &proposeFormulaTool{workbench: workbench, access: newFieldAccess(permissions)}
}

func (t *proposeFormulaTool) Name() string { return string(pagedraft.ActionProposeFormula) }

func (t *proposeFormulaTool) Description() string {
	return "Put a formula in front of the person for their editor, with its variables, a " +
		"plain explanation and two or three sample loads priced by the engine. Nothing is " +
		"saved: the person inserts it, tests it and saves it themselves."
}

func (t *proposeFormulaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The expression, built only from describe_formula_schema's names.",
			},
			"schemaId": map[string]any{"type": "string", "description": schemaIDDescription},
			"variables": map[string]any{
				"type": "array",
				"description": "The formula's own variables: any input that is not a " +
					"shipment variable, with a sensible default.",
				"maxItems": pagedraft.MaxFormulaVariables,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string", "description": "An identifier."},
						"type": map[string]any{
							"type":        "string",
							"enum":        []string{"Number", "String", "Boolean"},
							"description": "What the variable holds.",
						},
						"description": map[string]any{
							"type":        "string",
							"description": "What it is for, for a billing clerk.",
						},
						"defaultValue": map[string]any{
							"type":        []string{"number", "string", "boolean", "null"},
							"description": "The value it takes unless a load sets one.",
						},
					},
					"required":             []string{"name", "type"},
					"additionalProperties": false,
				},
			},
			"explanation": map[string]any{
				"type":        "string",
				"description": "What the formula charges, in plain words, term by term.",
			},
			"scenarios": scenarioSchema(),
		},
		"required":             []string{"expression", "explanation"},
		"additionalProperties": false,
	}
}

func (t *proposeFormulaTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceFormulaTemplate,
		scope:    agent.ToolScopeSelf,
		effect:   agent.ToolEffectPresent,
		rationale: "Hands a formula to the person's own editor for them to insert, test and " +
			"save; nothing is saved and nothing is sent.",
	})
}

func (t *proposeFormulaTool) SearchTerms() []string {
	return []string{
		"write formula", "suggest formula", "build formula", "draft", "rating method",
		"charge", "expression",
	}
}

func (t *proposeFormulaTool) mayAuthor(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) bool {
	actor := params.Actor
	if actor == nil || t.access.permissions == nil {
		return false
	}

	for _, operation := range []permission.Operation{permission.OpCreate, permission.OpUpdate} {
		result, err := t.access.permissions.Check(ctx, actor.PermissionCheck(
			permission.ResourceFormulaTemplate, operation,
		))
		if err == nil && result != nil && result.Allowed {
			return true
		}
	}

	return false
}

func (t *proposeFormulaTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	if !t.mayAuthor(ctx, params) {
		return nil, errors.New(
			"the person you are working for may read formulas but not write them, so no " +
				"formula was proposed; explain the formula instead, or tell them writing one " +
				"needs permission to create or update formula templates",
		)
	}

	expression, err := requireString(params.Params, "expression")
	if err != nil {
		return nil, err
	}
	explanation, err := requireString(params.Params, "explanation")
	if err != nil {
		return nil, err
	}
	var variables []pagedraft.FormulaVariable
	if raw, ok := params.Params["variables"]; ok && raw != nil {
		if err = decodeParam(params.Params, "variables", &variables); err != nil {
			return nil, err
		}
	}
	scenarios, err := decodeScenarios(params.Params)
	if err != nil {
		return nil, err
	}

	proposal, err := t.workbench.Propose(ctx, &formulaassistantservice.ProposeRequest{
		TenantInfo:  formulaTenant(params),
		SchemaID:    optionalString(params.Params, "schemaId"),
		Expression:  expression,
		Variables:   variables,
		Explanation: explanation,
		Scenarios:   scenarios,
	})
	if err != nil {
		return nil, err
	}

	note := "The formula is in front of the person with its sample loads. " + pricedByEngineNote
	if !proposal.Check.Valid {
		note = "The engine could not run this formula: " + proposal.Check.Error +
			". Tell the person, fix it and propose it again."
	}

	return pagedraft.EditResult{
		Draft: pagedraft.Edit{
			Surface: pagedraft.SurfaceFormula,
			Action:  pagedraft.ActionProposeFormula,
			Formula: proposal,
		},
		Note: note,
	}, nil
}
