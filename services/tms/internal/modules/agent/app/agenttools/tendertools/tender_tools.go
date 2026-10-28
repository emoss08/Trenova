package tendertools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const maxTenderLines = 10

// tenderStarter is the slice of the tender service the tools use: the two
// ways a move is offered to carriers.
type tenderStarter interface {
	CreateWaterfall(
		ctx context.Context,
		req *tenderservice.CreateWaterfallTenderRequest,
	) (*tenderservice.CreateWaterfallResult, error)
	CreateSpot(
		ctx context.Context,
		req *tenderservice.CreateSpotTenderRequest,
	) (*tender.Tender, error)
	PreviewWaterfall(
		ctx context.Context,
		req *tenderservice.CreateWaterfallTenderRequest,
	) (*tenderservice.TenderPreview, error)
	PreviewSpot(
		ctx context.Context,
		req *tenderservice.CreateSpotTenderRequest,
	) (*tenderservice.TenderPreview, error)
}

// tenderToRoutingGuideTool offers a move down its routing guide, carrier by
// carrier at the guide's own rates and time limits. It is the tender a
// dispatcher starts with one click, and the one an agent should reach for
// first: the guide is the organization's standing decision about who hauls
// the lane.
type TenderToRoutingGuideTool struct {
	tenders tenderStarter
}

func NewTenderToRoutingGuideTool(tenders tenderStarter) agenttoolports.AgentTool {
	return &TenderToRoutingGuideTool{tenders: tenders}
}

func (t *TenderToRoutingGuideTool) Name() string { return "dispatch_create_tender_from_routing_guide" }

func (t *TenderToRoutingGuideTool) Description() string {
	return "Offer an uncovered move to carriers down its routing guide, in the guide's " +
		"order at the guide's rates. Each carrier gets the guide's time to accept " +
		"before the next is asked. The guide is matched from the move's lane unless " +
		"one is named. The move must have no driver and no live tender. Use " +
		"carrier_shop first when the choice of carrier matters."
}

func (t *TenderToRoutingGuideTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentMoveId": agenttoolschema.RecordIDText(permission.ResourceShipmentMove,
				"The move to tender, from dispatch_get_board (moveId) or "+
					"shipment_get (its moves)."),
			"routingGuideId": agenttoolschema.RecordIDText(permission.ResourceRoutingGuide,
				"A specific routing guide to use instead of the one matched from "+
					"the lane: the routingGuideId carrier_shop returns. Omit to match from the "+
					"lane."),
		},
		"required":             []string{"shipmentMoveId"},
		"additionalProperties": false,
	}
}

func (t *TenderToRoutingGuideTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceTender,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Offers the load to carriers outside the organization.",
	}
}

func (t *TenderToRoutingGuideTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.tenders.CreateWaterfall(ctx, req)

	return err
}

func (t *TenderToRoutingGuideTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*tenderservice.CreateWaterfallTenderRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	moveID, err := common.RequirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return nil, err
	}

	req := &tenderservice.CreateWaterfallTenderRequest{
		TenantInfo:     common.TenantFrom(*params),
		ShipmentMoveID: moveID,
	}
	if guideID, ok, pErr := common.OptionalPulid(params.Params, "routingGuideId"); pErr != nil {
		return nil, pErr
	} else if ok {
		req.RoutingGuideID = &guideID
	}

	return req, nil
}

func (t *TenderToRoutingGuideTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, "shipmentMoveId", permission.ResourceShipmentMove)
}

// tenderToCarriersTool offers a move to named carriers at named rates, either
// all at once or one after another. It is the spot tender: what a dispatcher
// does when the guide has nobody or the load needs a specific carrier.
type TenderToCarriersTool struct {
	tenders tenderStarter
}

func NewTenderToCarriersTool(tenders tenderStarter) agenttoolports.AgentTool {
	return &TenderToCarriersTool{tenders: tenders}
}

func (t *TenderToCarriersTool) Name() string { return "dispatch_create_tender" }

func (t *TenderToCarriersTool) SearchTerms() []string {
	return []string{"tender to carriers"}
}

func (t *TenderToCarriersTool) Description() string {
	return "Offer an uncovered move to specific carriers at specific rates: broadcast " +
		"to all of them at once, or sequentially in the order given. Each line names " +
		"the carrier, the rate and how it is measured (flat or per mile), how long the " +
		"carrier has to accept, and how the offer is sent. Read carrier_shop first so " +
		"the rates come from contracts rather than guesses; a rate you cannot source is " +
		"a reason to raise an exception, not to invent one."
}

func (t *TenderToCarriersTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentMoveId": agenttoolschema.RecordIDText(permission.ResourceShipmentMove,
				"The move to tender, from dispatch_get_board (moveId) or "+
					"shipment_get (its moves)."),
			"mode": agenttoolschema.Enum(
				"Broadcast asks every carrier at once; Sequential asks them in order.",
				agenttoolschema.SpotTenderModes,
			),
			"lines": map[string]any{
				"type":        "array",
				"description": "One line per carrier, in the order they should be asked.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"carrierId": agenttoolschema.RecordIDText(
							permission.ResourceCarrier,
							"The carrier, from carrier_shop or carrier_list.",
						),
						"rate": map[string]any{
							"type":        "string",
							"description": "The offered rate as a decimal string.",
						},
						"rateMethod": agenttoolschema.Enum(
							"How the rate is measured. Defaults to Flat.",
							agenttoolschema.CarrierRateMethods,
						),
						"offerTtlSeconds": map[string]any{
							"type":        "integer",
							"description": "How long the carrier has to accept. Defaults to the organization's setting.",
						},
						"channel": agenttoolschema.Enum(
							"How the offer is sent.",
							agenttoolschema.TenderChannels,
						),
						"email": map[string]any{
							"type":        "string",
							"description": "Where to send an email offer, when not the carrier's own address.",
						},
					},
					"required":             []string{"carrierId", "rate"},
					"additionalProperties": false,
				},
			},
			"overrideInsuranceWarnings": map[string]any{
				"type":        "boolean",
				"description": "Offer to a carrier whose insurance is flagged anyway. Off by default.",
			},
		},
		"required":             []string{"shipmentMoveId", "mode", "lines"},
		"additionalProperties": false,
	}
}

func (t *TenderToCarriersTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceTender,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Offers the load to carriers outside the organization.",
	}
}

type tenderLineParam struct {
	CarrierID       string `json:"carrierId"`
	Rate            string `json:"rate"`
	RateMethod      string `json:"rateMethod"`
	OfferTTLSeconds int32  `json:"offerTtlSeconds"`
	Channel         string `json:"channel"`
	Email           string `json:"email"`
}

func (t *TenderToCarriersTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.tenders.CreateSpot(ctx, req)

	return err
}

func (t *TenderToCarriersTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*tenderservice.CreateSpotTenderRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	moveID, err := common.RequirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return nil, err
	}
	mode, err := common.RequireEnum(params.Params, "mode", agenttoolschema.SpotTenderModes.Values)
	if err != nil {
		return nil, err
	}

	var lineParams []tenderLineParam
	if err = common.DecodeParam(params.Params, "lines", &lineParams); err != nil {
		return nil, err
	}
	if len(lineParams) == 0 || len(lineParams) > maxTenderLines {
		return nil, errortypes.NewValidationError(
			"lines",
			errortypes.ErrInvalid,
			fmt.Sprintf("A spot tender needs between 1 and %d carrier lines", maxTenderLines),
		)
	}

	lines := make([]tenderservice.SpotTenderLine, 0, len(lineParams))
	for idx, line := range lineParams {
		parsed, lineErr := spotTenderLineOf(idx, line)
		if lineErr != nil {
			return nil, lineErr
		}
		lines = append(lines, parsed)
	}

	return &tenderservice.CreateSpotTenderRequest{
		TenantInfo:                common.TenantFrom(*params),
		ShipmentMoveID:            moveID,
		Mode:                      mode,
		Lines:                     lines,
		OverrideInsuranceWarnings: common.OptionalBool(params.Params, "overrideInsuranceWarnings"),
	}, nil
}

func spotTenderLineOf(idx int, line tenderLineParam) (tenderservice.SpotTenderLine, error) {
	carrierID, err := pulid.Parse(line.CarrierID)
	if err != nil {
		return tenderservice.SpotTenderLine{}, fmt.Errorf(
			"lines[%d].carrierId is not a carrier id",
			idx,
		)
	}
	rate, err := decimal.NewFromString(line.Rate)
	if err != nil {
		return tenderservice.SpotTenderLine{}, fmt.Errorf(
			"lines[%d].rate is not a decimal amount",
			idx,
		)
	}

	rateMethod, err := lineEnum(
		idx,
		"rateMethod",
		line.RateMethod,
		agenttoolschema.CarrierRateMethods,
	)
	if err != nil {
		return tenderservice.SpotTenderLine{}, err
	}
	channel, err := lineEnum(idx, "channel", line.Channel, agenttoolschema.TenderChannels)
	if err != nil {
		return tenderservice.SpotTenderLine{}, err
	}

	return tenderservice.SpotTenderLine{
		CarrierID:       carrierID,
		RateMethod:      rateMethod,
		Rate:            rate,
		OfferTTLSeconds: line.OfferTTLSeconds,
		Channel:         channel,
		Email:           line.Email,
	}, nil
}

// lineEnum reads one line's enum value, refusing anything outside the
// source and leaving an absent value to the service's default.
func lineEnum[T ~string](
	idx int,
	key, raw string,
	source agenttoolschema.EnumSource[T],
) (T, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	value := T(strings.TrimSpace(raw))
	if !slices.Contains(source.Values, value) {
		return "", common.ErrUnknownValue(
			fmt.Sprintf("lines[%d].%s", idx, key),
			raw,
			source.Names(),
		)
	}

	return value, nil
}

func (t *TenderToCarriersTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, "shipmentMoveId", permission.ResourceShipmentMove)
}
