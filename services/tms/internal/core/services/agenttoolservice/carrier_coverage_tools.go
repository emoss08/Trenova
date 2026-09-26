package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierassignmentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const maxCarrierAccessorials = 10

const (
	fieldRateMethod               = "rateMethod"
	fieldBaseRate                 = "baseRate"
	fieldFuelSurcharge            = "fuelSurcharge"
	fieldProNumber                = "proNumber"
	fieldExternalDriverName       = "externalDriverName"
	fieldExternalDriverPhone      = "externalDriverPhone"
	fieldExternalTractorNumber    = "externalTractorNumber"
	fieldExternalTrailerNumber    = "externalTrailerNumber"
	fieldReplace                  = "replace"
	fieldOverrideInsuranceWarning = "overrideInsuranceWarning"
	fieldAccessorials             = "accessorials"
	fieldDescription              = "description"
	fieldAmount                   = "amount"
	fieldAccessorialChargeID      = "accessorialChargeId"
)

// carrierCoverage is the slice of the carrier assignment service the carrier
// tools use: covering a move with a carrier and taking the carrier off it.
type carrierCoverage interface {
	AssignToMove(
		ctx context.Context,
		req *repositories.AssignMoveToCarrierRequest,
	) (*shipment.CarrierAssignment, error)
	PreviewAssignToMove(
		ctx context.Context,
		req *repositories.AssignMoveToCarrierRequest,
	) (*carrierassignmentservice.CoveragePlan, error)
	Cancel(ctx context.Context, req *repositories.CancelCarrierAssignmentRequest) error
	PreviewCancel(
		ctx context.Context,
		req *repositories.CancelCarrierAssignmentRequest,
	) (*carrierassignmentservice.CoveragePlan, error)
}

// assignMoveToCarrierTool covers a move with an outside carrier at a rate
// already agreed, which is the dispatch console's carrier assignment. The pay
// it records is what the carrier is settled against.
type assignMoveToCarrierTool struct {
	carriers carrierCoverage
}

func newAssignMoveToCarrierTool(carriers carrierCoverage) serviceports.AgentTool {
	return &assignMoveToCarrierTool{carriers: carriers}
}

func (t *assignMoveToCarrierTool) Name() string { return "assign_move_to_carrier" }

func (t *assignMoveToCarrierTool) Description() string {
	return "Cover a move with an outside carrier at a rate the carrier already agreed to. " +
		"Use it outside a tender: a contract carrier on its contract rate, or a carrier who " +
		"accepted a load by phone or email. Read shop_carriers first so the rate comes from " +
		"a contract or a quote; a rate you cannot source is a reason to ask, never to guess. " +
		"The move must have no driver. Nothing is sent to the carrier: generate the rate " +
		"confirmation with generate_rate_confirmation and send it with " +
		"send_rate_confirmation next. To offer a load and wait for an answer, tender it " +
		"instead."
}

func (t *assignMoveToCarrierTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		previewFieldShipmentMoveID: jsonschemautils.Text(
			"The move to cover, from get_dispatch_board (moveId) or get_shipment (its moves).",
		),
		fieldCarrierID: jsonschemautils.Text("The carrier, from shop_carriers or list_carriers."),
		fieldRateMethod: jsonschemautils.Enum(
			"Flat pays baseRate for the move; PerMile pays baseRate for each mile of the "+
				"move's computed distance.",
			string(shipment.CarrierRateMethodFlat),
			string(shipment.CarrierRateMethodPerMile),
		),
		fieldBaseRate: jsonschemautils.Text(
			"The agreed linehaul rate as a decimal string, such as \"1850.00\", greater than zero.",
		),
		fieldFuelSurcharge: jsonschemautils.Text(
			"The agreed fuel surcharge as a decimal string, when it is paid apart.",
		),
		fieldAccessorials: jsonschemautils.DescribedArray(
			"Extra charges the carrier is paid on this move, each agreed with them.",
			jsonschemautils.Object(map[string]any{
				fieldDescription: jsonschemautils.Text(
					"What the charge is for, as the carrier will read it.",
				),
				fieldAmount: jsonschemautils.Text("The amount as a decimal string."),
				fieldAccessorialChargeID: jsonschemautils.Text(
					"The accessorial charge it is, from list_accessorial_charges.",
				),
			}, fieldDescription, fieldAmount),
			maxCarrierAccessorials,
		),
		fieldProNumber: jsonschemautils.Text(
			"The carrier's own reference for the load, when they gave one.",
		),
		fieldExternalDriverName: jsonschemautils.Text(
			"The carrier's driver, when the carrier named them.",
		),
		fieldExternalDriverPhone: jsonschemautils.Text(
			"The carrier's driver's phone number, when the carrier gave it.",
		),
		fieldExternalTractorNumber: jsonschemautils.Text(
			"The carrier's truck number, when the carrier gave it.",
		),
		fieldExternalTrailerNumber: jsonschemautils.Text(
			"The carrier's trailer number, when the carrier gave it.",
		),
		fieldReplace: jsonschemautils.Boolean(
			"Replace the carrier already on the move, canceling their assignment and voiding " +
				"their rate confirmation. Off by default.",
		),
		fieldOverrideInsuranceWarning: jsonschemautils.Boolean(
			"Cover the move although the carrier's insurance expires soon. A carrier that is " +
				"blocked can never be used. Off by default.",
		),
	}, previewFieldShipmentMoveID, fieldCarrierID, fieldRateMethod, fieldBaseRate)
}

func (t *assignMoveToCarrierTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpAssign,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Condition: &serviceports.TierCondition{
			Description: "Replacing a carrier already on the move, or overriding the new " +
				"carrier's insurance warning, is a proposal a person decides; any other " +
				"assignment runs once a person approves it.",
			Limit: carrierAssignmentTierLimit,
		},
		Rationale: "Commits the organization to pay an outside carrier the rate it names " +
			"for the move; nothing is sent to the carrier, and canceling the assignment " +
			"releases the move.",
	}
}

func carrierAssignmentTierLimit(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	if optionalBool(params.Params, fieldReplace) ||
		optionalBool(params.Params, fieldOverrideInsuranceWarning) {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *assignMoveToCarrierTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}
	if multiErr := request.Validate(); multiErr != nil {
		return multiErr
	}

	return nil
}

func (t *assignMoveToCarrierTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.carriers.AssignToMove(ctx, request)

	return err
}

type carrierAccessorialParam struct {
	Description         string `json:"description"`
	Amount              string `json:"amount"`
	AccessorialChargeID string `json:"accessorialChargeId"`
}

// request mirrors the console's input. The rate is never left to the
// carrier's contract here: contract pricing saves a quote as it prices, so a
// proposal could not show the pay a person is approving.
func (t *assignMoveToCarrierTool) request(
	params *serviceports.ToolExecuteParams,
) (*repositories.AssignMoveToCarrierRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	moveID, err := requirePulid(params.Params, previewFieldShipmentMoveID)
	if err != nil {
		return nil, err
	}
	carrierID, err := requirePulid(params.Params, fieldCarrierID)
	if err != nil {
		return nil, err
	}
	rateMethod, err := carrierRateMethodArg(params.Params)
	if err != nil {
		return nil, err
	}
	baseRate, err := decimalArg(params.Params, fieldBaseRate, true)
	if err != nil {
		return nil, err
	}
	if !baseRate.IsPositive() {
		return nil, errortypes.NewValidationError(fieldBaseRate, errortypes.ErrInvalid,
			"The base rate must be greater than zero; a carrier is never covered for nothing")
	}
	fuelSurcharge, err := decimalArg(params.Params, fieldFuelSurcharge, false)
	if err != nil {
		return nil, err
	}
	accessorials, err := carrierAccessorialsArg(params.Params)
	if err != nil {
		return nil, err
	}

	return &repositories.AssignMoveToCarrierRequest{
		TenantInfo:     tenantFrom(*params),
		ShipmentMoveID: moveID,
		CarrierID:      carrierID,
		RateMethod:     rateMethod,
		BaseRate:       baseRate,
		FuelSurcharge:  fuelSurcharge,
		Accessorials:   accessorials,
		ProNumber:      strings.TrimSpace(optionalString(params.Params, fieldProNumber)),
		ExternalDriverName: strings.TrimSpace(
			optionalString(params.Params, fieldExternalDriverName),
		),
		ExternalDriverPhone: strings.TrimSpace(
			optionalString(params.Params, fieldExternalDriverPhone),
		),
		ExternalTractorNumber: strings.TrimSpace(
			optionalString(params.Params, fieldExternalTractorNumber),
		),
		ExternalTrailerNumber: strings.TrimSpace(
			optionalString(params.Params, fieldExternalTrailerNumber),
		),
		Replace:                  optionalBool(params.Params, fieldReplace),
		OverrideInsuranceWarning: optionalBool(params.Params, fieldOverrideInsuranceWarning),
	}, nil
}

func (t *assignMoveToCarrierTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, previewFieldShipmentMoveID, permission.ResourceShipmentMove)
}

// cancelCarrierAssignmentTool takes the carrier off a move that has not
// started, which is the console's cancel carrier assignment.
type cancelCarrierAssignmentTool struct {
	carriers carrierCoverage
}

func newCancelCarrierAssignmentTool(carriers carrierCoverage) serviceports.AgentTool {
	return &cancelCarrierAssignmentTool{carriers: carriers}
}

func (t *cancelCarrierAssignmentTool) Name() string { return "cancel_carrier_assignment" }

func (t *cancelCarrierAssignmentTool) Description() string {
	return "Take the outside carrier off a move that has not started, leaving it uncovered, " +
		"and void the carrier's rate confirmation. Use it when the carrier fell off the " +
		"load or the move is being covered another way. Say why in reason, as the desk will " +
		"read it on the move's history. A move already in transit keeps its carrier."
}

func (t *cancelCarrierAssignmentTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		previewFieldShipmentMoveID: jsonschemautils.Text(
			"The move whose carrier comes off, from get_dispatch_board (moveId) or " +
				"get_shipment (its moves).",
		),
		fieldReason: jsonschemautils.Text(
			"Why the carrier is coming off, in a sentence a person can check.",
		),
	}, previewFieldShipmentMoveID, fieldReason)
}

func (t *cancelCarrierAssignmentTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpUnassign,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Condition: &serviceports.TierCondition{
			Description: "Taking off a carrier who has confirmed the rate breaks an executed " +
				"agreement, so it is a proposal a person decides, as is one the service " +
				"would refuse or that cannot be read; a carrier who has not confirmed comes " +
				"off once a person approves it.",
			Limit: t.tierLimit,
		},
		Rationale: "Releases the organization's commitment to pay an outside carrier and " +
			"voids the carrier's rate confirmation, whose sign link stops working; covering " +
			"the move again makes a new agreement the carrier has to confirm afresh.",
	}
}

func (t *cancelCarrierAssignmentTool) tierLimit(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	request, err := t.request(&params)
	if err != nil {
		return agent.TierPropose
	}

	plan, err := t.carriers.PreviewCancel(ctx, request)
	if err != nil || plan.CanceledBefore == nil ||
		plan.CanceledBefore.Status == shipment.CarrierAssignmentStatusConfirmed {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *cancelCarrierAssignmentTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}
	if multiErr := request.Validate(); multiErr != nil {
		return multiErr
	}

	return nil
}

func (t *cancelCarrierAssignmentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	return t.carriers.Cancel(ctx, request)
}

func (t *cancelCarrierAssignmentTool) request(
	params *serviceports.ToolExecuteParams,
) (*repositories.CancelCarrierAssignmentRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	moveID, err := requirePulid(params.Params, previewFieldShipmentMoveID)
	if err != nil {
		return nil, err
	}
	reason, err := requireString(params.Params, fieldReason)
	if err != nil {
		return nil, err
	}

	return &repositories.CancelCarrierAssignmentRequest{
		TenantInfo:     tenantFrom(*params),
		ShipmentMoveID: moveID,
		Reason:         strings.TrimSpace(reason),
	}, nil
}

func (t *cancelCarrierAssignmentTool) Target(
	params map[string]any,
) (serviceports.ToolTarget, bool) {
	return targetOf(params, previewFieldShipmentMoveID, permission.ResourceShipmentMove)
}

// carrierRateMethodArg refuses anything but the two ways a carrier is paid.
func carrierRateMethodArg(params map[string]any) (shipment.CarrierRateMethod, error) {
	raw := optionalString(params, fieldRateMethod)
	method := shipment.CarrierRateMethod(raw)
	if method != shipment.CarrierRateMethodFlat && method != shipment.CarrierRateMethodPerMile {
		return "", fmt.Errorf(
			"parameter %q must be exactly Flat or PerMile, not %q", fieldRateMethod, raw,
		)
	}

	return method, nil
}

// decimalArg reads an amount written as a decimal string. It is never
// negative; a missing optional amount is zero.
func decimalArg(params map[string]any, key string, required bool) (decimal.Decimal, error) {
	raw := strings.TrimSpace(optionalString(params, key))
	if raw == "" {
		if required {
			return decimal.Zero, fmt.Errorf("missing required parameter %q", key)
		}

		return decimal.Zero, nil
	}

	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf(
			"parameter %q must be a decimal amount such as 1850.00, not %q", key, raw,
		)
	}
	if value.IsNegative() {
		return decimal.Zero, fmt.Errorf("parameter %q cannot be negative", key)
	}

	return value, nil
}

func carrierAccessorialsArg(params map[string]any) ([]repositories.CarrierAccessorialInput, error) {
	if _, present := params[fieldAccessorials]; !present {
		return nil, nil
	}

	var lines []carrierAccessorialParam
	if err := decodeParam(params, fieldAccessorials, &lines); err != nil {
		return nil, err
	}
	if len(lines) > maxCarrierAccessorials {
		return nil, fmt.Errorf(
			"parameter %q holds %d charges, more than the %d one assignment carries",
			fieldAccessorials, len(lines), maxCarrierAccessorials,
		)
	}

	out := make([]repositories.CarrierAccessorialInput, 0, len(lines))
	for idx, line := range lines {
		amount, err := decimalArg(map[string]any{fieldAmount: line.Amount}, fieldAmount, true)
		if err != nil {
			return nil, fmt.Errorf("accessorials[%d]: %w", idx, err)
		}
		input := repositories.CarrierAccessorialInput{
			Description: strings.TrimSpace(line.Description),
			Amount:      amount,
		}
		if strings.TrimSpace(line.AccessorialChargeID) != "" {
			chargeID, pErr := pulid.Parse(strings.TrimSpace(line.AccessorialChargeID))
			if pErr != nil {
				return nil, fmt.Errorf("accessorials[%d].accessorialChargeId is not an id", idx)
			}
			input.AccessorialChargeID = &chargeID
		}
		out = append(out, input)
	}

	return out, nil
}
