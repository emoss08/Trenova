package carriertools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carrierpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
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
		req *carrierassignmentports.AssignMoveToCarrierRequest,
	) (*shipment.CarrierAssignment, error)
	PreviewAssignToMove(
		ctx context.Context,
		req *carrierassignmentports.AssignMoveToCarrierRequest,
	) (*carrierassignmentservice.CoveragePlan, error)
	Cancel(ctx context.Context, req *carrierassignmentports.CancelCarrierAssignmentRequest) error
	PreviewCancel(
		ctx context.Context,
		req *carrierassignmentports.CancelCarrierAssignmentRequest,
	) (*carrierassignmentservice.CoveragePlan, error)
}

// assignMoveToCarrierTool covers a move with an outside carrier at a rate
// already agreed, which is the dispatch console's carrier assignment. The pay
// it records is what the carrier is settled against.
type AssignMoveToCarrierTool struct {
	carriers carrierCoverage
}

func NewAssignMoveToCarrierTool(carriers carrierCoverage) agenttoolports.AgentTool {
	return &AssignMoveToCarrierTool{carriers: carriers}
}

func (t *AssignMoveToCarrierTool) Name() string { return "dispatch_create_carrier_assignment" }

func (t *AssignMoveToCarrierTool) Recipe() []string {
	return []string{"shipment_get", "carrier_shop", "dispatch_create_carrier_assignment"}
}

func (t *AssignMoveToCarrierTool) Description() string {
	return "Cover a move with an outside carrier at a rate the carrier already agreed to. " +
		"Use it outside a tender: a contract carrier on its contract rate, or a carrier who " +
		"accepted a load by phone or email. Read carrier_shop first so the rate comes from " +
		"a contract or a quote; a rate you cannot source is a reason to ask, never to guess. " +
		"The move must have no driver. Nothing is sent to the carrier: generate the rate " +
		"confirmation with rate_generate_confirmation and send it with " +
		"rate_send_confirmation next. To offer a load and wait for an answer, tender it " +
		"instead."
}

func (t *AssignMoveToCarrierTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		changepreview.PreviewFieldShipmentMoveID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentMove,
			"The move to cover, from dispatch_get_board (moveId) or shipment_get (its moves).",
		),
		carrierpreview.FieldCarrierID: agenttoolschema.RecordIDText(
			permission.ResourceCarrier,
			"The carrier, from carrier_shop or carrier_list.",
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
				changepreview.FieldDescription: jsonschemautils.Text(
					"What the charge is for, as the carrier will read it.",
				),
				fieldAmount: jsonschemautils.Text("The amount as a decimal string."),
				fieldAccessorialChargeID: agenttoolschema.RecordIDText(
					permission.ResourceAccessorialCharge,
					"The accessorial charge it is, from masterdata_list_accessorial_charges.",
				),
			}, changepreview.FieldDescription, fieldAmount),
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
	}, changepreview.PreviewFieldShipmentMoveID, carrierpreview.FieldCarrierID, fieldRateMethod, fieldBaseRate)
}

func (t *AssignMoveToCarrierTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpAssign,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Condition: &agenttoolports.TierCondition{
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
	params *agenttoolports.ToolExecuteParams,
) agent.AutonomyTier {
	if toolparams.OptionalBool(params.Params, fieldReplace) ||
		toolparams.OptionalBool(params.Params, fieldOverrideInsuranceWarning) {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *AssignMoveToCarrierTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(params)
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
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(params)
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
	params *agenttoolports.ToolExecuteParams,
) (*carrierassignmentports.AssignMoveToCarrierRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	moveID, err := toolparams.RequirePulid(params.Params, changepreview.PreviewFieldShipmentMoveID)
	if err != nil {
		return nil, err
	}
	carrierID, err := toolparams.RequirePulid(params.Params, carrierpreview.FieldCarrierID)
	if err != nil {
		return nil, err
	}
	rateMethod, err := carrierRateMethodArg(params.Params)
	if err != nil {
		return nil, err
	}
	baseRate, err := toolparams.DecimalArg(params.Params, fieldBaseRate, true)
	if err != nil {
		return nil, err
	}
	if !baseRate.IsPositive() {
		return nil, errortypes.NewValidationError(fieldBaseRate, errortypes.ErrInvalid,
			"The base rate must be greater than zero; a carrier is never covered for nothing")
	}
	fuelSurcharge, err := toolparams.DecimalArg(params.Params, fieldFuelSurcharge, false)
	if err != nil {
		return nil, err
	}
	accessorials, err := carrierAccessorialsArg(params.Params)
	if err != nil {
		return nil, err
	}

	return &carrierassignmentports.AssignMoveToCarrierRequest{
		TenantInfo:     toolparams.TenantFrom(params),
		ShipmentMoveID: moveID,
		CarrierID:      carrierID,
		RateMethod:     rateMethod,
		BaseRate:       baseRate,
		FuelSurcharge:  fuelSurcharge,
		Accessorials:   accessorials,
		ProNumber:      strings.TrimSpace(toolparams.OptionalString(params.Params, fieldProNumber)),
		ExternalDriverName: strings.TrimSpace(
			toolparams.OptionalString(params.Params, fieldExternalDriverName),
		),
		ExternalDriverPhone: strings.TrimSpace(
			toolparams.OptionalString(params.Params, fieldExternalDriverPhone),
		),
		ExternalTractorNumber: strings.TrimSpace(
			toolparams.OptionalString(params.Params, fieldExternalTractorNumber),
		),
		ExternalTrailerNumber: strings.TrimSpace(
			toolparams.OptionalString(params.Params, fieldExternalTrailerNumber),
		),
		Replace: toolparams.OptionalBool(params.Params, fieldReplace),
		OverrideInsuranceWarning: toolparams.OptionalBool(
			params.Params,
			fieldOverrideInsuranceWarning,
		),
	}, nil
}

func (t *AssignMoveToCarrierTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		changepreview.PreviewFieldShipmentMoveID,
		permission.ResourceShipmentMove,
	)
}

// cancelCarrierAssignmentTool takes the carrier off a move that has not
// started, which is the console's cancel carrier assignment.
type CancelCarrierAssignmentTool struct {
	carriers carrierCoverage
}

func NewCancelCarrierAssignmentTool(carriers carrierCoverage) agenttoolports.AgentTool {
	return &CancelCarrierAssignmentTool{carriers: carriers}
}

func (t *CancelCarrierAssignmentTool) Name() string { return "dispatch_cancel_carrier_assignment" }

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
		changepreview.PreviewFieldShipmentMoveID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentMove,
			"The move whose carrier comes off, from dispatch_get_board (moveId) or "+
				"shipment_get (its moves).",
		),
		changepreview.FieldReason: jsonschemautils.Text(
			"Why the carrier is coming off, in a sentence a person can check.",
		),
	}, changepreview.PreviewFieldShipmentMoveID, changepreview.FieldReason)
}

func (t *CancelCarrierAssignmentTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpUnassign,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Condition: &agenttoolports.TierCondition{
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
	params *agenttoolports.ToolExecuteParams,
) agent.AutonomyTier {
	request, err := t.request(params)
	if err != nil {
		return agent.TierPropose
	}

	plan, err := t.carriers.PreviewCancel(ctx, request)
	if err != nil || plan.CanceledBefore == nil ||
		plan.CanceledBefore.Status == shipmentvocab.CarrierAssignmentStatusConfirmed {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *CancelCarrierAssignmentTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(params)
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
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(params)
	if err != nil {
		return err
	}

	return t.carriers.Cancel(ctx, request)
}

func (t *CancelCarrierAssignmentTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*carrierassignmentports.CancelCarrierAssignmentRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	moveID, err := toolparams.RequirePulid(params.Params, changepreview.PreviewFieldShipmentMoveID)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireString(params.Params, changepreview.FieldReason)
	if err != nil {
		return nil, err
	}

	return &carrierassignmentports.CancelCarrierAssignmentRequest{
		TenantInfo:     toolparams.TenantFrom(params),
		ShipmentMoveID: moveID,
		Reason:         strings.TrimSpace(reason),
	}, nil
}

func (t *CancelCarrierAssignmentTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		changepreview.PreviewFieldShipmentMoveID,
		permission.ResourceShipmentMove,
	)
}

// carrierRateMethodArg refuses anything but the two ways a carrier is paid.
func carrierRateMethodArg(params map[string]any) (shipmentvocab.CarrierRateMethod, error) {
	method := shipmentvocab.CarrierRateMethod(toolparams.OptionalString(params, fieldRateMethod))
	if !method.IsValid() {
		return "", fmt.Errorf(
			"parameter %q must be exactly Flat or PerMile, not %q", fieldRateMethod, method,
		)
	}

	return method, nil
}

func carrierAccessorialsArg(
	params map[string]any,
) ([]carrierassignmentports.CarrierAccessorialInput, error) {
	if _, present := params[fieldAccessorials]; !present {
		return nil, nil
	}

	var lines []carrierAccessorialParam
	if err := toolparams.DecodeParam(params, fieldAccessorials, &lines); err != nil {
		return nil, err
	}
	if len(lines) > maxCarrierAccessorials {
		return nil, fmt.Errorf(
			"parameter %q holds %d charges, more than the %d one assignment carries",
			fieldAccessorials, len(lines), maxCarrierAccessorials,
		)
	}

	out := make([]carrierassignmentports.CarrierAccessorialInput, 0, len(lines))
	for idx, line := range lines {
		amount, err := toolparams.DecimalArg(
			map[string]any{fieldAmount: line.Amount},
			fieldAmount,
			true,
		)
		if err != nil {
			return nil, fmt.Errorf("accessorials[%d]: %w", idx, err)
		}
		input := carrierassignmentports.CarrierAccessorialInput{
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
