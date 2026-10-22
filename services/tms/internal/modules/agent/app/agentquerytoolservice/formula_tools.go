package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	formulaFieldDescription = "description"
	formulaFieldRequired    = "required"
	entityFormulaLibrary    = "formula library"
	termFormula             = "formula"
	termTariff              = "tariff"
)

const (
	paramSource        = "source"
	paramFormulaKind   = "kind"
	paramExplanation   = "explanation"
	pricedByEngineNote = "Every amount here was computed by Trenova's Tariff engine. Quote " +
		"these amounts; never state an amount you did not get from a tool."
	proposalAppliedNote = "The page applied this to the editor; the person can undo it. It " +
		"shows in the page draft on your next turn."
	sourceParamDescription = "The whole Tariff source file, exactly as it should read."
)

type formulaWorkbench interface {
	Language(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*formulaassistantservice.Language, error)
	Check(
		ctx context.Context,
		req *formulaassistantservice.CheckRequest,
	) (*formulaassistantservice.Checked, error)
	Preview(
		ctx context.Context,
		req *formulaassistantservice.PreviewRequest,
	) (*formulaassistantservice.Preview, error)
	Propose(
		ctx context.Context,
		req *formulaassistantservice.ProposeRequest,
	) (*formulaassistantservice.Proposed, error)
}

func formulaToolProviders() []any {
	return []any{
		newListFormulaTemplatesTool,
		newGetFormulaTemplateTool,
		newListFormulaLibrariesTool,
		newGetFormulaLibraryTool,
		provideDescribeFormulaLanguageTool,
		provideCheckFormulaTool,
		provideRunFormulaTestsTool,
		providePreviewFormulaTool,
		provideProposeFormulaTool,
		provideProposeLibraryTool,
	}
}

func formulaTenant(params *agentports.QueryToolParams) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
}

var formulaStatuses = []string{
	string(formulatemplate.StatusActive),
	string(formulatemplate.StatusInactive),
	string(formulatemplate.StatusDraft),
	string(formulatemplate.StatusInReview),
}

type formulaTemplateRow struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Status        string `json:"status"`
	VersionNumber int64  `json:"versionNumber"`
	TestCount     int32  `json:"testCount"`
}

func newListFormulaTemplatesTool(
	repo formulatemplateports.FormulaTemplateRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "formula_list_templates",
		entityPlural: "formula templates",
		summary: "List formula templates, the Tariff rating methods a shipment, rule or " +
			"agreement is priced with. Use it to turn a rating method a person named into its " +
			"id. A row carries only a template's name, description, status, version and test " +
			"count: read one whole, with its source and inputs, with formula_get_template.",
		resource: permission.ResourceFormulaTemplate,
		config:   querybuilder.GetFieldConfiguration((*formulatemplate.FormulaTemplate)(nil)),
		fields: []listField{
			{Name: "name", Kind: filterText, Sortable: true},
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: formulaStatuses,
				Note:   "only Active ones price shipments",
			},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listPage, error) {
			result, err := repo.List(
				ctx,
				&formulatemplateports.ListFormulaTemplatesRequest{Filter: opts},
			)
			if err != nil {
				return listPage{}, err
			}

			return offsetListPage(result.Items, func(item *formulatemplate.FormulaTemplate) any {
				return formulaTemplateRow{
					ID:            item.ID.String(),
					Name:          item.Name,
					Description:   item.Description,
					Status:        string(item.Status),
					VersionNumber: item.CurrentVersionNumber,
					TestCount:     item.TestCount,
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
		name:   "formula_get_template",
		entity: "formula template",
		searchTerms: []string{
			"formula template details", "read a formula", "tariff source", "formula inputs",
		},
		resource: permission.ResourceFormulaTemplate,
		summary: "Retrieve one formula template by id, whole: its Tariff source, the inputs " +
			"it declares, the libraries it imports, its status and version. Read it before " +
			"explaining or changing a formula. Use formula_list_templates first when you have " +
			"a name rather than an id.",
		paramName: "formulaTemplateId",
		idSource:  "from formula_list_templates",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, formulatemplateports.GetFormulaTemplateByIDRequest{
				TemplateID: id,
				TenantInfo: tenant,
			})
		},
	}, permissions)
}

type formulaLibraryRow struct {
	ID            string `json:"id"`
	Path          string `json:"path"`
	Description   string `json:"description,omitempty"`
	Status        string `json:"status"`
	VersionNumber int64  `json:"versionNumber"`
	TestCount     int32  `json:"testCount"`
}

func newListFormulaLibrariesTool(
	repo formulalibraryports.FormulaLibraryRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "formula_list_libraries",
		entityPlural: "formula libraries",
		searchTerms:  []string{"list libraries", "shared libraries", "formula libraries"},
		summary: "List the organisation's formula libraries: Tariff modules of shared " +
			"functions a template imports with use org.<name>. A row carries the library's " +
			"path, description, status, version and test count: read one whole, with its " +
			"source, with formula_get_library.",
		resource:      permission.ResourceFormulaLibrary,
		config:        querybuilder.GetFieldConfiguration((*formulalibrary.FormulaLibrary)(nil)),
		pagesByCursor: true,
		fields: []listField{
			{Name: "name", Kind: filterText, Sortable: true},
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: formulaStatuses,
				Note:   "templates import the latest Active version",
			},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, cursor pagination.CursorInfo) (listPage, error) {
			result, err := repo.ListConnection(
				ctx,
				&formulalibraryports.ListLibraryConnectionRequest{
					Filter: opts,
					Cursor: cursor,
				},
			)
			if err != nil {
				return listPage{}, err
			}

			return cursorListPage(result, func(item *formulalibrary.FormulaLibrary) any {
				return formulaLibraryRow{
					ID:            item.ID.String(),
					Path:          formulalibrary.Path(item.Name),
					Description:   item.Description,
					Status:        string(item.Status),
					VersionNumber: item.CurrentVersionNumber,
					TestCount:     item.TestCount,
				}
			})
		},
	})
}

func newGetFormulaLibraryTool(
	repo formulalibraryports.FormulaLibraryRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:   "formula_get_library",
		entity: entityFormulaLibrary,
		searchTerms: []string{
			entityFormulaLibrary, "library source", "shared functions", "org module", "use org",
		},
		resource: permission.ResourceFormulaLibrary,
		summary: "Retrieve one formula library by id, whole: its Tariff source, the libraries " +
			"it imports, its status and version. Read it before explaining a function a " +
			"template imports or changing the library.",
		paramName: "formulaLibraryId",
		idSource:  "from formula_list_libraries",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
				LibraryID:  id,
				TenantInfo: tenant,
			})
		},
	}, permissions)
}

type describeFormulaLanguageTool struct {
	workbench formulaWorkbench
}

func provideDescribeFormulaLanguageTool(
	workbench *formulaassistantservice.Service,
) agentports.AgentQueryTool {
	return &describeFormulaLanguageTool{workbench: workbench}
}

func (t *describeFormulaLanguageTool) Name() string { return "formula_describe_language" }

func (t *describeFormulaLanguageTool) Description() string {
	return "Describe what a Tariff formula may use here: record fields, types and units, " +
		"functions, rate tables, accessorial codes and libraries. It covers every record " +
		"field (shipment, lane, customer, fuel, rating) with its type, the methods, the std " +
		"modules' functions and this organisation's importable org libraries. Read it before " +
		"writing or explaining a formula so every name you use exists. Give query (\"reefer\", \"stops\", \"ltl\", \"accessorials\") " +
		"for the matching entries in full, and a library's source."
}

func (t *describeFormulaLanguageTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramQuery: stringParam("Words for what the formula needs, such as \"reefer\", " +
				"\"weight\" or a library's name: only the entries that match, in full."),
		},
		"additionalProperties": false,
	}
}

func (t *describeFormulaLanguageTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFormulaTemplate})
}

func (t *describeFormulaLanguageTool) SearchTerms() []string {
	return []string{
		"formula language", "tariff language", "record fields", "functions", "units",
		"rate table", "syntax",
	}
}

func (t *describeFormulaLanguageTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	language, err := t.workbench.Language(ctx, formulaTenant(params))
	if err != nil {
		return nil, err
	}

	return languageView(language, optionalString(params.Params, paramQuery))
}

func rawSource(params map[string]any) (string, error) {
	source, ok := params[paramSource].(string)
	if !ok || source == "" {
		return "", fmt.Errorf("parameter %q must be the whole Tariff source", paramSource)
	}
	return source, nil
}

func formulaKind(params map[string]any) (formulaassistantservice.Kind, error) {
	kind := formulaassistantservice.Kind(optionalString(params, paramFormulaKind))
	if kind == "" {
		return formulaassistantservice.KindTemplate, nil
	}
	if !kind.IsValid() {
		return "", fmt.Errorf("parameter %q must be template or library", paramFormulaKind)
	}
	return kind, nil
}

func sourceParams() map[string]any {
	return map[string]any{
		paramSource: stringParam(sourceParamDescription),
		paramFormulaKind: agenttoolschema.Enum(
			"template for a formula template (tariff \"…\"), library for an org library "+
				"(library \"org.…\"); template when left out.",
			formulaKinds,
		),
	}
}

type checkFormulaTool struct {
	workbench formulaWorkbench
}

func provideCheckFormulaTool(workbench *formulaassistantservice.Service) agentports.AgentQueryTool {
	return &checkFormulaTool{workbench: workbench}
}

func (t *checkFormulaTool) Name() string { return "formula_check" }

func (t *checkFormulaTool) Description() string {
	return "Check a Tariff source against this organisation's tables, codes and libraries. " +
		"It returns every error and warning with its line and column and the fixes the " +
		"checker offers, the inputs it declares, the libraries it imports and the charges " +
		"it bills. Check before you show or explain source; fix every error."
}

func (t *checkFormulaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           sourceParams(),
		formulaFieldRequired:   []string{paramSource},
		"additionalProperties": false,
	}
}

func (t *checkFormulaTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource:  permission.ResourceFormulaTemplate,
		rationale: "Checks source the model holds; nothing is saved and nothing is sent.",
	})
}

func (t *checkFormulaTool) SearchTerms() []string {
	return []string{
		"check", "validate", "compile", "errors", "diagnostics", termFormula, termTariff,
	}
}

type checkedFormula struct {
	OK       bool                            `json:"ok"`
	Kind     string                          `json:"kind,omitempty"`
	Name     string                          `json:"name,omitempty"`
	Findings any                             `json:"findings"`
	Inputs   []formulaassistantservice.Input `json:"inputs"`
	Imports  []string                        `json:"imports"`
	Charges  []string                        `json:"charges"`
}

func (t *checkFormulaTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	checked, err := checkParams(ctx, t.workbench, params)
	if err != nil {
		return nil, err
	}

	return checkedFormula{
		OK:       checked.OK,
		Kind:     checked.Kind,
		Name:     checked.Name,
		Findings: checked.Findings,
		Inputs:   checked.Inputs,
		Imports:  checked.Imports,
		Charges:  checked.Charges,
	}, nil
}

func checkParams(
	ctx context.Context,
	workbench formulaWorkbench,
	params *agentports.QueryToolParams,
) (*formulaassistantservice.Checked, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	source, err := rawSource(params.Params)
	if err != nil {
		return nil, err
	}
	kind, err := formulaKind(params.Params)
	if err != nil {
		return nil, err
	}

	return workbench.Check(ctx, &formulaassistantservice.CheckRequest{
		TenantInfo: formulaTenant(params),
		Kind:       kind,
		Source:     source,
	})
}

type runFormulaTestsTool struct {
	workbench formulaWorkbench
}

func provideRunFormulaTestsTool(
	workbench *formulaassistantservice.Service,
) agentports.AgentQueryTool {
	return &runFormulaTestsTool{workbench: workbench}
}

func (t *runFormulaTestsTool) Name() string { return "formula_run_tests" }

func (t *runFormulaTestsTool) Description() string {
	return "Run the test blocks of a Tariff source with Trenova's engine: each test's name, " +
		"line, whether it passed, and what each failed expectation actually came to. A " +
		"source that does not check runs no tests; its errors come back instead. Run the " +
		"tests before you propose source, and write a test for every case you claim it prices."
}

func (t *runFormulaTestsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           sourceParams(),
		formulaFieldRequired:   []string{paramSource},
		"additionalProperties": false,
	}
}

func (t *runFormulaTestsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource:  permission.ResourceFormulaTemplate,
		rationale: "Runs the tests in source the model holds; nothing is saved and nothing is sent.",
	})
}

func (t *runFormulaTestsTool) SearchTerms() []string {
	return []string{"test", "run tests", "expect", "verify", termFormula, termTariff}
}

type testedFormula struct {
	OK       bool   `json:"ok"`
	Findings any    `json:"findings,omitempty"`
	Tests    any    `json:"tests"`
	Note     string `json:"note"`
}

func (t *runFormulaTestsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	checked, err := checkParams(ctx, t.workbench, params)
	if err != nil {
		return nil, err
	}

	result := testedFormula{OK: checked.OK, Tests: checked.Tests, Note: pricedByEngineNote}
	switch {
	case !checked.OK:
		result.Findings = checked.Findings
		result.Note = "The source does not check, so no test ran. Fix the findings first."
	case checked.Tests.Total == 0:
		result.Note = "The source has no test blocks. Write test \"…\" { given … expect … } " +
			"for each case the person cares about."
	}
	return result, nil
}

type previewFormulaTool struct {
	workbench formulaWorkbench
	access    fieldAccess
}

func providePreviewFormulaTool(
	workbench *formulaassistantservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &previewFormulaTool{workbench: workbench, access: newFieldAccess(permissions)}
}

func (t *previewFormulaTool) Name() string { return "formula_preview" }

func (t *previewFormulaTool) Description() string {
	return "Rate a formula template's Tariff source for a saved shipment or the sample record. " +
		"With the input values given, it returns the bill line by line, or the refusal or " +
		"error and where in the source it came from. Use it to learn what a formula charges " +
		"a real load; say an amount only when a tool gave it."
}

func (t *previewFormulaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramSource: stringParam(sourceParamDescription),
			"inputs": map[string]any{
				"type": "object",
				formulaFieldDescription: "Values for the template's inputs, by name, as wire values: a " +
					"number {\"$n\":\"2.5\"}, money {\"$q\":\"175\",\"u\":\"$\"}, a rate " +
					"{\"$q\":\"2.85\",\"u\":\"$/mi\"}, a percent {\"$q\":\"18\",\"u\":\"%\"}, a " +
					"date {\"$date\":\"2026-01-31\"}, text and true/false as themselves. " +
					"Inputs with a default may be left out.",
				"maxProperties": formulaassistantPreviewInputs,
			},
			"shipmentId": agenttoolschema.RecordIDText(permission.ResourceShipment,
				"Optional: rate this saved shipment instead of the sample record, "+
					"from shipment_list."),
		},
		formulaFieldRequired:   []string{paramSource},
		"additionalProperties": false,
	}
}

const formulaassistantPreviewInputs = 100

func (t *previewFormulaTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceFormulaTemplate,
		rationale: "Rates source with the engine, for the sample record or a shipment the " +
			"caller may read; nothing is saved and nothing is sent.",
	})
}

func (t *previewFormulaTool) SearchTerms() []string {
	return []string{"price", "rate", "preview", "bill", "charge", "shipment", termFormula}
}

type previewedFormula struct {
	*formulaassistantservice.Preview
	ShipmentID string `json:"shipmentId,omitempty"`
	Note       string `json:"note"`
}

func (t *previewFormulaTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	source, err := rawSource(params.Params)
	if err != nil {
		return nil, err
	}
	var inputs tarifftypes.Inputs
	if raw, ok := params.Params["inputs"]; ok && raw != nil {
		if err = decodeParam(params.Params, "inputs", &inputs); err != nil {
			return nil, err
		}
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
					"was rated; preview the sample record instead",
			)
		}
	}

	preview, err := t.workbench.Preview(ctx, &formulaassistantservice.PreviewRequest{
		TenantInfo: formulaTenant(params),
		Source:     source,
		Inputs:     inputs,
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, err
	}

	result := previewedFormula{Preview: preview, Note: pricedByEngineNote}
	if !preview.OK {
		result.Note = "The source does not check, so nothing was rated. Fix the findings first."
	}
	if shipmentID.IsNotNil() {
		result.ShipmentID = shipmentID.String()
	}
	return result, nil
}

type proposeSourceSpec struct {
	action      pagedraft.Action
	surface     pagedraft.Surface
	kind        formulaassistantservice.Kind
	resource    permission.Resource
	noun        string
	nameLimit   int
	nameExample string
	description string
	searchTerms []string
}

type proposeSourceTool struct {
	spec      proposeSourceSpec
	workbench formulaWorkbench
	access    fieldAccess
}

func provideProposeFormulaTool(
	workbench *formulaassistantservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newProposeSourceTool(&proposeFormulaSpec, workbench, permissions)
}

func provideProposeLibraryTool(
	workbench *formulaassistantservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newProposeSourceTool(&proposeLibrarySpec, workbench, permissions)
}

var proposeFormulaSpec = proposeSourceSpec{
	action:      pagedraft.ActionProposeFormula,
	surface:     pagedraft.SurfaceFormula,
	kind:        formulaassistantservice.KindTemplate,
	resource:    permission.ResourceFormulaTemplate,
	noun:        "formula template",
	nameLimit:   pagedraft.MaxTemplateNameLength,
	nameExample: "\"Linehaul per mile with fuel\"",
	description: "Change the formula template in the person's editor: its whole Tariff " +
		"source, and for a new template its name and description. The source is checked and " +
		"its tests run first; the page applies it only when it checks clean and every test " +
		"passes, and it needs at least one test. The person sees it in their editor, can undo " +
		"it, and nothing is saved until they save. Call it for every change the person asks " +
		"for, with the whole source as it should be after the change, keeping everything they " +
		"did not ask to change exactly as the page draft shows it. Never describe a change " +
		"without making it.",
	searchTerms: []string{
		"write formula", "suggest formula", "build formula", "draft", "rating method",
		"charge", termTariff,
	},
}

var proposeLibrarySpec = proposeSourceSpec{
	action:      pagedraft.ActionProposeLibrary,
	surface:     pagedraft.SurfaceFormulaLibrary,
	kind:        formulaassistantservice.KindLibrary,
	resource:    permission.ResourceFormulaLibrary,
	noun:        entityFormulaLibrary,
	nameLimit:   pagedraft.MaxLibraryNameLength,
	nameExample: "accessorials, which templates import as org.accessorials",
	description: "Change the formula library in the person's editor: its whole Tariff " +
		"source (library \"org.<name>\", its functions, types and tests), and for a new " +
		"library its name and description. The source is checked and its tests run first; " +
		"the page applies it only when it checks clean and every test passes, and it needs at " +
		"least one test. The person can undo it, and nothing is saved until they save. A " +
		"change reaches templates only after the library is approved again.",
	searchTerms: []string{
		"write library", "shared function", "add function", "org library", "draft library",
		"tariff module",
	},
}

func newProposeSourceTool(
	spec *proposeSourceSpec,
	workbench formulaWorkbench,
	permissions permissionports.PermissionEngine,
) *proposeSourceTool {
	return &proposeSourceTool{
		spec:      *spec,
		workbench: workbench,
		access:    newFieldAccess(permissions),
	}
}

func (t *proposeSourceTool) Name() string { return string(t.spec.action) }

func (t *proposeSourceTool) Recipe() []string {
	return []string{
		"formula_describe_language",
		"formula_check",
		"formula_run_tests",
		string(t.spec.action),
	}
}

func (t *proposeSourceTool) Description() string { return t.spec.description }

func (t *proposeSourceTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramSource: stringParam(sourceParamDescription + " Declare every rate, fee, " +
				"minimum and break as an input with a doc comment, never a figure written into " +
				"a charge, so one " + t.spec.noun + " prices every customer; include a test " +
				"block for each case you describe."),
			paramExplanation: stringParam("What the source does, in plain words, and what " +
				"changed."),
			"name": map[string]any{
				"type":      "string",
				"maxLength": t.spec.nameLimit,
				formulaFieldDescription: "Only when you draft a new " + t.spec.noun + " and the page " +
					"draft has no name yet: a short name, such as " + t.spec.nameExample + ".",
			},
			formulaFieldDescription: map[string]any{
				"type":      "string",
				"maxLength": pagedraft.MaxDescriptionLength,
				formulaFieldDescription: "Only when you draft a new " + t.spec.noun + " and the page " +
					"draft has no description yet: one or two sentences for the reviewer who " +
					"approves it.",
			},
		},
		formulaFieldRequired:   []string{paramSource, paramExplanation},
		"additionalProperties": false,
	}
}

func (t *proposeSourceTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: t.spec.resource,
		scope:    agent.ToolScopeSelf,
		effect:   agent.ToolEffectPresent,
		rationale: "Changes the person's own editor, which they can undo; nothing is saved " +
			"and nothing is sent.",
	})
}

func (t *proposeSourceTool) SearchTerms() []string { return t.spec.searchTerms }

func (t *proposeSourceTool) mayAuthor(
	ctx context.Context,
	params *agentports.QueryToolParams,
) bool {
	actor := params.Actor
	if actor == nil || t.access.permissions == nil {
		return false
	}

	for _, operation := range []permission.Operation{permission.OpCreate, permission.OpUpdate} {
		result, err := t.access.permissions.Check(ctx, actor.PermissionCheck(
			t.spec.resource, operation,
		))
		if err == nil && result != nil && result.Allowed {
			return true
		}
	}

	return false
}

type refusedProposal struct {
	Applied  bool   `json:"applied"`
	Findings any    `json:"findings,omitempty"`
	Tests    any    `json:"tests,omitempty"`
	Note     string `json:"note"`
}

func (t *proposeSourceTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	if !t.mayAuthor(ctx, params) {
		return nil, fmt.Errorf(
			"the person you are working for may read but not write a %s, so nothing was "+
				"proposed; explain the source instead, or tell them writing one needs "+
				"permission to create or update it", t.spec.noun,
		)
	}

	source, err := rawSource(params.Params)
	if err != nil {
		return nil, err
	}
	explanation, err := requireString(params.Params, paramExplanation)
	if err != nil {
		return nil, err
	}

	proposed, err := t.workbench.Propose(ctx, &formulaassistantservice.ProposeRequest{
		TenantInfo:  formulaTenant(params),
		Kind:        t.spec.kind,
		Name:        optionalString(params.Params, "name"),
		Description: optionalString(params.Params, "description"),
		Source:      source,
		Explanation: explanation,
	})
	if err != nil {
		return nil, err
	}

	if proposed.Proposal == nil {
		return refusalOf(proposed.Checked), nil
	}

	edit := pagedraft.Edit{Surface: t.spec.surface, Action: t.spec.action}
	if t.spec.kind == formulaassistantservice.KindLibrary {
		edit.Library = proposed.Proposal
	} else {
		edit.Formula = proposed.Proposal
	}
	return pagedraft.EditResult{
		Draft: edit,
		Note:  proposalAppliedNote + " " + pricedByEngineNote,
	}, nil
}

func refusalOf(checked *formulaassistantservice.Checked) refusedProposal {
	switch {
	case !checked.OK:
		return refusedProposal{
			Findings: checked.Findings,
			Note: "Nothing was shown: the source does not check. Fix every error finding and " +
				"propose it again.",
		}
	case checked.Tests.Total == 0:
		return refusedProposal{
			Note: "Nothing was shown: the source has no tests. Add a test block for each case " +
				"you describe and propose it again.",
		}
	default:
		return refusedProposal{
			Tests: checked.Tests,
			Note: "Nothing was shown: a test fails. Fix the source or, when the person asked " +
				"for the change the test catches, the test, and propose it again.",
		}
	}
}
