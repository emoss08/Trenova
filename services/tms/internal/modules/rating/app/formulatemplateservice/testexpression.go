package formulatemplateservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	previewEvaluationTimeout = 2 * time.Second

	CodeTimeout        = "timeout"
	CodeNegativeCharge = "negativeCharge"
)

type TestExpressionRequest struct {
	Expression        string
	SchemaID          string
	Variables         map[string]any
	ShipmentID        *pulid.ID
	TenantInfo        pagination.TenantInfo
	Breakdowns        []*formulatypes.BreakdownDefinition
	MinCharge         decimal.NullDecimal
	MaxCharge         decimal.NullDecimal
	RoundingMode      ratetypes.RoundingMode
	RoundingPrecision int32
	TraceStatements   bool
}

func (r *TestExpressionRequest) chargePolicy() formulatypes.ChargePolicy {
	return formulatypes.ChargePolicy{
		MinCharge:         r.MinCharge,
		MaxCharge:         r.MaxCharge,
		RoundingMode:      r.RoundingMode,
		RoundingPrecision: r.RoundingPrecision,
	}
}

type TestExpressionResponse struct {
	Valid             bool                                   `json:"valid"`
	Result            *decimal.Decimal                       `json:"result"`
	Diagnostics       []formula.ScopedDiagnostic             `json:"diagnostics"`
	Error             *formulaerrors.EvaluationError         `json:"error"`
	Statements        []formulatemplatetypes.Statement       `json:"statements"`
	Breakdown         []formulatemplatetypes.BreakdownAmount `json:"breakdown"`
	ResolvedVariables map[string]any                         `json:"resolvedVariables"`
	Guardrail         *formulatemplatetypes.GuardrailResult  `json:"guardrail"`
	Rounding          *formulatemplatetypes.RoundingResult   `json:"rounding"`
	Receipt           *formulatypes.Receipt                  `json:"receipt"`
}

func (r *TestExpressionResponse) FirstErrorDiagnostic() *formula.ScopedDiagnostic {
	for index := range r.Diagnostics {
		if r.Diagnostics[index].IsError() {
			return &r.Diagnostics[index]
		}
	}
	return nil
}

func (r *TestExpressionResponse) Failure() *rules.Message {
	if diagnostic := r.FirstErrorDiagnostic(); diagnostic != nil {
		return &diagnostic.Message
	}
	if r.Error != nil {
		return &r.Error.Err.Message
	}
	return nil
}

func (r *TestExpressionResponse) FailureCode() string {
	if diagnostic := r.FirstErrorDiagnostic(); diagnostic != nil {
		return diagnostic.Code
	}
	if r.Error != nil {
		return r.Error.Err.Code
	}
	return ""
}

func (r *TestExpressionResponse) FailureText(locale i18n.Locale) string {
	if failure := r.Failure(); failure != nil {
		return failure.Localize(locale)
	}
	return ""
}

func (r *TestExpressionResponse) Unguarded() []string {
	names := make([]string, 0, 2)
	for index := range r.Diagnostics {
		diagnostic := &r.Diagnostics[index]
		if diagnostic.Code != rules.DiagnosticNullable || diagnostic.Name == "" {
			continue
		}
		names = appendUnique(names, diagnostic.Name)
	}
	return names
}

func (s *Service) DescribeSchema(
	schemaID string,
) (*formulatemplatetypes.SchemaDescription, error) {
	return s.formulaService.DescribeSchema(schemaID)
}

func previewBoundsError(req *TestExpressionRequest) error {
	if len(req.Expression) > formulatemplate.MaxExpressionLength {
		return errortypes.NewValidationError(
			"expression",
			errortypes.ErrInvalid,
			"Expression cannot exceed 10,000 characters",
		)
	}
	if len(req.Breakdowns) > formulatypes.MaxBreakdownDefinitions {
		return errortypes.NewValidationError(
			"breakdowns",
			errortypes.ErrInvalid,
			"A template cannot have more than 20 breakdown definitions",
		)
	}
	for index, def := range req.Breakdowns {
		if def != nil && len(def.Expression) > formulatemplate.MaxExpressionLength {
			return errortypes.NewValidationError(
				formula.PreviewScope(index),
				errortypes.ErrInvalid,
				"Breakdown expression cannot exceed 10,000 characters",
			)
		}
	}
	return nil
}

func (s *Service) TestExpression(
	ctx context.Context,
	req *TestExpressionRequest,
) (*TestExpressionResponse, error) {
	if err := previewBoundsError(req); err != nil {
		return nil, err
	}

	ctx = contextvariablecache.With(ratetablecache.With(ctx))

	var entity any
	if req.ShipmentID != nil {
		shipment, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:              *req.ShipmentID,
			TenantInfo:      req.TenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
		})
		if err != nil {
			return nil, err
		}
		if err = s.hydrateShipment(ctx, shipment, req.TenantInfo); err != nil {
			return nil, err
		}
		entity = shipment
	}

	leashed, cancel := context.WithTimeout(ctx, previewEvaluationTimeout)
	defer cancel()

	preview, err := s.formulaService.Preview(leashed, &formula.PreviewRequest{
		SchemaID:        req.SchemaID,
		Expression:      req.Expression,
		Breakdowns:      req.Breakdowns,
		Variables:       req.Variables,
		Entity:          entity,
		TenantInfo:      req.TenantInfo,
		TraceStatements: req.TraceStatements,
	})
	if err != nil {
		if ctx.Err() == nil && errors.Is(leashed.Err(), context.DeadlineExceeded) {
			return timedOut(), nil
		}
		return nil, err
	}

	return previewResponse(i18n.FromContext(ctx), req, preview), nil
}

func timeoutMessage() rules.Message {
	return rules.Message{
		Template: "The formula took longer than {0} to price, so it was stopped",
		Args:     []any{previewEvaluationTimeout.String()},
	}
}

func timedOut() *TestExpressionResponse {
	return &TestExpressionResponse{
		Diagnostics: []formula.ScopedDiagnostic{},
		Error: formulaerrors.NewEvaluationError(formula.MainScope, &rules.FormulaError{
			Code:    CodeTimeout,
			Message: timeoutMessage(),
		}),
	}
}

func previewResponse(
	locale i18n.Locale,
	req *TestExpressionRequest,
	preview *formula.Preview,
) *TestExpressionResponse {
	resp := &TestExpressionResponse{
		Diagnostics:       preview.Diagnostics,
		Error:             preview.Error,
		Statements:        preview.Statements,
		ResolvedVariables: preview.Variables,
		Receipt:           preview.Receipt,
	}
	if resp.Diagnostics == nil {
		resp.Diagnostics = []formula.ScopedDiagnostic{}
	}

	evaluated := preview.Evaluated
	if evaluated == nil {
		return resp
	}

	policy := req.chargePolicy()
	resp.Breakdown = formula.RoundBreakdown(
		policy,
		evaluated.Breakdown(locale),
		evaluated.Amount,
	)
	if evaluated.Error != nil {
		return resp
	}

	amount, guardrail, rounding, err := formula.ApplyChargePolicy(policy, evaluated.Amount)
	resp.Guardrail = guardrail
	resp.Rounding = rounding
	if err != nil {
		resp.Error = formulaerrors.NewEvaluationError(formula.MainScope, &rules.FormulaError{
			Code: CodeNegativeCharge,
			Message: formula.NegativeChargeMessage(
				amount.StringFixed(policy.Normalized().RoundingPrecision),
			),
		})
		return resp
	}

	resp.Result = &amount
	resp.Valid = preview.Valid()
	return resp
}
