package detentionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// OccurrenceChange is one occurrence before and after a person's decision,
// computed by the transition the decision itself applies. Shipment is every
// occurrence on the same shipment as it stands, so a preview can say what the
// shipment's detention charge comes to on either side.
type OccurrenceChange struct {
	Before   *detention.DetentionOccurrence
	After    *detention.DetentionOccurrence
	Shipment []*detention.DetentionOccurrence
	Now      int64
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
		&repositories.GetDetentionOccurrenceByIDRequest{
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
		&repositories.GetOccurrencesByShipmentRequest{
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

// NoticePreview is the notice SendOccurrenceNotice would send now: rendered
// by the same template, to the same recipients, and the occurrence as the
// send would leave it.
type NoticePreview struct {
	Before     *detention.DetentionOccurrence
	After      *detention.DetentionOccurrence
	Kind       detention.NoticeKind
	Recipients []string
	Content    NoticeContent
	Attachment string
	Sender     *services.EmailSender
}

// PreviewOccurrenceNotice renders the notice without scheduling or sending
// it. The PDF is named rather than rendered: it is printed from the same
// context as the body, and printing it is work a preview does not need.
func (s *Service) PreviewOccurrenceNotice(
	ctx context.Context,
	p *SendOccurrenceNoticeParams,
) (*NoticePreview, error) {
	plan, err := s.planOccurrenceNotice(ctx, p)
	if err != nil {
		return nil, err
	}

	occurrence := plan.schedule.Occurrence
	before := *occurrence

	schedule := plan.schedule
	schedule.AttachPDF = false
	content, err := s.renderScheduledNotice(ctx, &schedule)
	if err != nil {
		return nil, err
	}

	sent := &detention.DetentionNotice{}
	sent.MarkSent(plan.now, occurrence.NoticeDeadlineAt, "")
	after := before
	applyNoticeOutcome(&after, sent)

	preview := &NoticePreview{
		Before:     &before,
		After:      &after,
		Kind:       schedule.Kind,
		Recipients: schedule.Recipients,
		Content:    content,
	}
	if plan.schedule.AttachPDF {
		preview.Attachment = noticePDFFileName(occurrence)
	}

	if s.senders != nil {
		sender, sErr := s.senders.ResolveSender(ctx, &services.SendEmailRequest{
			TenantInfo: p.TenantInfo,
			Purpose:    email.PurposeNotifications,
			To:         schedule.Recipients,
		})
		if sErr != nil {
			return nil, sErr
		}
		preview.Sender = sender
	}

	return preview, nil
}

type occurrenceNoticePlan struct {
	schedule ScheduleNoticeParams
	now      int64
}

// planOccurrenceNotice decides everything about a notice before it is
// written: that one is required, who receives it, which kind the moment
// calls for, and whether the policy attaches it on paper.
func (s *Service) planOccurrenceNotice(
	ctx context.Context,
	p *SendOccurrenceNoticeParams,
) (*occurrenceNoticePlan, error) {
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

	if occurrence.NotificationStatus == detention.NotificationStatusNotRequired {
		return nil, errortypes.NewValidationError(
			"occurrenceId", errortypes.ErrInvalidOperation,
			"The governing policy does not require a customer notice for this stop")
	}

	recipients, err := s.noticeRecipients(ctx, occurrence, p.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now()

	var sentBy *pulid.ID
	if !p.Automatic && !p.UserID.IsNil() {
		sentBy = &p.UserID
	}

	return &occurrenceNoticePlan{
		schedule: ScheduleNoticeParams{
			Occurrence:   occurrence,
			Kind:         noticeKindFor(occurrence, now),
			ScheduledFor: now,
			Recipients:   recipients,
			Automatic:    p.Automatic,
			SentByID:     sentBy,
			FacilityName: occurrence.LocationName,
			ShipmentRef:  occurrence.ShipmentProNumber,
			AttachPDF:    s.attachesNoticePDF(ctx, occurrence, p.TenantInfo, p.Policy),
		},
		now: now,
	}, nil
}

func (s *Service) renderScheduledNotice(
	ctx context.Context,
	p *ScheduleNoticeParams,
) (NoticeContent, error) {
	return s.BuildNotice(ctx, &BuildNoticeParams{
		Occurrence:   p.Occurrence,
		Kind:         p.Kind,
		FacilityName: p.FacilityName,
		ShipmentRef:  p.ShipmentRef,
		Location:     s.tenantLocation(ctx, p.Occurrence.OrganizationID),
		AttachPDF:    p.AttachPDF,
	})
}

// applyNoticeOutcome moves the occurrence's notification status to what a
// notice's delivery says happened.
func applyNoticeOutcome(
	occurrence *detention.DetentionOccurrence,
	notice *detention.DetentionNotice,
) {
	switch {
	case notice.DeliveryStatus.IsFailure():
		occurrence.NotificationStatus = detention.NotificationStatusFailed
	case notice.SatisfiesRequirement:
		occurrence.NotificationStatus = detention.NotificationStatusSent
		occurrence.NoticeSentAt = notice.SentAt
	default:
		occurrence.NotificationStatus = detention.NotificationStatusLate
		occurrence.NoticeSentAt = notice.SentAt
	}
}
