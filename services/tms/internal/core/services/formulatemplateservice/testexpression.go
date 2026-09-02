package formulatemplateservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/services/formula/contextvariablecache"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type TestExpressionRequest struct {
	Expression string
	SchemaID   string
	Variables  map[string]any
	ShipmentID *pulid.ID
	TenantInfo pagination.TenantInfo
	Breakdowns []*formulatypes.BreakdownDefinition
	MinCharge  decimal.NullDecimal
	MaxCharge  decimal.NullDecimal
	// RoundingMode and RoundingPrecision are the policy under test. An empty
	// mode means the default policy, exactly as it does on a stored template.
	RoundingMode      ratetypes.RoundingMode
	RoundingPrecision int32
}

func (r *TestExpressionRequest) chargePolicy() formulatypes.ChargePolicy {
	return formulatypes.ChargePolicy{
		MinCharge:         r.MinCharge,
		MaxCharge:         r.MaxCharge,
		RoundingMode:      r.RoundingMode,
		RoundingPrecision: r.RoundingPrecision,
	}
}

const (
	msgExpressionValidationFailed = "Expression validation failed"

	// previewEvaluationTimeout is the leash on an interactive preview. It runs
	// on every keystroke, so a runaway expression should fail fast rather than
	// hold the Studio for the engine's full batch ceiling.
	previewEvaluationTimeout = 2 * time.Second
)

// ExpressionWarning is advice the preview carries alongside a valid result:
// the expression works on the sample, and would fail on a real record shaped
// a particular way. Scope says which expression, in the same field-path form
// validation errors use, so the Studio can apply the fix in place.
type ExpressionWarning struct {
	Scope      string `json:"scope"`
	Field      string `json:"field"`
	Type       string `json:"type,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

type TestExpressionResponse struct {
	Valid             bool                                   `json:"valid"`
	Result            any                                    `json:"result,omitempty"`
	Error             string                                 `json:"error,omitempty"`
	Message           string                                 `json:"message"`
	Breakdown         []formulatemplatetypes.BreakdownAmount `json:"breakdown,omitempty"`
	ResolvedVariables map[string]any                         `json:"resolvedVariables,omitempty"`
	Guardrail         *formulatemplatetypes.GuardrailResult  `json:"guardrail,omitempty"`
	Rounding          *formulatemplatetypes.RoundingResult   `json:"rounding,omitempty"`
	Warnings          []ExpressionWarning                    `json:"warnings,omitempty"`
	Receipt           *formulatypes.Receipt                  `json:"receipt,omitempty"`
}

func (s *Service) DescribeSchema(
	schemaID string,
) (*formulatemplatetypes.SchemaDescription, error) {
	return s.formulaService.DescribeSchema(schemaID)
}

func (s *Service) TestExpression(
	ctx context.Context,
	req *TestExpressionRequest,
) *TestExpressionResponse {
	ctx = engine.WithEvaluationTimeout(
		contextvariablecache.With(ratetablecache.With(ctx)),
		previewEvaluationTimeout,
	)

	err := s.formulaService.ValidateLookupTables(ctx, req.Expression, req.TenantInfo)
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: msgExpressionValidationFailed,
		}
	}

	var resp *TestExpressionResponse
	if req.ShipmentID != nil {
		resp = s.testExpressionAgainstShipment(ctx, req)
	} else {
		lookup, lookupErr := s.lookupForTest(ctx, req)
		if lookupErr != nil {
			return &TestExpressionResponse{
				Valid:   false,
				Error:   lookupErr.Error(),
				Message: "Failed to load rate tables",
			}
		}
		resp = s.testExpressionWithEnv(ctx, req, lookup)
	}

	if resp.Valid {
		resp.Warnings = s.nullableFieldWarnings(ctx, req)
	}

	return resp
}

// nullableFieldWarnings runs the unguarded-field check over the expression and
// every breakdown line. A check that cannot run (a schema the engine does not
// know, an expression that will not parse) contributes nothing: the result
// already carries the error that matters.
func (s *Service) nullableFieldWarnings(
	ctx context.Context,
	req *TestExpressionRequest,
) []ExpressionWarning {
	warnings := s.warningsForScope(ctx, "expression", req.Expression, req)

	for i, def := range req.Breakdowns {
		if def == nil {
			continue
		}
		scope := fmt.Sprintf("breakdownDefinitions[%d].expression", i)
		warnings = append(warnings, s.warningsForScope(ctx, scope, def.Expression, req)...)
	}

	return warnings
}

func (s *Service) warningsForScope(
	ctx context.Context,
	scope, expression string,
	req *TestExpressionRequest,
) []ExpressionWarning {
	found, err := s.formulaService.UnguardedNullableFields(
		ctx,
		expression,
		req.SchemaID,
		req.Variables,
	)
	if err != nil || len(found) == 0 {
		return nil
	}

	warnings := make([]ExpressionWarning, 0, len(found))
	for _, item := range found {
		warnings = append(warnings, ExpressionWarning{
			Scope: scope,
			Field: item.Field,
			Type:  item.Type,
			Message: fmt.Sprintf(
				"%s can be empty on a shipment, and this expression would then fail to rate "+
					"instead of pricing it.",
				item.Field,
			),
			Suggestion: item.Suggestion,
		})
	}

	return warnings
}

// lookupForTest loads the tenant's rate tables only when the expression or a
// breakdown line names one. A preview re-runs on every keystroke, and most
// formulas never touch a table; loading every matrix for them would turn the
// live preview into the slowest thing on the page. When a table is named the
// real tables are used, so the number an author sees is the number a shipment
// would be charged.
func (s *Service) lookupForTest(
	ctx context.Context,
	req *TestExpressionRequest,
) (formulatemplatetypes.RateTableLookup, error) {
	if !referencesLookupTables(req.Expression, req.Breakdowns) {
		return nil, nil
	}

	return s.formulaService.BuildLookup(ctx, req.TenantInfo)
}

func referencesLookupTables(
	expression string,
	breakdowns []*formulatypes.BreakdownDefinition,
) bool {
	if tables, err := engine.ExtractLookupTables(expression); err == nil && len(tables) > 0 {
		return true
	}

	for _, def := range breakdowns {
		if def == nil {
			continue
		}
		if tables, err := engine.ExtractLookupTables(def.Expression); err == nil &&
			len(tables) > 0 {
			return true
		}
	}

	return false
}

func (s *Service) testExpressionAgainstShipment(
	ctx context.Context,
	req *TestExpressionRequest,
) *TestExpressionResponse {
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              *req.ShipmentID,
		TenantInfo:      req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Failed to load shipment",
		}
	}

	resp, err := s.formulaService.EvaluateExpression(ctx, &formula.EvaluateExpressionRequest{
		Expression: req.Expression,
		Entity:     entity,
		SchemaID:   req.SchemaID,
		Variables:  req.Variables,
		Breakdowns: req.Breakdowns,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression evaluation failed",
		}
	}

	amount, guardrail, rounding := formula.ApplyChargePolicy(req.chargePolicy(), resp.Amount)

	return &TestExpressionResponse{
		Valid:             true,
		Result:            amount,
		Breakdown:         resp.Breakdown,
		ResolvedVariables: maputils.WithoutFuncValues(resp.Variables),
		Guardrail:         guardrail,
		Rounding:          rounding,
		Receipt:           resp.Receipt,
		Message:           "Expression evaluated against shipment",
	}
}

func (s *Service) testExpressionWithEnv(
	ctx context.Context,
	req *TestExpressionRequest,
	lookup formulatemplatetypes.RateTableLookup,
) *TestExpressionResponse {
	env, err := s.formulaService.BuildValidationEnvironmentForTenant(
		ctx,
		req.TenantInfo,
		req.SchemaID,
		req.Variables,
	)
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: msgExpressionValidationFailed,
		}
	}

	if err = s.formulaService.ValidateExpressionWithEnv(ctx, req.Expression, env); err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: msgExpressionValidationFailed,
		}
	}

	result, err := s.formulaService.EvaluateWithEnv(ctx, &formulatemplatetypes.EnvEvaluationRequest{
		Expression: req.Expression,
		Env:        env,
		Lookup:     lookup,
	})
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression evaluation failed",
		}
	}

	amount, guardrail, rounding := formula.ApplyChargePolicy(req.chargePolicy(), result.Amount)

	resp := &TestExpressionResponse{
		Valid:     true,
		Result:    amount,
		Guardrail: guardrail,
		Rounding:  rounding,
		Receipt:   result.Receipt,
		Message:   "Expression is valid",
	}

	if len(req.Breakdowns) > 0 {
		var lookups []formulatypes.LookupTrace
		resp.Breakdown, lookups = s.evaluateBreakdownsWithEnv(ctx, req.Breakdowns, env, lookup)
		if resp.Receipt != nil {
			resp.Receipt.Lookups = append(resp.Receipt.Lookups, lookups...)
		}
	}

	return resp
}

// evaluateBreakdownsWithEnv prices each breakdown line and returns the
// rate-table lookups the lines made, re-scoped to the line's name so the
// receipt can attribute them.
func (s *Service) evaluateBreakdownsWithEnv(
	ctx context.Context,
	defs []*formulatypes.BreakdownDefinition,
	env map[string]any,
	lookup formulatemplatetypes.RateTableLookup,
) ([]formulatemplatetypes.BreakdownAmount, []formulatypes.LookupTrace) {
	items := make([]formulatemplatetypes.BreakdownAmount, 0, len(defs))
	lookups := make([]formulatypes.LookupTrace, 0, len(defs))
	for _, def := range defs {
		if def == nil {
			continue
		}

		item := formulatemplatetypes.BreakdownAmount{Name: def.Name, Label: def.Label}
		result, err := s.formulaService.EvaluateWithEnv(
			ctx,
			&formulatemplatetypes.EnvEvaluationRequest{
				Expression: def.Expression,
				Env:        env,
				Lookup:     lookup,
			},
		)
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Amount = result.Amount
			if result.Receipt != nil {
				for _, entry := range result.Receipt.Lookups {
					entry.Scope = def.Name
					lookups = append(lookups, entry)
				}
			}
		}

		items = append(items, item)
	}

	return items, lookups
}
