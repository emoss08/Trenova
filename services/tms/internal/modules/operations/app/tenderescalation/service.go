package tenderescalation

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Escalate follows an exhausted tender with the next round the
// organization's policy asks for: the same carriers again, all at once or in
// rank order, at the escalation rate and expiry. It stops at the policy's
// round limit, and never when the move has since been covered, held or
// canceled, because the new tender is held to every check a person's tender
// is.
//
// It never fails the exhaustion it follows. A round that cannot start is
// told to dispatch, and the move stays on the exception queue as exhausted.
func (s *Service) Escalate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exhausted *tender.Tender,
) {
	policy := s.effects.Policy(ctx, tenantInfo)
	mode, escalates := escalationMode(policy.ExhaustedAction)
	if !escalates {
		return
	}
	limit := min(policy.MaxEscalations, tender.MaxEscalationRound)
	if exhausted.EscalationRound >= limit {
		s.notifyEscalationStopped(ctx, tenantInfo, exhausted,
			"the escalation limit was reached")
		return
	}

	move, sh, err := s.creator.LoadTenderableMove(ctx, tenantInfo, exhausted.ShipmentMoveID)
	if err != nil {
		s.l.Info("exhausted tender not escalated",
			zap.String("tenderId", exhausted.ID.String()), zap.Error(err))
		s.notifyEscalationStopped(ctx, tenantInfo, exhausted, err.Error())
		return
	}

	offers := exhausted.EscalationPlan()
	if len(offers) == 0 {
		s.notifyEscalationStopped(ctx, tenantInfo, exhausted, "no carrier is left to ask")
		return
	}
	sell := sh.TotalChargeAmount.Decimal
	for _, offer := range offers {
		ceiling := tender.RateCeiling(
			sell,
			policy.CounterMarginFloor,
			offer.RateMethod == shipment.CarrierRateMethodPerMile,
			move.Distance,
		)
		offer.Rate = tender.EscalationRate(offer.Rate, policy.EscalationRateIncrease, ceiling)
		offer.OfferTTLSeconds = policy.EscalationOfferTTLSecs
	}

	from := exhausted.ID
	next := &tender.Tender{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		ShipmentID:      move.ShipmentID,
		ShipmentMoveID:  move.ID,
		Mode:            mode,
		Status:          tender.StatusActive,
		EscalatedFromID: &from,
		EscalationRound: exhausted.EscalationRound + 1,
		Offers:          offers,
	}
	created, err := s.creator.PersistAndStart(ctx, tenantInfo, next)
	if err != nil {
		s.l.Warn("failed to start the escalation tender",
			zap.String("tenderId", exhausted.ID.String()), zap.Error(err))
		s.notifyEscalationStopped(ctx, tenantInfo, exhausted, err.Error())
		return
	}

	s.announceEscalation(ctx, tenantInfo, exhausted, created, &policy)
}

func (s *Service) announceEscalation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exhausted, created *tender.Tender,
	policy *dispatchcontrol.TenderPolicy,
) {
	increase := policy.EscalationRateIncrease.Round(2).String()
	s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderEscalated(
		shipmenteventservice.TenantRefFor(tenantInfo),
		shipmenteventservice.TenderRef{
			ShipmentID: created.ShipmentID,
			MoveID:     created.ShipmentMoveID,
			TenderID:   created.ID,
		},
		created.Mode,
		created.EscalationRound,
		increase,
		len(created.Offers),
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.effects.NotifyDispatch(ctx, tenantInfo, &tendereffects.DispatchNotification{
		EventType: "tender_escalated",
		Priority:  tendereffects.NotificationPriorityMed,
		Title:     "Exhausted tender escalated",
		Message: fmt.Sprintf(
			"Nobody accepted the tender, so it went back to %d %s at up to %s%% more (round %d of %d)",
			len(created.Offers),
			tendercreate.PluralCarriers(len(created.Offers)),
			increase,
			created.EscalationRound,
			policy.MaxEscalations,
		),
		CorrelationID: created.ID.String(),
		Link:          tendereffects.DispatchConsoleMoveLink(created.ShipmentMoveID),
	})
	s.effects.LogAudit(&tendereffects.AuditParams{
		TenantInfo: tenantInfo,
		TenderID:   created.ID,
		Operation:  permission.OpCreate,
		Comment: fmt.Sprintf(
			"Escalation tender created from exhausted tender %s (%s, round %d)",
			exhausted.ID.String(), created.Mode.String(), created.EscalationRound,
		),
		Current: created,
	})
}

func (s *Service) notifyEscalationStopped(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exhausted *tender.Tender,
	reason string,
) {
	s.effects.NotifyDispatch(ctx, tenantInfo, &tendereffects.DispatchNotification{
		EventType: "tender_escalation_stopped",
		Priority:  tendereffects.NotificationPriorityHigh,
		Title:     "Exhausted tender was not escalated",
		Message: fmt.Sprintf(
			"The tender for move %s was not offered again because %s. The move still needs coverage",
			exhausted.ShipmentMoveID.String(),
			reason,
		),
		CorrelationID: exhausted.ID.String(),
		Link:          tendereffects.DispatchConsoleMoveLink(exhausted.ShipmentMoveID),
	})
}

type Service struct {
	l       *zap.Logger
	creator *tendercreate.Service
	effects *tendereffects.Effects
}

type Params struct {
	fx.In

	Logger  *zap.Logger
	Creator *tendercreate.Service
	Effects *tendereffects.Effects
}

func New(p Params) *Service {
	return &Service{
		l:       p.Logger.Named("service.tender"),
		creator: p.Creator,
		effects: p.Effects,
	}
}
