package detentionservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/shared/intutils"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// OccurrenceDetail bundles an occurrence with everything needed to judge it:
// the evidence trail, the notices sent, and the live defensibility assessment.
type OccurrenceDetail struct {
	Occurrence     *detention.DetentionOccurrence     `json:"occurrence"`
	Evidence       []*detention.DetentionEvidence     `json:"evidence"`
	Notices        []*detention.DetentionNotice       `json:"notices"`
	Collectability detention.CollectabilityAssessment `json:"collectability"`
	Receipt        string                             `json:"receipt"`
}

func (s *Service) GetOccurrenceDetail(
	ctx context.Context,
	req *repositories.GetDetentionOccurrenceByIDRequest,
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
		Collectability: detention.AssessCollectability(
			occurrence, occurrence.Evidence,
		),
	}

	if occurrence.CalculationTrace != nil {
		detail.Receipt = occurrence.CalculationTrace.Receipt()
	}

	return detail, nil
}

func (s *Service) ListOccurrences(
	ctx context.Context,
	req *repositories.ListDetentionOccurrencesRequest,
) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
	return s.occurrenceRepo.List(ctx, req)
}

func (s *Service) ListByShipment(
	ctx context.Context,
	req *repositories.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	return s.occurrenceRepo.GetByShipment(ctx, req)
}

// DeskEntry is one card on the live detention board.
type DeskEntry struct {
	Occurrence            *detention.DetentionOccurrence `json:"occurrence"`
	MinutesUntilFreeEnds  int32                          `json:"minutesUntilFreeEnds"`
	MinutesUntilNoticeDue *int32                         `json:"minutesUntilNoticeDue"`
	NoticeWindowOpen      bool                           `json:"noticeWindowOpen"`
	AmountAtRisk          decimal.Decimal                `json:"amountAtRisk"`
	Urgency               string                         `json:"urgency"`
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
	entries := make([]*DeskEntry, 0, len(occurrences))

	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}

		entry := &DeskEntry{
			Occurrence:            occurrence,
			MinutesUntilFreeEnds:  intutils.SafeToInt32((occurrence.FreeTimeExpiresAt - now) / 60),
			MinutesUntilNoticeDue: occurrence.MinutesUntilNoticeDue(now),
			NoticeWindowOpen:      occurrence.NoticeWindowOpen(now),
			AmountAtRisk:          occurrence.BillableAmount,
			Urgency:               deskUrgency(occurrence, now),
		}

		entries = append(entries, entry)
	}

	return entries, nil
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

// Waive forgives a detention charge with a coded reason, so discretionary
// revenue loss stays measurable instead of disappearing into free text.
func (s *Service) Waive(
	ctx context.Context,
	p WaiveParams,
) (*detention.DetentionOccurrence, error) {
	if _, err := detention.WaiverReasonFromString(string(p.Reason)); err != nil {
		return nil, errortypes.NewValidationError(
			"waiverReason", errortypes.ErrInvalid, "A coded waiver reason is required")
	}

	occurrence, err := s.occurrenceRepo.GetByID(
		ctx,
		&repositories.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: p.OccurrenceID,
			TenantInfo:   p.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	original := *occurrence
	now := s.now()

	if wErr := occurrence.Waive(p.Reason, p.Note, p.UserID, now); wErr != nil {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, wErr.Error())
	}

	saved, err := s.occurrenceRepo.Update(ctx, occurrence)
	if err != nil {
		return nil, err
	}

	s.appendEvidence(ctx, saved, detention.EvidenceKindWaiver,
		detention.EvidenceSourceManual,
		fmt.Sprintf("Waived %s %s as %s: %s",
			saved.WaivedAmount.StringFixed(2), saved.Currency, p.Reason, p.Note),
		now)

	s.audit(&original, saved, p.UserID, "Detention charge waived")

	return saved, nil
}

type ApproveParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
}

func (s *Service) Approve(
	ctx context.Context,
	p ApproveParams,
) (*detention.DetentionOccurrence, error) {
	occurrence, err := s.occurrenceRepo.GetByID(
		ctx,
		&repositories.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: p.OccurrenceID,
			TenantInfo:   p.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	original := *occurrence
	now := s.now()

	if aErr := occurrence.Approve(p.UserID, now); aErr != nil {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, aErr.Error())
	}

	saved, err := s.occurrenceRepo.Update(ctx, occurrence)
	if err != nil {
		return nil, err
	}

	s.appendEvidence(ctx, saved, detention.EvidenceKindStatusChange,
		detention.EvidenceSourceManual,
		fmt.Sprintf("Approved for billing at %s %s",
			saved.BillableAmount.StringFixed(2), saved.Currency),
		now)

	s.audit(&original, saved, p.UserID, "Detention charge approved")

	return saved, nil
}

type DisputeParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	Note         string
	UserID       pulid.ID
}

// Dispute records a customer rejection without discarding the original
// computation, which is exactly what the claim needs to be worked.
func (s *Service) Dispute(
	ctx context.Context,
	p DisputeParams,
) (*detention.DetentionOccurrence, error) {
	occurrence, err := s.occurrenceRepo.GetByID(
		ctx,
		&repositories.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: p.OccurrenceID,
			TenantInfo:   p.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	original := *occurrence
	now := s.now()

	if dErr := occurrence.Dispute(p.Note, now); dErr != nil {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, dErr.Error())
	}

	saved, err := s.occurrenceRepo.Update(ctx, occurrence)
	if err != nil {
		return nil, err
	}

	s.appendEvidence(ctx, saved, detention.EvidenceKindDispute,
		detention.EvidenceSourceManual,
		"Customer disputed the charge: "+p.Note, now)

	s.audit(&original, saved, p.UserID, "Detention charge disputed")

	return saved, nil
}

// DisputePacket is the assembled claim file: the terms that governed the
// charge, how the number was derived, what proves it, and where the evidence
// is thin. It is the artifact that turns a 45-minute argument into an
// attachment.
type DisputePacket struct {
	Occurrence     *detention.DetentionOccurrence     `json:"occurrence"`
	Snapshot       *detention.PolicySnapshot          `json:"policySnapshot"`
	Receipt        string                             `json:"receipt"`
	Evidence       []*detention.DetentionEvidence     `json:"evidence"`
	Notices        []*detention.DetentionNotice       `json:"notices"`
	Collectability detention.CollectabilityAssessment `json:"collectability"`
	ChainVerified  bool                               `json:"chainVerified"`
	GeneratedAt    int64                              `json:"generatedAt"`
}

// BuildDisputePacket assembles the claim file for one occurrence.
func (s *Service) BuildDisputePacket(
	ctx context.Context,
	occurrenceID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*DisputePacket, error) {
	detail, err := s.GetOccurrenceDetail(ctx, &repositories.GetDetentionOccurrenceByIDRequest{
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
		ChainVerified:  detention.VerifyChain(detail.Evidence) == -1,
		GeneratedAt:    s.now(),
	}, nil
}

func (s *Service) audit(
	original, updated *detention.DetentionOccurrence,
	userID pulid.ID,
	comment string,
) {
	if s.auditService == nil {
		return
	}

	if err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDetentionPolicy,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	},
		auditservice.WithComment(comment),
		auditservice.WithDiff(original, updated),
	); err != nil {
		s.l.Error("failed to log detention audit action", zap.Error(err))
	}
}
