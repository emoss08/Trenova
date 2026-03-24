package shipmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type stopRuleContext struct {
	moveIndex         int
	stopIndex         int
	move              *shipment.ShipmentMove
	stop              *shipment.Stop
	now               int64
	isCreate          bool
	seenStopIDs       map[pulid.ID]string
	seenStopSequences map[int64]int
	multiErr          *errortypes.MultiError
}

func createStopValidationRule() validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.
		NewTenantedRule[*shipment.Shipment]("shipment_stop_validation").
		OnBoth().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *shipment.Shipment,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			now := timeutils.NowUnix()
			seenStopIDs := make(map[pulid.ID]string)

			for moveIndex, move := range entity.Moves {
				if move == nil {
					continue
				}

				seenStopSequences := make(map[int64]int, len(move.Stops))

				for stopIndex, stop := range move.Stops {
					if stop == nil {
						continue
					}

					ruleCtx := stopRuleContext{
						moveIndex:         moveIndex,
						stopIndex:         stopIndex,
						move:              move,
						stop:              stop,
						now:               now,
						isCreate:          valCtx.IsCreate(),
						seenStopIDs:       seenStopIDs,
						seenStopSequences: seenStopSequences,
						multiErr:          multiErr,
					}

					validateStopIdentifiers(ruleCtx)
					validateStopSequence(ruleCtx)
					validateStopChronology(ruleCtx)
				}
			}

			return nil
		})
}

func validateStopIdentifiers(ctx stopRuleContext) {
	if ctx.isCreate {
		if !ctx.stop.ID.IsNil() {
			addStopValidationError(
				ctx,
				"id",
				errortypes.ErrInvalid,
				"Stop ID must not be provided when creating a shipment",
			)
		}

		if !ctx.stop.ShipmentMoveID.IsNil() {
			addStopValidationError(
				ctx,
				"shipmentMoveId",
				errortypes.ErrInvalid,
				"Stop shipment move ID must not be provided when creating a shipment",
			)
		}
	} else if !ctx.stop.ShipmentMoveID.IsNil() &&
		!ctx.move.ID.IsNil() &&
		ctx.stop.ShipmentMoveID != ctx.move.ID {
		addStopValidationError(
			ctx,
			"shipmentMoveId",
			errortypes.ErrInvalid,
			"Stop shipment move ID must match the parent move",
		)
	}

	if ctx.stop.ID.IsNil() {
		return
	}

	currentPath := stopFieldPath(ctx.moveIndex, ctx.stopIndex, "id")
	if firstPath, ok := ctx.seenStopIDs[ctx.stop.ID]; ok {
		ctx.multiErr.Add(
			currentPath,
			errortypes.ErrDuplicate,
			fmt.Sprintf("Stop ID duplicates %s", firstPath),
		)
		return
	}

	ctx.seenStopIDs[ctx.stop.ID] = currentPath
}

func validateStopSequence(ctx stopRuleContext) {
	if firstIndex, ok := ctx.seenStopSequences[ctx.stop.Sequence]; ok {
		ctx.multiErr.Add(
			stopFieldPath(ctx.moveIndex, ctx.stopIndex, "sequence"),
			errortypes.ErrDuplicate,
			fmt.Sprintf(
				"Stop sequence duplicates moves[%d].stops[%d].sequence",
				ctx.moveIndex,
				firstIndex,
			),
		)
		return
	}

	ctx.seenStopSequences[ctx.stop.Sequence] = ctx.stopIndex
}

func validateStopChronology(ctx stopRuleContext) {
	if ctx.stop.ScheduleType != shipment.StopScheduleTypeOpen &&
		ctx.stop.ScheduleType != shipment.StopScheduleTypeAppointment {
		addStopValidationError(
			ctx,
			"scheduleType",
			errortypes.ErrInvalid,
			"Schedule type must be Open or Appointment",
		)
	}

	if ctx.stop.ScheduledWindowStart <= 0 {
		addStopValidationError(
			ctx,
			"scheduledWindowStart",
			errortypes.ErrRequired,
			"Scheduled window start is required",
		)
	}

	if ctx.stop.ScheduledWindowEnd != nil && *ctx.stop.ScheduledWindowEnd < ctx.stop.ScheduledWindowStart {
		addStopValidationError(
			ctx,
			"scheduledWindowEnd",
			errortypes.ErrInvalid,
			"Scheduled window end must be greater than or equal to the scheduled window start",
		)
	}

	if ctx.stop.ActualArrival != nil &&
		ctx.stop.ActualDeparture != nil &&
		*ctx.stop.ActualDeparture < *ctx.stop.ActualArrival {
		addStopValidationError(
			ctx,
			"actualDeparture",
			errortypes.ErrInvalid,
			"Actual departure must be greater than or equal to actual arrival",
		)
	}

	if ctx.stop.ActualArrival != nil && *ctx.stop.ActualArrival > ctx.now {
		addStopValidationError(
			ctx,
			"actualArrival",
			errortypes.ErrInvalid,
			"Actual arrival cannot be in the future",
		)
	}

	if ctx.stop.ActualDeparture != nil && *ctx.stop.ActualDeparture > ctx.now {
		addStopValidationError(
			ctx,
			"actualDeparture",
			errortypes.ErrInvalid,
			"Actual departure cannot be in the future",
		)
	}
}

func addStopValidationError(
	ctx stopRuleContext,
	field string,
	code errortypes.ErrorCode,
	message string,
) {
	ctx.multiErr.Add(stopFieldPath(ctx.moveIndex, ctx.stopIndex, field), code, message)
}

func stopFieldPath(moveIndex, stopIndex int, field string) string {
	return fmt.Sprintf("moves[%d].stops[%d].%s", moveIndex, stopIndex, field)
}
