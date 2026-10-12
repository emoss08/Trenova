package triageshipmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	amountPlaces         = 2
	arrivalSourceUnknown = "Not recorded"
)

var arrivalSources = map[detention.EvidenceSource]string{
	detention.EvidenceSourceTelematics: "ELD",
	detention.EvidenceSourceGeofence:   "Geofence",
	detention.EvidenceSourceDriverApp:  "Driver app",
	detention.EvidenceSourceEDI:        "EDI",
	detention.EvidenceSourceDocument:   "Document",
	detention.EvidenceSourceManual:     "A person",
	detention.EvidenceSourceSystem:     "Trenova",
}

func (s *Service) disputedOccurrence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	withEvidence bool,
) (*detention.DetentionOccurrence, error) {
	occurrences, err := s.occurrences.GetByShipment(
		ctx,
		&detentionports.GetOccurrencesByShipmentRequest{
			ShipmentID:      shipmentID,
			TenantInfo:      tenantInfo,
			IncludeEvidence: withEvidence,
		},
	)
	if err != nil {
		return nil, err
	}

	var chosen *detention.DetentionOccurrence
	for _, occurrence := range occurrences {
		if chosen == nil {
			chosen = occurrence

			continue
		}
		amount, chosenAmount := disputedAmount(occurrence), disputedAmount(chosen)
		if amount.GreaterThan(chosenAmount) ||
			(amount.Equal(chosenAmount) && occurrence.ClockStartAt > chosen.ClockStartAt) {
			chosen = occurrence
		}
	}

	return chosen, nil
}

func disputedAmount(occurrence *detention.DetentionOccurrence) decimal.Decimal {
	return occurrence.BillableAmount.Add(occurrence.WaivedAmount)
}

func (s *Service) StopTimes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*inboundmessageports.StopTimes, error) {
	occurrence, err := s.disputedOccurrence(ctx, tenantInfo, shipmentID, true)
	if err != nil {
		return nil, err
	}
	if occurrence == nil {
		return nil, nil //nolint:nilnil // the shipment has no detention to show
	}

	return stopTimesOf(occurrence), nil
}

func stopTimesOf(occurrence *detention.DetentionOccurrence) *inboundmessageports.StopTimes {
	out := &inboundmessageports.StopTimes{
		ShipmentID:             occurrence.ShipmentID,
		StopName:               strings.TrimSpace(occurrence.LocationName),
		AppointmentStart:       occurrence.AppointmentStart,
		AppointmentEnd:         occurrence.AppointmentEnd,
		ArrivedAt:              occurrence.ArrivedAt,
		DepartedAt:             occurrence.DepartedAt,
		LateMinutes:            max(int(occurrence.LateByMinutes), 0),
		BilledDetentionMinutes: max(int(occurrence.RoundedMinutes), 0),
		DetentionPaysWhenLate:  paysWhenLate(occurrence.PolicySnapshot),
		ArrivalSource:          arrivalSourceUnknown,
	}
	if occurrence.BillableAmount.IsPositive() {
		amount := occurrence.BillableAmount.StringFixed(amountPlaces)
		out.BilledDetentionAmount = &amount
	}

	for _, evidence := range occurrence.Evidence {
		if evidence.Kind != detention.EvidenceKindArrival {
			continue
		}
		if label, ok := arrivalSources[evidence.Source]; ok {
			out.ArrivalSource = label
		} else {
			out.ArrivalSource = string(evidence.Source)
		}

		break
	}

	return out
}

func paysWhenLate(snapshot *detention.PolicySnapshot) bool {
	if snapshot == nil {
		return true
	}

	return snapshot.LateArrivalRule != detention.LateArrivalRuleForfeit
}

func (s *Service) SettleDetention(
	ctx context.Context,
	req *inboundmessageports.SettleTriageDetentionRequest,
) (*inboundmessageports.SettledTriageDetention, error) {
	occurrence, err := s.disputedOccurrence(ctx, req.TenantInfo, req.ShipmentID, false)
	if err != nil {
		return nil, err
	}
	if occurrence == nil {
		return &inboundmessageports.SettledTriageDetention{}, nil
	}

	switch req.Decision {
	case inboundmessage.StopsDecisionWaive:
		return s.waive(ctx, req, occurrence)
	case inboundmessage.StopsDecisionStandBy:
		settled, disputeErr := s.detention.Dispute(ctx, detentionservice.DisputeParams{
			OccurrenceID: occurrence.ID,
			TenantInfo:   req.TenantInfo,
			Note:         req.Note,
			UserID:       req.UserID,
		})
		if disputeErr != nil {
			return nil, disputeErr
		}

		return settledOf(settled.BillableAmount, settled.Currency), nil
	default:
		return &inboundmessageports.SettledTriageDetention{}, nil
	}
}

func (s *Service) waive(
	ctx context.Context,
	req *inboundmessageports.SettleTriageDetentionRequest,
	occurrence *detention.DetentionOccurrence,
) (*inboundmessageports.SettledTriageDetention, error) {
	params := detentionservice.WaiveParams{
		OccurrenceID: occurrence.ID,
		TenantInfo:   req.TenantInfo,
		Reason:       detention.WaiverReasonOther,
		Note:         req.Note,
		UserID:       req.UserID,
	}
	if occurrence.Status == detention.OccurrenceStatusWaived {
		return settledOf(occurrence.WaivedAmount, occurrence.Currency), nil
	}
	if s.credits == nil {
		return waivedOf(s.detention.Waive(ctx, params))
	}
	if occurrence.AdditionalChargeID == nil {
		return waivedOf(s.detention.WaiveUninvoiced(ctx, params))
	}

	credit, err := s.credits.CreditBilledCharge(ctx, &detentionports.CreditBilledDetentionRequest{
		TenantInfo: req.TenantInfo,
		Actor:      req.Actor,
		ShipmentID: occurrence.ShipmentID,
		ChargeID:   *occurrence.AdditionalChargeID,
		Reason:     "Detention waived: " + req.Note,
		Waiver: detentionports.DetentionWaiver{
			OccurrenceID: occurrence.ID,
			Reason:       params.Reason,
			Note:         req.Note,
			RequestedBy:  req.UserID,
		},
	})
	if err != nil {
		return nil, err
	}
	if credit == nil {
		return waivedOf(s.detention.WaiveUninvoiced(ctx, params))
	}

	settled := settledOf(credit.Amount, credit.Currency)
	settled.InvoiceNumbers = credit.InvoiceNumbers()
	settled.CreditMemoNumbers = credit.CreditMemoNumbers()
	settled.CreditPending = credit.Pending()

	return settled, nil
}

func waivedOf(
	waived *detention.DetentionOccurrence,
	err error,
) (*inboundmessageports.SettledTriageDetention, error) {
	if err != nil {
		return nil, err
	}

	return settledOf(waived.WaivedAmount, waived.Currency), nil
}

func settledOf(
	amount decimal.Decimal,
	currency string,
) *inboundmessageports.SettledTriageDetention {
	return &inboundmessageports.SettledTriageDetention{
		Amount:   amount.StringFixed(amountPlaces),
		Currency: currency,
		Settled:  true,
	}
}
