package movetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamMoveID = "moveId"

const ParamNewDeliveryLocationID = "newDeliveryLocationId"

const ParamRelayPickupStart = "relayPickupStart"

const paramRelayPickupEnd = "relayPickupEnd"

const ParamNewDeliveryStart = "newDeliveryStart"

const paramNewDeliveryEnd = "newDeliveryEnd"

type moveSplitter interface {
	SplitMove(
		ctx context.Context,
		req *shipmentports.SplitMoveRequest,
	) (*shipmentports.SplitMoveResponse, error)
	PreviewSplitMove(
		ctx context.Context,
		req *shipmentports.SplitMoveRequest,
	) (*shipmentports.MoveSplitPlan, error)
}

func NewSplitMoveTool(moves moveSplitter) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "split_move_at_relay",
		Artifact: common.ShipmentRecordEntity,
		Description: "Split a two-stop move at its delivery stop so the freight is dropped " +
			"there as a relay and a new move carries it on to a new delivery location. The " +
			"relay pickup window and the new delivery window come from the person who asked " +
			"or the customer's instructions, never from a guess: the move keeps its driver " +
			"only as far as the relay. Only a New or Assigned pickup-to-delivery move splits.",
		Resource:    permission.ResourceShipmentMove,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		Rationale: "Adds a move and turns a delivery into a relay inside Trenova; the two new " +
			"windows are times only a person can vouch for, so a person always approves it.",
		Properties: map[string]any{
			ParamMoveID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The two-stop move to split, from get_shipment (its moves) "+
					"or get_dispatch_board. Never guess one.",
			),
			ParamNewDeliveryLocationID: agenttoolschema.RecordIDText(
				permission.ResourceLocation,
				"Where the freight finally goes, from "+
					"list_locations. It must differ from the current delivery location.",
			),
			ParamRelayPickupStart: agenttoolschema.DateTime(
				"When the new move's pickup at the relay " +
					"opens; after the current delivery window ends.",
			),
			paramRelayPickupEnd: agenttoolschema.DateTime("When that relay pickup window closes."),
			ParamNewDeliveryStart: agenttoolschema.DateTime(
				"When the final delivery window opens; after " +
					"the relay pickup window.",
			),
			paramNewDeliveryEnd: agenttoolschema.DateTime("When the final delivery window closes."),
			common.FieldPieces: common.IntegerProperty(
				"Pieces going on from the relay.",
				1,
				1000000,
			),
			common.FieldWeight: common.IntegerProperty(
				"Weight going on from the relay.",
				1,
				1000000,
			),
		},
		Required: []string{
			ParamMoveID, ParamNewDeliveryLocationID, ParamRelayPickupStart, ParamNewDeliveryStart,
		},
		Target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return common.TargetOf(params, ParamMoveID, permission.ResourceShipmentMove)
		},
	}, common.ReceivablePlan[*shipmentports.SplitMoveRequest, *shipmentports.MoveSplitPlan]{
		Request: splitMoveRequest,
		Plan: func(
			ctx context.Context,
			req *shipmentports.SplitMoveRequest,
			_ *agentports.ToolExecuteParams,
		) (*shipmentports.MoveSplitPlan, error) {
			return moves.PreviewSplitMove(ctx, req)
		},
		Refused: func(*shipmentports.SplitMoveRequest) string {
			return "Would split the move at a relay."
		},
		Render: renderMoveSplit,
		Run: func(
			ctx context.Context,
			req *shipmentports.SplitMoveRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			response, err := moves.SplitMove(ctx, req)
			if err != nil {
				return nil, err
			}
			result := &agent.ToolExecutionResult{Action: "split", Kind: "move"}
			if response != nil && response.NewMove != nil {
				result.IDs = map[string]string{
					ParamMoveID: req.MoveID.String(),
					"newMoveId": response.NewMove.ID.String(),
				}
			}

			return result, nil
		},
	})
}

func splitMoveRequest(
	params *agentports.ToolExecuteParams,
) (*shipmentports.SplitMoveRequest, error) {
	moveID, err := common.RequirePulid(params.Params, ParamMoveID)
	if err != nil {
		return nil, err
	}
	locationID, err := common.RequirePulid(params.Params, ParamNewDeliveryLocationID)
	if err != nil {
		return nil, err
	}
	relayStart, err := common.RequireDateTime(params.Params, ParamRelayPickupStart)
	if err != nil {
		return nil, err
	}
	relayEnd, err := common.OptionalDateTime(params.Params, paramRelayPickupEnd)
	if err != nil {
		return nil, err
	}
	deliveryStart, err := common.RequireDateTime(params.Params, ParamNewDeliveryStart)
	if err != nil {
		return nil, err
	}
	deliveryEnd, err := common.OptionalDateTime(params.Params, paramNewDeliveryEnd)
	if err != nil {
		return nil, err
	}

	req := &shipmentports.SplitMoveRequest{
		TenantInfo:            common.TenantFrom(*params),
		MoveID:                moveID,
		NewDeliveryLocationID: locationID,
		SplitPickupTimes: shipmentports.SplitStopTimes{
			ScheduledWindowStart: relayStart,
			ScheduledWindowEnd:   relayEnd,
		},
		NewDeliveryTimes: shipmentports.SplitStopTimes{
			ScheduledWindowStart: deliveryStart,
			ScheduledWindowEnd:   deliveryEnd,
		},
	}
	if pieces := common.OptionalInt64(params.Params, common.FieldPieces); pieces > 0 {
		req.Pieces = &pieces
	}
	if weight := common.OptionalInt64(params.Params, common.FieldWeight); weight > 0 {
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
	ParamRelayPickupStart: assistantartifact.DisplayDateTime,
	paramRelayPickupEnd:   assistantartifact.DisplayDateTime,
	ParamNewDeliveryStart: assistantartifact.DisplayDateTime,
	paramNewDeliveryEnd:   assistantartifact.DisplayDateTime,
})

func renderMoveSplit(
	_ *shipmentports.SplitMoveRequest,
	plan *shipmentports.MoveSplitPlan,
) (*agent.ToolPreview, error) {
	original := plan.Move.Stops[1]
	relay, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceShipmentMove,
			ID:       plan.Move.ID,
			Label:    fmt.Sprintf("Move %d, %s", plan.Move.Sequence+1, common.StopLabel(original)),
			Version:  common.MoveVersion(plan.Move),
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
		common.StopLabel(original),
	), relay, created), nil
}
