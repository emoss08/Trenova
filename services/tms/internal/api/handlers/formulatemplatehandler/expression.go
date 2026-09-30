package formulatemplatehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type testExpressionRequest struct {
	Expression string                              `json:"expression"`
	SchemaID   string                              `json:"schemaId"`
	Variables  map[string]any                      `json:"variables"`
	ShipmentID string                              `json:"shipmentId"`
	Breakdowns []*formulatypes.BreakdownDefinition `json:"breakdowns"`
	MinCharge  *string                             `json:"minCharge"`
	MaxCharge  *string                             `json:"maxCharge"`
	// RoundingMode and RoundingPrecision are the charge policy under test;
	// omitted means the default the template would store.
	RoundingMode      string `json:"roundingMode"`
	RoundingPrecision *int32 `json:"roundingPrecision"`
}

// @Summary Test a formula expression
// @ID testFormulaExpression
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body testExpressionRequest true "Expression test request"
// @Success 200 {object} formulatemplateservice.TestExpressionResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/test [post]
func (h *Handler) testExpression(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req testExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if req.SchemaID == "" {
		req.SchemaID = "shipment"
	}

	serviceReq := &formulatemplateservice.TestExpressionRequest{
		Expression: req.Expression,
		SchemaID:   req.SchemaID,
		Variables:  req.Variables,
		Breakdowns: req.Breakdowns,
		TenantInfo: pagination.FromAuthAsUser(authCtx),
	}

	minCharge, err := parseGuardrailCharge("minCharge", req.MinCharge)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	maxCharge, err := parseGuardrailCharge("maxCharge", req.MaxCharge)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if minCharge.Valid && maxCharge.Valid &&
		minCharge.Decimal.GreaterThan(maxCharge.Decimal) {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"minCharge",
			errortypes.ErrInvalid,
			"Minimum charge cannot exceed maximum charge",
		))
		return
	}
	serviceReq.MinCharge = minCharge
	serviceReq.MaxCharge = maxCharge

	policy, err := parseRoundingPolicy(req.RoundingMode, req.RoundingPrecision)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	serviceReq.RoundingMode = policy.RoundingMode
	serviceReq.RoundingPrecision = policy.RoundingPrecision

	if req.ShipmentID != "" {
		shipmentID, err := pulid.MustParse(req.ShipmentID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}

		if !h.allowShipmentRead(c, authCtx) {
			return
		}

		serviceReq.ShipmentID = &shipmentID
	}

	result := h.service.TestExpression(c.Request.Context(), serviceReq)

	c.JSON(http.StatusOK, result)
}

func (h *Handler) allowShipmentRead(c *gin.Context, authCtx *authctx.AuthContext) bool {
	result, err := h.permEngine.Check(
		c.Request.Context(),
		middleware.BuildPermissionCheckRequest(
			authCtx,
			permission.ResourceShipment.String(),
			permission.OpRead,
		),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return false
	}

	if !result.Allowed {
		h.eh.HandleError(c, errortypes.NewAuthorizationError(
			"You don't have permission to read shipments",
		))
		return false
	}

	return true
}

func parseRoundingPolicy(mode string, precision *int32) (formulatypes.ChargePolicy, error) {
	policy := formulatypes.ChargePolicy{RoundingMode: ratetypes.RoundingMode(mode)}

	if mode != "" && !policy.RoundingMode.IsValid() {
		return policy, errortypes.NewValidationError(
			"roundingMode",
			errortypes.ErrInvalid,
			"Must be one of HalfUp, HalfEven, Up, Down, or None",
		)
	}

	if precision != nil {
		if *precision < 0 || *precision > formulatypes.MaxRoundingPrecision {
			return policy, errortypes.NewValidationError(
				"roundingPrecision",
				errortypes.ErrInvalid,
				"Must be between 0 and 4",
			)
		}
		policy.RoundingPrecision = *precision
	} else if mode != "" {
		policy.RoundingPrecision = formulatypes.DefaultRoundingPrecision
	}

	return policy, nil
}

func parseGuardrailCharge(field string, raw *string) (decimal.NullDecimal, error) {
	if raw == nil || *raw == "" {
		return decimal.NullDecimal{}, nil
	}

	value, err := decimal.NewFromString(*raw)
	if err != nil {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Must be a valid decimal number",
		)
	}

	if value.IsNegative() {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Cannot be negative",
		)
	}

	return decimal.NullDecimal{Decimal: value, Valid: true}, nil
}
