package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
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
}

// tenderToRoutingGuideTool offers a move down its routing guide, carrier by
// carrier at the guide's own rates and time limits. It is the tender a
// dispatcher starts with one click, and the one an agent should reach for
// first: the guide is the organization's standing decision about who hauls
// the lane.
type tenderToRoutingGuideTool struct {
	tenders tenderStarter
}

func newTenderToRoutingGuideTool(tenders tenderStarter) serviceports.AgentTool {
	return &tenderToRoutingGuideTool{tenders: tenders}
}

func (t *tenderToRoutingGuideTool) Name() string { return "tender_move_to_routing_guide" }

func (t *tenderToRoutingGuideTool) Description() string {
	return "Offer an uncovered move to carriers down its routing guide, in the guide's " +
		"order at the guide's rates. Each carrier gets the guide's time to accept " +
		"before the next is asked. The guide is matched from the move's lane unless " +
		"one is named. The move must have no driver and no live tender. Use " +
		"shop_carriers first when the choice of carrier matters."
}

func (t *tenderToRoutingGuideTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentMoveId": map[string]any{
				"type": "string",
				"description": "The move to tender, from get_dispatch_board (moveId) or " +
					"get_shipment (its moves).",
			},
			"routingGuideId": map[string]any{
				"type": "string",
				"description": "A specific routing guide to use instead of the one matched from " +
					"the lane: the routingGuideId shop_carriers returns. Omit to match from the " +
					"lane.",
			},
		},
		"required":             []string{"shipmentMoveId"},
		"additionalProperties": false,
	}
}

func (t *tenderToRoutingGuideTool) Reversible() bool { return true }

func (t *tenderToRoutingGuideTool) PermissionResource() permission.Resource {
	return permission.ResourceTender
}

func (t *tenderToRoutingGuideTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

// RequiresIdempotencyKey is true because a second tender for the same move
// is refused by the service, and a retry should not read as a failure.
func (t *tenderToRoutingGuideTool) RequiresIdempotencyKey() bool { return true }

func (t *tenderToRoutingGuideTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *tenderToRoutingGuideTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	moveID, err := requirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return err
	}

	req := &tenderservice.CreateWaterfallTenderRequest{
		TenantInfo:     tenantFrom(params),
		ShipmentMoveID: moveID,
	}
	if guideID, ok, pErr := optionalPulid(params.Params, "routingGuideId"); pErr != nil {
		return pErr
	} else if ok {
		req.RoutingGuideID = &guideID
	}

	_, err = t.tenders.CreateWaterfall(ctx, req)

	return err
}

func (t *tenderToRoutingGuideTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentMoveId", permission.ResourceShipmentMove)
}

// tenderToCarriersTool offers a move to named carriers at named rates, either
// all at once or one after another. It is the spot tender: what a dispatcher
// does when the guide has nobody or the load needs a specific carrier.
type tenderToCarriersTool struct {
	tenders tenderStarter
}

func newTenderToCarriersTool(tenders tenderStarter) serviceports.AgentTool {
	return &tenderToCarriersTool{tenders: tenders}
}

func (t *tenderToCarriersTool) Name() string { return "tender_move_to_carriers" }

func (t *tenderToCarriersTool) Description() string {
	return "Offer an uncovered move to specific carriers at specific rates: broadcast " +
		"to all of them at once, or sequentially in the order given. Each line names " +
		"the carrier, the rate and how it is measured (flat or per mile), how long the " +
		"carrier has to accept, and how the offer is sent. Read shop_carriers first so " +
		"the rates come from contracts rather than guesses; a rate you cannot source is " +
		"a reason to raise an exception, not to invent one."
}

func (t *tenderToCarriersTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentMoveId": map[string]any{
				"type": "string",
				"description": "The move to tender, from get_dispatch_board (moveId) or " +
					"get_shipment (its moves).",
			},
			"mode": map[string]any{
				"type": "string",
				"enum": []string{
					string(tender.ModeSpotBroadcast),
					string(tender.ModeSpotSequential),
				},
				"description": "Broadcast asks every carrier at once; Sequential asks them in order.",
			},
			"lines": map[string]any{
				"type":        "array",
				"description": "One line per carrier, in the order they should be asked.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"carrierId": map[string]any{
							"type":        "string",
							"description": "The carrier, from shop_carriers or list_carriers.",
						},
						"rate": map[string]any{
							"type":        "string",
							"description": "The offered rate as a decimal string.",
						},
						"rateMethod": map[string]any{
							"type": "string",
							"enum": []string{"Flat", "PerMile"},
						},
						"offerTtlSeconds": map[string]any{
							"type":        "integer",
							"description": "How long the carrier has to accept. Defaults to the organization's setting.",
						},
						"channel": map[string]any{
							"type": "string",
							"enum": []string{"Email", "EDI"},
						},
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

func (t *tenderToCarriersTool) Reversible() bool { return true }

func (t *tenderToCarriersTool) PermissionResource() permission.Resource {
	return permission.ResourceTender
}

func (t *tenderToCarriersTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *tenderToCarriersTool) RequiresIdempotencyKey() bool { return true }

func (t *tenderToCarriersTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

type tenderLineParam struct {
	CarrierID       string `json:"carrierId"`
	Rate            string `json:"rate"`
	RateMethod      string `json:"rateMethod"`
	OfferTTLSeconds int32  `json:"offerTtlSeconds"`
	Channel         string `json:"channel"`
	Email           string `json:"email"`
}

func (t *tenderToCarriersTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	moveID, err := requirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return err
	}
	mode, err := requireString(params.Params, "mode")
	if err != nil {
		return err
	}

	var lineParams []tenderLineParam
	if err = decodeParam(params.Params, "lines", &lineParams); err != nil {
		return err
	}
	if len(lineParams) == 0 || len(lineParams) > maxTenderLines {
		return errortypes.NewValidationError(
			"lines",
			errortypes.ErrInvalid,
			fmt.Sprintf("A spot tender needs between 1 and %d carrier lines", maxTenderLines),
		)
	}

	lines := make([]tenderservice.SpotTenderLine, 0, len(lineParams))
	for idx, line := range lineParams {
		parsed, lineErr := spotTenderLineOf(idx, line)
		if lineErr != nil {
			return lineErr
		}
		lines = append(lines, parsed)
	}

	_, err = t.tenders.CreateSpot(ctx, &tenderservice.CreateSpotTenderRequest{
		TenantInfo:                tenantFrom(params),
		ShipmentMoveID:            moveID,
		Mode:                      tender.Mode(mode),
		Lines:                     lines,
		OverrideInsuranceWarnings: optionalBool(params.Params, "overrideInsuranceWarnings"),
	})

	return err
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

	return tenderservice.SpotTenderLine{
		CarrierID:       carrierID,
		RateMethod:      shipment.CarrierRateMethod(line.RateMethod),
		Rate:            rate,
		OfferTTLSeconds: line.OfferTTLSeconds,
		Channel:         tender.Channel(line.Channel),
		Email:           line.Email,
	}, nil
}

func (t *tenderToCarriersTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentMoveId", permission.ResourceShipmentMove)
}
