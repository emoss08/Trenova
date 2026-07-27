package detentionservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// NoticeContent is the rendered notice, kept separate from delivery so the same
// text can be previewed in the UI before anyone commits to sending it.
type NoticeContent struct {
	Subject string
	Text    string
	HTML    string
}

// BuildNoticeParams carries everything the notice cites. Every number in the
// message comes from the frozen snapshot rather than live config, so the notice
// and the eventual invoice quote identical terms.
type BuildNoticeParams struct {
	Occurrence    *detention.DetentionOccurrence
	Kind          detention.NoticeKind
	FacilityName  string
	ShipmentRef   string
	ArrivalSource detention.EvidenceSource
	Location      *time.Location
}

// BuildNotice renders a detention notice.
//
// The notice names the arrival timestamp and where it came from, the free time
// the contract grants, the exact moment it expires, and the rate that applies
// after. A customer who receives that twice while the truck is on their dock
// rarely disputes the invoice that follows.
func BuildNotice(p BuildNoticeParams) NoticeContent {
	occ := p.Occurrence
	loc := p.Location
	if loc == nil {
		loc = time.UTC
	}

	snap := occ.PolicySnapshot
	facility := p.FacilityName
	if facility == "" {
		facility = "the facility"
	}

	ref := p.ShipmentRef
	if ref == "" {
		ref = occ.ShipmentID.String()
	}

	subject := noticeSubject(p.Kind, ref, facility)

	var b strings.Builder

	b.WriteString(noticeOpening(p.Kind, ref, facility))
	b.WriteString("\n\n")

	if occ.ArrivedAt != nil {
		b.WriteString(fmt.Sprintf("Arrived: %s", formatStamp(*occ.ArrivedAt, loc)))
		if p.ArrivalSource != "" {
			b.WriteString(fmt.Sprintf(" (%s)", p.ArrivalSource))
		}
		b.WriteString("\n")
	}

	if snap != nil {
		b.WriteString(fmt.Sprintf("Contracted free time: %s\n",
			formatMinutes(snap.FreeMinutes)))
		b.WriteString(fmt.Sprintf("Free time expires: %s\n",
			formatStamp(occ.FreeTimeExpiresAt, loc)))
		b.WriteString(fmt.Sprintf("Detention rate after free time: %s\n",
			noticeRateLabel(snap)))
	}

	switch p.Kind {
	case detention.NoticeKindFinal:
		if occ.DepartedAt != nil {
			b.WriteString(fmt.Sprintf("Departed: %s\n", formatStamp(*occ.DepartedAt, loc)))
		}
		b.WriteString(fmt.Sprintf("Total time on site: %s\n",
			formatMinutes(occ.RawDwellMinutes)))
		b.WriteString(fmt.Sprintf("Billable detention: %s\n",
			formatMinutes(occ.RoundedMinutes)))
		b.WriteString(fmt.Sprintf("Detention charge: %s %s\n",
			occ.BillableAmount.StringFixed(2), occ.Currency))
	case detention.NoticeKindUpdate:
		b.WriteString(fmt.Sprintf("Detention accrued so far: %s (%s %s)\n",
			formatMinutes(occ.RoundedMinutes),
			occ.BillableAmount.StringFixed(2), occ.Currency))
	case detention.NoticeKindWarning, detention.NoticeKindStarted:
		if occ.RoundedMinutes > 0 {
			b.WriteString(fmt.Sprintf("Detention accrued so far: %s (%s %s)\n",
				formatMinutes(occ.RoundedMinutes),
				occ.BillableAmount.StringFixed(2), occ.Currency))
		}
	}

	b.WriteString("\n")
	b.WriteString(noticeClosing(p.Kind))

	text := b.String()

	return NoticeContent{
		Subject: subject,
		Text:    text,
		HTML:    "<pre style=\"font-family:inherit;white-space:pre-wrap\">" + text + "</pre>",
	}
}

func noticeSubject(kind detention.NoticeKind, ref, facility string) string {
	switch kind {
	case detention.NoticeKindWarning:
		return fmt.Sprintf("Free time expiring soon — shipment %s at %s", ref, facility)
	case detention.NoticeKindStarted:
		return fmt.Sprintf("Detention has started — shipment %s at %s", ref, facility)
	case detention.NoticeKindUpdate:
		return fmt.Sprintf("Detention update — shipment %s at %s", ref, facility)
	case detention.NoticeKindFinal:
		return fmt.Sprintf("Detention summary — shipment %s at %s", ref, facility)
	default:
		return fmt.Sprintf("Detention notice — shipment %s at %s", ref, facility)
	}
}

func noticeOpening(kind detention.NoticeKind, ref, facility string) string {
	switch kind {
	case detention.NoticeKindWarning:
		return fmt.Sprintf(
			"Our driver on shipment %s is approaching the end of contracted free time at %s.",
			ref, facility)
	case detention.NoticeKindStarted:
		return fmt.Sprintf(
			"Contracted free time on shipment %s at %s has been exhausted and detention is now accruing.",
			ref, facility)
	case detention.NoticeKindUpdate:
		return fmt.Sprintf(
			"Our driver on shipment %s remains at %s and detention continues to accrue.",
			ref, facility)
	case detention.NoticeKindFinal:
		return fmt.Sprintf(
			"Our driver has departed %s on shipment %s. Final detention is summarized below.",
			facility, ref)
	default:
		return fmt.Sprintf("Detention notice for shipment %s at %s.", ref, facility)
	}
}

func noticeClosing(kind detention.NoticeKind) string {
	if kind == detention.NoticeKindFinal {
		return "This charge will appear on the invoice for this shipment. " +
			"Reply to this message if any detail needs review."
	}

	return "Please expedite loading or unloading to avoid further charges. " +
		"Reply to this message if any detail needs review."
}

func noticeRateLabel(snap *detention.PolicySnapshot) string {
	if snap.RateSource == detention.RateSourceTiers && len(snap.Tiers) > 0 {
		parts := make([]string, 0, len(snap.Tiers))
		for _, tier := range snap.Tiers {
			parts = append(parts, fmt.Sprintf("%s at %s",
				tierBandTerm(tier), rateLabel(tier.Rate, tier.RateUnit)))
		}
		return strings.Join(parts, "; ")
	}

	return rateLabel(snap.FlatRate, snap.FlatRateUnit)
}

func formatStamp(ts int64, loc *time.Location) string {
	return time.Unix(ts, 0).In(loc).Format("Mon 02 Jan 2006 15:04 MST")
}

// ScheduleNoticeParams describes an outbound notice to queue.
type ScheduleNoticeParams struct {
	Occurrence   *detention.DetentionOccurrence
	Kind         detention.NoticeKind
	ScheduledFor int64
	Recipients   []string
	Automatic    bool
	SentByID     *pulid.ID
	FacilityName string
	ShipmentRef  string
}

// ScheduleNotice queues a notice without sending it. Scheduling and sending are
// separate so a warning can be lined up the moment the driver arrives, long
// before the free-time clock runs out.
func (s *Service) ScheduleNotice(
	ctx context.Context,
	p ScheduleNoticeParams,
) (*detention.DetentionNotice, error) {
	if p.Occurrence == nil {
		return nil, errortypes.NewValidationError(
			"occurrence", errortypes.ErrRequired, "Detention occurrence is required")
	}

	occ := p.Occurrence
	loc := s.tenantLocation(ctx, occ.OrganizationID)

	content := BuildNotice(BuildNoticeParams{
		Occurrence:   occ,
		Kind:         p.Kind,
		FacilityName: p.FacilityName,
		ShipmentRef:  p.ShipmentRef,
		Location:     loc,
	})

	notice := &detention.DetentionNotice{
		OrganizationID:        occ.OrganizationID,
		BusinessUnitID:        occ.BusinessUnitID,
		DetentionOccurrenceID: occ.ID,
		Kind:                  p.Kind,
		Channel:               detention.NoticeChannelEmail,
		DeliveryStatus:        detention.NoticeDeliveryStatusQueued,
		Recipients:            p.Recipients,
		Subject:               content.Subject,
		Body:                  content.Text,
		ScheduledFor:          p.ScheduledFor,
		SentByID:              p.SentByID,
		WasAutomatic:          p.Automatic,
		QuotedFreeMinutes:     occ.FreeMinutesGranted,
		QuotedAmount:          decimal.NewNullDecimal(occ.BillableAmount),
	}

	if occ.PolicySnapshot != nil {
		notice.QuotedRate = decimal.NewNullDecimal(occ.PolicySnapshot.FlatRate)
	}

	multiErr := errortypes.NewMultiError()
	notice.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.noticeRepo.Create(ctx, notice)
}

// SendNotice delivers a queued notice and records the outcome on both the
// notice and its occurrence. A send that lands after the contractual deadline
// is recorded as Late rather than Sent, because that distinction is what the
// billing gate keys on.
func (s *Service) SendNotice(
	ctx context.Context,
	notice *detention.DetentionNotice,
	occurrence *detention.DetentionOccurrence,
) (*detention.DetentionNotice, error) {
	if notice == nil || occurrence == nil {
		return nil, errortypes.NewValidationError(
			"notice", errortypes.ErrRequired, "Notice and occurrence are required")
	}

	log := s.l.With(
		zap.String("operation", "SendNotice"),
		zap.String("noticeId", notice.ID.String()),
	)

	tenantInfo := pagination.TenantInfo{
		OrgID: notice.OrganizationID,
		BuID:  notice.BusinessUnitID,
	}

	now := s.now()

	if s.emailService == nil {
		notice.MarkFailed(now, "No email service is configured")
		return s.persistNoticeOutcome(ctx, notice, occurrence, now)
	}

	_, err := s.emailService.Send(ctx, &services.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeNotifications,
		To:             notice.Recipients,
		Subject:        notice.Subject,
		Text:           notice.Body,
		HTML:           "<pre style=\"font-family:inherit;white-space:pre-wrap\">" + notice.Body + "</pre>",
		OpenTracking:   true,
		IdempotencyKey: "detention-notice-" + notice.ID.String(),
	})
	if err != nil {
		log.Warn("failed to send detention notice", zap.Error(err))
		notice.MarkFailed(now, err.Error())
		return s.persistNoticeOutcome(ctx, notice, occurrence, now)
	}

	notice.MarkSent(now, occurrence.NoticeDeadlineAt, "")

	return s.persistNoticeOutcome(ctx, notice, occurrence, now)
}

func (s *Service) persistNoticeOutcome(
	ctx context.Context,
	notice *detention.DetentionNotice,
	occurrence *detention.DetentionOccurrence,
	now int64,
) (*detention.DetentionNotice, error) {
	saved, err := s.noticeRepo.Update(ctx, notice)
	if err != nil {
		return nil, err
	}

	if err = s.applyNoticeToOccurrence(ctx, saved, occurrence, now); err != nil {
		return nil, err
	}

	return saved, nil
}

// applyNoticeToOccurrence moves the occurrence's notification status in step
// with what actually happened to the notice, and records the attempt as
// evidence either way. A bounce is as important to the claim file as a success.
func (s *Service) applyNoticeToOccurrence(
	ctx context.Context,
	notice *detention.DetentionNotice,
	occurrence *detention.DetentionOccurrence,
	now int64,
) error {
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

	if _, err := s.occurrenceRepo.Update(ctx, occurrence); err != nil {
		return err
	}

	summary := fmt.Sprintf("%s notice %s to %s",
		notice.Kind, noticeOutcomeVerb(notice), strings.Join(notice.Recipients, ", "))

	s.appendEvidence(ctx, occurrence, detention.EvidenceKindNotice,
		detention.EvidenceSourceSystem, summary, now)

	return nil
}

func noticeOutcomeVerb(notice *detention.DetentionNotice) string {
	switch {
	case notice.DeliveryStatus.IsFailure():
		return "failed to send"
	case notice.SatisfiesRequirement:
		return "sent within the contractual window"
	default:
		return "sent after the contractual deadline"
	}
}

type SendOccurrenceNoticeParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
	Automatic    bool
}

// SendOccurrenceNotice schedules and delivers a customer notice for one
// occurrence in a single step: the desk's "Send notice" action and the
// automated sweep both land here. Recipients come from the customer's email
// profile, because the notice must reach whoever receives the invoice.
func (s *Service) SendOccurrenceNotice(
	ctx context.Context,
	p SendOccurrenceNoticeParams,
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
	original := *occurrence

	var sentBy *pulid.ID
	if !p.Automatic && !p.UserID.IsNil() {
		sentBy = &p.UserID
	}

	notice, err := s.ScheduleNotice(ctx, ScheduleNoticeParams{
		Occurrence:   occurrence,
		Kind:         noticeKindFor(occurrence, now),
		ScheduledFor: now,
		Recipients:   recipients,
		Automatic:    p.Automatic,
		SentByID:     sentBy,
		FacilityName: occurrence.LocationName,
		ShipmentRef:  occurrence.ShipmentProNumber,
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.SendNotice(ctx, notice, occurrence); err != nil {
		return nil, err
	}

	s.audit(&original, occurrence, p.UserID, "Detention notice sent")

	return occurrence, nil
}

// noticeKindFor picks the message the moment calls for: a warning while free
// time is still running, a start-of-charges notice once it lapses, an update on
// a repeat send, and a final summary after the clock has stopped.
func noticeKindFor(occurrence *detention.DetentionOccurrence, now int64) detention.NoticeKind {
	switch {
	case occurrence.ClockStopAt != nil:
		return detention.NoticeKindFinal
	case now < occurrence.FreeTimeExpiresAt:
		return detention.NoticeKindWarning
	case occurrence.NoticeSentAt == nil:
		return detention.NoticeKindStarted
	default:
		return detention.NoticeKindUpdate
	}
}

func (s *Service) noticeRecipients(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	tenantInfo pagination.TenantInfo,
) ([]string, error) {
	if s.customerRepo == nil {
		return nil, errortypes.NewValidationError(
			"occurrenceId", errortypes.ErrSystemError,
			"Customer lookup is not available")
	}

	entity, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         occurrence.CustomerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeEmailProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	var recipients []string
	if entity.EmailProfile != nil {
		recipients = stringutils.SplitEmailList(entity.EmailProfile.ToRecipients)
	}

	if len(recipients) == 0 {
		return nil, errortypes.NewValidationError(
			"occurrenceId", errortypes.ErrInvalidOperation,
			"The customer has no notice recipients configured — add a customer email profile first")
	}

	return recipients, nil
}

type NoticeSweepResult struct {
	Due     int `json:"due"`
	Sent    int `json:"sent"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// SweepNoticesDue delivers every pending notice whose window has opened, for
// policies that opted into automatic sending. Policies that leave sending to a
// human stay on the desk with their countdown; the sweep never overrides that
// choice. A configuration failure (no recipients) is stamped onto the
// occurrence so it stops re-queuing and surfaces to a person instead.
func (s *Service) SweepNoticesDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*NoticeSweepResult, error) {
	now := s.now()

	occurrences, err := s.occurrenceRepo.ListNoticesDue(ctx, &repositories.ListNoticesDueRequest{
		TenantInfo: tenantInfo,
		Before:     now,
	})
	if err != nil {
		return nil, err
	}

	result := &NoticeSweepResult{Due: len(occurrences)}
	autoSendByPolicy := make(map[pulid.ID]bool, 4)

	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}

		if !s.policyAutoSends(ctx, occurrence, tenantInfo, autoSendByPolicy) {
			result.Skipped++
			continue
		}

		if _, sendErr := s.SendOccurrenceNotice(ctx, SendOccurrenceNoticeParams{
			OccurrenceID: occurrence.ID,
			TenantInfo:   tenantInfo,
			Automatic:    true,
		}); sendErr != nil {
			result.Failed++
			s.l.Warn("automatic detention notice failed",
				zap.String("occurrenceId", occurrence.ID.String()),
				zap.Error(sendErr))

			var validationErr *errortypes.Error
			if errors.As(sendErr, &validationErr) {
				s.markNoticeUnsendable(ctx, occurrence, validationErr.Message, now)
			}
			continue
		}

		result.Sent++
	}

	return result, nil
}

func (s *Service) policyAutoSends(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	tenantInfo pagination.TenantInfo,
	cache map[pulid.ID]bool,
) bool {
	if occurrence.DetentionPolicyID == nil || occurrence.DetentionPolicyID.IsNil() {
		return false
	}

	policyID := *occurrence.DetentionPolicyID
	if autoSend, ok := cache[policyID]; ok {
		return autoSend
	}

	policy, err := s.policyRepo.GetByID(ctx, &repositories.GetDetentionPolicyByIDRequest{
		DetentionPolicyID: policyID,
		TenantInfo:        tenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to load policy for notice sweep",
			zap.String("policyId", policyID.String()),
			zap.Error(err))
		cache[policyID] = false
		return false
	}

	cache[policyID] = policy.AutoSendNotice
	return policy.AutoSendNotice
}

func (s *Service) markNoticeUnsendable(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	reason string,
	now int64,
) {
	occurrence.NotificationStatus = detention.NotificationStatusFailed
	if _, err := s.occurrenceRepo.Update(ctx, occurrence); err != nil {
		s.l.Error("failed to mark detention notice unsendable",
			zap.String("occurrenceId", occurrence.ID.String()),
			zap.Error(err))
		return
	}

	s.appendEvidence(ctx, occurrence, detention.EvidenceKindNotice,
		detention.EvidenceSourceSystem,
		"Automatic notice could not be sent: "+reason, now)
}

// RecordNoticeDelivery folds a provider webhook back into the claim file.
func (s *Service) RecordNoticeDelivery(
	ctx context.Context,
	notice *detention.DetentionNotice,
	status detention.NoticeDeliveryStatus,
	reason string,
) (*detention.DetentionNotice, error) {
	now := s.now()

	switch status {
	case detention.NoticeDeliveryStatusDelivered:
		notice.MarkDelivered(now)
	case detention.NoticeDeliveryStatusOpened:
		notice.MarkOpened(now)
	case detention.NoticeDeliveryStatusBounced:
		notice.MarkBounced(now, reason)
	case detention.NoticeDeliveryStatusFailed:
		notice.MarkFailed(now, reason)
	case detention.NoticeDeliveryStatusQueued, detention.NoticeDeliveryStatusSent:
		return notice, nil
	}

	return s.noticeRepo.Update(ctx, notice)
}

func (s *Service) appendEvidence(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	kind detention.EvidenceKind,
	source detention.EvidenceSource,
	summary string,
	observedAt int64,
) {
	if s.evidenceRepo == nil {
		return
	}

	entry := &detention.DetentionEvidence{
		OrganizationID:        occurrence.OrganizationID,
		BusinessUnitID:        occurrence.BusinessUnitID,
		DetentionOccurrenceID: occurrence.ID,
		Kind:                  kind,
		Source:                source,
		Summary:               summary,
		ObservedAt:            observedAt,
		RecordedAt:            s.now(),
	}

	if _, err := s.evidenceRepo.Append(ctx, &repositories.AppendEvidenceRequest{
		Entry: entry,
	}); err != nil {
		s.l.Warn("failed to append detention evidence",
			zap.String("occurrenceId", occurrence.ID.String()), zap.Error(err))
	}
}
