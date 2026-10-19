package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	schemaIDDescription = "The formula schema, as the page draft names it; shipment when " +
		"it names none."
	pricedByEngineNote = "Every amount here was computed by Trenova's formula engine. Quote " +
		"these amounts; never state an amount you did not get from a tool."
	proposalAppliedNote = "The page applied this to the editor; the person can undo it. It " +
		"shows in the page draft on your next turn. "
	paramPolicy = "policy"
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
		newGetFormulaTemplateTool,
		provideDescribeFormulaSchemaTool,
		provideTestFormulaExpressionTool,
		provideProposeFormulaTool,
	}
}

func formulaTenant(params *agentports.QueryToolParams) pagination.TenantInfo {
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
	repo formulatemplateports.FormulaTemplateRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_formula_templates",
		entityPlural: "formula templates",
		summary: "List formula templates — the rating methods a shipment is priced with. " +
			"Use it to turn a rating method a person named into the id a shipment needs, or " +
			"to find a formula to read. A row carries only a template's name, description, " +
			"type and status: to show, explain or price one template, read it whole with " +
			"get_formula_template using the row's id.",
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
			result, err := repo.List(
				ctx,
				&formulatemplateports.ListFormulaTemplatesRequest{Filter: opts},
			)
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

func newGetFormulaTemplateTool(
	repo formulatemplateports.FormulaTemplateRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:   "get_formula_template",
		entity: "formula template",
		searchTerms: []string{
			"formula template details", "read a formula", "formula variables",
			"minimum charge", "rounding",
		},
		resource: permission.ResourceFormulaTemplate,
		summary: "Retrieve one formula template by id, whole: its formula, variables and " +
			"their defaults, breakdown lines, minimum, maximum, rounding, status and version. " +
			"Read it before explaining or changing a formula. Use list_formula_templates first " +
			"when you have a name rather than an id.",
		paramName: "formulaTemplateId",
		idSource:  "from list_formula_templates",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, formulatemplateports.GetFormulaTemplateByIDRequest{
				TemplateID: id,
				TenantInfo: tenant,
			})
		},
	}, permissions)
}

type describeFormulaSchemaTool struct {
	workbench formulaWorkbench
}

func provideDescribeFormulaSchemaTool(
	workbench *formulaassistantservice.Service,
) agentports.AgentQueryTool {
	return newDescribeFormulaSchemaTool(workbench)
}

func newDescribeFormulaSchemaTool(workbench formulaWorkbench) agentports.AgentQueryTool {
	return &describeFormulaSchemaTool{workbench: workbench}
}

func (t *describeFormulaSchemaTool) Name() string { return "describe_formula_schema" }

func (t *describeFormulaSchemaTool) Description() string {
	return "Describe what a rating formula may use: every shipment variable (one line each), " +
		"the functions by signature, and the rate tables it can look up. Read it before " +
		"writing or explaining a formula, so every name you use is one the engine knows. " +
		"Give query (\"reefer\", \"stops\", \"round\") for the matching variables and " +
		"functions in full, with examples. A name not listed is not a shipment field: make " +
		"it a template variable or ask, never guess."
}

func (t *describeFormulaSchemaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"schemaId": map[string]any{"type": "string", "description": schemaIDDescription},
			paramQuery: stringParam("Words for what the formula needs, such as \"reefer\" or " +
				"\"weight\": only the variables and functions that match, described in full."),
		},
		"additionalProperties": false,
	}
}

func (t *describeFormulaSchemaTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFormulaTemplate})
}

func (t *describeFormulaSchemaTool) SearchTerms() []string {
	return []string{
		"formula", "variables", "functions", "expression", "rate table", "lookup", "rating",
	}
}

func (t *describeFormulaSchemaTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	reference, err := t.workbench.Reference(ctx, formulaTenant(params),
		optionalString(params.Params, "schemaId"))
	if err != nil {
		return nil, err
	}

	return schemaView(reference, optionalString(params.Params, paramQuery)), nil
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
					"type": "object",
					"description": "Values for this load, by variable name or schema field " +
						"path (serviceType.code), so a load can say which service or " +
						"equipment it is.",
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

type policyParam struct {
	MinCharge         *float64 `json:"minCharge"`
	MaxCharge         *float64 `json:"maxCharge"`
	RoundingMode      string   `json:"roundingMode"`
	RoundingPrecision *int     `json:"roundingPrecision"`
}

func policySchema(description string) map[string]any {
	return map[string]any{
		"type":        "object",
		"description": description,
		"properties": map[string]any{
			"minCharge": map[string]any{
				"type": []string{"number", "null"},
				"description": "The least the template bills for a load, whatever the " +
					"expression comes to; null removes the minimum.",
			},
			"maxCharge": map[string]any{
				"type": []string{"number", "null"},
				"description": "The most the template bills for a load, whatever the " +
					"expression comes to; null removes the maximum.",
			},
			"roundingMode": agenttoolschema.Enum(
				"How the charge is rounded after the bounds; HalfUp unless the person "+
					"chose otherwise.",
				formulaRoundingModes,
			),
			"roundingPrecision": map[string]any{
				"type":    "integer",
				"minimum": 0,
				"maximum": int(formulatypes.MaxRoundingPrecision),
				"description": "The decimal places the charge is rounded to, 0 to " +
					strconv.Itoa(int(formulatypes.MaxRoundingPrecision)) + "; 2 is the cent.",
			},
		},
		"additionalProperties": false,
	}
}

func decodePolicy(params map[string]any) (*pagedraft.FormulaPolicy, error) {
	if raw, ok := params[paramPolicy]; !ok || raw == nil {
		return nil, nil
	}

	var decoded policyParam
	if err := decodeParam(params, paramPolicy, &decoded); err != nil {
		return nil, err
	}

	return &pagedraft.FormulaPolicy{
		MinCharge:         amountString(decoded.MinCharge),
		MaxCharge:         amountString(decoded.MaxCharge),
		RoundingMode:      decoded.RoundingMode,
		RoundingPrecision: decoded.RoundingPrecision,
	}, nil
}

func amountString(value *float64) *string {
	if value == nil {
		return nil
	}
	text := decimal.NewFromFloat(*value).String()

	return &text
}

type testFormulaExpressionTool struct {
	workbench formulaWorkbench
	access    fieldAccess
}

func provideTestFormulaExpressionTool(
	workbench *formulaassistantservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newTestFormulaExpressionTool(workbench, permissions)
}

func newTestFormulaExpressionTool(
	workbench formulaWorkbench,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
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
			paramPolicy: policySchema("The page draft's charge policy, so the amounts " +
				"match what is billed."),
			"shipmentId": agenttoolschema.RecordIDText(permission.ResourceShipment,
				"Optional: price a saved shipment instead of sample loads, "+
					"from shipment_list."),
		},
		"required":             []string{"expression"},
		"additionalProperties": false,
	}
}

func (t *testFormulaExpressionTool) Policy() agentports.ToolPolicy {
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
	params *agentports.QueryToolParams,
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
	policy, err := decodePolicy(params.Params)
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
		Policy:     policy,
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newProposeFormulaTool(workbench, permissions)
}

func newProposeFormulaTool(
	workbench formulaWorkbench,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &proposeFormulaTool{workbench: workbench, access: newFieldAccess(permissions)}
}

func (t *proposeFormulaTool) Name() string { return string(pagedraft.ActionProposeFormula) }

func (t *proposeFormulaTool) Recipe() []string {
	return []string{
		"describe_formula_schema",
		"test_formula_expression",
		string(pagedraft.ActionProposeFormula),
	}
}

func (t *proposeFormulaTool) Description() string {
	return "Change the formula in the person's editor: the expression, its variables, the " +
		"charge policy, breakdown lines, and for a new template its name and description. " +
		"The page applies the change the moment you call this; the person sees " +
		"it in their editor, can undo it, and nothing is saved until they save. Call it for " +
		"every change the person asks for — add, remove, fix, rename, set a default, change " +
		"the minimum — with the whole formula as it should be after the change. Keep " +
		"everything they did not ask to change exactly as the page draft shows it. Never " +
		"describe a change without making it."
}

func (t *proposeFormulaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type": "string",
				"description": "The expression, built only from describe_formula_schema's names " +
					"and this formula's variables. Write every rate, fee, minimum, percentage " +
					"and weight or mileage break as a variable, never as a number in the " +
					"expression: one template prices every customer, and a figure written into " +
					"it can only be changed by editing the formula. baseRate * totalDistance, not " +
					"2.85 * totalDistance; totalWeight < breakOne, not totalWeight < 500. Only a " +
					"unit conversion such as / 100 for hundredweight stays a number. It returns " +
					"the charge for the whole load, never a rate: a figure per mile, hour or stop " +
					"is multiplied by totalDistance, the hours or the stops in the expression. " +
					"Write it from the terms the person states, even when they say the terms are " +
					"their contracts': open an agreement or program only when they ask you to take " +
					"the terms from one. A quantity no shipment field holds, such as hours at a " +
					"stop, is a variable with a default: declare it and propose rather than asking " +
					"how it will arrive. So is a condition no field marks directly, such as team " +
					"service: compare the nearest field to a variable (serviceType.code == " +
					"teamServiceCode) or make it a yes/no variable, and say in the explanation " +
					"what to set it to. Lay it out as the editor does: for more than two " +
					"terms, one let per charge on its own line, named for what it is and " +
					"ending with a semicolon (let fuelSurcharge = …;), then the sum of the " +
					"names on the last line; never one long line or one chain of ternaries.",
			},
			"schemaId": map[string]any{"type": "string", "description": schemaIDDescription},
			"variables": map[string]any{
				"type": "array",
				"description": "The formula's own variables: every rate, fee, minimum, " +
					"percentage and break the expression uses, and any input no shipment " +
					"variable holds, such as hours at a stop. Something billed per stop, per " +
					"hour or per event is written for one occurrence from such an input, cap " +
					"included; say so in the explanation rather than asking how the data " +
					"arrives. Give the figure the person named as its default.",
				"maxItems": pagedraft.MaxFormulaVariables,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string", "description": "An identifier."},
						"type": agenttoolschema.Enum(
							"What the variable holds.",
							formulaVariableTypes,
						),
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
			"name": map[string]any{
				"type":      "string",
				"maxLength": pagedraft.MaxTemplateNameLength,
				"description": "Only when you draft a new template from scratch and the page " +
					"draft has no name yet: a short name for the rating method, such as " +
					"\"Linehaul per mile with fuel\". Leave it out when changing an existing " +
					"formula.",
			},
			"description": map[string]any{
				"type":      "string",
				"maxLength": pagedraft.MaxVariableTextLength,
				"description": "Only when you draft a new template from scratch and the page " +
					"draft has no description yet: one or two sentences on what it prices and " +
					"when it applies, for the reviewer who approves it.",
			},
			"templateType": agenttoolschema.Enum(
				"Only when you draft a new template from scratch: FreightCharge for the "+
					"linehaul price of a load, AccessorialCharge for an extra service or fee "+
					"billed on top of it.",
				formulaTemplateTypeSource,
			),
			"breakdown": map[string]any{
				"type":     "array",
				"maxItems": pagedraft.MaxBreakdownLines,
				"description": "Only when the person asks for itemized lines on the invoice: " +
					"each line prices one part of the charge with its own expression, and the " +
					"lines add up to the formula's result. Leave it out otherwise.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
							"description": "A camelCase identifier, unique among the lines, " +
								"such as fuelSurcharge.",
						},
						"label": map[string]any{
							"type":        "string",
							"maxLength":   pagedraft.MaxBreakdownLabel,
							"description": "What the invoice line says, such as \"Fuel surcharge\".",
						},
						"expression": map[string]any{
							"type": "string",
							"description": "The line's amount, built from the same names as " +
								"the formula.",
						},
					},
					"required":             []string{"name", "label", "expression"},
					"additionalProperties": false,
				},
			},
			paramPolicy: policySchema("The charge policy the template applies after the " +
				"expression. Always pass the page draft's policy here, changed only where the " +
				"person asked (a minimum, a maximum, the rounding), so the amounts you quote " +
				"match what the studio bills. Never write a minimum, a maximum or rounding " +
				"into the expression."),
			"scenarios": scenarioSchema(),
		},
		"required":             []string{"expression", "explanation"},
		"additionalProperties": false,
	}
}

func (t *proposeFormulaTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceFormulaTemplate,
		scope:    agent.ToolScopeSelf,
		effect:   agent.ToolEffectPresent,
		rationale: "Changes the person's own editor, which they can undo; nothing is saved " +
			"and nothing is sent.",
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
	params *agentports.QueryToolParams,
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
	params *agentports.QueryToolParams,
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
	details := pagedraft.FormulaDetails{
		Name:         optionalString(params.Params, "name"),
		Description:  optionalString(params.Params, "description"),
		TemplateType: optionalString(params.Params, "templateType"),
	}
	if raw, ok := params.Params["breakdown"]; ok && raw != nil {
		if err = decodeParam(params.Params, "breakdown", &details.Breakdown); err != nil {
			return nil, err
		}
	}
	if details.Policy, err = decodePolicy(params.Params); err != nil {
		return nil, err
	}

	proposal, err := t.workbench.Propose(ctx, &formulaassistantservice.ProposeRequest{
		TenantInfo:  formulaTenant(params),
		SchemaID:    optionalString(params.Params, "schemaId"),
		Expression:  expression,
		Variables:   variables,
		Details:     details,
		Explanation: explanation,
		Scenarios:   scenarios,
	})
	if err != nil {
		return nil, err
	}

	note := proposalAppliedNote + pricedByEngineNote
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
