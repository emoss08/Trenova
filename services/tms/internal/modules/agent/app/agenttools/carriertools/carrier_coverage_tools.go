package carriertools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxCarrierAccessorials = 10

const fieldRateMethod = "rateMethod"

const fieldBaseRate = "baseRate"

const fieldFuelSurcharge = "fuelSurcharge"

const fieldProNumber = "proNumber"

const fieldExternalDriverName = "externalDriverName"

const fieldExternalDriverPhone = "externalDriverPhone"

const fieldExternalTractorNumber = "externalTractorNumber"

const fieldExternalTrailerNumber = "externalTrailerNumber"

const fieldReplace = "replace"

const fieldOverrideInsuranceWarning = "overrideInsuranceWarning"

const fieldAccessorials = "accessorials"

const fieldAmount = "amount"

const fieldAccessorialChargeID = "accessorialChargeId"

// carrierCoverage is the slice of the carrier assignment service the carrier
// tools use: covering a move with a carrier and taking the carrier off it.
type carrierCoverage interface {
	AssignToMove(
		ctx context.Context,
		req *shipmentports.AssignMoveToCarrierRequest,
	) (*shipment.CarrierAssignment, error)
	PreviewAssignToMove(
		ctx context.Context,
		req *shipmentports.AssignMoveToCarrierRequest,
	) (*carrierassignmentservice.CoveragePlan, error)
	Cancel(ctx context.Context, req *shipmentports.CancelCarrierAssignmentRequest) error
	PreviewCancel(
		ctx context.Context,
		req *shipmentports.CancelCarrierAssignmentRequest,
	) (*carrierassignmentservice.CoveragePlan, error)
}

// assignMoveToCarrierTool covers a move with an outside carrier at a rate
// already agreed, which is the dispatch console's carrier assignment. The pay
// it records is what the carrier is settled against.
type AssignMoveToCarrierTool struct {
	carriers carrierCoverage
}

func NewAssignMoveToCarrierTool(carriers carrierCoverage) agentports.AgentTool {
	return &AssignMoveToCarrierTool{carriers: carriers}
}

func (t *AssignMoveToCarrierTool) Name() string { return "assign_move_to_carrier" }

func (t *AssignMoveToCarrierTool) Recipe() []string {
	return []string{"get_shipment", "shop_carriers", "assign_move_to_carrier"}
}

func (t *AssignMoveToCarrierTool) Description() string {
	return "Cover a move with an outside carrier at a rate the carrier already agreed to. " +
		"Use it outside a tender: a contract carrier on its contract rate, or a carrier who " +
		"accepted a load by phone or email. Read shop_carriers first so the rate comes from " +
		"a contract or a quote; a rate you cannot source is a reason to ask, never to guess. " +
		"The move must have no driver. Nothing is sent to the carrier: generate the rate " +
		"confirmation with generate_rate_confirmation and send it with " +
		"send_rate_confirmation next. To offer a load and wait for an answer, tender it " +
		"instead."
}

func (t *AssignMoveToCarrierTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		common.PreviewFieldShipmentMoveID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentMove,
			"The move to cover, from get_dispatch_board (moveId) or get_shipment (its moves).",
		),
		common.FieldCarrierID: agenttoolschema.RecordIDText(
			permission.ResourceCarrier,
			"The carrier, from shop_carriers or list_carriers.",
		),
		fieldRateMethod: agenttoolschema.Enum(
			"Flat pays baseRate for the move; PerMile pays baseRate for each mile of the "+
				"move's computed distance.",
			agenttoolschema.CarrierRateMethods,
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
				common.FieldDescription: jsonschemautils.Text(
					"What the charge is for, as the carrier will read it.",
				),
				fieldAmount: jsonschemautils.Text("The amount as a decimal string."),
				fieldAccessorialChargeID: agenttoolschema.RecordIDText(
					permission.ResourceAccessorialCharge,
					"The accessorial charge it is, from list_accessorial_charges.",
				),
			}, common.FieldDescription, fieldAmount),
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
	}, common.PreviewFieldShipmentMoveID, common.FieldCarrierID, fieldRateMethod, fieldBaseRate)
}

func (t *AssignMoveToCarrierTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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
		Condition: &agentports.TierCondition{
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
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	if common.OptionalBool(params.Params, fieldReplace) ||
		common.OptionalBool(params.Params, fieldOverrideInsuranceWarning) {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *AssignMoveToCarrierTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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

func (t *AssignMoveToCarrierTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
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
func (t *AssignMoveToCarrierTool) request(
	params *agentports.ToolExecuteParams,
) (*shipmentports.AssignMoveToCarrierRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	moveID, err := common.RequirePulid(params.Params, common.PreviewFieldShipmentMoveID)
	if err != nil {
		return nil, err
	}
	carrierID, err := common.RequirePulid(params.Params, common.FieldCarrierID)
	if err != nil {
		return nil, err
	}
	rateMethod, err := carrierRateMethodArg(params.Params)
	if err != nil {
		return nil, err
	}
	baseRate, err := common.DecimalArg(params.Params, fieldBaseRate, true)
	if err != nil {
		return nil, err
	}
	if !baseRate.IsPositive() {
		return nil, errortypes.NewValidationError(fieldBaseRate, errortypes.ErrInvalid,
			"The base rate must be greater than zero; a carrier is never covered for nothing")
	}
	fuelSurcharge, err := common.DecimalArg(params.Params, fieldFuelSurcharge, false)
	if err != nil {
		return nil, err
	}
	accessorials, err := carrierAccessorialsArg(params.Params)
	if err != nil {
		return nil, err
	}

	return &shipmentports.AssignMoveToCarrierRequest{
		TenantInfo:     common.TenantFrom(*params),
		ShipmentMoveID: moveID,
		CarrierID:      carrierID,
		RateMethod:     rateMethod,
		BaseRate:       baseRate,
		FuelSurcharge:  fuelSurcharge,
		Accessorials:   accessorials,
		ProNumber:      strings.TrimSpace(common.OptionalString(params.Params, fieldProNumber)),
		ExternalDriverName: strings.TrimSpace(
			common.OptionalString(params.Params, fieldExternalDriverName),
		),
		ExternalDriverPhone: strings.TrimSpace(
			common.OptionalString(params.Params, fieldExternalDriverPhone),
		),
		ExternalTractorNumber: strings.TrimSpace(
			common.OptionalString(params.Params, fieldExternalTractorNumber),
		),
		ExternalTrailerNumber: strings.TrimSpace(
			common.OptionalString(params.Params, fieldExternalTrailerNumber),
		),
		Replace:                  common.OptionalBool(params.Params, fieldReplace),
		OverrideInsuranceWarning: common.OptionalBool(params.Params, fieldOverrideInsuranceWarning),
	}, nil
}

func (t *AssignMoveToCarrierTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.PreviewFieldShipmentMoveID, permission.ResourceShipmentMove)
}

// cancelCarrierAssignmentTool takes the carrier off a move that has not
// started, which is the console's cancel carrier assignment.
type CancelCarrierAssignmentTool struct {
	carriers carrierCoverage
}

func NewCancelCarrierAssignmentTool(carriers carrierCoverage) agentports.AgentTool {
	return &CancelCarrierAssignmentTool{carriers: carriers}
}

func (t *CancelCarrierAssignmentTool) Name() string { return "cancel_carrier_assignment" }

func (t *CancelCarrierAssignmentTool) SearchTerms() []string {
	return []string{"drop carrier", "uncover move", "remove outside carrier"}
}

func (t *CancelCarrierAssignmentTool) Description() string {
	return "Take the outside carrier off a move that has not started, leaving it uncovered, " +
		"and void the carrier's rate confirmation. Use it when the carrier fell off the " +
		"load or the move is being covered another way. Say why in reason, as the desk will " +
		"read it on the move's history. A move already in transit keeps its carrier."
}

func (t *CancelCarrierAssignmentTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		common.PreviewFieldShipmentMoveID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentMove,
			"The move whose carrier comes off, from get_dispatch_board (moveId) or "+
				"get_shipment (its moves).",
		),
		common.FieldReason: jsonschemautils.Text(
			"Why the carrier is coming off, in a sentence a person can check.",
		),
	}, common.PreviewFieldShipmentMoveID, common.FieldReason)
}

func (t *CancelCarrierAssignmentTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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
		Condition: &agentports.TierCondition{
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

func (t *CancelCarrierAssignmentTool) tierLimit(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
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

func (t *CancelCarrierAssignmentTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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

func (t *CancelCarrierAssignmentTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	return t.carriers.Cancel(ctx, request)
}

func (t *CancelCarrierAssignmentTool) request(
	params *agentports.ToolExecuteParams,
) (*shipmentports.CancelCarrierAssignmentRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	moveID, err := common.RequirePulid(params.Params, common.PreviewFieldShipmentMoveID)
	if err != nil {
		return nil, err
	}
	reason, err := common.RequireString(params.Params, common.FieldReason)
	if err != nil {
		return nil, err
	}

	return &shipmentports.CancelCarrierAssignmentRequest{
		TenantInfo:     common.TenantFrom(*params),
		ShipmentMoveID: moveID,
		Reason:         strings.TrimSpace(reason),
	}, nil
}

func (t *CancelCarrierAssignmentTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.PreviewFieldShipmentMoveID, permission.ResourceShipmentMove)
}

// carrierRateMethodArg refuses anything but the two ways a carrier is paid.
func carrierRateMethodArg(params map[string]any) (shipment.CarrierRateMethod, error) {
	raw := common.OptionalString(params, fieldRateMethod)
	method := shipment.CarrierRateMethod(raw)
	if method != shipment.CarrierRateMethodFlat && method != shipment.CarrierRateMethodPerMile {
		return "", fmt.Errorf(
			"parameter %q must be exactly Flat or PerMile, not %q", fieldRateMethod, raw,
		)
	}

	return method, nil
}

func carrierAccessorialsArg(
	params map[string]any,
) ([]shipmentports.CarrierAccessorialInput, error) {
	if _, present := params[fieldAccessorials]; !present {
		return nil, nil
	}

	var lines []carrierAccessorialParam
	if err := common.DecodeParam(params, fieldAccessorials, &lines); err != nil {
		return nil, err
	}
	if len(lines) > maxCarrierAccessorials {
		return nil, fmt.Errorf(
			"parameter %q holds %d charges, more than the %d one assignment carries",
			fieldAccessorials, len(lines), maxCarrierAccessorials,
		)
	}

	out := make([]shipmentports.CarrierAccessorialInput, 0, len(lines))
	for idx, line := range lines {
		amount, err := common.DecimalArg(map[string]any{fieldAmount: line.Amount}, fieldAmount, true)
		if err != nil {
			return nil, fmt.Errorf("accessorials[%d]: %w", idx, err)
		}
		input := shipmentports.CarrierAccessorialInput{
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
