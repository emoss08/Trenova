package trackingeventservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/trackingevent"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	reasonNotYetApplied = "Not yet applied to the stop"
	reasonInvoiced      = "This load has been invoiced"
	reasonBillingQueue  = "This load is in the billing queue"
)

type Params struct {
	fx.In

	DB           ports.DBConnection
	Logger       *zap.Logger
	Repo         repositories.TrackingEventRepository
	MoveRepo     repositories.ShipmentMoveRepository
	ShipmentRepo repositories.ShipmentRepository
	MoveService  services.ShipmentMoveService
}

type Service struct {
	db           ports.DBConnection
	l            *zap.Logger
	repo         repositories.TrackingEventRepository
	moveRepo     repositories.ShipmentMoveRepository
	shipmentRepo repositories.ShipmentRepository
	moveService  services.ShipmentMoveService
}

func New(p Params) services.TrackingEventService {
	return &Service{
		db:           p.DB,
		l:            p.Logger.Named("service.tracking-event"),
		repo:         p.Repo,
		moveRepo:     p.MoveRepo,
		shipmentRepo: p.ShipmentRepo,
		moveService:  p.MoveService,
	}
}

func (s *Service) RecordObserved(
	ctx context.Context,
	params *services.RecordObservedStopEventParams,
) (*services.StopEventResult, error) {
	if multiErr := validateObserved(params); multiErr != nil {
		return nil, multiErr
	}

	move, err := s.loadMove(ctx, params.TenantInfo, params.MoveID, params.StopID)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	entity := &trackingevent.TrackingEvent{
		ID:             trackingevent.NewID(),
		OrganizationID: params.TenantInfo.OrgID,
		BusinessUnitID: params.TenantInfo.BuID,
		ShipmentID:     move.ShipmentID,
		ShipmentMoveID: move.ID,
		StopID:         params.StopID,
		Source:         params.Source,
		SourceKey:      strings.TrimSpace(params.SourceKey),
		Kind:           params.Kind,
		MatchMethod:    params.MatchMethod,
		EventAt:        params.EventAt,
		ReceivedAt:     now,
		Latitude:       params.Latitude,
		Longitude:      params.Longitude,
		SourceStatus:   truncate(params.SourceStatus, trackingevent.MaxSourceStatusLength),
		RawReference:   truncate(params.RawReference, trackingevent.MaxRawReferenceLength),
		Outcome:        trackingevent.OutcomePending,
		OutcomeReason:  reasonNotYetApplied,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if multiErr := validateEntity(entity); multiErr != nil {
		return nil, multiErr
	}

	stored, inserted, err := s.repo.Insert(ctx, entity)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return replayed(stored, nil), nil
	}

	reconciled, err := s.reconcile(ctx, params.TenantInfo, move.ID)
	if err != nil {
		if !errortypes.IsRefusal(err) {
			return nil, err
		}
		reason := truncate(errortypes.Summary(err), trackingevent.MaxOutcomeReasonLength)
		if recordErr := s.repo.RecordVerdicts(ctx, params.TenantInfo, []repositories.TrackingEventVerdict{{
			ID:      stored.ID,
			Outcome: trackingevent.OutcomeRefused,
			Reason:  reason,
		}}, timeutils.NowUnix()); recordErr != nil {
			return nil, recordErr
		}
		stored.Outcome = trackingevent.OutcomeRefused
		stored.OutcomeReason = reason
		return &services.StopEventResult{
			Event:   stored,
			Move:    move,
			Outcome: trackingevent.OutcomeRefused,
			Reason:  reason,
		}, nil
	}

	return reconciled.resultFor(stored), nil
}

func (s *Service) RecordReported(
	ctx context.Context,
	params *services.RecordReportedStopEventParams,
) (*services.StopEventResult, error) {
	if multiErr := validateReported(params); multiErr != nil {
		return nil, multiErr
	}

	now := timeutils.NowUnix()
	eventAt := now
	if params.OccurredAt != nil {
		eventAt = *params.OccurredAt
	}
	kind := visitForAction(params.Action)

	var result *services.StopEventResult
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		move, err := s.loadMove(txCtx, params.TenantInfo, params.MoveID, params.StopID)
		if err != nil {
			return err
		}

		sourceKey := strings.TrimSpace(params.SourceKey)
		if sourceKey == "" {
			sourceKey = trackingevent.NewID().String()
		}
		entity := &trackingevent.TrackingEvent{
			ID:             trackingevent.NewID(),
			OrganizationID: params.TenantInfo.OrgID,
			BusinessUnitID: params.TenantInfo.BuID,
			ShipmentID:     move.ShipmentID,
			ShipmentMoveID: move.ID,
			StopID:         params.StopID,
			Source:         params.Source,
			SourceKey:      sourceKey,
			Kind:           kind,
			MatchMethod:    trackingevent.MatchDirect,
			EventAt:        eventAt,
			ReceivedAt:     now,
			Latitude:       params.Latitude,
			Longitude:      params.Longitude,
			ReportedByID:   params.ReportedByID,
			Outcome:        trackingevent.OutcomeApplied,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if multiErr := validateEntity(entity); multiErr != nil {
			return multiErr
		}

		stored, inserted, err := s.repo.Insert(txCtx, entity)
		if err != nil {
			return err
		}
		if !inserted {
			if stored.StopID != params.StopID || stored.Kind != kind {
				return errortypes.NewValidationError(
					"sourceKey",
					errortypes.ErrDuplicate,
					"This report was already recorded for a different stop or action",
				)
			}
			current, getErr := s.moveRepo.GetByID(txCtx, &repositories.GetMoveByIDRequest{
				MoveID:            params.MoveID,
				TenantInfo:        params.TenantInfo,
				ExpandMoveDetails: true,
			})
			if getErr != nil {
				return getErr
			}
			result = replayed(stored, current)
			return nil
		}

		recorded, err := s.moveService.RecordStopActual(txCtx, &repositories.RecordStopActualRequest{
			TenantInfo: params.TenantInfo,
			MoveID:     params.MoveID,
			StopID:     params.StopID,
			Action:     params.Action,
			OccurredAt: &eventAt,
		})
		if err != nil {
			return err
		}

		reconciled, err := s.reconcile(txCtx, params.TenantInfo, params.MoveID)
		if err != nil {
			return err
		}
		if reconciled.move == nil {
			reconciled.move = recorded
		}
		result = reconciled.resultFor(stored)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) Reconcile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() || moveID.IsNil() {
		return errortypes.NewValidationError("moveId", errortypes.ErrRequired, "Move ID is required")
	}
	_, err := s.reconcile(ctx, tenantInfo, moveID)
	return err
}

func (s *Service) ListForShipment(
	ctx context.Context,
	req *repositories.ListTrackingEventsByShipmentRequest,
) ([]*trackingevent.TrackingEvent, error) {
	if req == nil || req.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}
	return s.repo.ListByShipment(ctx, req)
}

type reconciliation struct {
	move     *shipment.ShipmentMove
	verdicts map[pulid.ID]trackingevent.Verdict
}

func (r reconciliation) resultFor(event *trackingevent.TrackingEvent) *services.StopEventResult {
	verdict, ok := r.verdicts[event.ID]
	if ok {
		event.Outcome = verdict.Outcome
		event.OutcomeReason = verdict.Reason
	}
	return &services.StopEventResult{
		Event:   event,
		Move:    r.move,
		Outcome: event.Outcome,
		Reason:  event.OutcomeReason,
	}
}

func (s *Service) reconcile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (reconciliation, error) {
	var out reconciliation
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var events []*trackingevent.TrackingEvent
		var opts trackingevent.Options
		loaded := false

		result, err := s.moveService.ReconcileStopActuals(txCtx, &services.ReconcileStopActualsRequest{
			TenantInfo: tenantInfo,
			MoveID:     moveID,
			Plan: func(planCtx context.Context, move *shipment.ShipmentMove) ([]shipment.StopActualChange, error) {
				if !loaded {
					var loadErr error
					events, loadErr = s.repo.ListByMove(planCtx, tenantInfo, moveID)
					if loadErr != nil {
						return nil, loadErr
					}
					opts, loadErr = s.projectionOptions(planCtx, tenantInfo, move.ShipmentID)
					if loadErr != nil {
						return nil, loadErr
					}
					loaded = true
				}
				projection := trackingevent.Project(move, events, opts)
				out.verdicts = projection.Verdicts
				return projection.Changes, nil
			},
		})
		if err != nil {
			return err
		}
		out.move = result.Move

		return s.repo.RecordVerdicts(
			txCtx,
			tenantInfo,
			changedVerdicts(events, out.verdicts),
			timeutils.NowUnix(),
		)
	})
	if err != nil {
		return reconciliation{}, err
	}
	return out, nil
}

func (s *Service) projectionOptions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (trackingevent.Options, error) {
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return trackingevent.Options{}, err
	}
	switch {
	case entity.StatusEquals(shipment.StatusInvoiced):
		return trackingevent.Options{LockedReason: reasonInvoiced}, nil
	case !entity.BillingTransferStatus.IsOutsideBillingQueue():
		return trackingevent.Options{LockedReason: reasonBillingQueue}, nil
	default:
		return trackingevent.Options{}, nil
	}
}

func (s *Service) loadMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID, stopID pulid.ID,
) (*shipment.ShipmentMove, error) {
	move, err := s.moveRepo.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            moveID,
		TenantInfo:        tenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}
	for _, stop := range move.Stops {
		if stop != nil && stop.ID == stopID {
			return move, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Stop not found on this load")
}

func changedVerdicts(
	events []*trackingevent.TrackingEvent,
	verdicts map[pulid.ID]trackingevent.Verdict,
) []repositories.TrackingEventVerdict {
	changed := make([]repositories.TrackingEventVerdict, 0, len(verdicts))
	for _, event := range events {
		verdict, ok := verdicts[event.ID]
		if !ok {
			continue
		}
		if verdict.Outcome == event.Outcome && verdict.Reason == event.OutcomeReason {
			continue
		}
		changed = append(changed, repositories.TrackingEventVerdict{
			ID:      event.ID,
			Outcome: verdict.Outcome,
			Reason:  verdict.Reason,
		})
	}
	return changed
}

func replayed(
	event *trackingevent.TrackingEvent,
	move *shipment.ShipmentMove,
) *services.StopEventResult {
	return &services.StopEventResult{
		Event:    event,
		Move:     move,
		Outcome:  event.Outcome,
		Reason:   event.OutcomeReason,
		Replayed: true,
	}
}

func visitForAction(action repositories.StopActualAction) shipment.VisitKind {
	if action == repositories.StopActualActionDepart {
		return shipment.VisitDeparture
	}
	return shipment.VisitArrival
}

func validateEntity(entity *trackingevent.TrackingEvent) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
