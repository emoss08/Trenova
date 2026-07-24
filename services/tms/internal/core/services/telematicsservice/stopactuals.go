package telematicsservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type resolvedStop struct {
	moveID pulid.ID
	stop   *shipment.Stop
}

func (s *Service) autoStopActualsEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) bool {
	if s.dispatchControlRepo == nil {
		return false
	}
	dc, err := s.dispatchControlRepo.GetOrCreate(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		s.l.Warn("failed to load dispatch control for auto stop actuals",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
		return false
	}
	return dc.EnableAutoStopActuals
}

func (s *Service) applyStopEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *telematics.TelematicsEvent,
	action repositories.StopActualAction,
) {
	if !s.autoStopActualsEnabled(ctx, tenantInfo) {
		return
	}

	assignment, err := s.resolveActiveAssignment(ctx, tenantInfo, record.TractorID, record.WorkerID)
	if err != nil || assignment == nil {
		return
	}

	resolved, err := s.resolveStop(ctx, tenantInfo, assignment.ShipmentMoveID, record.LocationID)
	if err != nil || resolved == nil {
		return
	}

	_, err = s.shipmentMoveService.RecordStopActual(
		ctx,
		&repositories.RecordStopActualRequest{
			TenantInfo: tenantInfo,
			MoveID:     resolved.moveID,
			StopID:     resolved.stop.ID,
			Action:     action,
		},
	)
	if err != nil {
		if errortypes.IsBusinessError(err) || errortypes.IsError(err) {
			return
		}
		s.l.Warn("failed to auto-record stop actual",
			zap.String("stopId", resolved.stop.ID.String()),
			zap.String("action", string(action)),
			zap.Error(err))
		return
	}

	s.l.Info("auto-recorded stop actual from telematics",
		zap.String("organizationId", tenantInfo.OrgID.String()),
		zap.String("stopId", resolved.stop.ID.String()),
		zap.String("action", string(action)))
}

func (s *Service) resolveActiveAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tractorID pulid.ID,
	workerID pulid.ID,
) (*shipment.Assignment, error) {
	if s.assignmentRepo == nil {
		return nil, nil
	}
	if !tractorID.IsNil() {
		assignment, err := s.assignmentRepo.FindActiveByTractorID(ctx, tenantInfo, tractorID)
		if err != nil {
			return nil, err
		}
		if assignment != nil {
			return assignment, nil
		}
	}
	if !workerID.IsNil() {
		return s.assignmentRepo.FindActiveByWorkerID(ctx, tenantInfo, workerID)
	}
	return nil, nil
}

func (s *Service) resolveStop(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	locationID pulid.ID,
) (*resolvedStop, error) {
	move, err := s.shipmentMoveRepo.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            moveID,
		TenantInfo:        tenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}
	if move == nil {
		return nil, nil
	}

	if !locationID.IsNil() {
		if stop := matchStopByLocation(move.Stops, locationID); stop != nil {
			return &resolvedStop{moveID: move.ID, stop: stop}, nil
		}
	}
	if stop := nextOpenStop(move.Stops); stop != nil {
		return &resolvedStop{moveID: move.ID, stop: stop}, nil
	}
	return nil, nil
}

func matchStopByLocation(stops []*shipment.Stop, locationID pulid.ID) *shipment.Stop {
	for _, stop := range stops {
		if stop == nil || stop.Status == shipment.StopStatusCanceled {
			continue
		}
		if stop.LocationID == locationID && stop.ActualDeparture == nil {
			return stop
		}
	}
	return nil
}

func nextOpenStop(stops []*shipment.Stop) *shipment.Stop {
	var best *shipment.Stop
	for _, stop := range stops {
		if stop == nil || stop.Status == shipment.StopStatusCanceled {
			continue
		}
		if stop.ActualDeparture != nil {
			continue
		}
		if best == nil || stop.Sequence < best.Sequence {
			best = stop
		}
	}
	return best
}

var errNoActiveShipment = errors.New("no active shipment for telematics subject")

func stopExternalLocationID(externalIDs map[string]string) (pulid.ID, bool) {
	if locationID, ok := externalIDs["trenovaLocationId"]; ok {
		if parsed, parseErr := pulid.Parse(locationID); parseErr == nil {
			return parsed, true
		}
	}
	return pulid.Nil, false
}
