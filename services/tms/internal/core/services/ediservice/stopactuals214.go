package ediservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type stopActualTarget string

const (
	stopTargetOrigin      = stopActualTarget("origin")
	stopTargetDestination = stopActualTarget("destination")
)

type stopActuals214Plan struct {
	target          stopActualTarget
	action          repositories.StopActualAction
	arriveIfMissing bool
}

// stopActualsPlanForStatusCode maps the carrier 214 status codes that carry a
// concrete arrival/departure meaning onto the stop-actual pipeline; every
// other code stays comment-and-event only.
func stopActualsPlanForStatusCode(statusCode string) (stopActuals214Plan, bool) {
	switch statusCode {
	case "X3":
		return stopActuals214Plan{
			target: stopTargetOrigin,
			action: repositories.StopActualActionArrive,
		}, true
	case "AF":
		return stopActuals214Plan{
			target: stopTargetOrigin,
			action: repositories.StopActualActionDepart,
		}, true
	case "X1":
		return stopActuals214Plan{
			target: stopTargetDestination,
			action: repositories.StopActualActionArrive,
		}, true
	case "D1":
		return stopActuals214Plan{
			target:          stopTargetDestination,
			action:          repositories.StopActualActionDepart,
			arriveIfMissing: true,
		}, true
	default:
		return stopActuals214Plan{}, false
	}
}

func resolve214TargetStop(
	move *shipment.ShipmentMove,
	target stopActualTarget,
) *shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(move.Stops))
	for _, stop := range move.Stops {
		if stop != nil && stop.Status != shipment.StopStatusCanceled {
			stops = append(stops, stop)
		}
	}
	sort.SliceStable(stops, func(i, j int) bool { return stops[i].Sequence < stops[j].Sequence })

	switch target {
	case stopTargetOrigin:
		for _, stop := range stops {
			if stop.IsOriginStop() {
				return stop
			}
		}
	case stopTargetDestination:
		for i := len(stops) - 1; i >= 0; i-- {
			if stops[i].IsDestinationStop() {
				return stops[i]
			}
		}
	}
	return nil
}

func is214StopActualBusinessFailure(err error) bool {
	return errortypes.IsBusinessError(err) ||
		errortypes.IsError(err) ||
		errortypes.IsNotFoundError(err)
}

// applyTendered214StopActuals drives the move/stop state machine from a mapped
// carrier 214 status. Business-rule rejections (out-of-order arrivals,
// already-recorded actuals, canceled moves, delivery holds) degrade to
// warnings so the inbound file keeps its comment-and-event record; only
// infrastructure failures propagate.
func (s *Service) applyTendered214StopActuals(
	ctx context.Context,
	req *RecordTenderedShipmentStatusRequest,
) ([]string, error) {
	plan, ok := stopActualsPlanForStatusCode(req.StatusCode)
	if !ok || s.shipmentMoves == nil || s.shipmentMoveRepo == nil {
		return nil, nil
	}
	moveID := req.Offer.Tender.ShipmentMoveID
	if moveID.IsNil() {
		return []string{
			fmt.Sprintf(
				"shipment status %s could not update stop actuals: the accepted tender offer has no shipment move",
				req.StatusCode,
			),
		}, nil
	}
	move, err := s.shipmentMoveRepo.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            moveID,
		TenantInfo:        req.TenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		if is214StopActualBusinessFailure(err) {
			return []string{fmt.Sprintf(
				"shipment status %s could not resolve the tendered move: %s",
				req.StatusCode,
				err.Error(),
			)}, nil
		}
		return nil, err
	}
	stop := resolve214TargetStop(move, plan.target)
	if stop == nil {
		return []string{fmt.Sprintf(
			"shipment status %s has no active %s stop on the tendered move",
			req.StatusCode,
			plan.target,
		)}, nil
	}

	var occurredAt *int64
	if req.EventAt > 0 {
		eventAt := req.EventAt
		occurredAt = &eventAt
	}

	actions := make([]repositories.StopActualAction, 0, 2)
	if plan.arriveIfMissing && stop.ActualArrival == nil {
		actions = append(actions, repositories.StopActualActionArrive)
	}
	actions = append(actions, plan.action)
	for _, action := range actions {
		if _, err = s.shipmentMoves.RecordStopActual(ctx, &repositories.RecordStopActualRequest{
			TenantInfo: req.TenantInfo,
			MoveID:     move.ID,
			StopID:     stop.ID,
			Action:     action,
			OccurredAt: occurredAt,
		}); err != nil {
			if is214StopActualBusinessFailure(err) {
				return []string{fmt.Sprintf(
					"shipment status %s stop %s was not applied: %s",
					req.StatusCode,
					strings.ToLower(string(action)),
					err.Error(),
				)}, nil
			}
			return nil, err
		}
	}
	return nil, nil
}
