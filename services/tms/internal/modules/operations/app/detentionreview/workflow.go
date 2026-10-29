package detentionreview

import (
	"cmp"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncollectability"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// OccurrenceDetail bundles an occurrence with everything needed to judge it:
// the evidence trail, the notices sent, and the live defensibility assessment.
type OccurrenceDetail struct {
	Occurrence        *detention.DetentionOccurrence                   `json:"occurrence"`
	Evidence          []*detentionevidence.DetentionEvidence           `json:"evidence"`
	Notices           []*detentionnotice.DetentionNotice               `json:"notices"`
	Collectability    detentioncollectability.CollectabilityAssessment `json:"collectability"`
	Receipt           string                                           `json:"receipt"`
	CurrentlyBillable *ClockPrice                                      `json:"currentlyBillable,omitempty"`
}

// DeskEntry is one card on the live detention board.
type DeskEntry struct {
	Occurrence            *detention.DetentionOccurrence `json:"occurrence"`
	MinutesUntilFreeEnds  int32                          `json:"minutesUntilFreeEnds"`
	MinutesUntilNoticeDue *int32                         `json:"minutesUntilNoticeDue"`
	NoticeWindowOpen      bool                           `json:"noticeWindowOpen"`
	AmountAtRisk          decimal.Decimal                `json:"amountAtRisk"`
	BillableMinutes       int32                          `json:"billableMinutes"`
	PricedAt              int64                          `json:"pricedAt"`
	Urgency               string                         `json:"urgency"`
}

type ClockPrice struct {
	BillableMinutes int32
	BillableAmount  decimal.Decimal
	AsOf            int64
}

func priceShipmentClocks(
	occurrences []*detention.DetentionOccurrence,
	now int64,
	location *time.Location,
	priced map[pulid.ID]detentionengine.ComputeResult,
) {
	ordered := slices.DeleteFunc(
		slices.Clone(occurrences),
		func(o *detention.DetentionOccurrence) bool {
			return o == nil
		},
	)
	slices.SortStableFunc(ordered, func(a, b *detention.DetentionOccurrence) int {
		return cmp.Compare(a.ClockStartAt, b.ClockStartAt)
	})

	dayAccrued := map[string]decimal.Decimal{}
	shipmentAccrued := decimal.Zero
	for _, occurrence := range ordered {
		if occurrence.IsFrozen() || occurrence.PolicySnapshot == nil {
			shipmentAccrued = shipmentAccrued.Add(occurrence.BillableAmount)
			continue
		}

		computed := detentionengine.Compute(detentionengine.ComputeInput{
			Snapshot:         *occurrence.PolicySnapshot,
			StopType:         occurrence.StopType,
			ScheduleType:     occurrence.ScheduleType,
			ArrivedAt:        occurrence.ArrivedAt,
			DepartedAt:       occurrence.DepartedAt,
			AppointmentStart: occurrence.AppointmentStart,
			AppointmentEnd:   occurrence.AppointmentEnd,
			NoticeSentAt:     occurrence.NoticeSentAt,
			Now:              now,
			Location:         location,
			DayAccrued:       dayAccrued,
			ShipmentAccrued:  shipmentAccrued,
		})
		if !computed.Applicable {
			continue
		}
		for day, amount := range computed.DayAllocation {
			dayAccrued[day] = dayAccrued[day].Add(amount)
		}
		shipmentAccrued = shipmentAccrued.Add(computed.BillableAmount)
		if occurrence.IsOpen {
			priced[occurrence.ID] = computed
		}
	}
}

// UrgencyAwaitingApproval marks a charge whose clock has stopped and that is
// holding its shipment off an invoice until someone approves or waives it.
const UrgencyAwaitingApproval = "AwaitingApproval"

// DeskFilter narrows the detention board: the charges waiting on approval
// beside the running clocks, one urgency, one shipment, or a PRO number.
type DeskFilter struct {
	IncludeAwaitingApproval bool
	Urgency                 string
	ShipmentID              pulid.ID
	ProNumber               string
}

func newDeskEntry(
	occurrence *detention.DetentionOccurrence,
	now int64,
	urgency string,
) *DeskEntry {
	return &DeskEntry{
		Occurrence:            occurrence,
		MinutesUntilFreeEnds:  intutils.SafeToInt32((occurrence.FreeTimeExpiresAt - now) / 60),
		MinutesUntilNoticeDue: occurrence.MinutesUntilNoticeDue(now),
		NoticeWindowOpen:      occurrence.NoticeWindowOpen(now),
		AmountAtRisk:          occurrence.BillableAmount,
		BillableMinutes:       occurrence.BillableMinutes,
		PricedAt:              occurrence.UpdatedAt,
		Urgency:               urgency,
	}
}

// deskUrgency ranks what a dispatcher should look at first.
func deskUrgency(occurrence *detention.DetentionOccurrence, now int64) string {
	if occurrence.NotificationStatus == detention.NotificationStatusMissed ||
		occurrence.SuppressedByGate {
		return "Lost"
	}

	if occurrence.NotificationStatus == detention.NotificationStatusPending {
		if remaining := occurrence.MinutesUntilNoticeDue(now); remaining != nil {
			if *remaining <= 0 {
				return "NoticeOverdue"
			}
			if *remaining <= 30 {
				return "NoticeDueSoon"
			}
		}
	}

	if now >= occurrence.FreeTimeExpiresAt {
		return "Accruing"
	}

	if occurrence.FreeTimeExpiresAt-now <= 1800 {
		return "FreeTimeEnding"
	}

	return "Normal"
}

// WaiveParams describes a discretionary write-off.
type WaiveParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	Reason       detention.WaiverReason
	Note         string
	UserID       pulid.ID
}

type ApproveParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
	// Note is why the charge stands, recorded on its evidence chain with the
	// approval.
	Note string
}

type DisputeParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	Note         string
	UserID       pulid.ID
}

// EscalateParams names the clock being handed over and why.
type EscalateParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	Reason       string
	UserID       pulid.ID
}

// DisputePacket is the assembled claim file: the terms that governed the
// charge, how the number was derived, what proves it, and where the evidence
// is thin. It is the artifact that turns a 45-minute argument into an
// attachment.
type DisputePacket struct {
	Occurrence     *detention.DetentionOccurrence                   `json:"occurrence"`
	Snapshot       *detentionpolicy.PolicySnapshot                  `json:"policySnapshot"`
	Receipt        string                                           `json:"receipt"`
	Evidence       []*detentionevidence.DetentionEvidence           `json:"evidence"`
	Notices        []*detentionnotice.DetentionNotice               `json:"notices"`
	Collectability detentioncollectability.CollectabilityAssessment `json:"collectability"`
	ChainVerified  bool                                             `json:"chainVerified"`
	GeneratedAt    int64                                            `json:"generatedAt"`
}
