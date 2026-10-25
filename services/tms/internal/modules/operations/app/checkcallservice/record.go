package checkcallservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const carrierLinkReferencePrefix = "carrier_tracking_link:"

type recordInput struct {
	tenant       pagination.TenantInfo
	move         *shipment.ShipmentMove
	source       checkcall.Source
	sourceKey    string
	reportedByID pulid.ID
	linkID       pulid.ID
	details      *checkcallports.CheckCallDetails
	reportedAt   int64
}

func (s *Service) record(
	ctx context.Context,
	in *recordInput,
) (*checkcallports.CheckCallResult, error) {
	var result *checkcallports.CheckCallResult
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var visit *trackingeventports.StopEventResult
		if in.details.Kind.IsStopVisit() {
			var err error
			if visit, err = s.recordStopVisit(txCtx, in); err != nil {
				return err
			}
		}

		entity := s.newCheckCall(in)
		if visit != nil && visit.Event != nil {
			entity.TrackingEventID = visit.Event.ID
		}
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		stored, inserted, err := s.repo.Insert(txCtx, entity)
		if err != nil {
			return err
		}
		result = &checkcallports.CheckCallResult{CheckCall: stored, Replayed: !inserted}
		if visit != nil {
			result.Outcome = visit.Outcome
			result.Reason = visit.Reason
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(s.observers) > 0 {
		tenant, shipmentID, moveID := in.tenant, in.move.ShipmentID, in.move.ID
		kernelports.AfterCommit(ctx, func(runCtx context.Context) {
			for _, observer := range s.observers {
				if observer != nil {
					observer.CheckCallRecorded(runCtx, tenant, shipmentID, moveID)
				}
			}
		})
	}
	return result, nil
}

func (s *Service) newCheckCall(in *recordInput) *checkcall.CheckCall {
	now := s.now()
	d := in.details
	entity := &checkcall.CheckCall{
		ID:               checkcall.NewID(),
		OrganizationID:   in.tenant.OrgID,
		BusinessUnitID:   in.tenant.BuID,
		ShipmentID:       in.move.ShipmentID,
		ShipmentMoveID:   in.move.ID,
		Source:           in.source,
		SourceKey:        in.sourceKey,
		Kind:             d.Kind,
		ReportedAt:       in.reportedAt,
		ReceivedAt:       now,
		Latitude:         d.Latitude,
		Longitude:        d.Longitude,
		City:             d.City,
		StateCode:        d.StateCode,
		Delayed:          d.Delayed,
		DelayReason:      d.DelayReason,
		EstimatedArrival: d.EstimatedArrival,
		Note:             d.Note,
		ContactName:      d.ContactName,
		ReportedByID:     in.reportedByID,
		TrackingLinkID:   in.linkID,
		CreatedAt:        now,
	}
	if d.Kind.IsStopVisit() {
		entity.StopID = d.StopID
	}
	return entity
}

func (s *Service) recordStopVisit(
	ctx context.Context,
	in *recordInput,
) (*trackingeventports.StopEventResult, error) {
	d := in.details
	if in.source == checkcall.SourceCarrier {
		return s.trackingEvents.RecordObserved(
			ctx,
			&trackingeventports.RecordObservedStopEventParams{
				TenantInfo:   in.tenant,
				MoveID:       in.move.ID,
				StopID:       d.StopID,
				Kind:         visitKind(d.Kind),
				Source:       trackingevent.SourceCarrier,
				SourceKey:    in.sourceKey,
				MatchMethod:  trackingevent.MatchDirect,
				EventAt:      in.reportedAt,
				Latitude:     d.Latitude,
				Longitude:    d.Longitude,
				SourceStatus: string(d.Kind),
				RawReference: carrierLinkReferencePrefix + in.linkID.String(),
			},
		)
	}

	source := trackingevent.SourceDispatcher
	if in.source == checkcall.SourceAgent {
		source = trackingevent.SourceAgent
	}
	reportedAt := in.reportedAt
	return s.trackingEvents.RecordReported(ctx, &trackingeventports.RecordReportedStopEventParams{
		TenantInfo:   in.tenant,
		MoveID:       in.move.ID,
		StopID:       d.StopID,
		Action:       checkcallports.StopActualAction(d.Kind),
		OccurredAt:   &reportedAt,
		Source:       source,
		SourceKey:    in.sourceKey,
		ReportedByID: in.reportedByID,
		Latitude:     d.Latitude,
		Longitude:    d.Longitude,
	})
}

func (s *Service) Log(
	ctx context.Context,
	req *checkcallports.LogCheckCallRequest,
) (*checkcallports.CheckCallResult, error) {
	in, err := s.prepareLog(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.record(ctx, in)
}

func (s *Service) PreviewLog(
	ctx context.Context,
	req *checkcallports.LogCheckCallRequest,
) (*checkcallports.CheckCallPreview, error) {
	in, err := s.prepareLog(ctx, req)
	if err != nil {
		return nil, err
	}

	entity := s.newCheckCall(in)
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &checkcallports.CheckCallPreview{CheckCall: entity, Move: in.move}, nil
}

func (s *Service) prepareLog(
	ctx context.Context,
	req *checkcallports.LogCheckCallRequest,
) (*recordInput, error) {
	if multiErr := validateLogRequest(req); multiErr != nil {
		return nil, multiErr
	}
	details := req.Details
	reportedAt, multiErr := normalizeDetails(normalizeInput{
		details: &details,
		now:     s.now(),
		policy:  refuseFutureTimes,
	})
	if multiErr != nil {
		return nil, multiErr
	}

	move, err := s.loadMove(ctx, req.TenantInfo, req.MoveID)
	if err != nil {
		return nil, err
	}
	if move.Status == shipment.MoveStatusCanceled {
		return nil, errortypes.NewBusinessError("This load is canceled, so it takes no check calls")
	}
	if details.Kind.IsStopVisit() && findStop(move, details.StopID) == nil {
		return nil, errortypes.NewNotFoundError("Stop not found on this load")
	}

	sourceKey := strings.TrimSpace(req.SourceKey)
	if sourceKey == "" {
		sourceKey = checkcall.NewID().String()
	}
	if len([]rune(sourceKey)) > checkcall.MaxSourceKeyLength {
		return nil, errortypes.NewValidationError(
			"sourceKey",
			errortypes.ErrInvalid,
			"Source key cannot be longer than 255 characters",
		)
	}

	return &recordInput{
		tenant:       req.TenantInfo,
		move:         move,
		source:       req.Source,
		sourceKey:    sourceKey,
		reportedByID: req.ReportedByID,
		details:      &details,
		reportedAt:   reportedAt,
	}, nil
}

func (s *Service) ListForShipment(
	ctx context.Context,
	req *checkcallports.ListCheckCallsByShipmentRequest,
) ([]*checkcall.CheckCall, error) {
	if req == nil || req.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}
	return s.repo.ListByShipment(ctx, req)
}

func (s *Service) loadMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.ShipmentMove, error) {
	return s.moveRepo.GetByID(ctx, &shipmentports.GetMoveByIDRequest{
		MoveID:            moveID,
		TenantInfo:        tenantInfo,
		ExpandMoveDetails: true,
	})
}
