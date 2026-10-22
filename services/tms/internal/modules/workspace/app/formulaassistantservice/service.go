package formulaassistantservice

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	DefaultSchemaID     = "shipment"
	MaxScenarios        = 5
	MaxScenarioNameLen  = 120
	MaxExplanationRunes = 4000
	maxScenarioVars     = pagedraft.MaxFormulaVariables + 40
)

type Params struct {
	fx.In

	FormulaService  *formula.Service
	TemplateService *formulatemplateservice.Service
	RateMatrixRepo  ratematrixports.RateMatrixRepository
}

type Service struct {
	formulaService  *formula.Service
	templateService *formulatemplateservice.Service
	rateMatrixRepo  ratematrixports.RateMatrixRepository
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		formulaService:  p.FormulaService,
		templateService: p.TemplateService,
		rateMatrixRepo:  p.RateMatrixRepo,
	}
}

type RateTable struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Axes      int    `json:"axes"`
	Functions string `json:"functions"`
}

type FunctionReference struct {
	Name        string `json:"name"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Category    string `json:"category"`
	Operator    bool   `json:"operator,omitempty"`
	Predicate   bool   `json:"predicate,omitempty"`
}

type DisabledReference struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type Reference struct {
	SchemaID   string                  `json:"schemaId"`
	Variables  []rules.FormulaVariable `json:"variables"`
	Functions  []FunctionReference     `json:"functions"`
	Disabled   []DisabledReference     `json:"disabled"`
	RateTables []RateTable             `json:"rateTables"`
}

type Scenario struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Variables   map[string]any `json:"variables"`
}

type TestRequest struct {
	TenantInfo pagination.TenantInfo
	SchemaID   string
	Expression string
	Variables  map[string]any
	ShipmentID pulid.ID
	Scenarios  []Scenario
	Policy     *pagedraft.FormulaPolicy
	// declaredOnly checks the expression with Variables alone, for a proposal:
	// the studio holds only its declared variables.
	declaredOnly bool
}

type TestOutcome struct {
	SchemaID  string                     `json:"schemaId"`
	Check     pagedraft.FormulaCheck     `json:"check"`
	Policy    *pagedraft.FormulaPolicy   `json:"policy,omitempty"`
	Warnings  []string                   `json:"warnings,omitempty"`
	Resolved  map[string]any             `json:"resolvedVariables,omitempty"`
	Scenarios []pagedraft.PricedScenario `json:"scenarios,omitempty"`
}

type ProposeRequest struct {
	TenantInfo  pagination.TenantInfo
	SchemaID    string
	Expression  string
	Variables   []pagedraft.FormulaVariable
	Details     pagedraft.FormulaDetails
	Explanation string
	Scenarios   []Scenario
}

func SchemaIDOrDefault(schemaID string) string {
	if trimmed := strings.TrimSpace(schemaID); trimmed != "" {
		return trimmed
	}

	return DefaultSchemaID
}

func (s *Service) Reference(
	ctx context.Context,
	tenant pagination.TenantInfo,
	schemaID string,
) (*Reference, error) {
	schemaID = SchemaIDOrDefault(schemaID)
	description, err := s.formulaService.DescribeSchema(schemaID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"schemaId", errortypes.ErrInvalid,
			"There is no formula schema named "+schemaID,
		)
	}

	tables, err := s.rateTables(ctx, tenant)
	if err != nil {
		return nil, err
	}

	catalog, err := rules.Catalog(ctx)
	if err != nil {
		return nil, err
	}

	locale := i18n.FromContext(ctx)
	functions := make([]FunctionReference, 0, len(catalog.Functions)+len(catalog.Builtins))
	for _, group := range [][]rules.FormulaFunction{catalog.Functions, catalog.Builtins} {
		for index := range group {
			function := &group[index]
			functions = append(functions, FunctionReference{
				Name:        function.Name,
				Signature:   function.Signature,
				Description: function.Description.Localize(locale),
				Example:     function.Example.Localize(locale),
				Category:    function.Category,
				Operator:    function.Operator,
				Predicate:   function.Predicate,
			})
		}
	}
	disabled := make([]DisabledReference, 0, len(catalog.Disabled))
	for index := range catalog.Disabled {
		disabled = append(disabled, DisabledReference{
			Name:    catalog.Disabled[index].Name,
			Message: catalog.Disabled[index].Message.Localize(locale),
		})
	}

	return &Reference{
		SchemaID:   schemaID,
		Variables:  description.Variables,
		Functions:  functions,
		Disabled:   disabled,
		RateTables: tables,
	}, nil
}

func (s *Service) rateTables(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]RateTable, error) {
	data, err := s.rateMatrixRepo.GetLookupData(
		ctx,
		&ratematrixports.GetRateMatrixLookupDataRequest{TenantInfo: tenant},
	)
	if err != nil {
		return nil, fmt.Errorf("read the rate tables a formula may look up: %w", err)
	}

	tables := make([]RateTable, 0, len(data))
	for _, entry := range data {
		if entry == nil || entry.Matrix == nil {
			continue
		}
		table := RateTable{
			Code: entry.Matrix.Code,
			Name: entry.Matrix.Name,
			Axes: len(entry.Matrix.Dimensions),
		}
		switch table.Axes {
		case 1:
			table.Functions = "lookup, lookupOr"
		case 2:
			table.Functions = "lookup2, lookup2Or"
		default:
			continue
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func (s *Service) Test(ctx context.Context, req *TestRequest) (*TestOutcome, error) {
	if err := validateTest(req); err != nil {
		return nil, err
	}

	schemaID := SchemaIDOrDefault(req.SchemaID)
	check := policedRequest(&formulatemplateservice.TestExpressionRequest{
		Expression: req.Expression,
		SchemaID:   schemaID,
		Variables:  checkVariables(req),
		TenantInfo: req.TenantInfo,
	}, req.Policy)
	if req.ShipmentID.IsNotNil() {
		shipmentID := req.ShipmentID
		check.ShipmentID = &shipmentID
	}

	result, err := s.templateService.TestExpression(ctx, check)
	if err != nil {
		return nil, err
	}
	locale := i18n.FromContext(ctx)
	outcome := &TestOutcome{
		SchemaID: schemaID,
		Check:    checkOf(locale, result),
		Policy:   req.Policy.Resolved(),
		Resolved: result.ResolvedVariables,
	}
	for index := range result.Diagnostics {
		diagnostic := &result.Diagnostics[index]
		if diagnostic.Severity == rules.SeverityWarning {
			outcome.Warnings = append(outcome.Warnings, diagnostic.Message.Localize(locale))
		}
	}
	if outcome.Check.Valid {
		outcome.Scenarios = s.price(ctx, &pricing{
			tenant:     req.TenantInfo,
			schemaID:   schemaID,
			expression: req.Expression,
			defaults:   req.Variables,
			policy:     req.Policy,
		}, req.Scenarios)
	}

	return outcome, nil
}

func policedRequest(
	req *formulatemplateservice.TestExpressionRequest,
	policy *pagedraft.FormulaPolicy,
) *formulatemplateservice.TestExpressionRequest {
	charge := policy.ChargePolicy()
	req.MinCharge = charge.MinCharge
	req.MaxCharge = charge.MaxCharge
	req.RoundingMode = charge.RoundingMode
	req.RoundingPrecision = charge.RoundingPrecision

	return req
}

// checkVariables are the values the expression is checked with: the shared
// ones, and for a name only the loads set, the first load's value. A template
// variable given a value per load is as declared as one given once; checking
// with the shared values alone refused the expression as naming an unknown
// variable before any load was priced.
func checkVariables(req *TestRequest) map[string]any {
	values := maps.Clone(req.Variables)
	if req.declaredOnly {
		return values
	}
	if values == nil {
		values = make(map[string]any)
	}
	for _, scenario := range req.Scenarios {
		for name, value := range scenario.Variables {
			if _, set := values[name]; !set && value != nil {
				values[name] = value
			}
		}
	}

	return values
}

func (s *Service) Propose(
	ctx context.Context,
	req *ProposeRequest,
) (*pagedraft.FormulaProposal, error) {
	if err := validatePropose(req); err != nil {
		return nil, err
	}

	defaults := make(map[string]any, len(req.Variables))
	for _, variable := range req.Variables {
		if variable.DefaultValue != nil {
			defaults[variable.Name] = variable.DefaultValue
		}
	}

	outcome, err := s.Test(ctx, &TestRequest{
		TenantInfo:   req.TenantInfo,
		SchemaID:     req.SchemaID,
		Expression:   req.Expression,
		Variables:    defaults,
		Scenarios:    req.Scenarios,
		Policy:       req.Details.Policy,
		declaredOnly: true,
	})
	if err != nil {
		return nil, err
	}

	details := req.Details.Normalized()
	check := outcome.Check
	if check.Valid && len(details.Breakdown) > 0 {
		check = s.checkBreakdown(ctx, req, outcome.SchemaID, defaults, details.Breakdown, check)
	}

	return &pagedraft.FormulaProposal{
		SchemaID:       outcome.SchemaID,
		Expression:     strings.TrimSpace(req.Expression),
		Variables:      req.Variables,
		FormulaDetails: details,
		Explanation:    strings.TrimSpace(req.Explanation),
		Check:          check,
		Scenarios:      outcome.Scenarios,
	}, nil
}

func (s *Service) checkBreakdown(
	ctx context.Context,
	req *ProposeRequest,
	schemaID string,
	defaults map[string]any,
	lines []pagedraft.BreakdownLine,
	check pagedraft.FormulaCheck,
) pagedraft.FormulaCheck {
	definitions := make([]*formulatypes.BreakdownDefinition, 0, len(lines))
	for _, line := range lines {
		definitions = append(definitions, &formulatypes.BreakdownDefinition{
			Name:       line.Name,
			Label:      line.Label,
			Expression: line.Expression,
		})
	}

	result, err := s.templateService.TestExpression(ctx, policedRequest(
		&formulatemplateservice.TestExpressionRequest{
			Expression: req.Expression,
			SchemaID:   schemaID,
			Variables:  defaults,
			TenantInfo: req.TenantInfo,
			Breakdowns: definitions,
		}, req.Details.Policy))
	if err != nil {
		return pagedraft.FormulaCheck{Error: errortypes.LocalizedText(ctx, err)}
	}
	locale := i18n.FromContext(ctx)
	for index, line := range lines {
		scope := formula.PreviewScope(index)
		for position := range result.Diagnostics {
			diagnostic := &result.Diagnostics[position]
			if diagnostic.Scope == scope && diagnostic.IsError() {
				return lineFailure(line, diagnostic.Message.Localize(locale), diagnostic.Code)
			}
		}
		if result.Error != nil && result.Error.Scope == scope {
			return lineFailure(line, result.Error.Err.Localize(locale), result.Error.Err.Code)
		}
	}
	if result.FirstErrorDiagnostic() != nil || result.Result == nil {
		return checkOf(locale, result)
	}

	return check
}

type pricing struct {
	tenant     pagination.TenantInfo
	schemaID   string
	expression string
	defaults   map[string]any
	policy     *pagedraft.FormulaPolicy
}

func (s *Service) price(
	ctx context.Context,
	run *pricing,
	scenarios []Scenario,
) []pagedraft.PricedScenario {
	priced := make([]pagedraft.PricedScenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		variables := make(map[string]any, len(run.defaults)+len(scenario.Variables))
		maps.Copy(variables, run.defaults)
		for name, value := range scenario.Variables {
			if value != nil {
				variables[name] = value
			}
		}

		result, err := s.templateService.TestExpression(ctx, policedRequest(
			&formulatemplateservice.TestExpressionRequest{
				Expression: run.expression,
				SchemaID:   run.schemaID,
				Variables:  variables,
				TenantInfo: run.tenant,
			}, run.policy))
		check := pagedraft.FormulaCheck{Error: errortypes.LocalizedText(ctx, err)}
		if err == nil {
			check = checkOf(i18n.FromContext(ctx), result)
		}
		priced = append(priced, pagedraft.PricedScenario{
			Name:        strings.TrimSpace(scenario.Name),
			Description: strings.TrimSpace(scenario.Description),
			Variables:   variables,
			Amount:      check.Result,
			Valid:       check.Valid,
			Error:       check.Error,
			Code:        check.Code,
		})
	}

	return priced
}

func lineFailure(line pagedraft.BreakdownLine, message, code string) pagedraft.FormulaCheck {
	return pagedraft.FormulaCheck{
		Error: fmt.Sprintf(
			"Breakdown line %q (%s) does not run: %s",
			line.Label,
			line.Name,
			strings.TrimSpace(message),
		),
		Code: code,
	}
}

func checkOf(
	locale i18n.Locale,
	result *formulatemplateservice.TestExpressionResponse,
) pagedraft.FormulaCheck {
	if result == nil {
		return pagedraft.FormulaCheck{
			Error: i18n.Translate(locale, "The expression could not be tested"),
		}
	}
	if diagnostic := result.FirstErrorDiagnostic(); diagnostic != nil {
		return pagedraft.FormulaCheck{
			Error: diagnostic.Message.Localize(locale),
			Code:  diagnostic.Code,
		}
	}
	if result.Error != nil {
		return pagedraft.FormulaCheck{
			Error: result.Error.Err.Localize(locale),
			Code:  result.Error.Err.Code,
		}
	}
	if !result.Valid || result.Result == nil {
		return pagedraft.FormulaCheck{
			Error: i18n.Translate(locale, "The expression could not be tested"),
		}
	}

	return pagedraft.FormulaCheck{Valid: true, Result: result.Result.String()}
}

func validateTest(req *TestRequest) error {
	multiErr := errortypes.NewMultiError()

	expression := strings.TrimSpace(req.Expression)
	switch {
	case expression == "":
		multiErr.Add("expression", errortypes.ErrRequired, "An expression is required")
	case utf8.RuneCountInString(expression) > pagedraft.MaxExpressionLength:
		multiErr.Add("expression", errortypes.ErrInvalid,
			"Expression cannot exceed {0} characters", pagedraft.MaxExpressionLength)
	}
	validateValues(multiErr, "variables", req.Variables)
	req.Policy.Validate("policy", multiErr)
	if len(req.Scenarios) > MaxScenarios {
		multiErr.Add("scenarios", errortypes.ErrInvalid,
			"At most {0} scenarios are priced at once", MaxScenarios)
	}
	for idx, scenario := range req.Scenarios {
		field := fmt.Sprintf("scenarios[%d]", idx)
		name := strings.TrimSpace(scenario.Name)
		if name == "" {
			multiErr.Add(field+".name", errortypes.ErrRequired, "A scenario needs a name")
		}
		if utf8.RuneCountInString(name) > MaxScenarioNameLen ||
			utf8.RuneCountInString(scenario.Description) > pagedraft.MaxVariableTextLength {
			multiErr.Add(field, errortypes.ErrInvalid, "Scenario text is too long")
		}
		validateValues(multiErr, field+".variables", scenario.Variables)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateValues(multiErr *errortypes.MultiError, field string, values map[string]any) {
	if len(values) > maxScenarioVars {
		multiErr.Add(field, errortypes.ErrInvalid, "Too many variable values")
		return
	}
	for name, value := range values {
		if !pagedraft.ValidValuePath(name) {
			multiErr.Add(field+"."+name, errortypes.ErrInvalid,
				"Variable name is invalid: use a formula variable's name or a schema field "+
					"path such as serviceType.code")
			continue
		}
		if !pagedraft.ScalarValue(value, pagedraft.MaxVariableTextLength) {
			multiErr.Add(field+"."+name, errortypes.ErrInvalid,
				"A value must be a number, text or true/false")
		}
	}
}

func validatePropose(req *ProposeRequest) error {
	multiErr := errortypes.NewMultiError()
	if utf8.RuneCountInString(req.Explanation) > MaxExplanationRunes {
		multiErr.Add("explanation", errortypes.ErrInvalid,
			"Explanation cannot exceed {0} characters", MaxExplanationRunes)
	}
	if strings.TrimSpace(req.Explanation) == "" {
		multiErr.Add("explanation", errortypes.ErrRequired,
			"Say in plain words what the formula charges")
	}
	draft := pagedraft.Formula{
		SchemaID:   SchemaIDOrDefault(req.SchemaID),
		Expression: req.Expression,
		Variables:  req.Variables,
	}
	(&pagedraft.Draft{Surface: pagedraft.SurfaceFormula, Formula: &draft}).
		Validate("", multiErr)
	req.Details.Validate("", multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
