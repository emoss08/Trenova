package emailservice

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/emailjobs/emailcontract"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	Repo               emailports.EmailRepository
	InvoiceRepo        invoiceports.InvoiceRepository
	Validator          *Validator
	IntegrationService *integrationservice.Service
	EncryptionService  *encryptionservice.Service
	AuditService       auditports.AuditService
	WorkflowStarter    kernelports.WorkflowStarter
	Storage            kernelports.Client
	Plans              platformplanports.PlanService `optional:"true"`
	Realtime           realtimeports.RealtimeService `optional:"true"`
}

// ObserverParams are the observers told when a message changes. They are
// registered once the graph is built rather than taken by New: an observer may
// itself depend, through its own dependencies, on sending email, and taking it
// at construction made the email service depend on everything each observer
// depends on.
type ObserverParams struct {
	fx.In

	Email     *Service
	Observers []emailports.EmailMessageObserver `group:"email_message_observers"`
}

type Service struct {
	l                  *zap.Logger
	repo               emailports.EmailRepository
	invoiceRepo        invoiceports.InvoiceRepository
	validator          *Validator
	integrationService *integrationservice.Service
	encryptionService  *encryptionservice.Service
	auditService       auditports.AuditService
	providerSenders    map[email.Provider]ProviderSender
	workflowStarter    kernelports.WorkflowStarter
	storage            kernelports.Client
	plans              platformplanports.PlanService
	realtime           realtimeports.RealtimeService
	observers          atomic.Pointer[[]emailports.EmailMessageObserver]
	observersMu        sync.Mutex
}

type HandleProviderEventParams struct {
	TenantInfo        pagination.TenantInfo
	Event             *email.Event
	ProviderMessageID string
	SuppressionReason email.SuppressionReason
}

func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.email"),
		repo:               p.Repo,
		invoiceRepo:        p.InvoiceRepo,
		validator:          p.Validator,
		integrationService: p.IntegrationService,
		encryptionService:  p.EncryptionService,
		auditService:       p.AuditService,
		providerSenders: map[email.Provider]ProviderSender{
			email.ProviderResend:   NewResendSender(),
			email.ProviderPostmark: NewPostmarkSender(),
		},
		workflowStarter: p.WorkflowStarter,
		storage:         p.Storage,
		plans:           p.Plans,
		realtime:        p.Realtime,
	}
}

// RegisterObservers hands the email service the observers the graph provides.
func RegisterObservers(p ObserverParams) {
	p.Email.Observe(p.Observers...)
}

// Observe adds observers told of every message change from now on.
func (s *Service) Observe(observers ...emailports.EmailMessageObserver) {
	if len(observers) == 0 {
		return
	}

	s.observersMu.Lock()
	defer s.observersMu.Unlock()

	var current []emailports.EmailMessageObserver
	if loaded := s.observers.Load(); loaded != nil {
		current = *loaded
	}
	next := make([]emailports.EmailMessageObserver, 0, len(current)+len(observers))
	next = append(next, current...)
	next = append(next, observers...)
	s.observers.Store(&next)
}

func (s *Service) ListProfiles(
	ctx context.Context,
	req *emailports.ListEmailProfilesRequest,
) (*pagination.ListResult[*email.Profile], error) {
	return s.repo.ListProfiles(ctx, req)
}

func (s *Service) ListProfilesConnection(
	ctx context.Context,
	req *emailports.ListEmailProfileConnectionRequest,
) (*pagination.CursorListResult[*email.Profile], error) {
	return s.repo.ListProfilesConnection(ctx, req)
}

func (s *Service) SelectProfileOptions(
	ctx context.Context,
	req *emailports.EmailProfileSelectOptionsRequest,
) (*pagination.ListResult[*email.Profile], error) {
	return s.repo.SelectProfileOptions(ctx, req)
}

func (s *Service) GetProfile(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
) (*email.Profile, error) {
	return s.repo.GetProfile(ctx, req)
}

func (s *Service) CreateProfile(
	ctx context.Context,
	profile *email.Profile,
	userID pulid.ID,
) (*email.Profile, error) {
	if multiErr := s.validator.ValidateProfile(ctx, profile); multiErr != nil {
		return nil, multiErr
	}
	created, err := s.repo.CreateProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	s.logAudit(emailProfileAuditParams{
		Current: created,
		UserID:  userID,
		Op:      permission.OpCreate,
		Comment: "Email profile created",
	})
	return created, nil
}

func (s *Service) UpdateProfile(
	ctx context.Context,
	profile *email.Profile,
	userID pulid.ID,
) (*email.Profile, error) {
	if multiErr := s.validator.ValidateProfile(ctx, profile); multiErr != nil {
		return nil, multiErr
	}
	original, err := s.repo.GetProfile(ctx, emailports.GetEmailEntityRequest{
		ID: profile.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: profile.OrganizationID,
			BuID:  profile.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	profile.Version = original.Version
	updated, err := s.repo.UpdateProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	s.logAudit(emailProfileAuditParams{
		Current:  updated,
		Previous: original,
		UserID:   userID,
		Op:       permission.OpUpdate,
		Comment:  "Email profile updated",
	})
	return updated, nil
}

func (s *Service) DeleteProfile(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
	userID pulid.ID,
) error {
	original, err := s.repo.GetProfile(ctx, req)
	if err != nil {
		return err
	}
	if err = s.repo.DeleteProfile(ctx, req); err != nil {
		return err
	}
	s.logAudit(emailProfileAuditParams{
		Current:  original,
		Previous: original,
		UserID:   userID,
		Op:       permission.OpDelete,
		Comment:  "Email profile deleted",
	})
	return nil
}

func (s *Service) ListAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*email.ProfileAssignment, error) {
	return s.repo.ListAssignments(ctx, tenantInfo)
}

func (s *Service) UpsertAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignments []*email.ProfileAssignment,
	userID pulid.ID,
) ([]*email.ProfileAssignment, error) {
	seen := make(map[email.Purpose]struct{}, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}
		if !email.IsValidPurpose(assignment.Purpose) {
			return nil, errortypes.NewValidationError(
				"purpose",
				errortypes.ErrInvalid,
				"Invalid email purpose",
			)
		}
		if assignment.ProfileID.IsNil() {
			return nil, errortypes.NewValidationError(
				"profileId",
				errortypes.ErrRequired,
				"Email profile is required",
			)
		}
		if _, ok := seen[assignment.Purpose]; ok {
			return nil, errortypes.NewValidationError(
				"purpose",
				errortypes.ErrDuplicate,
				"Email purpose is duplicated",
			)
		}
		seen[assignment.Purpose] = struct{}{}

		profile, err := s.repo.GetProfile(ctx, emailports.GetEmailEntityRequest{
			ID:         assignment.ProfileID,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if profile.Status != email.ProfileStatusActive {
			return nil, errortypes.NewValidationError(
				"profileId",
				errortypes.ErrInvalid,
				"Email profile must be active",
			)
		}
	}
	updated, err := s.repo.UpsertAssignments(ctx, tenantInfo, assignments)
	if err != nil {
		return nil, err
	}
	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceEmailProfile,
		ResourceID:     tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(map[string]any{"assignments": updated}),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}, auditservice.WithComment("Email profile assignments updated")); err != nil {
		s.l.Error("failed to log email assignment audit", zap.Error(err))
	}
	return updated, nil
}

func (s *Service) Send(
	ctx context.Context,
	req *emailports.SendEmailRequest,
) (*email.Message, error) {
	if err := s.requireOutbound(ctx, req.TenantInfo, req.Purpose); err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateSend(ctx, req); multiErr != nil {
		return nil, multiErr
	}
	profile, err := s.resolveProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	split, err := s.splitSuppressed(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(split.To) == 0 {
		return nil, errortypes.NewBusinessError(
			"Every To recipient is on the suppression list: {0}",
			strings.Join(split.Suppressed, ", "),
		)
	}
	req.To, req.CC, req.BCC = split.To, split.CC, split.BCC

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = newIdempotencyKey()
	}
	fromEmail := senderEmail(req, profile)
	msg := &email.Message{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		ProfileID:      profile.ID,
		Purpose:        req.Purpose,
		Provider:       profile.Provider,
		IdempotencyKey: req.IdempotencyKey,
		Status:         email.MessageStatusQueued,
		FromEmail:      fromEmail,
		FromName:       profile.SenderName,
		ReplyToEmail:   profile.ReplyToEmail,
		ToRecipients:   stringutils.NormalizeEmailAddresses(req.To),
		CCRecipients:   stringutils.NormalizeEmailAddresses(req.CC),
		BCCRecipients:  stringutils.NormalizeEmailAddresses(req.BCC),
		Subject:        strings.TrimSpace(req.Subject),
		BodyTextSize:   int64(len(req.Text)),
		BodyHTMLSize:   int64(len(req.HTML)),
	}
	stored, created, err := s.repo.CreateMessageOnce(ctx, msg)
	if err != nil {
		return nil, err
	}
	if !created {
		return s.replaySend(ctx, req, msg, stored)
	}
	msg = stored
	if err = s.persistAttachments(ctx, msg, req.Attachments); err != nil {
		if _, updateErr := s.markFailed(ctx, msg, err); updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}
	if err = s.startSendWorkflow(ctx, msg, req); err != nil {
		if _, updateErr := s.markFailed(ctx, msg, err); updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}
	return msg, nil
}

func (s *Service) requireOutbound(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purpose email.Purpose,
) error {
	if purpose == email.PurposeAuthentication {
		return nil
	}

	return planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityEmailOutbound,
	)
}

func (s *Service) replaySend(
	ctx context.Context,
	req *emailports.SendEmailRequest,
	requested *email.Message,
	stored *email.Message,
) (*email.Message, error) {
	if !stored.SameRecipientsAs(requested) {
		return nil, errortypes.NewBusinessError(
			"This email was already sent to different recipients under the same idempotency key",
		)
	}
	stored.Replayed = true

	switch stored.Status {
	case email.MessageStatusQueued:
		if err := s.startSendWorkflow(ctx, stored, req); err != nil {
			return nil, err
		}
		return stored, nil
	case email.MessageStatusFailed:
		return s.resendFailed(ctx, req, stored)
	case email.MessageStatusSending,
		email.MessageStatusSent,
		email.MessageStatusDelivered,
		email.MessageStatusBounced,
		email.MessageStatusComplained,
		email.MessageStatusOpened,
		email.MessageStatusClicked,
		email.MessageStatusSuppressed:
		return stored, nil
	default:
		return stored, nil
	}
}

func (s *Service) resendFailed(
	ctx context.Context,
	req *emailports.SendEmailRequest,
	stored *email.Message,
) (*email.Message, error) {
	if len(req.Attachments) > 0 {
		existing, err := s.repo.ListAttachments(ctx, emailports.ListEmailAttachmentsRequest{
			MessageID:  stored.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if len(existing) == 0 {
			if err = s.persistAttachments(ctx, stored, req.Attachments); err != nil {
				return nil, err
			}
		}
	}

	stored.Status = email.MessageStatusQueued
	stored.LastError = ""
	stored.FailedAt = 0
	stored.UpdatedAt = timeutils.NowUnix()
	requeued, err := s.repo.UpdateMessage(ctx, stored)
	if err != nil {
		return nil, err
	}
	requeued.Replayed = true
	if err = s.startSendWorkflow(ctx, requeued, req); err != nil {
		if _, updateErr := s.markFailed(ctx, requeued, err); updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}
	return requeued, nil
}

func (s *Service) TestSend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
	req *emailports.TestEmailProfileRequest,
) (*email.Message, error) {
	return s.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		ProfileID:      profileID,
		Purpose:        email.PurposeGeneral,
		To:             []string{req.To},
		Subject:        req.Subject,
		HTML:           req.HTML,
		Text:           req.Text,
		IdempotencyKey: "test-" + newIdempotencyKey(),
	})
}

func (s *Service) SendPersisted(
	ctx context.Context,
	req *emailports.SendPersistedEmailRequest,
) (*email.Message, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Email send request is required",
		)
	}
	msg, err := s.repo.GetMessage(ctx, emailports.GetEmailEntityRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if planErr := s.requireOutbound(ctx, pagination.TenantInfo{
		OrgID: msg.OrganizationID,
		BuID:  msg.BusinessUnitID,
	}, msg.Purpose); planErr != nil {
		return s.markFailed(ctx, msg, fmt.Errorf("%w: %w", ErrNonRetryableSend, planErr))
	}
	msg.Status = email.MessageStatusSending
	msg.Attempts++
	if msg, err = s.repo.UpdateMessage(ctx, msg); err != nil {
		return nil, err
	}

	sender, ok := s.providerSenders[msg.Provider]
	if !ok {
		return s.markFailed(
			ctx,
			msg,
			fmt.Errorf(
				"%w: no sender registered for email provider %s",
				ErrNonRetryableSend,
				msg.Provider,
			),
		)
	}

	cfg, err := s.integrationService.GetRuntimeConfig(ctx, pagination.TenantInfo{
		OrgID: msg.OrganizationID,
		BuID:  msg.BusinessUnitID,
	}, sender.IntegrationType())
	if err != nil {
		return s.markFailed(ctx, msg, providerConfigurationError(msg.Provider, err))
	}
	attachments, err := s.providerAttachments(ctx, req.TenantInfo, msg.ID)
	if err != nil {
		if errors.Is(err, ErrNonRetryableSend) {
			return s.markFailed(ctx, msg, err)
		}
		return s.markFailed(ctx, msg, fmt.Errorf("%w: %w", ErrRetryableSend, err))
	}

	result, err := sender.Send(ctx, SendProviderRequest{
		Config: cfg.Config,
		Message: SendProviderMessage{
			IdempotencyKey: msg.IdempotencyKey,
			From:           stringutils.FormatEmailAddress(msg.FromName, msg.FromEmail),
			ReplyTo:        msg.ReplyToEmail,
			To:             msg.ToRecipients,
			CC:             msg.CCRecipients,
			BCC:            msg.BCCRecipients,
			Subject:        msg.Subject,
			HTML:           req.HTML,
			Text:           req.Text,
			Attachments:    attachments,
			Headers:        req.Headers,
			OpenTracking:   req.OpenTracking,
		},
	})
	if err != nil {
		return s.markFailed(ctx, msg, s.describeSenderRejection(ctx, msg, err))
	}

	msg.Status = email.MessageStatusSent
	msg.ProviderMessageID = result.ProviderMessageID
	msg.SentAt = timeutils.NowUnix()
	msg.LastError = ""
	msg, err = s.repo.UpdateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}
	s.messageChanged(ctx, msg, nil)
	return msg, nil
}

func (s *Service) persistAttachments(
	ctx context.Context,
	msg *email.Message,
	attachments []emailports.EmailAttachment,
) error {
	if len(attachments) == 0 {
		return nil
	}
	entities := make([]*email.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		entity, err := s.persistAttachment(ctx, msg, attachment)
		if err != nil {
			return err
		}
		entities = append(entities, entity)
	}
	created, err := s.repo.CreateAttachments(ctx, entities)
	if err != nil {
		return err
	}
	msg.Attachments = created
	return nil
}

func (s *Service) persistAttachment(
	ctx context.Context,
	msg *email.Message,
	attachment emailports.EmailAttachment,
) (*email.Attachment, error) {
	fileName := fileutils.SafeFilename(attachment.FileName)
	if fileName == "" || fileName == "." || fileName == "/" {
		return nil, errortypes.NewValidationError(
			"attachments.fileName",
			errortypes.ErrRequired,
			"Attachment file name is required",
		)
	}
	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if !attachment.Seal.IsValid() {
		return nil, errortypes.NewValidationError(
			"attachments.seal",
			errortypes.ErrInvalid,
			"Attachment seal is not recognized",
		)
	}
	if attachment.Seal != email.AttachmentSealNone && len(attachment.Content) > 0 {
		return nil, errortypes.NewValidationError(
			"attachments.seal",
			errortypes.ErrInvalid,
			"A sealed attachment must reference a stored object, not carry content",
		)
	}

	objectKey := strings.TrimSpace(attachment.ObjectKey)
	sizeBytes := attachment.SizeBytes
	if len(attachment.Content) > 0 {
		objectKey = fileutils.GenerateStoragePath(
			msg.OrganizationID.String(),
			"email-attachments/"+msg.ID.String(),
			fileName,
		)
		info, err := s.storage.Upload(ctx, &kernelports.UploadParams{
			Key:         objectKey,
			ContentType: contentType,
			Size:        int64(len(attachment.Content)),
			Body:        bytes.NewReader(attachment.Content),
			Metadata: map[string]string{
				"message-id": msg.ID.String(),
			},
		})
		if err != nil {
			return nil, err
		}
		sizeBytes = info.Size
	}
	if objectKey == "" {
		return nil, errortypes.NewValidationError(
			"attachments.objectKey",
			errortypes.ErrRequired,
			"Attachment content or object key is required",
		)
	}

	return &email.Attachment{
		BusinessUnitID: msg.BusinessUnitID,
		OrganizationID: msg.OrganizationID,
		MessageID:      msg.ID,
		FileName:       fileName,
		ContentType:    contentType,
		ObjectKey:      objectKey,
		SizeBytes:      sizeBytes,
		Seal:           attachment.Seal,
	}, nil
}

func (s *Service) providerAttachments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) ([]ProviderAttachment, error) {
	attachments, err := s.repo.ListAttachments(ctx, emailports.ListEmailAttachmentsRequest{
		MessageID:  messageID,
		TenantInfo: tenantInfo,
	})
	if err != nil || len(attachments) == 0 {
		return []ProviderAttachment{}, err
	}
	result := make([]ProviderAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		content, contentErr := s.attachmentContent(ctx, attachment)
		if contentErr != nil {
			return nil, contentErr
		}
		result = append(result, ProviderAttachment{
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			Content:     content,
		})
	}
	return result, nil
}

func (s *Service) attachmentContent(
	ctx context.Context,
	attachment *email.Attachment,
) ([]byte, error) {
	download, err := s.storage.Download(ctx, attachment.ObjectKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = download.Body.Close() }()

	switch attachment.Seal {
	case email.AttachmentSealNone:
		return io.ReadAll(download.Body)
	case email.AttachmentSealDocument:
		return s.unsealDocument(ctx, attachment, download)
	default:
		return nil, fmt.Errorf(
			"%w: attachment %s has an unknown seal %q",
			ErrNonRetryableSend,
			attachment.ID,
			attachment.Seal,
		)
	}
}

func (s *Service) unsealDocument(
	ctx context.Context,
	attachment *email.Attachment,
	download *kernelports.DownloadResult,
) ([]byte, error) {
	if s.encryptionService == nil {
		return nil, fmt.Errorf(
			"%w: attachment %s is not a sealed document",
			ErrNonRetryableSend,
			attachment.ID,
		)
	}

	sealedSize := download.Size
	if sealedSize <= 0 {
		sealedSize = -1
	}
	opened, err := s.encryptionService.OpenBlob(
		ctx,
		download.Body,
		sealedSize,
		encryptionservice.AAD{
			Purpose:        encryptionservice.PurposeDocument,
			OrganizationID: attachment.OrganizationID,
			BusinessUnitID: attachment.BusinessUnitID,
			ResourceID:     attachment.ObjectKey,
		},
	)
	if err != nil {
		return nil, unsealError(attachment, err)
	}
	content, err := io.ReadAll(opened)
	if err != nil {
		return nil, unsealError(attachment, err)
	}

	return content, nil
}

func unsealError(attachment *email.Attachment, err error) error {
	if errors.Is(err, encryptionservice.ErrBlobRejected) {
		return fmt.Errorf(
			"%w: attachment %s could not be unsealed: %w",
			ErrNonRetryableSend,
			attachment.ID,
			err,
		)
	}
	return fmt.Errorf("read attachment %s: %w", attachment.ID, err)
}

func (s *Service) startSendWorkflow(
	ctx context.Context,
	msg *email.Message,
	req *emailports.SendEmailRequest,
) error {
	if !s.workflowStarter.Enabled() {
		return kernelports.ErrWorkflowStarterDisabled
	}

	workflowID := fmt.Sprintf(
		"email-send-%s-%s-%s",
		msg.OrganizationID.String(),
		msg.BusinessUnitID.String(),
		msg.ID.String(),
	)
	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:            workflowID,
			TaskQueue:     temporaltype.EmailTaskQueue,
			StaticSummary: "Send email message " + msg.ID.String(),
		},
		emailcontract.SendEmailWorkflowName,
		&emailcontract.SendEmailPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: msg.OrganizationID,
				BusinessUnitID: msg.BusinessUnitID,
				Timestamp:      timeutils.NowUnix(),
			},
			MessageID:    msg.ID,
			HTML:         req.HTML,
			Text:         req.Text,
			Headers:      req.Headers,
			OpenTracking: req.OpenTracking,
		},
	)
	return err
}

func (s *Service) HandleProviderEvent(
	ctx context.Context,
	params HandleProviderEventParams,
) error {
	tenantInfo := params.TenantInfo
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	event := params.Event
	if event.MessageID.IsNil() {
		if params.ProviderMessageID != "" {
			msg, lookupErr := s.repo.GetMessageByProviderID(
				ctx,
				emailports.GetEmailMessageByProviderIDRequest{
					Provider:          event.Provider,
					ProviderMessageID: params.ProviderMessageID,
					TenantInfo:        tenantInfo,
				},
			)
			if lookupErr != nil && !errortypes.IsNotFoundError(lookupErr) {
				return lookupErr
			}
			if lookupErr == nil {
				event.MessageID = msg.ID
			}
		}
	}
	inserted, err := s.repo.CreateEvent(ctx, event)
	if err != nil || !inserted {
		return err
	}
	if event.MessageID.IsNil() {
		return nil
	}
	msg, err := s.repo.GetMessage(ctx, emailports.GetEmailEntityRequest{
		ID:         event.MessageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	changed, err := s.applyProviderEvent(ctx, msg, event, tenantInfo)
	if err != nil {
		return err
	}
	if changed {
		if _, err = s.repo.UpdateMessage(ctx, msg); err != nil {
			return err
		}
		s.messageChanged(ctx, msg, event)
	}
	if params.SuppressionReason == "" || strings.TrimSpace(event.Recipient) == "" {
		return nil
	}
	_, err = s.repo.CreateSuppression(ctx, &email.Suppression{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		EmailAddress:   event.Recipient,
		Reason:         params.SuppressionReason,
		Provider:       event.Provider,
		SourceEventID:  event.ProviderEventID,
	})
	return err
}

func (s *Service) applyProviderEvent(
	ctx context.Context,
	msg *email.Message,
	event *email.Event,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	next, changed := email.NextMessageStatus(msg.Status, event.Type)
	if !changed {
		return false, nil
	}
	now := timeutils.NowUnix()
	if next == email.MessageStatusBounced {
		bounced, err := s.bouncedRecipients(ctx, msg, event, tenantInfo)
		if err != nil {
			return false, err
		}
		if remaining := unbouncedRecipients(msg, bounced); len(remaining) > 0 {
			msg.LastError = "Bounced for " + strings.Join(bounced, ", ")
			return true, nil
		}
		msg.LastError = "Bounced for every recipient"
	}
	msg.Status = next
	switch {
	case next == email.MessageStatusDelivered:
		msg.DeliveredAt = now
	case next.IsFailure():
		msg.FailedAt = now
	}
	return true, nil
}

func (s *Service) bouncedRecipients(
	ctx context.Context,
	msg *email.Message,
	event *email.Event,
	tenantInfo pagination.TenantInfo,
) ([]string, error) {
	if strings.TrimSpace(event.Recipient) == "" {
		return nil, nil
	}
	recipients, err := s.repo.ListEventRecipients(ctx, emailports.ListEmailEventRecipientsRequest{
		MessageID:  msg.ID,
		Type:       email.EventTypeBounced,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	return stringutils.NormalizeEmailList(recipients), nil
}

func unbouncedRecipients(msg *email.Message, bounced []string) []string {
	if len(bounced) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(bounced))
	for _, address := range bounced {
		seen[address] = struct{}{}
	}
	all := stringutils.NormalizeEmailList(
		slices.Concat(msg.ToRecipients, msg.CCRecipients, msg.BCCRecipients),
	)
	remaining := make([]string, 0, len(all))
	for _, address := range all {
		if _, ok := seen[address]; !ok {
			remaining = append(remaining, address)
		}
	}
	return remaining
}

func (s *Service) ListMessages(
	ctx context.Context,
	req *emailports.ListEmailMessagesRequest,
) (*pagination.ListResult[*email.Message], error) {
	return s.repo.ListMessages(ctx, req)
}

func (s *Service) GetMessage(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
) (*email.Message, error) {
	return s.repo.GetMessage(ctx, req)
}

func (s *Service) ListSuppressions(
	ctx context.Context,
	req *emailports.ListEmailSuppressionsRequest,
) (*pagination.ListResult[*email.Suppression], error) {
	return s.repo.ListSuppressions(ctx, req)
}

func (s *Service) CreateSuppression(
	ctx context.Context,
	suppression *email.Suppression,
) (*email.Suppression, error) {
	if !strings.Contains(suppression.EmailAddress, "@") {
		return nil, errortypes.NewValidationError(
			"emailAddress",
			errortypes.ErrInvalid,
			"Email address is invalid",
		)
	}
	if suppression.Reason == "" {
		suppression.Reason = email.SuppressionReasonManual
	}
	return s.repo.CreateSuppression(ctx, suppression)
}

func (s *Service) DeleteSuppression(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
) error {
	return s.repo.DeleteSuppression(ctx, req)
}

func (s *Service) ResolveTenantByWebhookToken(
	ctx context.Context,
	token string,
) (pagination.TenantInfo, string, error) {
	sender, ok := s.providerSenders[email.ProviderResend]
	if !ok {
		return pagination.TenantInfo{}, "", errortypes.NewValidationError(
			"provider",
			errortypes.ErrInvalid,
			"Unsupported email provider",
		)
	}
	cfg, err := s.repo.GetEmailWebhookConfig(ctx, emailports.GetEmailWebhookConfigRequest{
		IntegrationType: sender.IntegrationType(),
		Token:           token,
	})
	if err != nil {
		return pagination.TenantInfo{}, "", err
	}
	signingSecret, err := s.integrationService.OpenSecret(
		cfg.TenantInfo,
		sender.IntegrationType(),
		integration.WebhookSigningSecretField,
		cfg.SigningSecret,
	)
	if err != nil {
		return pagination.TenantInfo{}, "", err
	}
	return cfg.TenantInfo, signingSecret, nil
}

func (s *Service) ResolveTenantByProviderWebhookToken(
	ctx context.Context,
	provider email.Provider,
	token string,
) (pagination.TenantInfo, error) {
	sender, ok := s.providerSenders[provider]
	if !ok {
		return pagination.TenantInfo{}, errortypes.NewValidationError(
			"provider",
			errortypes.ErrInvalid,
			"Unsupported email provider",
		)
	}
	cfg, err := s.repo.GetEmailWebhookConfig(ctx, emailports.GetEmailWebhookConfigRequest{
		IntegrationType: sender.IntegrationType(),
		Token:           token,
	})
	if err != nil {
		return pagination.TenantInfo{}, err
	}
	return cfg.TenantInfo, nil
}

func (s *Service) resolveProfile(
	ctx context.Context,
	req *emailports.SendEmailRequest,
) (*email.Profile, error) {
	if !req.ProfileID.IsNil() {
		return s.repo.GetProfile(ctx, emailports.GetEmailEntityRequest{
			ID:         req.ProfileID,
			TenantInfo: req.TenantInfo,
		})
	}
	profile, err := s.repo.GetAssignedProfile(ctx, req.TenantInfo, req.Purpose)
	if err == nil {
		return profile, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	profile, err = s.repo.GetAssignedProfile(ctx, req.TenantInfo, email.PurposeGeneral)
	if errortypes.IsNotFoundError(err) {
		return nil, noEmailProfile()
	}

	return profile, err
}

// noEmailProfile says what a send with no profile behind it is missing.
// "EmailProfile not found" told an agent nothing it could act on, and it
// kept trying; this says nothing can be emailed until one is set up.
func noEmailProfile() error {
	return errortypes.NewBusinessError(
		"No email profile is set up for this organization, so nothing can be emailed until " +
			"an admin adds one in the email settings. Say so, and offer another way to reach them",
	).WithInternal(emailports.ErrNoEmailProfile)
}

func (s *Service) markFailed(
	ctx context.Context,
	msg *email.Message,
	err error,
) (*email.Message, error) {
	msg.Status = email.MessageStatusFailed
	msg.LastError = err.Error()
	msg.FailedAt = timeutils.NowUnix()
	updated, updateErr := s.repo.UpdateMessage(ctx, msg)
	if updateErr != nil {
		return nil, updateErr
	}
	s.messageChanged(ctx, updated, nil)
	return updated, err
}

func (s *Service) messageChanged(ctx context.Context, msg *email.Message, event *email.Event) {
	s.syncInvoiceAttempts(ctx, msg)
	if msg == nil {
		return
	}
	observers := s.observers.Load()
	if observers == nil {
		return
	}
	for _, observer := range *observers {
		observer.EmailMessageChanged(ctx, emailports.EmailMessageChange{Message: msg, Event: event})
	}
}

func (s *Service) describeSenderRejection(
	ctx context.Context,
	msg *email.Message,
	err error,
) error {
	var rejected *SenderRejectedError
	if !errors.As(err, &rejected) {
		return err
	}
	rejected.Address = msg.FromEmail
	rejected.Origin = s.profileSenderOrigin(ctx, msg)
	return rejected
}

func (s *Service) profileSenderOrigin(ctx context.Context, msg *email.Message) string {
	profile, err := s.repo.GetProfile(ctx, emailports.GetEmailEntityRequest{
		ID: msg.ProfileID,
		TenantInfo: pagination.TenantInfo{
			OrgID: msg.OrganizationID,
			BuID:  msg.BusinessUnitID,
		},
	})
	if err != nil {
		if s.l != nil {
			s.l.Warn(
				"failed to load email profile to describe a rejected sender",
				zap.Error(err),
				zap.String("messageId", msg.ID.String()),
			)
		}
		return ""
	}
	if !strings.EqualFold(
		strings.TrimSpace(profile.SenderEmail),
		strings.TrimSpace(msg.FromEmail),
	) {
		return ""
	}
	return fmt.Sprintf(
		"the sender email of the %q email profile (Organization settings > Email profiles)",
		profile.Name,
	)
}

func (s *Service) syncInvoiceAttempts(ctx context.Context, msg *email.Message) {
	if s.invoiceRepo == nil || msg == nil || msg.ID.IsNil() {
		return
	}
	updated, err := s.invoiceRepo.SyncEmailAttemptsForMessage(ctx, msg.ID, pagination.TenantInfo{
		OrgID: msg.OrganizationID,
		BuID:  msg.BusinessUnitID,
	})
	if err != nil {
		if s.l != nil {
			s.l.Error(
				"failed to sync invoice email attempts",
				zap.Error(err),
				zap.String("messageId", msg.ID.String()),
			)
		}
		return
	}
	for _, entity := range updated {
		realtimeinvalidation.PublishAfterCommit(ctx, s.realtime, s.l, realtimeinvalidation.ChangeOf(
			permission.ResourceInvoice.String(),
			entity,
			invoicerealtime.RealtimeActionSendUpdated,
			permissionports.SystemAuditActor(),
			invoicerealtime.NewRealtimeSnapshot(entity),
		))
	}
}

type emailProfileAuditParams struct {
	Current  *email.Profile
	Previous *email.Profile
	UserID   pulid.ID
	Op       permission.Operation
	Comment  string
}

func (s *Service) logAudit(p emailProfileAuditParams) {
	if p.Current == nil {
		return
	}
	if err := s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceEmailProfile,
		ResourceID:     p.Current.ID.String(),
		Operation:      p.Op,
		UserID:         p.UserID,
		CurrentState:   jsonutils.MustToJSON(p.Current),
		PreviousState:  jsonutils.MustToJSON(p.Previous),
		OrganizationID: p.Current.OrganizationID,
		BusinessUnitID: p.Current.BusinessUnitID,
	}, auditservice.WithComment(p.Comment), auditservice.WithDiff(p.Previous, p.Current)); err != nil {
		s.l.Error("failed to log email profile audit", zap.Error(err))
	}
}

func newIdempotencyKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return pulid.MustNew("emlidm_").String()
	}
	return hex.EncodeToString(b[:])
}
