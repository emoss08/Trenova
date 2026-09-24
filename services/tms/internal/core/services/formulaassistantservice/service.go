package formulaassistantservice

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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
	RateMatrixRepo  repositories.RateMatrixRepository
}

type Service struct {
	formulaService  *formula.Service
	templateService *formulatemplateservice.Service
	rateMatrixRepo  repositories.RateMatrixRepository
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

type Reference struct {
	SchemaID   string                                    `json:"schemaId"`
	Variables  []formulatemplatetypes.SchemaVariableInfo `json:"variables"`
	Functions  []formulatemplatetypes.SchemaFunctionInfo `json:"functions"`
	RateTables []RateTable                               `json:"rateTables"`
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
}

type TestOutcome struct {
	SchemaID  string                     `json:"schemaId"`
	Check     pagedraft.FormulaCheck     `json:"check"`
	Warnings  []string                   `json:"warnings,omitempty"`
	Resolved  map[string]any             `json:"resolvedVariables,omitempty"`
	Scenarios []pagedraft.PricedScenario `json:"scenarios,omitempty"`
}

type ProposeRequest struct {
	TenantInfo  pagination.TenantInfo
	SchemaID    string
	Expression  string
	Variables   []pagedraft.FormulaVariable
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

	return &Reference{
		SchemaID:   schemaID,
		Variables:  description.Variables,
		Functions:  description.Functions,
		RateTables: tables,
	}, nil
}

func (s *Service) rateTables(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]RateTable, error) {
	data, err := s.rateMatrixRepo.GetLookupData(
		ctx,
		&repositories.GetRateMatrixLookupDataRequest{TenantInfo: tenant},
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
	check := &formulatemplateservice.TestExpressionRequest{
		Expression: req.Expression,
		SchemaID:   schemaID,
		Variables:  maps.Clone(req.Variables),
		TenantInfo: req.TenantInfo,
	}
	if req.ShipmentID.IsNotNil() {
		shipmentID := req.ShipmentID
		check.ShipmentID = &shipmentID
	}

	result := s.templateService.TestExpression(ctx, check)
	outcome := &TestOutcome{
		SchemaID: schemaID,
		Check:    checkOf(result),
		Resolved: result.ResolvedVariables,
	}
	for _, warning := range result.Warnings {
		outcome.Warnings = append(outcome.Warnings, warning.Message)
	}
	if outcome.Check.Valid {
		outcome.Scenarios = s.price(ctx, req.TenantInfo, schemaID, req.Expression,
			req.Variables, req.Scenarios)
	}

	return outcome, nil
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
		TenantInfo: req.TenantInfo,
		SchemaID:   req.SchemaID,
		Expression: req.Expression,
		Variables:  defaults,
		Scenarios:  req.Scenarios,
	})
	if err != nil {
		return nil, err
	}

	return &pagedraft.FormulaProposal{
		SchemaID:    outcome.SchemaID,
		Expression:  strings.TrimSpace(req.Expression),
		Variables:   req.Variables,
		Explanation: strings.TrimSpace(req.Explanation),
		Check:       outcome.Check,
		Scenarios:   outcome.Scenarios,
	}, nil
}

func (s *Service) price(
	ctx context.Context,
	tenant pagination.TenantInfo,
	schemaID, expression string,
	defaults map[string]any,
	scenarios []Scenario,
) []pagedraft.PricedScenario {
	priced := make([]pagedraft.PricedScenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		variables := make(map[string]any, len(defaults)+len(scenario.Variables))
		maps.Copy(variables, defaults)
		for name, value := range scenario.Variables {
			if value != nil {
				variables[name] = value
			}
		}

		result := s.templateService.TestExpression(ctx,
			&formulatemplateservice.TestExpressionRequest{
				Expression: expression,
				SchemaID:   schemaID,
				Variables:  variables,
				TenantInfo: tenant,
			})
		check := checkOf(result)
		priced = append(priced, pagedraft.PricedScenario{
			Name:        strings.TrimSpace(scenario.Name),
			Description: strings.TrimSpace(scenario.Description),
			Variables:   variables,
			Amount:      check.Result,
			Valid:       check.Valid,
			Error:       check.Error,
		})
	}

	return priced
}

func checkOf(result *formulatemplateservice.TestExpressionResponse) pagedraft.FormulaCheck {
	if result == nil {
		return pagedraft.FormulaCheck{Error: "The expression could not be tested"}
	}
	if !result.Valid {
		message := strings.TrimSpace(result.Error)
		if message == "" {
			message = strings.TrimSpace(result.Message)
		}

		return pagedraft.FormulaCheck{Error: message}
	}

	amount, ok := AmountOf(result.Result)
	if !ok {
		return pagedraft.FormulaCheck{
			Error: "The expression evaluated to something other than an amount",
		}
	}

	return pagedraft.FormulaCheck{Valid: true, Result: amount}
}

func AmountOf(value any) (string, bool) {
	switch v := value.(type) {
	case decimal.Decimal:
		return v.String(), true
	case *decimal.Decimal:
		if v == nil {
			return "", false
		}

		return v.String(), true
	case float64:
		return decimal.NewFromFloat(v).String(), true
	case float32:
		return decimal.NewFromFloat32(v).String(), true
	case int:
		return decimal.NewFromInt(int64(v)).String(), true
	case int32:
		return decimal.NewFromInt32(v).String(), true
	case int64:
		return decimal.NewFromInt(v).String(), true
	default:
		return "", false
	}
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
		if !pagedraft.ValidVariableName(name) {
			multiErr.Add(field+"."+name, errortypes.ErrInvalid, "Variable name is invalid")
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
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
