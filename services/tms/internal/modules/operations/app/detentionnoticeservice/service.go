package detentionnoticeservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/detentionjobs/detentioncontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func (s *Service) DisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
) (*DisputeRecord, error) {
	occ := p.Occurrence
	if occ == nil {
		return nil, errortypes.NewValidationError(
			"occurrence", errortypes.ErrRequired, "Detention occurrence is required")
	}

	sent, err := s.sentNoticeDocument(ctx, p)
	if err != nil {
		return nil, err
	}
	if sent.IsNotNil() {
		return &DisputeRecord{DocumentID: sent}, nil
	}

	fileName := disputeRecordFileName(occ)
	existing, err := s.storedDisputeRecord(ctx, p.TenantInfo, occ, fileName)
	if err != nil {
		return nil, err
	}
	if existing.IsNotNil() {
		return &DisputeRecord{DocumentID: existing, FileName: fileName}, nil
	}

	if s.documentStore == nil {
		return nil, nil //nolint:nilnil // nothing can be stored on this deployment
	}

	return s.storeDisputeRecord(ctx, p, fileName)
}

func (s *Service) sentNoticeDocument(
	ctx context.Context,
	p *DisputeRecordParams,
) (pulid.ID, error) {
	notices, err := s.noticeRepo.List(ctx, &detentionports.ListDetentionNoticesRequest{
		OccurrenceID: p.Occurrence.ID,
		TenantInfo:   p.TenantInfo,
	})
	if err != nil {
		return pulid.Nil, err
	}

	var chosen *detentionnotice.DetentionNotice
	for _, notice := range notices {
		if notice.PDFDocumentID == nil || notice.PDFDocumentID.IsNil() ||
			notice.DeliveryStatus.IsFailure() {
			continue
		}
		if chosen == nil || betterRecord(notice, chosen) {
			chosen = notice
		}
	}
	if chosen == nil {
		return pulid.Nil, nil
	}

	return *chosen.PDFDocumentID, nil
}

func (s *Service) storedDisputeRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	occ *detention.DetentionOccurrence,
	fileName string,
) (pulid.ID, error) {
	if s.documents == nil {
		return pulid.Nil, nil
	}

	docs, err := s.documents.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceID:   occ.ShipmentID.String(),
		ResourceType: disputeRecordResourceType,
	})
	if err != nil {
		return pulid.Nil, err
	}

	for _, doc := range docs {
		if doc.OriginalName == fileName && doc.Mailable() {
			return doc.ID, nil
		}
	}

	return pulid.Nil, nil
}

func (s *Service) storeDisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
	fileName string,
) (*DisputeRecord, error) {
	occ := p.Occurrence
	pdf, err := s.renderDisputeRecord(ctx, p)
	if err != nil {
		return nil, err
	}

	docType, err := s.documentTypeRepo.GetByCode(
		ctx,
		documenttypeports.GetDocumentTypeByCodeRequest{
			Code:       documenttype.CodeDetentionNotice,
			TenantInfo: p.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	stored, err := s.documentStore.Store(ctx, &documentports.StoreDocumentRequest{
		TenantInfo: p.TenantInfo,
		Actor: permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeSystem,
			PrincipalID:    permissionports.SystemPrincipalID,
			OrganizationID: p.TenantInfo.OrgID,
			BusinessUnitID: p.TenantInfo.BuID,
		},
		Content:        pdf.Content,
		FileName:       fileName,
		ContentType:    disputeRecordContentType,
		ResourceType:   disputeRecordResourceType,
		ResourceID:     occ.ShipmentID.String(),
		DocumentTypeID: docType.ID,
		Description:    disputeRecordDescription,
		Tags:           disputeRecordTags,
	})
	if err != nil {
		return nil, err
	}

	s.recordDisputeRecord(ctx, p, stored, pdf)

	return &DisputeRecord{DocumentID: stored.ID, FileName: fileName, Rendered: true}, nil
}

func (s *Service) renderDisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
) (*detentionnoticerender.NoticePDF, error) {
	if s.templates == nil || s.contextBuilder == nil {
		return nil, errortypes.NewBusinessError(
			"Detention records cannot be rendered on this deployment",
		)
	}

	occ := p.Occurrence
	location, err := s.effects.TenantLocation(ctx, p.TenantInfo)
	if err != nil {
		return nil, err
	}

	data, err := s.contextBuilder.BuildFrom(ctx, &detentionnoticerender.NoticeContextParams{
		Occurrence:    occ,
		Kind:          detentionnotice.NoticeKindFinal,
		FacilityName:  occ.LocationName,
		ShipmentRef:   occ.ShipmentProNumber,
		ArrivalSource: arrivalSourceOf(occ),
		TenantInfo:    p.TenantInfo,
		Location:      location,
	})
	if err != nil {
		return nil, err
	}

	return s.renderNoticePDF(ctx, p.TenantInfo, occ, nil, data)
}

func (s *Service) recordDisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
	stored *document.Document,
	pdf *detentionnoticerender.NoticePDF,
) {
	if pdf.Provenance == nil {
		return
	}

	documentID := stored.ID
	if _, err := s.templates.RecordGeneratedDocument(
		ctx,
		&documenttemplateports.RecordGeneratedDocumentRequest{
			TenantInfo:    p.TenantInfo,
			Kind:          documenttemplate.KindDetentionNoticePDF,
			Rendered:      pdf.Provenance,
			ReferenceType: noticePDFReferenceType,
			ReferenceID:   p.Occurrence.ID,
			DocumentID:    &documentID,
			FileName:      stored.OriginalName,
			FilePath:      stored.StoragePath,
			FileSize:      stored.FileSize,
			Checksum:      stored.ChecksumSHA256,
		},
	); err != nil {
		s.l.Warn("could not record the generated detention record",
			zap.String("occurrenceId", p.Occurrence.ID.String()),
			zap.String("documentId", stored.ID.String()),
			zap.Error(err))
	}
}

type Service struct {
	l                *zap.Logger
	policyRepo       detentionports.DetentionPolicyRepository
	occurrenceRepo   detentionports.DetentionOccurrenceRepository
	noticeRepo       detentionports.DetentionNoticeRepository
	customerRepo     customerports.CustomerRepository
	documentTypeRepo documenttypeports.DocumentTypeRepository
	emailService     emailports.EmailService
	uploadService    documentuploadports.DocumentUploadService
	workflowStarter  kernelports.WorkflowStarter
	templates        documenttemplateports.DocumentTemplateResolver
	contextBuilder   *detentionnoticerender.ContextBuilder
	publisher        agentports.AgentEventPublisher
	senders          emailports.EmailSenderResolver
	documentStore    documentports.DocumentStore
	documents        documentports.DocumentRepository
	now              func() int64
	effects          *detentioneffects.Effects
}

type Deps struct {
	Logger           *zap.Logger
	PolicyRepo       detentionports.DetentionPolicyRepository
	OccurrenceRepo   detentionports.DetentionOccurrenceRepository
	NoticeRepo       detentionports.DetentionNoticeRepository
	CustomerRepo     customerports.CustomerRepository
	DocumentTypeRepo documenttypeports.DocumentTypeRepository
	EmailService     emailports.EmailService
	UploadService    documentuploadports.DocumentUploadService
	WorkflowStarter  kernelports.WorkflowStarter
	Templates        documenttemplateports.DocumentTemplateResolver
	ContextBuilder   *detentionnoticerender.ContextBuilder
	Publisher        agentports.AgentEventPublisher
	Senders          emailports.EmailSenderResolver
	DocumentStore    documentports.DocumentStore
	Documents        documentports.DocumentRepository
	Now              func() int64
	Effects          *detentioneffects.Effects
}

func New(d Deps) *Service {
	return &Service{
		l:                d.Logger,
		policyRepo:       d.PolicyRepo,
		occurrenceRepo:   d.OccurrenceRepo,
		noticeRepo:       d.NoticeRepo,
		customerRepo:     d.CustomerRepo,
		documentTypeRepo: d.DocumentTypeRepo,
		emailService:     d.EmailService,
		uploadService:    d.UploadService,
		workflowStarter:  d.WorkflowStarter,
		templates:        d.Templates,
		contextBuilder:   d.ContextBuilder,
		publisher:        d.Publisher,
		senders:          d.Senders,
		documentStore:    d.DocumentStore,
		documents:        d.Documents,
		now:              d.Now,
		effects:          d.Effects,
	}
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
	p *detentionnoticerender.BuildNoticeParams,
) (detentionnoticerender.NoticeContent, error) {
	if s.templates == nil || s.contextBuilder == nil {
		return detentionnoticerender.NoticeContent{}, errortypes.NewBusinessError(
			"Detention notice rendering is not configured on this deployment",
		)
	}

	occ := p.Occurrence
	if occ == nil {
		return detentionnoticerender.NoticeContent{}, errortypes.NewValidationError(
			"occurrence", errortypes.ErrRequired, "Detention occurrence is required")
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: occ.OrganizationID,
		BuID:  occ.BusinessUnitID,
	}

	data, err := s.contextBuilder.BuildFrom(ctx, &detentionnoticerender.NoticeContextParams{
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
		return detentionnoticerender.NoticeContent{}, err
	}

	var customerID *pulid.ID
	if p.CustomerID != nil && !p.CustomerID.IsNil() {
		customerID = p.CustomerID
	}

	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
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
		return detentionnoticerender.NoticeContent{}, err
	}

	content := detentionnoticerender.NoticeContent{
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
	content, err := s.renderScheduledNotice(ctx, &p)
	if err != nil {
		return nil, err
	}

	notice := &detentionnotice.DetentionNotice{
		OrganizationID:        occ.OrganizationID,
		BusinessUnitID:        occ.BusinessUnitID,
		DetentionOccurrenceID: occ.ID,
		Kind:                  p.Kind,
		Channel:               detentionnotice.NoticeChannelEmail,
		DeliveryStatus:        detentionnotice.NoticeDeliveryStatusQueued,
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

// SendNotice delivers a queued notice and records the outcome on both the
// notice and its occurrence. A send that lands after the contractual deadline
// is recorded as Late rather than Sent, because that distinction is what the
// billing gate keys on.
func (s *Service) SendNotice(
	ctx context.Context,
	p *SendNoticeParams,
) (*detentionnotice.DetentionNotice, error) {
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

	_, err := s.emailService.Send(ctx, &emailports.SendEmailRequest{
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

func (s *Service) persistNoticeOutcome(
	ctx context.Context,
	notice *detentionnotice.DetentionNotice,
	occurrence *detention.DetentionOccurrence,
	now int64,
) (*detentionnotice.DetentionNotice, error) {
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
	notice *detentionnotice.DetentionNotice,
	occurrence *detention.DetentionOccurrence,
	now int64,
) error {
	applyNoticeOutcome(occurrence, notice)

	if _, err := s.occurrenceRepo.Update(ctx, occurrence); err != nil {
		return err
	}

	summary := fmt.Sprintf("%s notice %s to %s",
		notice.Kind, noticeOutcomeVerb(notice), strings.Join(notice.Recipients, ", "))

	s.effects.AppendEvidence(ctx, occurrence, detentionevidence.EvidenceKindNotice,
		detentionevidence.EvidenceSourceSystem, summary, now)

	return nil
}

// SendOccurrenceNotice schedules and delivers a customer notice for one
// occurrence in a single step: the desk's "Send notice" action and the
// automated sweep both land here. Recipients come from the customer's email
// profile, because the notice must reach whoever receives the invoice.
func (s *Service) SendOccurrenceNotice(
	ctx context.Context,
	p SendOccurrenceNoticeParams,
) (*detention.DetentionOccurrence, error) {
	plan, err := s.planOccurrenceNotice(ctx, &p)
	if err != nil {
		return nil, err
	}

	occurrence := plan.schedule.Occurrence
	original := *occurrence

	scheduled, err := s.ScheduleNotice(ctx, plan.schedule)
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

	s.effects.Audit(&original, occurrence, p.UserID, "Detention notice sent")

	return occurrence, nil
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

	entity, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         occurrence.CustomerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeEmailProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	var recipients []string
	if entity.EmailProfile != nil {
		recipients = mailtext.SplitEmailList(entity.EmailProfile.ToRecipients)
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
) (*detentionports.NoticeSweepResult, error) {
	now := s.now()

	occurrences, err := s.occurrenceRepo.ListNoticesDue(ctx, &detentionports.ListNoticesDueRequest{
		TenantInfo: tenantInfo,
		Before:     now,
	})
	if err != nil {
		return nil, err
	}

	result := &detentionports.NoticeSweepResult{Due: len(occurrences)}
	policyCache := make(map[pulid.ID]*detentionpolicy.DetentionPolicy, 4)

	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}

		policy := s.cachedNoticePolicy(ctx, occurrence, tenantInfo, policyCache)
		if policy == nil || !policy.AutoSendNotice {
			result.Skipped++
			// A policy that leaves sending to a person is a decision, not a
			// gap: the notice stays on the desk and whichever agent covers
			// detention is woken to draft it. A policy that could not be
			// read is a configuration problem and wakes nobody.
			if policy != nil {
				agentports.PublishAgentEvent(ctx, s.publisher, agentports.AgentEvent{
					Kind:       agenttrigger.EventDetentionNoticeDue,
					SubjectID:  occurrence.ID,
					TenantInfo: tenantInfo,
				})
			}
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
	governing *detentionpolicy.DetentionPolicy,
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
) *detentionpolicy.DetentionPolicy {
	if occurrence.DetentionPolicyID == nil || occurrence.DetentionPolicyID.IsNil() {
		return nil
	}

	policyID := *occurrence.DetentionPolicyID
	policy, err := s.policyRepo.GetByID(ctx, &detentionports.GetDetentionPolicyByIDRequest{
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
	cache map[pulid.ID]*detentionpolicy.DetentionPolicy,
) *detentionpolicy.DetentionPolicy {
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

	s.effects.AppendEvidence(ctx, occurrence, detentionevidence.EvidenceKindNotice,
		detentionevidence.EvidenceSourceSystem,
		"Automatic notice could not be sent: "+reason, now)
}

// RecordNoticeDelivery folds a provider webhook back into the claim file.
func (s *Service) RecordNoticeDelivery(
	ctx context.Context,
	notice *detentionnotice.DetentionNotice,
	status detentionnotice.NoticeDeliveryStatus,
	reason string,
) (*detentionnotice.DetentionNotice, error) {
	now := s.now()

	switch status {
	case detentionnotice.NoticeDeliveryStatusDelivered:
		notice.MarkDelivered(now)
	case detentionnotice.NoticeDeliveryStatusOpened:
		notice.MarkOpened(now)
	case detentionnotice.NoticeDeliveryStatusBounced:
		notice.MarkBounced(now, reason)
	case detentionnotice.NoticeDeliveryStatusFailed:
		notice.MarkFailed(now, reason)
	case detentionnotice.NoticeDeliveryStatusQueued, detentionnotice.NoticeDeliveryStatusSent:
		return notice, nil
	}

	return s.noticeRepo.Update(ctx, notice)
}

// renderNoticePDF prints the notice against the context its email was rendered
// from.
//
// Passing the already-built context rather than a reference id is what
// guarantees the attachment and the email quote the same figures. Rendering the
// PDF from a fresh context would re-read the occurrence, which may have moved on
// between the two renders — and a notice whose attachment disagrees with its body
// is worse than no attachment at all.
func (s *Service) renderNoticePDF(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	occurrence *detention.DetentionOccurrence,
	customerID *pulid.ID,
	data any,
) (*detentionnoticerender.NoticePDF, error) {
	rendered, err := s.templates.RenderDocument(ctx, &documenttemplateports.RenderDocumentRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindDetentionNoticePDF,
		CustomerID:  customerID,
		Data:        data,
		ReferenceID: occurrence.ID,
		Title:       noticePDFTitle(occurrence),
	})
	if err != nil {
		return nil, err
	}

	if len(rendered.PDF) == 0 {
		return nil, errortypes.NewBusinessError(
			"The detention notice PDF rendered no content",
		)
	}

	provenance := *rendered
	provenance.PDF = nil
	provenance.HTML = ""

	return &detentionnoticerender.NoticePDF{
		Content:    rendered.PDF,
		FileName:   noticePDFFileName(occurrence),
		Provenance: &provenance,
	}, nil
}

// fileNoticePDF stores the sent notice PDF as a shipment document.
//
// It runs after the email has gone out, and every failure here is logged rather
// than returned: the customer has the notice, the occurrence records that it was
// sent, and the charge stands on that. Losing the filing copy is a bookkeeping
// gap, not a lost notice.
func (s *Service) fileNoticePDF(
	ctx context.Context,
	notice *detentionnotice.DetentionNotice,
	occurrence *detention.DetentionOccurrence,
	pdf *detentionnoticerender.NoticePDF,
) {
	log := s.l.With(
		zap.String("operation", "fileNoticePDF"),
		zap.String("noticeId", notice.ID.String()),
	)

	if s.uploadService == nil {
		log.Warn("document uploads are not configured; the notice PDF was not filed")
		return
	}

	filing := &noticePDFFiling{
		TenantInfo: pagination.TenantInfo{
			OrgID:  notice.OrganizationID,
			BuID:   notice.BusinessUnitID,
			UserID: noticeActorUserID(notice),
		},
		Notice:     notice,
		Occurrence: occurrence,
		PDF:        pdf,
	}

	sessionID, err := s.uploadNoticePDF(ctx, filing)
	if err != nil {
		log.Warn("failed to stage the detention notice PDF for storage", zap.Error(err))
		return
	}

	if err = s.startNoticePDFStorage(ctx, filing, sessionID); err != nil {
		log.Warn("failed to start detention notice PDF storage", zap.Error(err))
	}
}

// uploadNoticePDF pushes the bytes into an upload session, leaving finalization
// — the document row, the thumbnail, the search projection — to the same
// workflow every other stored document goes through.
func (s *Service) uploadNoticePDF(
	ctx context.Context,
	f *noticePDFFiling,
) (pulid.ID, error) {
	docType, err := s.documentTypeRepo.GetByCode(
		ctx,
		documenttypeports.GetDocumentTypeByCodeRequest{
			Code:       documenttype.CodeDetentionNotice,
			TenantInfo: f.TenantInfo,
		},
	)
	if err != nil {
		return pulid.Nil, err
	}

	session, err := s.uploadService.CreateSession(ctx, &documentuploadports.CreateSessionRequest{
		TenantInfo:     f.TenantInfo,
		Actor:          f.actor(),
		ResourceID:     f.Occurrence.ShipmentID.String(),
		ResourceType:   "shipment",
		FileName:       f.PDF.FileName,
		FileSize:       f.size(),
		ContentType:    "application/pdf",
		Description:    "Detention notice sent to the customer",
		Tags:           []string{"detention", "notice", "generated"},
		DocumentTypeID: docType.ID.String(),
	})
	if err != nil {
		return pulid.Nil, err
	}

	if _, err = s.uploadService.UploadPart(ctx, &documentuploadports.UploadPartRequest{
		TenantInfo: f.TenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(f.PDF.Content),
		Size:       f.size(),
	}); err != nil {
		return pulid.Nil, err
	}

	return session.ID, nil
}

// startNoticePDFStorage hands the staged upload to the workflow that finalizes
// it, or finalizes it inline where there is no worker to hand it to.
//
// Both paths end in AttachNoticePDFDocument, so the notice and its audit row are
// written by exactly one piece of code regardless of how the document was
// finalized.
func (s *Service) startNoticePDFStorage(
	ctx context.Context,
	f *noticePDFFiling,
	sessionID pulid.ID,
) error {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		session, err := s.uploadService.Complete(ctx, &documentuploadports.CompletionRequest{
			TenantInfo: f.TenantInfo,
			Actor:      f.actor(),
			SessionID:  sessionID,
		})
		if err != nil {
			return err
		}
		if session.DocumentID == nil || session.DocumentID.IsNil() {
			return errortypes.NewBusinessError(
				"The detention notice PDF upload did not produce a document",
			)
		}

		return s.AttachNoticePDFDocument(ctx, &detentionports.AttachNoticePDFDocumentParams{
			TenantInfo:   f.TenantInfo,
			NoticeID:     f.Notice.ID,
			OccurrenceID: f.Occurrence.ID,
			DocumentID:   *session.DocumentID,
			FileName:     f.PDF.FileName,
			FileSize:     f.size(),
			Provenance:   f.PDF.Provenance,
			UserID:       f.TenantInfo.UserID,
		})
	}

	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			// One workflow per notice: a retried send reuses the id and Temporal
			// rejects the duplicate rather than filing the notice twice.
			ID:        "detention-notice-pdf-" + f.Notice.ID.String(),
			TaskQueue: temporaltype.TaskQueueBilling.String(),
			StaticSummary: fmt.Sprintf(
				"Filing detention notice PDF for notice %s", f.Notice.ID.String(),
			),
		},
		detentioncontract.StoreNoticePDFWorkflowName,
		&detentioncontract.StoreNoticePDFPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: f.Notice.OrganizationID,
				BusinessUnitID: f.Notice.BusinessUnitID,
				UserID:         f.TenantInfo.UserID,
				Timestamp:      s.now(),
			},
			NoticeID:      f.Notice.ID,
			OccurrenceID:  f.Occurrence.ID,
			SessionID:     sessionID,
			FileName:      f.PDF.FileName,
			FileSize:      f.size(),
			Provenance:    f.PDF.Provenance,
			PrincipalType: permissionports.PrincipalTypeSystem,
			PrincipalID:   permissionports.SystemPrincipalID,
		},
	)

	return err
}

// AttachNoticePDFDocument links a finalized document to its notice and logs it
// against the template version that produced it.
//
// This is the step that makes a disputed detention charge answerable: the notice
// row points at the exact PDF that was attached, and the generated-documents row
// names the template version behind it even after the template has been edited.
func (s *Service) AttachNoticePDFDocument(
	ctx context.Context,
	p *detentionports.AttachNoticePDFDocumentParams,
) error {
	if p == nil || p.NoticeID.IsNil() || p.DocumentID.IsNil() {
		return errortypes.NewValidationError(
			"documentId", errortypes.ErrRequired,
			"A notice id and a document id are required to file a notice PDF")
	}

	if err := s.noticeRepo.AttachPDFDocument(
		ctx,
		&detentionports.AttachNoticePDFDocumentRequest{
			TenantInfo: p.TenantInfo,
			NoticeID:   p.NoticeID,
			DocumentID: p.DocumentID,
		},
	); err != nil {
		return err
	}

	if s.templates == nil || p.Provenance == nil {
		return nil
	}

	documentID := p.DocumentID
	if _, err := s.templates.RecordGeneratedDocument(
		ctx,
		&documenttemplateports.RecordGeneratedDocumentRequest{
			TenantInfo:    p.TenantInfo,
			Kind:          documenttemplate.KindDetentionNoticePDF,
			Rendered:      p.Provenance,
			ReferenceType: noticePDFReferenceType,
			ReferenceID:   p.OccurrenceID,
			DocumentID:    &documentID,
			FileName:      p.FileName,
			FileSize:      p.FileSize,
			UserID:        p.UserID,
		},
	); err != nil {
		// The document is filed and the notice points at it. A missing audit row
		// is worth a warning, not an error that would retry the whole filing and
		// duplicate the document.
		s.l.Warn("could not record the generated detention notice PDF",
			zap.String("noticeId", p.NoticeID.String()),
			zap.Error(err))
	}

	return nil
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

	sent := &detentionnotice.DetentionNotice{}
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
		sender, sErr := s.senders.ResolveSender(ctx, &emailports.SendEmailRequest{
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

// planOccurrenceNotice decides everything about a notice before it is
// written: that one is required, who receives it, which kind the moment
// calls for, and whether the policy attaches it on paper.
func (s *Service) planOccurrenceNotice(
	ctx context.Context,
	p *SendOccurrenceNoticeParams,
) (*occurrenceNoticePlan, error) {
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
) (detentionnoticerender.NoticeContent, error) {
	location, err := s.effects.TenantLocation(ctx, pagination.TenantInfo{
		OrgID: p.Occurrence.OrganizationID,
		BuID:  p.Occurrence.BusinessUnitID,
	})
	if err != nil {
		return detentionnoticerender.NoticeContent{}, err
	}

	return s.BuildNotice(ctx, &detentionnoticerender.BuildNoticeParams{
		Occurrence:   p.Occurrence,
		Kind:         p.Kind,
		FacilityName: p.FacilityName,
		ShipmentRef:  p.ShipmentRef,
		Location:     location,
		AttachPDF:    p.AttachPDF,
	})
}
