package shipmentsuggestionservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	undoWindow   = 10 * time.Minute
	actionLeeway = 5 * time.Minute
	undoReason   = "Undone from the shipment board"
)

func (s *Service) revert(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key shipmentsuggestion.Key,
	decidedAt int64,
) error {
	if s.now().Unix()-decidedAt > int64(undoWindow.Seconds()) {
		return nil
	}
	switch key.Kind {
	case shipmentsuggestion.KindCoverage, shipmentsuggestion.KindTender:
		return s.revertCoverage(ctx, tenantInfo, key.RecordID, decidedAt)
	default:
		return nil
	}
}

func (s *Service) revertCoverage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	decidedAt int64,
) error {
	since := decidedAt - int64(actionLeeway.Seconds())

	live, err := s.tenders.GetLiveByMoveID(ctx, repositories.GetLiveTenderByMoveRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
	})
	if err != nil {
		return err
	}
	if live != nil && live.CreatedAt >= since {
		return s.tenderGuard.CancelLiveTenderForMove(ctx, tenantInfo, moveID, undoReason)
	}

	assignment, err := s.liveAssignment(ctx, tenantInfo, moveID)
	if err != nil || assignment == nil || assignment.CreatedAt < since {
		return err
	}

	return s.assignments.Unassign(ctx, &repositories.UnassignShipmentMoveRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
	})
}

func (s *Service) liveAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.Assignment, error) {
	assignment, err := s.assignmentReads.GetByMoveID(ctx, tenantInfo, moveID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // the move has no driver to take off
		}
		return nil, err
	}

	return assignment, nil
}
