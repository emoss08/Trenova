package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramMoveID                = "moveId"
	paramNewDeliveryLocationID = "newDeliveryLocationId"
	paramRelayPickupStart      = "relayPickupStart"
	paramRelayPickupEnd        = "relayPickupEnd"
	paramNewDeliveryStart      = "newDeliveryStart"
	paramNewDeliveryEnd        = "newDeliveryEnd"
)

type moveSplitter interface {
	SplitMove(
		ctx context.Context,
		req *repositories.SplitMoveRequest,
	) (*repositories.SplitMoveResponse, error)
	PreviewSplitMove(
		ctx context.Context,
		req *repositories.SplitMoveRequest,
	) (*serviceports.MoveSplitPlan, error)
}

func newSplitMoveTool(moves moveSplitter) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "split_move_at_relay",
		artifact: shipmentRecordEntity,
		description: "Split a two-stop move at its delivery stop so the freight is dropped " +
			"there as a relay and a new move carries it on to a new delivery location. The " +
			"relay pickup window and the new delivery window come from the person who asked " +
			"or the customer's instructions, never from a guess: the move keeps its driver " +
			"only as far as the relay. Only a New or Assigned pickup-to-delivery move splits.",
		resource:    permission.ResourceShipmentMove,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		rationale: "Adds a move and turns a delivery into a relay inside Trenova; the two new " +
			"windows are times only a person can vouch for, so a person always approves it.",
		properties: map[string]any{
			paramMoveID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The two-stop move to split, from get_shipment (its moves) "+
					"or get_dispatch_board. Never guess one.",
			),
			paramNewDeliveryLocationID: agenttoolschema.RecordIDText(
				permission.ResourceLocation,
				"Where the freight finally goes, from "+
					"list_locations. It must differ from the current delivery location.",
			),
			paramRelayPickupStart: agenttoolschema.DateTime(
				"When the new move's pickup at the relay " +
					"opens; after the current delivery window ends.",
			),
			paramRelayPickupEnd: agenttoolschema.DateTime("When that relay pickup window closes."),
			paramNewDeliveryStart: agenttoolschema.DateTime(
				"When the final delivery window opens; after " +
					"the relay pickup window.",
			),
			paramNewDeliveryEnd: agenttoolschema.DateTime("When the final delivery window closes."),
			fieldPieces:         integerProperty("Pieces going on from the relay.", 1, 1000000),
			fieldWeight:         integerProperty("Weight going on from the relay.", 1, 1000000),
		},
		required: []string{
			paramMoveID, paramNewDeliveryLocationID, paramRelayPickupStart, paramNewDeliveryStart,
		},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramMoveID, permission.ResourceShipmentMove)
		},
	}, receivablePlan[*repositories.SplitMoveRequest, *serviceports.MoveSplitPlan]{
		request: splitMoveRequest,
		plan: func(
			ctx context.Context,
			req *repositories.SplitMoveRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.MoveSplitPlan, error) {
			return moves.PreviewSplitMove(ctx, req)
		},
		refused: func(*repositories.SplitMoveRequest) string {
			return "Would split the move at a relay."
		},
		render: renderMoveSplit,
		run: func(
			ctx context.Context,
			req *repositories.SplitMoveRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			response, err := moves.SplitMove(ctx, req)
			if err != nil {
				return nil, err
			}
			result := &agent.ToolExecutionResult{Action: "split", Kind: "move"}
			if response != nil && response.NewMove != nil {
				result.IDs = map[string]string{
					paramMoveID: req.MoveID.String(),
					"newMoveId": response.NewMove.ID.String(),
				}
			}

			return result, nil
		},
	})
}

func splitMoveRequest(
	params *serviceports.ToolExecuteParams,
) (*repositories.SplitMoveRequest, error) {
	moveID, err := requirePulid(params.Params, paramMoveID)
	if err != nil {
		return nil, err
	}
	locationID, err := requirePulid(params.Params, paramNewDeliveryLocationID)
	if err != nil {
		return nil, err
	}
	relayStart, err := requireDateTime(params.Params, paramRelayPickupStart)
	if err != nil {
		return nil, err
	}
	relayEnd, err := optionalDateTime(params.Params, paramRelayPickupEnd)
	if err != nil {
		return nil, err
	}
	deliveryStart, err := requireDateTime(params.Params, paramNewDeliveryStart)
	if err != nil {
		return nil, err
	}
	deliveryEnd, err := optionalDateTime(params.Params, paramNewDeliveryEnd)
	if err != nil {
		return nil, err
	}

	req := &repositories.SplitMoveRequest{
		TenantInfo:            tenantFrom(*params),
		MoveID:                moveID,
		NewDeliveryLocationID: locationID,
		SplitPickupTimes: repositories.SplitStopTimes{
			ScheduledWindowStart: relayStart,
			ScheduledWindowEnd:   relayEnd,
		},
		NewDeliveryTimes: repositories.SplitStopTimes{
			ScheduledWindowStart: deliveryStart,
			ScheduledWindowEnd:   deliveryEnd,
		},
	}
	if pieces := optionalInt64(params.Params, fieldPieces); pieces > 0 {
		req.Pieces = &pieces
	}
	if weight := optionalInt64(params.Params, fieldWeight); weight > 0 {
		req.Weight = &weight
	}

	return req, nil
}

type relayStopView struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type splitMoveView struct {
	RelayLocationID    pulid.ID `json:"relayLocationId"`
	RelayPickupStart   int64    `json:"relayPickupStart"`
	RelayPickupEnd     *int64   `json:"relayPickupEnd"`
	DeliveryLocationID pulid.ID `json:"deliveryLocationId"`
	NewDeliveryStart   int64    `json:"newDeliveryStart"`
	NewDeliveryEnd     *int64   `json:"newDeliveryEnd"`
	Pieces             *int64   `json:"pieces"`
	Weight             *int64   `json:"weight"`
	Sequence           int64    `json:"sequence"`
}

func splitViewOf(move *shipment.ShipmentMove) *splitMoveView {
	pickup, delivery := move.Stops[0], move.Stops[1]

	return &splitMoveView{
		RelayLocationID:    pickup.LocationID,
		RelayPickupStart:   pickup.ScheduledWindowStart,
		RelayPickupEnd:     pickup.ScheduledWindowEnd,
		DeliveryLocationID: delivery.LocationID,
		NewDeliveryStart:   delivery.ScheduledWindowStart,
		NewDeliveryEnd:     delivery.ScheduledWindowEnd,
		Pieces:             pickup.Pieces,
		Weight:             pickup.Weight,
		Sequence:           move.Sequence + 1,
	}
}

var splitDateTimes = toolpreview.Types(map[string]assistantartifact.DisplayType{
	paramRelayPickupStart: assistantartifact.DisplayDateTime,
	paramRelayPickupEnd:   assistantartifact.DisplayDateTime,
	paramNewDeliveryStart: assistantartifact.DisplayDateTime,
	paramNewDeliveryEnd:   assistantartifact.DisplayDateTime,
})

func renderMoveSplit(
	_ *repositories.SplitMoveRequest,
	plan *serviceports.MoveSplitPlan,
) (*agent.ToolPreview, error) {
	original := plan.Move.Stops[1]
	relay, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceShipmentMove,
			ID:       plan.Move.ID,
			Label:    fmt.Sprintf("Move %d, %s", plan.Move.Sequence+1, stopLabel(original)),
			Version:  moveVersion(plan.Move),
		},
		&relayStopView{Type: string(original.Type), Status: string(original.Status)},
		&relayStopView{
			Type:   string(plan.Split.RelayStop.Type),
			Status: string(plan.Split.RelayStop.Status),
		},
	)
	if err != nil {
		return nil, err
	}

	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceShipmentMove,
			Label:    fmt.Sprintf("New move %d from the relay", plan.Split.NewMove.Sequence+1),
		},
		splitViewOf(plan.Split.NewMove),
		toolpreview.WithRefs(map[string]permission.Resource{
			"relayLocationId":    permission.ResourceLocation,
			"deliveryLocationId": permission.ResourceLocation,
		}),
		splitDateTimes,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would make %s a relay drop and add a move carrying the freight on to a new "+
			"delivery. Moves after it move one place down.",
		stopLabel(original),
	), relay, created), nil
}
