package tendercancel

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l           *zap.Logger
	repo        tenderports.TenderRepository
	workflows   kernelports.WorkflowStarter
	invalidator shipmentinvalidationports.ShipmentInvalidator
	effects     *tendereffects.Effects
}

// Cancel withdraws a live tender. An Active tender is canceled through its
// workflow so an in-flight response cannot race the withdrawal; a NeedsReview
// tender no longer has a workflow and is canceled directly. The direct path
// also serves as the fallback when the workflow is lost.
func (s *Service) Cancel(ctx context.Context, req *CancelTenderRequest) error {
	entity, err := s.planCancel(ctx, req)
	if err != nil {
		return err
	}

	switch entity.Status {
	case tender.StatusActive:
		signalErr := s.workflows.SignalWorkflow(
			ctx,
			tendercontract.WorkflowID(entity.ID, entity.BounceCount),
			"",
			tendercontract.TenderSignalName,
			tendercontract.Signal{
				Kind:        tendercontract.SignalKindCancel,
				Reason:      req.Reason,
				ActorUserID: req.TenantInfo.UserID,
			},
		)
		if signalErr == nil {
			return nil
		}
		s.l.Warn("failed to signal tender workflow for cancel; canceling directly",
			zap.Error(signalErr), zap.String("tenderId", entity.ID.String()))
		return s.CancelDirect(ctx, req.TenantInfo, entity, req.Reason)
	case tender.StatusNeedsReview:
		return s.CancelDirect(ctx, req.TenantInfo, entity, req.Reason)
	case tender.StatusAccepted, tender.StatusExhausted, tender.StatusCanceled:
		return finishedTenderError(entity)
	default:
		return finishedTenderError(entity)
	}
}

func finishedTenderError(entity *tender.Tender) error {
	return errortypes.NewBusinessError("Tender has already finished").
		WithParam("tenderId", entity.ID.String()).
		WithParam("status", entity.Status.String())
}

// CancelLiveTenderForMove implements the TenderGuard hook: when a move gets
// covered or canceled outside the tender flow, any live tender on it is
// withdrawn so carriers stop receiving (and answering) dead offers.
func (s *Service) CancelLiveTenderForMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	reason string,
) error {
	live, err := s.repo.GetLiveByMoveID(ctx, tenderports.GetLiveTenderByMoveRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
	})
	if err != nil {
		return err
	}
	if live == nil {
		return nil
	}

	return s.Cancel(ctx, &CancelTenderRequest{
		TenantInfo: tenantInfo,
		TenderID:   live.ID,
		Reason:     reason,
	})
}

// CancelLiveTendersForShipment withdraws every live tender under a shipment,
// used when the shipment itself is canceled.
func (s *Service) CancelLiveTendersForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	reason string,
) error {
	live, err := s.liveTendersForShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		return err
	}

	for _, entity := range live {
		if cancelErr := s.Cancel(ctx, &CancelTenderRequest{
			TenantInfo: tenantInfo,
			TenderID:   entity.ID,
			Reason:     reason,
		}); cancelErr != nil {
			s.l.Error("failed to cancel live tender for canceled shipment",
				zap.Error(cancelErr), zap.String("tenderId", entity.ID.String()))
			err = cancelErr
		}
	}

	return err
}

// CancelDirect performs the withdrawal without the workflow: CAS the tender
// terminal, withdraw outstanding offers, revoke their tokens, and record the
// event. Used for NeedsReview tenders and as the lost-workflow fallback.
func (s *Service) CancelDirect(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
	reason string,
) error {
	now := timeutils.NowUnix()

	moved, err := s.repo.UpdateTenderStatus(ctx, &tenderports.UpdateTenderStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   entity.ID,
		FromStatus: []tender.Status{
			tender.StatusActive,
			tender.StatusNeedsReview,
		},
		ToStatus:           tender.StatusCanceled,
		CanceledByID:       tendereffects.UserIDPtr(tenantInfo),
		CancellationReason: reason,
		CanceledAt:         &now,
	})
	if err != nil {
		return err
	}
	if !moved {
		return errortypes.NewBusinessError("Tender has already finished").
			WithParam("tenderId", entity.ID.String())
	}

	for _, withdrawal := range offerWithdrawals {
		if _, err = s.repo.BulkUpdateOfferStatus(ctx, &tenderports.BulkOfferStatusRequest{
			TenantInfo: tenantInfo,
			TenderID:   entity.ID,
			FromStatus: []tender.OfferStatus{withdrawal.from},
			ToStatus:   withdrawal.to,
		}); err != nil {
			return err
		}
	}

	for _, offer := range entity.Offers {
		if offer == nil || offer.Channel != tender.ChannelEmail {
			continue
		}
		if err = s.repo.RevokeTokensForOffer(ctx, tenantInfo, offer.ID, now); err != nil {
			s.l.Warn("failed to revoke tender offer tokens on cancel",
				zap.Error(err), zap.String("offerId", offer.ID.String()))
		}
	}

	s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderWithdrawn(
		shipmenteventservice.TenantRefFor(tenantInfo),
		shipmenteventservice.TenderRef{
			ShipmentID: entity.ShipmentID,
			MoveID:     entity.ShipmentMoveID,
			TenderID:   entity.ID,
		},
		reason,
		shipmenteventservice.ActorFor(tenantInfo),
	))

	s.effects.LogAudit(&tendereffects.AuditParams{
		TenantInfo: tenantInfo,
		TenderID:   entity.ID,
		Operation:  permission.OpCancel,
		UserID:     tenantInfo.UserID,
		Comment:    "Tender canceled: " + reason,
		Previous:   entity,
		Current:    withdrawnTender(entity, tenantInfo, reason, now),
	})
	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationByUser(
			tenantInfo,
			entity.ShipmentID,
			"tender_canceled",
		),
	)

	return nil
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Repo        tenderports.TenderRepository
	Workflows   kernelports.WorkflowStarter
	Invalidator shipmentinvalidationports.ShipmentInvalidator
	Effects     *tendereffects.Effects
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.tender"),
		repo:        p.Repo,
		workflows:   p.Workflows,
		invalidator: p.Invalidator,
		effects:     p.Effects,
	}
}

// PreviewCancel is what Cancel would do, from the same checks, without
// signalling the workflow or writing.
func (s *Service) PreviewCancel(
	ctx context.Context,
	req *CancelTenderRequest,
) (*CancelPreview, error) {
	entity, err := s.planCancel(ctx, req)
	if err != nil {
		return nil, err
	}

	return &CancelPreview{
		Before: entity,
		After:  withdrawnTender(entity, req.TenantInfo, req.Reason, timeutils.NowUnix()),
	}, nil
}

// planCancel is everything Cancel decides before it acts: the tender, with
// its offers, is still live.
func (s *Service) planCancel(
	ctx context.Context,
	req *CancelTenderRequest,
) (*tender.Tender, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	entity, err := s.repo.GetByID(ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      req.TenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}
	if !entity.IsLive() {
		return nil, errortypes.NewBusinessError("Tender has already finished").
			WithParam("tenderId", entity.ID.String()).
			WithParam("status", entity.Status.String())
	}

	return entity, nil
}

// PreviewLiveTenders is what cancelling a shipment withdraws: every live
// tender on it, with the offers carriers hold.
func (s *Service) PreviewLiveTenders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*tender.Tender, error) {
	live, err := s.liveTendersForShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		return nil, err
	}

	withOffers := make([]*tender.Tender, 0, len(live))
	for _, entity := range live {
		loaded, gErr := s.repo.GetByID(ctx, tenderports.GetTenderByIDRequest{
			TenantInfo:    tenantInfo,
			TenderID:      entity.ID,
			IncludeOffers: true,
		})
		if gErr != nil {
			return nil, gErr
		}
		withOffers = append(withOffers, loaded)
	}

	return withOffers, nil
}

func (s *Service) liveTendersForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*tender.Tender, error) {
	tenders, err := s.repo.ListByShipment(ctx, tenderports.ListTendersByShipmentRequest{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, err
	}

	live := make([]*tender.Tender, 0, len(tenders))
	for _, entity := range tenders {
		if entity.IsLive() {
			live = append(live, entity)
		}
	}

	return live, nil
}

var _ tenderports.TenderGuard = (*Service)(nil)
