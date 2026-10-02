package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) PreviewUncancel(
	ctx context.Context,
	req *repositories.UncancelShipmentRequest,
) (*services.ShipmentChangePreview, error) {
	if req == nil {
		return nil, uncancelRequestRequired()
	}

	request := *req
	original, restored, err := s.planUncancel(ctx, &request)
	if err != nil {
		return nil, err
	}

	return &services.ShipmentChangePreview{Before: original, After: restored}, nil
}

func (s *service) planUncancel(
	ctx context.Context,
	req *repositories.UncancelShipmentRequest,
) (original, restored *shipment.Shipment, err error) {
	if req == nil {
		return nil, nil, uncancelRequestRequired()
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, nil, multiErr
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	}

	original, err = s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              req.ShipmentID,
		TenantInfo:      tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, nil, err
	}

	if !original.IsCanceled() {
		return nil, nil, errortypes.NewBusinessError("shipment is not canceled")
	}

	control, err := s.getShipmentControl(ctx, tenantInfo)
	if err != nil {
		return nil, nil, err
	}

	restored = original.CloneMoveGraph()
	s.coordinator.PrepareForUncancel(restored, delayThresholdMinutes(control))

	req.ExpectedVersion = original.Version
	req.RestoredStatus = restored.Status
	req.MoveStatuses, req.StopStatuses = uncancelRestores(original, restored)

	return original, restored, nil
}

func uncancelRequestRequired() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("request", errortypes.ErrRequired, "Uncancel request is required")

	return multiErr
}

func (s *service) PreviewTransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
	actor *services.RequestActor,
) (*services.ShipmentChangePreview, error) {
	original, err := s.planTransferOwnership(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := *original
	after.OwnerID = req.OwnerID

	return &services.ShipmentChangePreview{Before: original, After: &after}, nil
}

func (s *service) planTransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
	actor *services.RequestActor,
) (*shipment.Shipment, error) {
	if req == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("request", errortypes.ErrRequired, "Transfer ownership request is required")
		return nil, multiErr
	}

	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	auditActor := actor.AuditActor()
	if actor.PersonUserID().IsNil() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrInvalidOperation,
			"Shipment ownership transfer requires a user actor",
		)
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	if original.OwnerID == req.OwnerID {
		return nil, errortypes.NewValidationError(
			"ownerId",
			errortypes.ErrInvalid,
			"Shipment already belongs to this owner",
		)
	}

	if err = s.validateTransferActor(ctx, auditActor, original, req.TenantInfo.OrgID); err != nil {
		return nil, err
	}

	if err = s.validateTransferTarget(ctx, req); err != nil {
		return nil, err
	}

	return original, nil
}

func (s *service) PreviewRecalculateDistance(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*services.ShipmentDistancePreview, error) {
	if s.distanceCalculation == nil {
		return nil, errortypes.NewBusinessError("distance calculation service is not configured")
	}

	return s.distanceCalculation.PreviewRecalculateShipment(ctx, shipmentID, tenantInfo)
}

func (s *service) PreviewDuplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateShipmentRequest,
) (*services.ShipmentDuplicatePreview, error) {
	plan, err := s.planDuplicate(ctx, req)
	if err != nil {
		return nil, err
	}

	return &services.ShipmentDuplicatePreview{Source: plan.Source, Copies: plan.Copies}, nil
}

func (s *service) planDuplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateShipmentRequest,
) (*repositories.ShipmentDuplicatePlan, error) {
	if err := s.guardDuplicate(req); err != nil {
		return nil, err
	}

	plan, err := s.repo.PlanDuplicate(ctx, req)
	if err != nil {
		return nil, err
	}

	if err = (rateCoverage{billing: s.billingRepo}).refuseUnratedCopies(ctx, plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *service) guardDuplicate(req *repositories.BulkDuplicateShipmentRequest) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	if !s.workflowStarter.Enabled() {
		return errortypes.NewBusinessError("shipment duplication is not configured")
	}

	return nil
}

func uncancelRestores(
	original, restored *shipment.Shipment,
) ([]repositories.MoveStatusRestore, []repositories.StopStatusRestore) {
	moves := make([]repositories.MoveStatusRestore, 0, len(restored.Moves))
	stops := make([]repositories.StopStatusRestore, 0)

	for moveIndex, move := range restored.Moves {
		if move == nil || moveIndex >= len(original.Moves) || original.Moves[moveIndex] == nil {
			continue
		}

		before := original.Moves[moveIndex]
		if before.IsCanceled() && !move.IsCanceled() {
			moves = append(moves, repositories.MoveStatusRestore{
				MoveID: move.ID,
				Status: move.Status,
			})
		}

		for stopIndex, stop := range move.Stops {
			if stop == nil || stopIndex >= len(before.Stops) || before.Stops[stopIndex] == nil {
				continue
			}

			if before.Stops[stopIndex].IsCanceled() && !stop.IsCanceled() {
				stops = append(stops, repositories.StopStatusRestore{
					StopID: stop.ID,
					Status: stop.Status,
				})
			}
		}
	}

	return moves, stops
}
