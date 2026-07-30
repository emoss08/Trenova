package detentionservice

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
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
	// VersionID names the template version that produced this wording, so a
	// disputed charge can be traced to the exact notice text that was sent. It
	// is nil when the built-in rendered.
	VersionID *pulid.ID
	// PDF is the same notice on paper, rendered from the same context. It is nil
	// unless the governing policy attaches one.
	PDF *NoticePDF
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
	CustomerName  string
	// CustomerID lets a customer with an assigned notice template receive it.
	CustomerID *pulid.ID
	// AttachPDF asks for the notice on paper as well. The caller resolves it from
	// the governing policy rather than the service reading the policy again,
	// because the send path has already loaded it.
	AttachPDF bool
}

// BuildNotice renders a detention notice through the organization's template.
//
// The prose that used to be assembled here with a strings.Builder now lives in
// the built-in starter, word for word, with the per-kind wording expressed as
// {{ if eq .Kind ... }} branches. Moving it means an organization can reword its
// own notices — and that the HTML is produced by html/template rather than by
// wrapping plain text in <pre>, which is what previously let an angle bracket in
// a facility name become markup.
//
// A render failure falls back to the built-in rather than failing the notice: a
// detention notice that never goes out costs the charge it was evidence for.
func (s *Service) BuildNotice(
	ctx context.Context,
	p *BuildNoticeParams,
) (NoticeContent, error) {
	if s.templates == nil || s.contextBuilder == nil {
		return NoticeContent{}, errortypes.NewBusinessError(
			"Detention notice rendering is not configured on this deployment",
		)
	}

	occ := p.Occurrence
	if occ == nil {
		return NoticeContent{}, errortypes.NewValidationError(
			"occurrence", errortypes.ErrRequired, "Detention occurrence is required")
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: occ.OrganizationID,
		BuID:  occ.BusinessUnitID,
	}

	data, err := s.contextBuilder.BuildFrom(ctx, &buildNoticeContextParams{
		Occurrence:    occ,
		Kind:          p.Kind,
		FacilityName:  p.FacilityName,
		ShipmentRef:   p.ShipmentRef,
		CustomerName:  p.CustomerName,
		ArrivalSource: p.ArrivalSource,
		TenantInfo:    tenantInfo,
		Location:      p.Location,
	})
	if err != nil {
		return NoticeContent{}, err
	}

	var customerID *pulid.ID
	if p.CustomerID != nil && !p.CustomerID.IsNil() {
		customerID = p.CustomerID
	}

	rendered, err := s.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindDetentionNoticeEmail,
		CustomerID:  customerID,
		Data:        data,
		ReferenceID: occ.ID,
		// A notice is evidence for a charge. Losing it to a template mistake
		// would cost more than sending the shipped default wording.
		FallbackToBuiltIn: true,
	})
	if err != nil {
		return NoticeContent{}, err
	}

	content := NoticeContent{
		Subject:   rendered.Subject,
		Text:      rendered.Text,
		HTML:      rendered.HTML,
		VersionID: rendered.VersionID,
	}

	if p.AttachPDF {
		// The PDF is rendered from the same context object the email was, so the
		// attachment cannot quote a figure the body does not. A failure here
		// drops the attachment and keeps the notice: the email is the evidence
		// the charge rests on, and a renderer outage must not cost it.
		pdf, pdfErr := s.renderNoticePDF(ctx, tenantInfo, occ, customerID, data)
		if pdfErr != nil {
			s.l.Warn("could not render the detention notice PDF; sending without it",
				zap.String("occurrenceId", occ.ID.String()),
				zap.Error(pdfErr))
		} else {
			content.PDF = pdf
		}
	}

	return content, nil
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
	AttachPDF    bool
}

// ScheduledNotice is a queued notice plus the artifact the row cannot hold.
//
// The PDF travels in memory from the render to the send rather than being
// re-rendered there, because the notice body is a snapshot and an attachment
// rendered later could quote figures the snapshot does not.
type ScheduledNotice struct {
	Notice *detention.DetentionNotice
	PDF    *NoticePDF
}

// ScheduleNotice queues a notice without sending it. Scheduling and sending are
// separate so a warning can be lined up the moment the driver arrives, long
// before the free-time clock runs out.
func (s *Service) ScheduleNotice(
	ctx context.Context,
	p ScheduleNoticeParams,
) (*ScheduledNotice, error) {
	if p.Occurrence == nil {
		return nil, errortypes.NewValidationError(
			"occurrence", errortypes.ErrRequired, "Detention occurrence is required")
	}

	occ := p.Occurrence
	loc := s.tenantLocation(ctx, occ.OrganizationID)

	content, err := s.BuildNotice(ctx, &BuildNoticeParams{
		Occurrence:   occ,
		Kind:         p.Kind,
		FacilityName: p.FacilityName,
		ShipmentRef:  p.ShipmentRef,
		Location:     loc,
		AttachPDF:    p.AttachPDF,
	})
	if err != nil {
		return nil, err
	}

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
		// The rendered HTML is snapshotted so the send path interpolates
		// nothing: it mails exactly what was rendered and reviewed.
		BodyHTML:          content.HTML,
		TemplateVersionID: content.VersionID,
		ScheduledFor:      p.ScheduledFor,
		SentByID:          p.SentByID,
		WasAutomatic:      p.Automatic,
		QuotedFreeMinutes: occ.FreeMinutesGranted,
		QuotedAmount:      decimal.NewNullDecimal(occ.BillableAmount),
	}

	if occ.PolicySnapshot != nil {
		notice.QuotedRate = decimal.NewNullDecimal(occ.PolicySnapshot.FlatRate)
	}

	multiErr := errortypes.NewMultiError()
	notice.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.noticeRepo.Create(ctx, notice)
	if err != nil {
		return nil, err
	}

	return &ScheduledNotice{Notice: saved, PDF: content.PDF}, nil
}

// SendNoticeParams is a queued notice, the occurrence it belongs to, and the
// optional PDF rendered alongside its body.
type SendNoticeParams struct {
	Notice     *detention.DetentionNotice
	Occurrence *detention.DetentionOccurrence
	PDF        *NoticePDF
}

// SendNotice delivers a queued notice and records the outcome on both the
// notice and its occurrence. A send that lands after the contractual deadline
// is recorded as Late rather than Sent, because that distinction is what the
// billing gate keys on.
func (s *Service) SendNotice(
	ctx context.Context,
	p *SendNoticeParams,
) (*detention.DetentionNotice, error) {
	if p == nil || p.Notice == nil || p.Occurrence == nil {
		return nil, errortypes.NewValidationError(
			"notice", errortypes.ErrRequired, "Notice and occurrence are required")
	}

	notice, occurrence := p.Notice, p.Occurrence

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
		HTML:           noticeHTML(notice),
		Attachments:    noticeAttachments(p.PDF),
		OpenTracking:   true,
		IdempotencyKey: "detention-notice-" + notice.ID.String(),
	})
	if err != nil {
		log.Warn("failed to send detention notice", zap.Error(err))
		notice.MarkFailed(now, err.Error())
		return s.persistNoticeOutcome(ctx, notice, occurrence, now)
	}

	notice.MarkSent(now, occurrence.NoticeDeadlineAt, "")

	saved, err := s.persistNoticeOutcome(ctx, notice, occurrence, now)
	if err != nil {
		return nil, err
	}

	// Filing runs after the row exists, so the document has a notice to point
	// at, and after the send, so a storage problem can never delay the notice
	// that a charge depends on.
	if p.PDF != nil {
		s.fileNoticePDF(ctx, saved, occurrence, p.PDF)
	}

	return saved, nil
}

// noticeAttachments hands the rendered PDF to the email service, which stores it
// and references it by object key rather than carrying bytes through the send
// workflow.
func noticeAttachments(pdf *NoticePDF) []services.EmailAttachment {
	if pdf == nil || len(pdf.Content) == 0 {
		return nil
	}

	return []services.EmailAttachment{
		{
			FileName:    pdf.FileName,
			ContentType: "application/pdf",
			Content:     pdf.Content,
			SizeBytes:   int64(len(pdf.Content)),
		},
	}
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

	// Policy is the governing policy when the caller already has it. The sweep
	// does — it reads the policy to decide whether to send at all — so passing it
	// here saves re-reading the same row for every occurrence it sends.
	Policy *detention.DetentionPolicy
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

	scheduled, err := s.ScheduleNotice(ctx, ScheduleNoticeParams{
		Occurrence:   occurrence,
		Kind:         noticeKindFor(occurrence, now),
		ScheduledFor: now,
		Recipients:   recipients,
		Automatic:    p.Automatic,
		SentByID:     sentBy,
		FacilityName: occurrence.LocationName,
		ShipmentRef:  occurrence.ShipmentProNumber,
		AttachPDF:    s.attachesNoticePDF(ctx, occurrence, p.TenantInfo, p.Policy),
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.SendNotice(ctx, &SendNoticeParams{
		Notice:     scheduled.Notice,
		Occurrence: occurrence,
		PDF:        scheduled.PDF,
	}); err != nil {
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

// SweepNoticesDue delivers every pending notice whose window has opened, for
// policies that opted into automatic sending. Policies that leave sending to a
// human stay on the desk with their countdown; the sweep never overrides that
// choice. A configuration failure (no recipients) is stamped onto the
// occurrence so it stops re-queuing and surfaces to a person instead.
func (s *Service) SweepNoticesDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.NoticeSweepResult, error) {
	now := s.now()

	occurrences, err := s.occurrenceRepo.ListNoticesDue(ctx, &repositories.ListNoticesDueRequest{
		TenantInfo: tenantInfo,
		Before:     now,
	})
	if err != nil {
		return nil, err
	}

	result := &services.NoticeSweepResult{Due: len(occurrences)}
	policyCache := make(map[pulid.ID]*detention.DetentionPolicy, 4)

	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}

		policy := s.cachedNoticePolicy(ctx, occurrence, tenantInfo, policyCache)
		if policy == nil || !policy.AutoSendNotice {
			result.Skipped++
			continue
		}

		if _, sendErr := s.SendOccurrenceNotice(ctx, SendOccurrenceNoticeParams{
			OccurrenceID: occurrence.ID,
			TenantInfo:   tenantInfo,
			Automatic:    true,
			Policy:       policy,
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

// attachesNoticePDF reports whether the governing policy wants the notice on
// paper as well as in the body.
//
// A policy that cannot be loaded means no attachment rather than an error: the
// notice itself is what the charge rests on, and the customer receiving it
// without a duplicate copy attached is not a failure.
func (s *Service) attachesNoticePDF(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	tenantInfo pagination.TenantInfo,
	governing *detention.DetentionPolicy,
) bool {
	if governing != nil {
		return governing.AttachNoticePDF
	}

	policy := s.noticePolicy(ctx, occurrence, tenantInfo)

	return policy != nil && policy.AttachNoticePDF
}

// noticePolicy loads the policy governing an occurrence's notices, or nil when
// there is none to load.
func (s *Service) noticePolicy(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	tenantInfo pagination.TenantInfo,
) *detention.DetentionPolicy {
	if occurrence.DetentionPolicyID == nil || occurrence.DetentionPolicyID.IsNil() {
		return nil
	}

	policyID := *occurrence.DetentionPolicyID
	policy, err := s.policyRepo.GetByID(ctx, &repositories.GetDetentionPolicyByIDRequest{
		DetentionPolicyID: policyID,
		TenantInfo:        tenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to load the policy governing a detention notice",
			zap.String("policyId", policyID.String()),
			zap.Error(err))
		return nil
	}

	return policy
}

// cachedNoticePolicy is noticePolicy for the sweep, which walks many
// occurrences that share a handful of policies. A nil entry is cached too, so a
// policy that cannot be read is not read again for every occurrence under it.
func (s *Service) cachedNoticePolicy(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	tenantInfo pagination.TenantInfo,
	cache map[pulid.ID]*detention.DetentionPolicy,
) *detention.DetentionPolicy {
	if occurrence.DetentionPolicyID == nil || occurrence.DetentionPolicyID.IsNil() {
		return nil
	}

	policyID := *occurrence.DetentionPolicyID
	if policy, ok := cache[policyID]; ok {
		return policy
	}

	policy := s.noticePolicy(ctx, occurrence, tenantInfo)
	cache[policyID] = policy

	return policy
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

// noticeHTML returns the HTML snapshotted when the notice was built.
//
// It never constructs markup from Body. The previous implementation wrapped the
// plain text in <pre> at send time, which meant any angle bracket in a facility
// or customer name was interpolated into the email unescaped.
func noticeHTML(notice *detention.DetentionNotice) string {
	if strings.TrimSpace(notice.BodyHTML) != "" {
		return notice.BodyHTML
	}

	// A notice queued before this column existed has no snapshot. Escaping the
	// stored text is the safe reconstruction; it renders as plain text rather
	// than as whatever the text happens to look like.
	return "<pre style=\"font-family:inherit;white-space:pre-wrap\">" +
		html.EscapeString(notice.Body) + "</pre>"
}

// NoticeHTMLForTest exposes the send-path HTML reconstruction so the escaping
// guarantee can be asserted without standing up an email provider.
func NoticeHTMLForTest(notice *detention.DetentionNotice) string {
	return noticeHTML(notice)
}
