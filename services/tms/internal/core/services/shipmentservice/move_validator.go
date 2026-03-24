package shipmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

type moveRuleContext struct {
	moveIndex   int
	move        *shipment.ShipmentMove
	shipmentID  pulid.ID
	isCreate    bool
	seenMoveIDs map[pulid.ID]int
	multiErr    *errortypes.MultiError
}

func createMoveValidationRule() validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.
		NewTenantedRule[*shipment.Shipment]("shipment_move_validation").
		OnBoth().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *shipment.Shipment,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			seenMoveIDs := make(map[pulid.ID]int, len(entity.Moves))

			for moveIndex, move := range entity.Moves {
				if move == nil {
					continue
				}

				ruleCtx := moveRuleContext{
					moveIndex:   moveIndex,
					move:        move,
					shipmentID:  entity.ID,
					isCreate:    valCtx.IsCreate(),
					seenMoveIDs: seenMoveIDs,
					multiErr:    multiErr,
				}

				validateMoveIdentifiers(ruleCtx)
				validateMoveStopCount(ruleCtx)
				validateMoveStopOrder(ruleCtx)
				validateMoveStopChronology(ruleCtx)
			}

			return nil
		})
}

func validateMoveIdentifiers(ctx moveRuleContext) {
	if !ctx.move.ID.IsNil() {
		if firstIndex, ok := ctx.seenMoveIDs[ctx.move.ID]; ok {
			ctx.multiErr.Add(
				moveFieldPath(ctx.moveIndex, "id"),
				errortypes.ErrDuplicate,
				fmt.Sprintf("Move ID duplicates moves[%d].id", firstIndex),
			)
		} else {
			ctx.seenMoveIDs[ctx.move.ID] = ctx.moveIndex
		}
	}

	if ctx.isCreate {
		if !ctx.move.ID.IsNil() {
			ctx.multiErr.Add(
				moveFieldPath(ctx.moveIndex, "id"),
				errortypes.ErrInvalid,
				"Move ID must not be provided when creating a shipment",
			)
		}
		return
	}

	if !ctx.shipmentID.IsNil() && !ctx.move.ShipmentID.IsNil() && ctx.move.ShipmentID != ctx.shipmentID {
		ctx.multiErr.Add(
			moveFieldPath(ctx.moveIndex, "shipmentId"),
			errortypes.ErrInvalid,
			"Move shipment ID must match the shipment being updated",
		)
	}
}

func validateMoveStopCount(ctx moveRuleContext) {
	if len(ctx.move.Stops) >= 2 {
		return
	}

	ctx.multiErr.Add(
		moveFieldPath(ctx.moveIndex, "stops"),
		errortypes.ErrInvalid,
		"Move must contain at least two stops",
	)
}

func validateMoveStopOrder(ctx moveRuleContext) {
	if len(ctx.move.Stops) == 0 {
		return
	}

	firstStop := ctx.move.Stops[0]
	if firstStop != nil && !isPickupLikeStopType(firstStop.Type) {
		ctx.multiErr.Add(
			stopFieldPath(ctx.moveIndex, 0, "type"),
			errortypes.ErrInvalid,
			"First stop must be a pickup or split pickup",
		)
	}

	lastIndex := len(ctx.move.Stops) - 1
	lastStop := ctx.move.Stops[lastIndex]
	if lastStop != nil && !isDeliveryLikeStopType(lastStop.Type) {
		ctx.multiErr.Add(
			stopFieldPath(ctx.moveIndex, lastIndex, "type"),
			errortypes.ErrInvalid,
			"Last stop must be a delivery or split delivery",
		)
	}

	hasPickup := false
	for stopIndex, stop := range ctx.move.Stops {
		if stop == nil {
			continue
		}

		switch {
		case isPickupLikeStopType(stop.Type):
			hasPickup = true
		case isDeliveryLikeStopType(stop.Type):
			if !hasPickup {
				ctx.multiErr.Add(
					stopFieldPath(ctx.moveIndex, stopIndex, "type"),
					errortypes.ErrInvalid,
					"Delivery stop must be preceded by a pickup or split pickup",
				)
			}
		default:
			ctx.multiErr.Add(
				stopFieldPath(ctx.moveIndex, stopIndex, "type"),
				errortypes.ErrInvalid,
				"Stop type must be pickup, split pickup, delivery, or split delivery",
			)
		}
	}
}

func validateMoveStopChronology(ctx moveRuleContext) {
	if len(ctx.move.Stops) < 2 {
		return
	}

	for stopIndex := 0; stopIndex < len(ctx.move.Stops)-1; stopIndex++ {
		currentStop := ctx.move.Stops[stopIndex]
		nextStop := ctx.move.Stops[stopIndex+1]
		if currentStop == nil || nextStop == nil {
			continue
		}

		if currentStop.EffectiveScheduledWindowEnd() >= nextStop.ScheduledWindowStart {
			ctx.multiErr.Add(
				stopFieldPath(ctx.moveIndex, stopIndex, "scheduledWindowEnd"),
				errortypes.ErrInvalid,
				"Scheduled window end must be before the next stop's scheduled window start",
			)
		}

		if currentStop.ActualDeparture != nil &&
			nextStop.ActualArrival != nil &&
			*currentStop.ActualDeparture >= *nextStop.ActualArrival {
			ctx.multiErr.Add(
				stopFieldPath(ctx.moveIndex, stopIndex, "actualDeparture"),
				errortypes.ErrInvalid,
				"Actual departure must be before the next stop's actual arrival",
			)
		}
	}
}

func moveFieldPath(moveIndex int, field string) string {
	return fmt.Sprintf("moves[%d].%s", moveIndex, field)
}

func isPickupLikeStopType(stopType shipment.StopType) bool {
	return stopType == shipment.StopTypePickup || stopType == shipment.StopTypeSplitPickup
}

func isDeliveryLikeStopType(stopType shipment.StopType) bool {
	return stopType == shipment.StopTypeDelivery || stopType == shipment.StopTypeSplitDelivery
}
