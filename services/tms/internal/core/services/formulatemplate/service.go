package formulatemplate

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/formula/expression"
	"github.com/emoss08/trenova/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/formula/variables"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.FormulaTemplateRepository
	AuditService services.AuditService
	Variables    *variables.Registry
	Schemas      *schema.Registry
	Resolver     *schema.DefaultDataResolver
}

type Service struct {
	l         *zap.Logger
	repo      repositories.FormulaTemplateRepository
	as        services.AuditService
	variables *variables.Registry
	schemas   *schema.Registry
	resolver  *schema.DefaultDataResolver
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:         p.Logger.Named("service.formulatemplate"),
		as:        p.AuditService,
		variables: p.Variables,
		schemas:   p.Schemas,
		resolver:  p.Resolver,
	}
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListFormulaTemplateRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return s.repo.List(ctx, opts)
}

func (s *Service) Get(
	ctx context.Context,
	opts *repositories.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByID(ctx, opts)
}

func (s *Service) GetByCategory(
	ctx context.Context,
	category formulatemplate.Category,
	orgID pulid.ID,
	buID pulid.ID,
) ([]*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByCategory(ctx, category, orgID, buID)
}

func (s *Service) Create(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("name", template.Name),
		zap.String("buID", template.BusinessUnitID.String()),
		zap.String("orgID", template.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	if err := s.validateExpression(template); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, template)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFormulaTemplate,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Formula template created"),
		audit.WithCategory("configuration"),
		audit.WithMetadata(map[string]any{
			"name":     createdEntity.Name,
			"category": createdEntity.Category.String(),
		}),
		audit.WithTags("formula-template", "category-"+createdEntity.Category.String()),
	)
	if err != nil {
		log.Error("failed to log formula template creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("name", template.Name),
		zap.String("buID", template.BusinessUnitID.String()),
		zap.String("orgID", template.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	if err := s.validateExpression(template); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetFormulaTemplateByIDRequest{
		ID:    template.ID,
		OrgID: template.OrganizationID,
		BuID:  template.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, template)
	if err != nil {
		log.Error("failed to update formula template", zap.Error(err))
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFormulaTemplate,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Formula template updated"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCategory("configuration"),
		audit.WithMetadata(map[string]any{
			"name":     updatedEntity.Name,
			"category": updatedEntity.Category.String(),
		}),
		audit.WithTags("formula-template", "category-"+updatedEntity.Category.String()),
	)
	if err != nil {
		log.Error("failed to log formula template update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) SetDefault(
	ctx context.Context,
	req *repositories.SetDefaultFormulaTemplateRequest,
) error {
	return s.repo.SetDefault(ctx, req)
}

func (s *Service) CalculateShipmentRate(
	ctx context.Context,
	templateID pulid.ID,
	shp *shipment.Shipment,
	userID pulid.ID,
) (decimal.Decimal, error) {
	log := s.l.With(
		zap.String("operation", "CalculateShipmentRate"),
		zap.String("templateID", templateID.String()),
		zap.String("shipmentID", shp.ID.String()),
		zap.String("userID", userID.String()),
	)

	template, err := s.repo.GetByID(ctx, &repositories.GetFormulaTemplateByIDRequest{
		ID:     templateID,
		OrgID:  shp.OrganizationID,
		BuID:   shp.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		log.Error("failed to load formula template", zap.Error(err))
		return decimal.Zero, fmt.Errorf("failed to load formula template: %w", err)
	}

	if !template.IsActive {
		return decimal.Zero, fmt.Errorf("formula template %s is not active", templateID)
	}

	varCtx := s.createShipmentContext(template, shp)

	evaluator := expression.NewEvaluator(s.variables)
	value, err := evaluator.Evaluate(ctx, template.Expression, varCtx)
	if err != nil {
		log.Error("failed to evaluate formula", zap.Error(err))
		return decimal.Zero, fmt.Errorf("failed to evaluate formula: %w", err)
	}

	rate, err := s.toDecimal(value)
	if err != nil {
		log.Error("failed to convert result to decimal", zap.Error(err))
		return decimal.Zero, fmt.Errorf("failed to convert result to decimal: %w", err)
	}

	rate = s.applyRateConstraints(rate, template)

	log.Info("formula rate calculated successfully", zap.String("calculatedRate", rate.String()))

	return rate, nil
}

func (s *Service) validateExpression(
	template *formulatemplate.FormulaTemplate,
) error {
	tokens, err := expression.NewTokenizer(template.Expression).Tokenize()
	if err != nil {
		return fmt.Errorf("invalid expression syntax: %w", err)
	}

	parser := expression.NewParser(tokens)
	_, err = parser.Parse()
	if err != nil {
		return fmt.Errorf("expression parsing failed: %w", err)
	}

	for _, templateVar := range template.Variables {
		if templateVar.Required {
			if _, err = s.variables.Get(templateVar.Name); err != nil {
				if !s.isMetadataVariable(templateVar.Name) {
					return fmt.Errorf(
						"required variable '%s' not found in registry",
						templateVar.Name,
					)
				}
			}
		}
	}

	return nil
}

func (s *Service) createShipmentContext(
	template *formulatemplate.FormulaTemplate,
	shp *shipment.Shipment,
) variables.VariableContext {
	varCtx := variables.NewDefaultContext(shp, s.resolver)

	if template.Parameters != nil {
		for _, param := range template.Parameters {
			if param.DefaultValue != nil {
				varCtx.SetMetadata(param.Name, param.DefaultValue)
			}
		}
	}

	if len(shp.Moves) > 0 {
		totalMileage := 0.0
		for _, move := range shp.Moves {
			if move.Distance != nil {
				totalMileage += float64(*move.Distance)
			}
		}
		varCtx.SetMetadata("mileage", totalMileage)
		varCtx.SetMetadata("distance", totalMileage)
	}

	totalStops := 0
	for _, move := range shp.Moves {
		totalStops += len(move.Stops)
	}
	varCtx.SetMetadata("total_stops", float64(totalStops))
	varCtx.SetMetadata("move_count", float64(len(shp.Moves)))
	varCtx.SetMetadata("commodity_count", float64(len(shp.Commodities)))

	return varCtx
}

func (s *Service) toDecimal(result any) (decimal.Decimal, error) {
	switch v := result.(type) {
	case float64:
		return decimal.NewFromFloat(v), nil
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case decimal.Decimal:
		return v, nil
	case string:
		return decimal.NewFromString(v)
	default:
		return decimal.Zero, fmt.Errorf("cannot convert %T to decimal", result)
	}
}

func (s *Service) applyRateConstraints(
	rate decimal.Decimal,
	template *formulatemplate.FormulaTemplate,
) decimal.Decimal {
	if template.MinRate != nil {
		minRate := decimal.NewFromFloat(*template.MinRate)
		if rate.LessThan(minRate) {
			rate = minRate
		}
	}

	if template.MaxRate != nil {
		maxRate := decimal.NewFromFloat(*template.MaxRate)
		if rate.GreaterThan(maxRate) {
			rate = maxRate
		}
	}

	return rate
}

func (s *Service) isMetadataVariable(name string) bool {
	metadataVars := []string{
		"mileage", "distance", "total_stops", "move_count", "commodity_count",
	}

	for _, v := range metadataVars {
		if v == name {
			return true
		}
	}
	return false
}

func (s *Service) TestFormula(
	ctx context.Context,
	req *TestFormulaRequest,
) (*TestFormulaResponse, error) {
	testCtx := s.createTestContext(req)

	tokens, err := expression.NewTokenizer(req.Expression).Tokenize()
	if err != nil {
		return &TestFormulaResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid expression syntax: %v", err),
		}, nil
	}

	parser := expression.NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return &TestFormulaResponse{
			Success: false,
			Error:   fmt.Sprintf("Expression parsing failed: %v", err),
		}, nil
	}

	usedVars := s.extractUsedVariables(ast)

	evaluator := expression.NewEvaluator(s.variables)
	value, err := evaluator.Evaluate(ctx, req.Expression, testCtx)
	if err != nil {
		return &TestFormulaResponse{
			Success:       false,
			Error:         fmt.Sprintf("Evaluation failed: %v", err),
			UsedVariables: usedVars,
		}, nil
	}

	rate, err := s.toDecimal(value)
	if err != nil {
		return &TestFormulaResponse{
			Success:       false,
			Error:         fmt.Sprintf("Failed to convert result to decimal: %v", err),
			RawResult:     value,
			UsedVariables: usedVars,
		}, nil
	}

	if req.MinRate != nil && rate.LessThan(decimal.NewFromFloat(*req.MinRate)) {
		rate = decimal.NewFromFloat(*req.MinRate)
	}
	if req.MaxRate != nil && rate.GreaterThan(decimal.NewFromFloat(*req.MaxRate)) {
		rate = decimal.NewFromFloat(*req.MaxRate)
	}

	rateFloat, _ := rate.Float64()
	return &TestFormulaResponse{
		Success:          true,
		Result:           rateFloat,
		RawResult:        value,
		UsedVariables:    usedVars,
		EvaluationSteps:  s.generateEvaluationSteps(ctx, req.Expression, testCtx),
		AvailableContext: s.getAvailableContext(testCtx),
	}, nil
}

func (s *Service) createTestContext(
	req *TestFormulaRequest,
) variables.VariableContext {
	mockShipment := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
		Status:         shipment.StatusNew,
		ProNumber:      "TEST-001",
		BOL:            "TEST-BOL-001",
	}

	if req.TestData != nil {
		if weight, ok := req.TestData["weight"].(float64); ok {
			weightInt := int64(weight)
			mockShipment.Weight = &weightInt
		}
		if pieces, ok := req.TestData["pieces"].(float64); ok {
			piecesInt := int64(pieces)
			mockShipment.Pieces = &piecesInt
		}
		if tempMin, ok := req.TestData["temperature_min"].(float64); ok {
			tempMinInt := int16(tempMin)
			mockShipment.TemperatureMin = &tempMinInt
		}
		if tempMax, ok := req.TestData["temperature_max"].(float64); ok {
			tempMaxInt := int16(tempMax)
			mockShipment.TemperatureMax = &tempMaxInt
		}
	}

	varCtx := variables.NewDefaultContext(mockShipment, s.resolver)

	if req.TestData != nil { //nolint:nestif // this is fine for testing
		if distance, ok := req.TestData["distance"].(float64); ok {
			varCtx.SetMetadata("distance", distance)
			varCtx.SetMetadata("mileage", distance)
		}
		if stops, ok := req.TestData["total_stops"].(float64); ok {
			varCtx.SetMetadata("total_stops", stops)
		}
		if moveCount, ok := req.TestData["move_count"].(float64); ok {
			varCtx.SetMetadata("move_count", moveCount)
		}
		if commodityCount, ok := req.TestData["commodity_count"].(float64); ok {
			varCtx.SetMetadata("commodity_count", commodityCount)
		}

		for key, value := range req.TestData {
			if !s.isStandardField(key) {
				varCtx.SetMetadata(key, value)
			}
		}
	}

	if req.Parameters != nil {
		for key, value := range req.Parameters {
			varCtx.SetMetadata(key, value)
		}
	}

	return varCtx
}

func (s *Service) extractUsedVariables(node expression.Node) []string {
	vars := make(map[string]bool)
	s.extractVariablesRecursive(node, vars)

	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}
	return result
}

func (s *Service) extractVariablesRecursive(node expression.Node, vars map[string]bool) {
	switch n := node.(type) {
	case *expression.IdentifierNode:
		vars[n.Name] = true
	case *expression.BinaryOpNode:
		s.extractVariablesRecursive(n.Left, vars)
		s.extractVariablesRecursive(n.Right, vars)
	case *expression.UnaryOpNode:
		s.extractVariablesRecursive(n.Operand, vars)
	case *expression.ConditionalNode:
		s.extractVariablesRecursive(n.Condition, vars)
		s.extractVariablesRecursive(n.TrueExpr, vars)
		s.extractVariablesRecursive(n.FalseExpr, vars)
	case *expression.FunctionCallNode:
		for _, arg := range n.Arguments {
			s.extractVariablesRecursive(arg, vars)
		}
	}
}

func (s *Service) generateEvaluationSteps(
	ctx context.Context,
	expr string,
	varCtx variables.VariableContext,
) []EvaluationStep {
	tracingEval := expression.NewTracingEvaluator(s.variables)
	_, traceSteps, err := tracingEval.EvaluateWithTrace(ctx, expr, varCtx)

	evalSteps := make([]EvaluationStep, 0, len(traceSteps))
	for _, trace := range traceSteps {
		evalSteps = append(evalSteps, EvaluationStep{
			Step:        trace.Step,
			Description: trace.Description,
			Result:      trace.Result,
		})
	}

	if err != nil && len(evalSteps) == 0 {
		evalSteps = []EvaluationStep{
			{
				Step:        "Parse expression",
				Description: fmt.Sprintf("Parsing: %s", expr),
				Result:      fmt.Sprintf("Failed: %v", err),
			},
		}
	}

	return evalSteps
}

func (s *Service) getAvailableContext(varCtx variables.VariableContext) map[string]any {
	availableCtx := make(map[string]any)

	for _, name := range s.variables.ListNames() {
		if variable, err := s.variables.Get(name); err == nil {
			if value, vErr := variable.Resolve(varCtx); vErr == nil && value != nil {
				availableCtx[name] = value
			}
		}
	}

	return availableCtx
}

func (s *Service) isStandardField(field string) bool {
	standardFields := map[string]bool{
		"weight":          true,
		"pieces":          true,
		"temperature_min": true,
		"temperature_max": true,
		"pro_number":      true,
		"bol":             true,
	}
	return standardFields[field]
}
