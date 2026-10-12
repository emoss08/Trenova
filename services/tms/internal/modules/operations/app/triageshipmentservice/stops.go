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
		if chosen == nil || occurrence.BillableAmount.GreaterThan(chosen.BillableAmount) ||
			(occurrence.BillableAmount.Equal(chosen.BillableAmount) &&
				occurrence.ClockStartAt > chosen.ClockStartAt) {
			chosen = occurrence
		}
	}

	return chosen, nil
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

	var settled *detention.DetentionOccurrence
	switch req.Decision {
	case inboundmessage.StopsDecisionWaive:
		settled, err = s.detention.Waive(ctx, detentionservice.WaiveParams{
			OccurrenceID: occurrence.ID,
			TenantInfo:   req.TenantInfo,
			Reason:       detention.WaiverReasonOther,
			Note:         req.Note,
			UserID:       req.UserID,
		})
	case inboundmessage.StopsDecisionStandBy:
		settled, err = s.detention.Dispute(ctx, detentionservice.DisputeParams{
			OccurrenceID: occurrence.ID,
			TenantInfo:   req.TenantInfo,
			Note:         req.Note,
			UserID:       req.UserID,
		})
	default:
		return &inboundmessageports.SettledTriageDetention{}, nil
	}
	if err != nil {
		return nil, err
	}

	amount := settled.BillableAmount
	if req.Decision == inboundmessage.StopsDecisionWaive {
		amount = settled.WaivedAmount
	}

	return &inboundmessageports.SettledTriageDetention{
		Amount:   amount.StringFixed(amountPlaces),
		Currency: settled.Currency,
		Settled:  true,
	}, nil
}
