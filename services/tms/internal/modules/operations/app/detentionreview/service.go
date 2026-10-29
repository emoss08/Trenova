package detentionreview

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncollectability"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Service struct {
	occurrenceRepo detentionports.DetentionOccurrenceRepository
	now            func() int64
	effects        *detentioneffects.Effects
}

type Params struct {
	fx.In

	OccurrenceRepo detentionports.DetentionOccurrenceRepository
	Now            func() int64 `optional:"true"`
	Effects        *detentioneffects.Effects
}

func New(p Params) *Service {
	now := p.Now
	if now == nil {
		now = timeutils.NowUnix
	}
	return &Service{
		occurrenceRepo: p.OccurrenceRepo,
		now:            now,
		effects:        p.Effects,
	}
}

// PreviewApprove is Approve without the save: the occurrence as approving it
// would leave it, and the shipment's other occurrences.
func (s *Service) PreviewApprove(
	ctx context.Context,
	p *ApproveParams,
) (*OccurrenceChange, error) {
	change, err := s.planApprove(ctx, p)
	if err != nil {
		return nil, err
	}

	return s.withShipmentOccurrences(ctx, change, p.TenantInfo)
}

// PreviewWaive is Waive without the save.
func (s *Service) PreviewWaive(
	ctx context.Context,
	p *WaiveParams,
) (*OccurrenceChange, error) {
	change, err := s.planWaive(ctx, p)
	if err != nil {
		return nil, err
	}

	return s.withShipmentOccurrences(ctx, change, p.TenantInfo)
}

// PreviewDispute is Dispute without the save.
func (s *Service) PreviewDispute(
	ctx context.Context,
	p *DisputeParams,
) (*OccurrenceChange, error) {
	change, err := s.planDispute(ctx, p)
	if err != nil {
		return nil, err
	}

	return s.withShipmentOccurrences(ctx, change, p.TenantInfo)
}

func (s *Service) planDispute(ctx context.Context, p *DisputeParams) (*OccurrenceChange, error) {
	return s.planOccurrence(ctx, p.OccurrenceID, p.TenantInfo,
		func(occurrence *detention.DetentionOccurrence, now int64) error {
			return occurrence.Dispute(p.Note, now)
		})
}

func (s *Service) planApprove(ctx context.Context, p *ApproveParams) (*OccurrenceChange, error) {
	return s.planOccurrence(ctx, p.OccurrenceID, p.TenantInfo,
		func(occurrence *detention.DetentionOccurrence, now int64) error {
			return occurrence.Approve(p.UserID, now)
		})
}

func (s *Service) planWaive(ctx context.Context, p *WaiveParams) (*OccurrenceChange, error) {
	if _, err := detention.WaiverReasonFromString(string(p.Reason)); err != nil {
		return nil, errortypes.NewValidationError(
			"waiverReason", errortypes.ErrInvalid, "A coded waiver reason is required")
	}

	return s.planOccurrence(ctx, p.OccurrenceID, p.TenantInfo,
		func(occurrence *detention.DetentionOccurrence, now int64) error {
			return occurrence.Waive(p.Reason, p.Note, p.UserID, now)
		})
}

// planOccurrence loads an occurrence and applies a transition to a copy of
// it, keeping the record as it was for the audit trail and the preview.
func (s *Service) planOccurrence(
	ctx context.Context,
	occurrenceID pulid.ID,
	tenantInfo pagination.TenantInfo,
	transition func(*detention.DetentionOccurrence, int64) error,
) (*OccurrenceChange, error) {
	occurrence, err := s.occurrenceRepo.GetByID(
		ctx,
		&detentionports.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: occurrenceID,
			TenantInfo:   tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	before := *occurrence
	now := s.now()
	if tErr := transition(occurrence, now); tErr != nil {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, tErr.Error())
	}

	return &OccurrenceChange{Before: &before, After: occurrence, Now: now}, nil
}

func (s *Service) withShipmentOccurrences(
	ctx context.Context,
	change *OccurrenceChange,
	tenantInfo pagination.TenantInfo,
) (*OccurrenceChange, error) {
	occurrences, err := s.occurrenceRepo.GetByShipment(
		ctx,
		&detentionports.GetOccurrencesByShipmentRequest{
			ShipmentID: change.Before.ShipmentID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	change.Shipment = occurrences

	return change, nil
}

func (s *Service) GetOccurrenceDetail(
	ctx context.Context,
	req *detentionports.GetDetentionOccurrenceByIDRequest,
) (*OccurrenceDetail, error) {
	req.IncludeEvidence = true
	req.IncludeNotices = true

	occurrence, err := s.occurrenceRepo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	detail := &OccurrenceDetail{
		Occurrence: occurrence,
		Evidence:   occurrence.Evidence,
		Notices:    occurrence.Notices,
		Collectability: detentioncollectability.AssessCollectability(
			occurrence, occurrence.Evidence,
		),
	}

	if occurrence.CalculationTrace != nil {
		detail.Receipt = occurrence.CalculationTrace.Receipt()
	}

	current, err := s.PriceOpenClock(ctx, req.TenantInfo, occurrence)
	switch {
	case errors.Is(err, ErrClockNotPriced):
	case err != nil:
		return nil, err
	default:
		detail.CurrentlyBillable = current
	}

	return detail, nil
}

func (s *Service) ListOccurrences(
	ctx context.Context,
	req *detentionports.ListDetentionOccurrencesRequest,
) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
	return s.occurrenceRepo.List(ctx, req)
}

func (s *Service) ListByShipment(
	ctx context.Context,
	req *detentionports.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	return s.occurrenceRepo.GetByShipment(ctx, req)
}

// ListDesk returns every still-accruing occurrence ordered by how soon someone
// has to act. A notice deadline about to lapse outranks a bigger charge that
// nobody can lose yet.
func (s *Service) ListDesk(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*DeskEntry, error) {
	occurrences, err := s.occurrenceRepo.ListOpen(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now()
	priced, err := s.priceOpenClocks(ctx, tenantInfo, occurrences, now)
	if err != nil {
		return nil, err
	}
	entries := make([]*DeskEntry, 0, len(occurrences))

	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}

		entry := newDeskEntry(occurrence, now, deskUrgency(occurrence, now))
		if current, ok := priced[occurrence.ID]; ok {
			entry.AmountAtRisk = current.BillableAmount
			entry.BillableMinutes = current.BillableMinutes
			entry.PricedAt = now
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *Service) PriceOpenClock(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	occurrence *detention.DetentionOccurrence,
) (*ClockPrice, error) {
	if occurrence == nil {
		return nil, ErrClockNotPriced
	}

	now := s.now()
	priced, err := s.priceOpenClocks(
		ctx, tenantInfo, []*detention.DetentionOccurrence{occurrence}, now,
	)
	if err != nil {
		return nil, err
	}
	current, ok := priced[occurrence.ID]
	if !ok {
		return nil, ErrClockNotPriced
	}

	return &ClockPrice{
		BillableMinutes: current.BillableMinutes,
		BillableAmount:  current.BillableAmount,
		AsOf:            now,
	}, nil
}

func (s *Service) priceOpenClocks(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	open []*detention.DetentionOccurrence,
	now int64,
) (map[pulid.ID]detentionengine.ComputeResult, error) {
	shipments := make([]pulid.ID, 0, len(open))
	for _, occurrence := range open {
		if occurrence == nil || !occurrence.IsOpen || occurrence.IsFrozen() ||
			occurrence.PolicySnapshot == nil {
			continue
		}
		if !slices.Contains(shipments, occurrence.ShipmentID) {
			shipments = append(shipments, occurrence.ShipmentID)
		}
	}
	if len(shipments) == 0 {
		return map[pulid.ID]detentionengine.ComputeResult{}, nil
	}

	location, err := s.effects.TenantLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	priced := make(map[pulid.ID]detentionengine.ComputeResult, len(open))
	for _, shipmentID := range shipments {
		occurrences, listErr := s.occurrenceRepo.GetByShipment(ctx,
			&detentionports.GetOccurrencesByShipmentRequest{
				ShipmentID: shipmentID,
				TenantInfo: tenantInfo,
			})
		if listErr != nil {
			return nil, listErr
		}
		priceShipmentClocks(occurrences, now, location, priced)
	}

	return priced, nil
}

// ListAwaitingApproval returns every charge holding its shipment off an
// invoice, oldest clock first. These are not on the live board, whose clocks
// are still running; they are what the board's clocks became once they
// stopped over the approval threshold, missed a notice, or were escalated.
func (s *Service) ListAwaitingApproval(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*DeskEntry, error) {
	occurrences, err := s.occurrenceRepo.ListBillingHolds(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now()
	entries := make([]*DeskEntry, 0, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence == nil || !occurrence.HoldsBilling() {
			continue
		}

		entries = append(entries, newDeskEntry(occurrence, now, UrgencyAwaitingApproval))
	}

	return entries, nil
}

// ListDeskEntries is the board narrowed by filter. Naming the
// AwaitingApproval urgency reads the held charges whether or not they were
// asked for beside the running clocks.
func (s *Service) ListDeskEntries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	filter *DeskFilter,
) ([]*DeskEntry, error) {
	entries := make([]*DeskEntry, 0)
	if filter.Urgency != UrgencyAwaitingApproval {
		open, err := s.ListDesk(ctx, tenantInfo)
		if err != nil {
			return nil, err
		}
		entries = append(entries, open...)
	}
	if filter.IncludeAwaitingApproval || filter.Urgency == UrgencyAwaitingApproval {
		held, err := s.ListAwaitingApproval(ctx, tenantInfo)
		if err != nil {
			return nil, err
		}
		entries = append(entries, held...)
	}

	proNumber := strings.ToLower(strings.TrimSpace(filter.ProNumber))

	return slices.DeleteFunc(entries, func(entry *DeskEntry) bool {
		switch {
		case entry == nil || entry.Occurrence == nil:
			return true
		case filter.Urgency != "" && entry.Urgency != filter.Urgency:
			return true
		case filter.ShipmentID.IsNotNil() && entry.Occurrence.ShipmentID != filter.ShipmentID:
			return true
		case proNumber != "" &&
			!strings.Contains(strings.ToLower(entry.Occurrence.ShipmentProNumber), proNumber):
			return true
		}
		return false
	}), nil
}

// Waive forgives a detention charge with a coded reason, so discretionary
// revenue loss stays measurable instead of disappearing into free text.
func (s *Service) Waive(
	ctx context.Context,
	p *WaiveParams,
) (*detention.DetentionOccurrence, error) {
	change, err := s.planWaive(ctx, p)
	if err != nil {
		return nil, err
	}

	return s.saveWaiver(ctx, p, change, false)
}

func (s *Service) WaiveUninvoiced(
	ctx context.Context,
	p *WaiveParams,
) (*detention.DetentionOccurrence, error) {
	if _, err := detention.WaiverReasonFromString(string(p.Reason)); err != nil {
		return nil, errortypes.NewValidationError(
			"waiverReason", errortypes.ErrInvalid, "A coded waiver reason is required")
	}

	released := false
	change, err := s.planOccurrence(ctx, p.OccurrenceID, p.TenantInfo,
		func(occurrence *detention.DetentionOccurrence, now int64) error {
			if occurrence.Status == detention.OccurrenceStatusBilled {
				if releaseErr := occurrence.ReleaseBilling(); releaseErr != nil {
					return releaseErr
				}
				released = true
			}

			return occurrence.Waive(p.Reason, p.Note, p.UserID, now)
		})
	if err != nil {
		return nil, err
	}

	return s.saveWaiver(ctx, p, change, released)
}

func (s *Service) saveWaiver(
	ctx context.Context,
	p *WaiveParams,
	change *OccurrenceChange,
	released bool,
) (*detention.DetentionOccurrence, error) {
	saved, err := s.occurrenceRepo.Update(ctx, change.After)
	if err != nil {
		return nil, err
	}

	if released {
		s.effects.AppendEvidence(ctx, saved, detentionevidence.EvidenceKindStatusChange,
			detentionevidence.EvidenceSourceSystem,
			"Billing released: no standing invoice bills this charge",
			change.Now)
	}
	s.effects.AppendEvidence(ctx, saved, detentionevidence.EvidenceKindWaiver,
		detentionevidence.EvidenceSourceManual,
		fmt.Sprintf("Waived %s %s as %s: %s",
			saved.WaivedAmount.StringFixed(2), saved.Currency, p.Reason, p.Note),
		change.Now)

	s.effects.Audit(change.Before, saved, p.UserID, "Detention charge waived")
	s.effects.PublishBillingHoldChange(ctx, change.Before, saved, p.UserID)

	return saved, nil
}

func (s *Service) Approve(
	ctx context.Context,
	p *ApproveParams,
) (*detention.DetentionOccurrence, error) {
	change, err := s.planApprove(ctx, p)
	if err != nil {
		return nil, err
	}

	saved, err := s.occurrenceRepo.Update(ctx, change.After)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Approved for billing at %s %s",
		saved.BillableAmount.StringFixed(2), saved.Currency)
	if note := strings.TrimSpace(p.Note); note != "" {
		summary += ": " + note
	}
	s.effects.AppendEvidence(ctx, saved, detentionevidence.EvidenceKindStatusChange,
		detentionevidence.EvidenceSourceManual, summary, change.Now)

	s.effects.Audit(change.Before, saved, p.UserID, "Detention charge approved")
	s.effects.PublishBillingHoldChange(ctx, change.Before, saved, p.UserID)

	return saved, nil
}

// Dispute records a customer rejection without discarding the original
// computation, which is exactly what the claim needs to be worked.
func (s *Service) Dispute(
	ctx context.Context,
	p *DisputeParams,
) (*detention.DetentionOccurrence, error) {
	change, err := s.planDispute(ctx, p)
	if err != nil {
		return nil, err
	}
	original, occurrence, now := change.Before, change.After, change.Now

	saved, err := s.occurrenceRepo.Update(ctx, occurrence)
	if err != nil {
		return nil, err
	}

	s.effects.AppendEvidence(ctx, saved, detentionevidence.EvidenceKindDispute,
		detentionevidence.EvidenceSourceManual,
		"Customer disputed the charge: "+p.Note, now)

	s.effects.Audit(original, saved, p.UserID, "Detention charge disputed")
	s.effects.PublishBillingHoldChange(ctx, original, saved, p.UserID)

	return saved, nil
}

// Escalate hands a detention clock to a person without touching the money.
// It is what the desk reaches for when the notice window has closed, a gate
// is holding the notice back, or the customer has nobody on file to send it
// to: the charge stands, and somebody has to decide what happens to it.
func (s *Service) Escalate(
	ctx context.Context,
	p *EscalateParams,
) (*detention.DetentionOccurrence, error) {
	occurrence, err := s.occurrenceRepo.GetByID(
		ctx,
		&detentionports.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: p.OccurrenceID,
			TenantInfo:   p.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	original := *occurrence
	now := s.now()

	if eErr := occurrence.Escalate(now); eErr != nil {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, eErr.Error())
	}

	saved, err := s.occurrenceRepo.Update(ctx, occurrence)
	if err != nil {
		return nil, err
	}

	s.effects.AppendEvidence(ctx, saved, detentionevidence.EvidenceKindStatusChange,
		detentionevidence.EvidenceSourceManual,
		EscalationSummary(p.Reason), now)

	s.effects.Audit(&original, saved, p.UserID, "Detention escalated: "+p.Reason)
	s.effects.PublishBillingHoldChange(ctx, &original, saved, p.UserID)

	return saved, nil
}

// BuildDisputePacket assembles the claim file for one occurrence.
func (s *Service) BuildDisputePacket(
	ctx context.Context,
	occurrenceID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*DisputePacket, error) {
	detail, err := s.GetOccurrenceDetail(ctx, &detentionports.GetDetentionOccurrenceByIDRequest{
		OccurrenceID: occurrenceID,
		TenantInfo:   tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return &DisputePacket{
		Occurrence:     detail.Occurrence,
		Snapshot:       detail.Occurrence.PolicySnapshot,
		Receipt:        detail.Receipt,
		Evidence:       detail.Evidence,
		Notices:        detail.Notices,
		Collectability: detail.Collectability,
		ChainVerified:  detentionevidence.VerifyChain(detail.Evidence) == -1,
		GeneratedAt:    s.now(),
	}, nil
}
