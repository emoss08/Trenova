/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package formula

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/formula/expression"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.FormulaTemplateRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Variables    *variables.Registry
	Schemas      *schema.SchemaRegistry
	Resolver     *schema.DefaultDataResolver
}

type Service struct {
	l         *zerolog.Logger
	repo      repositories.FormulaTemplateRepository
	ps        services.PermissionService
	as        services.AuditService
	variables *variables.Registry
	schemas   *schema.SchemaRegistry
	resolver  *schema.DefaultDataResolver
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "formula").
		Logger()

	return &Service{
		l:         &log,
		repo:      p.Repo,
		ps:        p.PermService,
		as:        p.AuditService,
		variables: p.Variables,
		schemas:   p.Schemas,
		resolver:  p.Resolver,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	opts *repositories.ListFormulaTemplateOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, t := range result.Items {
		options = append(options, &types.SelectOption{
			Value: t.GetID(),
			Label: t.Name,
		})
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListFormulaTemplateOptions,
) (*ports.ListResult[*formulatemplate.FormulaTemplate], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read formula templates",
		)
	}

	return s.repo.List(ctx, opts)
}

func (s *Service) Get(
	ctx context.Context,
	opts *repositories.GetFormulaTemplateByIDOptions,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("templateID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this formula template",
		)
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get formula template")
		return nil, err
	}

	return entity, nil
}

func (s *Service) GetByCategory(
	ctx context.Context,
	category formulatemplate.Category,
	orgID pulid.ID,
	buID pulid.ID,
	userID pulid.ID,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := s.l.With().
		Str("operation", "GetByCategory").
		Str("category", category.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionRead,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read formula templates",
		)
	}

	return s.repo.GetByCategory(ctx, category, orgID, buID)
}

func (s *Service) Create(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", template.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionCreate,
				BusinessUnitID: template.BusinessUnitID,
				OrganizationID: template.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create a formula template",
		)
	}

	// valCtx := &validator.ValidationContext{
	// 	IsCreate: true,
	// 	IsUpdate: false,
	// }

	// if err := s.v.Validate(ctx, valCtx, template); err != nil {
	// 	return nil, err
	// }

	// * Validate the expression syntax
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
			Action:         permission.ActionCreate,
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
		log.Error().Err(err).Msg("failed to log formula template creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", template.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionUpdate,
				BusinessUnitID: template.BusinessUnitID,
				OrganizationID: template.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this formula template",
		)
	}

	// valCtx := &validator.ValidationContext{
	// 	IsUpdate: true,
	// 	IsCreate: false,
	// }
	//
	// if err := s.v.Validate(ctx, valCtx, template); err != nil {
	// 	return nil, err
	// }

	// * Validate the expression syntax
	if err := s.validateExpression(template); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetFormulaTemplateByIDOptions{
		ID:    template.ID,
		OrgID: template.OrganizationID,
		BuID:  template.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, template)
	if err != nil {
		log.Error().Err(err).Msg("failed to update formula template")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFormulaTemplate,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
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
		log.Error().Err(err).Msg("failed to log formula template update")
	}

	return updatedEntity, nil
}

func (s *Service) SetDefault(
	ctx context.Context,
	req *repositories.SetDefaultFormulaTemplateRequest,
) error {
	log := s.l.With().
		Str("operation", "SetDefault").
		Str("templateID", req.TemplateID.String()).
		Str("category", req.Category.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionUpdate,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to set default formula templates",
		)
	}

	return s.repo.SetDefault(ctx, req)
}

func (s *Service) Delete(
	ctx context.Context,
	id pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
	userID pulid.ID,
) error {
	log := s.l.With().
		Str("operation", "Delete").
		Str("templateID", id.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionDelete,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to delete this formula template",
		)
	}

	// Get the template before deletion for audit logging
	template, err := s.repo.GetByID(ctx, &repositories.GetFormulaTemplateByIDOptions{
		ID:    id,
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id, orgID, buID); err != nil {
		log.Error().Err(err).Msg("failed to delete formula template")
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFormulaTemplate,
			ResourceID:     id.String(),
			Action:         permission.ActionDelete,
			UserID:         userID,
			PreviousState:  jsonutils.MustToJSON(template),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment("Formula template deleted"),
		audit.WithCategory("configuration"),
		audit.WithMetadata(map[string]any{
			"name":     template.Name,
			"category": template.Category.String(),
		}),
		audit.WithTags("formula-template", "category-"+template.Category.String()),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log formula template deletion")
	}

	return nil
}

// * CalculateShipmentRate calculates the rate for a shipment using a formula template
func (s *Service) CalculateShipmentRate(
	ctx context.Context,
	templateID pulid.ID,
	shp *shipment.Shipment,
	userID pulid.ID,
) (decimal.Decimal, error) {
	log := s.l.With().
		Str("operation", "CalculateShipmentRate").
		Str("templateID", templateID.String()).
		Str("shipmentID", shp.ID.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionRead,
				BusinessUnitID: shp.BusinessUnitID,
				OrganizationID: shp.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return decimal.Zero, err
	}

	if !result.Allowed {
		return decimal.Zero, errors.NewAuthorizationError(
			"You do not have permission to calculate shipment rates",
		)
	}

	// * Load the formula template
	template, err := s.repo.GetByID(ctx, &repositories.GetFormulaTemplateByIDOptions{
		ID:     templateID,
		OrgID:  shp.OrganizationID,
		BuID:   shp.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to load formula template")
		return decimal.Zero, fmt.Errorf("failed to load formula template: %w", err)
	}

	// * Validate template is active
	if !template.IsActive {
		return decimal.Zero, fmt.Errorf("formula template %s is not active", templateID)
	}

	// * Create variable context with shipment data
	varCtx, err := s.createShipmentContext(template, shp)
	if err != nil {
		log.Error().Err(err).Msg("failed to create variable context")
		return decimal.Zero, fmt.Errorf("failed to create variable context: %w", err)
	}

	// * Create evaluator and evaluate the expression
	evaluator := expression.NewEvaluator(s.variables)
	value, err := evaluator.Evaluate(ctx, template.Expression, varCtx)
	if err != nil {
		log.Error().Err(err).Msg("failed to evaluate formula")
		return decimal.Zero, fmt.Errorf("failed to evaluate formula: %w", err)
	}

	// * Convert result to decimal
	rate, err := s.toDecimal(value)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert result to decimal")
		return decimal.Zero, fmt.Errorf("failed to convert result to decimal: %w", err)
	}

	// * Apply min/max constraints
	rate = s.applyRateConstraints(rate, template)

	log.Info().
		Str("calculatedRate", rate.String()).
		Msg("formula rate calculated successfully")

	return rate, nil
}

// * validateExpression validates a formula template expression
func (s *Service) validateExpression(
	template *formulatemplate.FormulaTemplate,
) error {
	// * Parse the expression to check syntax
	tokens, err := expression.NewTokenizer(template.Expression).Tokenize()
	if err != nil {
		return fmt.Errorf("invalid expression syntax: %w", err)
	}

	// * Parse to AST
	parser := expression.NewParser(tokens)
	_, err = parser.Parse()
	if err != nil {
		return fmt.Errorf("expression parsing failed: %w", err)
	}

	// * Validate required variables exist
	for _, templateVar := range template.Variables {
		if templateVar.Required {
			if _, err := s.variables.Get(templateVar.Name); err != nil {
				// * Check if it's a metadata variable or computed field
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

// createShipmentContext creates a variable context for a shipment
func (s *Service) createShipmentContext(
	template *formulatemplate.FormulaTemplate,
	shp *shipment.Shipment,
) (variables.VariableContext, error) {
	// * Create variable context with shipment data
	varCtx := variables.NewDefaultContext(shp, s.resolver)

	// * Add template parameters as metadata
	if template.Parameters != nil {
		for _, param := range template.Parameters {
			// * Use default value if provided
			if param.DefaultValue != nil {
				varCtx.SetMetadata(param.Name, param.DefaultValue)
			}
		}
	}

	// * Add common metadata that might be used in formulas
	// * These would typically come from other sources (e.g., route planning service)
	if len(shp.Moves) > 0 {
		// * Calculate total mileage from moves
		totalMileage := 0.0
		for _, move := range shp.Moves {
			if move.Distance != nil {
				totalMileage += float64(*move.Distance)
			}
		}
		varCtx.SetMetadata("mileage", totalMileage)
		varCtx.SetMetadata("distance", totalMileage)
	}

	// * Add stop count metadata
	totalStops := 0
	for _, move := range shp.Moves {
		totalStops += len(move.Stops)
	}
	varCtx.SetMetadata("total_stops", float64(totalStops))
	varCtx.SetMetadata("move_count", float64(len(shp.Moves)))
	varCtx.SetMetadata("commodity_count", float64(len(shp.Commodities)))

	return varCtx, nil
}

// toDecimal converts a result to decimal
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

// * applyRateConstraints applies min/max rate constraints
func (s *Service) applyRateConstraints(
	rate decimal.Decimal,
	template *formulatemplate.FormulaTemplate,
) decimal.Decimal {
	// * Apply minimum rate
	if template.MinRate != nil {
		minRate := decimal.NewFromFloat(*template.MinRate)
		if rate.LessThan(minRate) {
			rate = minRate
		}
	}

	// * Apply maximum rate
	if template.MaxRate != nil {
		maxRate := decimal.NewFromFloat(*template.MaxRate)
		if rate.GreaterThan(maxRate) {
			rate = maxRate
		}
	}

	return rate
}

// * isMetadataVariable checks if a variable is available via metadata
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

// * TestFormula allows testing a formula expression with sample data
func (s *Service) TestFormula(
	ctx context.Context,
	req *formula.TestFormulaRequest,
) (*formula.TestFormulaResponse, error) {
	log := s.l.With().
		Str("operation", "TestFormula").
		Logger()

	// * Check permissions - user must have read permission on formula templates
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceFormulaTemplate,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to test formulas",
		)
	}

	// * Create test context
	testCtx, err := s.createTestContext(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to create test context")
		return nil, fmt.Errorf("failed to create test context: %w", err)
	}

	// * Parse and validate the expression
	tokens, err := expression.NewTokenizer(req.Expression).Tokenize()
	if err != nil {
		return &formula.TestFormulaResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid expression syntax: %v", err),
		}, nil
	}

	// * Parse to AST
	parser := expression.NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return &formula.TestFormulaResponse{
			Success: false,
			Error:   fmt.Sprintf("Expression parsing failed: %v", err),
		}, nil
	}

	// * Extract used variables
	usedVars := s.extractUsedVariables(ast)

	// * Evaluate the expression
	evaluator := expression.NewEvaluator(s.variables)
	value, err := evaluator.Evaluate(ctx, req.Expression, testCtx)
	if err != nil {
		return &formula.TestFormulaResponse{
			Success:       false,
			Error:         fmt.Sprintf("Evaluation failed: %v", err),
			UsedVariables: usedVars,
		}, nil
	}

	// * Convert to decimal
	rate, err := s.toDecimal(value)
	if err != nil {
		return &formula.TestFormulaResponse{
			Success:       false,
			Error:         fmt.Sprintf("Failed to convert result to decimal: %v", err),
			RawResult:     value,
			UsedVariables: usedVars,
		}, nil
	}

	// * Apply constraints if provided
	if req.MinRate != nil && rate.LessThan(decimal.NewFromFloat(*req.MinRate)) {
		rate = decimal.NewFromFloat(*req.MinRate)
	}
	if req.MaxRate != nil && rate.GreaterThan(decimal.NewFromFloat(*req.MaxRate)) {
		rate = decimal.NewFromFloat(*req.MaxRate)
	}

	rateFloat, _ := rate.Float64()
	return &formula.TestFormulaResponse{
		Success:          true,
		Result:           rateFloat,
		RawResult:        value,
		UsedVariables:    usedVars,
		EvaluationSteps:  s.generateEvaluationSteps(ctx, req.Expression, testCtx),
		AvailableContext: s.getAvailableContext(testCtx),
	}, nil
}

// * createTestContext creates a variable context for testing
func (s *Service) createTestContext(
	req *formula.TestFormulaRequest,
) (variables.VariableContext, error) {
	// * Create a mock shipment with test data
	mockShipment := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
		Status:         shipment.StatusNew,
		ProNumber:      "TEST-001",
		BOL:            "TEST-BOL-001",
	}

	// * Apply test data to shipment
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

	// * Create variable context
	varCtx := variables.NewDefaultContext(mockShipment, s.resolver)

	// * Add metadata from test data
	if req.TestData != nil {
		// * Add common test metadata
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

		// * Add any custom metadata
		for key, value := range req.TestData {
			if !s.isStandardField(key) {
				varCtx.SetMetadata(key, value)
			}
		}
	}

	// * Add template parameters if provided
	if req.Parameters != nil {
		for key, value := range req.Parameters {
			varCtx.SetMetadata(key, value)
		}
	}

	return varCtx, nil
}

// * extractUsedVariables extracts variable names from the AST
func (s *Service) extractUsedVariables(node expression.Node) []string {
	vars := make(map[string]bool)
	s.extractVariablesRecursive(node, vars)

	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}
	return result
}

// * extractVariablesRecursive recursively extracts variables from AST nodes
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

// * generateEvaluationSteps generates a step-by-step evaluation trace
func (s *Service) generateEvaluationSteps(
	ctx context.Context,
	expr string,
	varCtx variables.VariableContext,
) []formula.EvaluationStep {
	// Use the tracing evaluator to get actual evaluation steps
	tracingEval := expression.NewTracingEvaluator(s.variables)
	_, traceSteps, err := tracingEval.EvaluateWithTrace(ctx, expr, varCtx)

	// Convert trace steps to evaluation steps
	evalSteps := make([]formula.EvaluationStep, 0, len(traceSteps))
	for _, trace := range traceSteps {
		evalSteps = append(evalSteps, formula.EvaluationStep{
			Step:        trace.Step,
			Description: trace.Description,
			Result:      trace.Result,
		})
	}

	// If there was an error, the trace will still contain the steps up to the error
	if err != nil && len(evalSteps) == 0 {
		// Fallback if no trace was generated
		evalSteps = []formula.EvaluationStep{
			{
				Step:        "Parse expression",
				Description: fmt.Sprintf("Parsing: %s", expr),
				Result:      fmt.Sprintf("Failed: %v", err),
			},
		}
	}

	return evalSteps
}

// * getAvailableContext returns available variables and their current values
func (s *Service) getAvailableContext(varCtx variables.VariableContext) map[string]any {
	availableCtx := make(map[string]any)

	// * Get all registered variables and try to resolve them
	for _, name := range s.variables.ListNames() {
		if variable, err := s.variables.Get(name); err == nil {
			if value, err := variable.Resolve(varCtx); err == nil && value != nil {
				availableCtx[name] = value
			}
		}
	}

	return availableCtx
}

// * isStandardField checks if a field is a standard shipment field
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
