package emailservice

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/emailjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
	Repo               repositories.EmailRepository
	InvoiceRepo        repositories.InvoiceRepository
	Validator          *Validator
	IntegrationService *integrationservice.Service
	EncryptionService  *encryptionservice.Service
	AuditService       services.AuditService
	WorkflowStarter    services.WorkflowStarter
	Storage            storage.Client
}

type Service struct {
	l                  *zap.Logger
	repo               repositories.EmailRepository
	invoiceRepo        repositories.InvoiceRepository
	validator          *Validator
	integrationService *integrationservice.Service
	encryptionService  *encryptionservice.Service
	auditService       services.AuditService
	providerSenders    map[email.Provider]ProviderSender
	workflowStarter    services.WorkflowStarter
	storage            storage.Client
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
	}
}

func (s *Service) ListProfiles(
	ctx context.Context,
	req *repositories.ListEmailProfilesRequest,
) (*pagination.ListResult[*email.Profile], error) {
	return s.repo.ListProfiles(ctx, req)
}

func (s *Service) ListProfilesConnection(
	ctx context.Context,
	req *repositories.ListEmailProfileConnectionRequest,
) (*pagination.CursorListResult[*email.Profile], error) {
	return s.repo.ListProfilesConnection(ctx, req)
}

func (s *Service) SelectProfileOptions(
	ctx context.Context,
	req *repositories.EmailProfileSelectOptionsRequest,
) (*pagination.ListResult[*email.Profile], error) {
	return s.repo.SelectProfileOptions(ctx, req)
}

func (s *Service) GetProfile(
	ctx context.Context,
	req repositories.GetEmailEntityRequest,
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
	original, err := s.repo.GetProfile(ctx, repositories.GetEmailEntityRequest{
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
	req repositories.GetEmailEntityRequest,
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
			return nil, errortypes.NewValidationError("purpose", errortypes.ErrInvalid, "Invalid email purpose")
		}
		if assignment.ProfileID.IsNil() {
			return nil, errortypes.NewValidationError("profileId", errortypes.ErrRequired, "Email profile is required")
		}
		if _, ok := seen[assignment.Purpose]; ok {
			return nil, errortypes.NewValidationError("purpose", errortypes.ErrDuplicate, "Email purpose is duplicated")
		}
		seen[assignment.Purpose] = struct{}{}

		profile, err := s.repo.GetProfile(ctx, repositories.GetEmailEntityRequest{
			ID:         assignment.ProfileID,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if profile.Status != email.ProfileStatusActive {
			return nil, errortypes.NewValidationError("profileId", errortypes.ErrInvalid, "Email profile must be active")
		}
	}
	updated, err := s.repo.UpsertAssignments(ctx, tenantInfo, assignments)
	if err != nil {
		return nil, err
	}
	if err = s.auditService.LogAction(&services.LogActionParams{
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
	req *services.SendEmailRequest,
) (*email.Message, error) {
	if multiErr := s.validator.ValidateSend(ctx, req); multiErr != nil {
		return nil, multiErr
	}
	profile, err := s.resolveProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, recipient := range req.To {
		suppressed, err := s.repo.HasSuppression(ctx, req.TenantInfo, recipient)
		if err != nil {
			return nil, err
		}
		if suppressed {
			return nil, errortypes.NewBusinessError("recipient is suppressed: " + recipient)
		}
	}

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = newIdempotencyKey()
	}
	fromEmail := strings.TrimSpace(req.FromEmail)
	if fromEmail == "" {
		fromEmail = profile.SenderEmail
	}
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
	msg, err = s.repo.CreateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}
	if err = s.persistAttachments(ctx, msg, req.Attachments); err != nil {
		if _, updateErr := s.markFailed(ctx, msg, err); updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}
	if err = s.startSendWorkflow(ctx, msg, req.HTML, req.Text, req.Headers, req.OpenTracking); err != nil {
		if _, updateErr := s.markFailed(ctx, msg, err); updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}
	return msg, nil
}

func (s *Service) TestSend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
	req *services.TestEmailProfileRequest,
) (*email.Message, error) {
	return s.Send(ctx, &services.SendEmailRequest{
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
	req *services.SendPersistedEmailRequest,
) (*email.Message, error) {
	if req == nil {
		return nil, errortypes.NewValidationError("request", errortypes.ErrRequired, "Email send request is required")
	}
	msg, err := s.repo.GetMessage(ctx, repositories.GetEmailEntityRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
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
			fmt.Errorf("%w: no sender registered for email provider %s", ErrNonRetryableSend, msg.Provider),
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
		return s.markFailed(ctx, msg, err)
	}

	msg.Status = email.MessageStatusSent
	msg.ProviderMessageID = result.ProviderMessageID
	msg.SentAt = timeutils.NowUnix()
	msg.LastError = ""
	msg, err = s.repo.UpdateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}
	s.syncInvoiceAttempts(ctx, msg)
	return msg, nil
}

func (s *Service) persistAttachments(
	ctx context.Context,
	msg *email.Message,
	attachments []services.EmailAttachment,
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
	attachment services.EmailAttachment,
) (*email.Attachment, error) {
	fileName := fileutils.SafeFilename(attachment.FileName)
	if fileName == "" || fileName == "." || fileName == "/" {
		return nil, errortypes.NewValidationError("attachments.fileName", errortypes.ErrRequired, "Attachment file name is required")
	}
	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectKey := strings.TrimSpace(attachment.ObjectKey)
	sizeBytes := attachment.SizeBytes
	if len(attachment.Content) > 0 {
		objectKey = fileutils.GenerateStoragePath(
			msg.OrganizationID.String(),
			"email-attachments/"+msg.ID.String(),
			fileName,
		)
		info, err := s.storage.Upload(ctx, &storage.UploadParams{
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
		return nil, errortypes.NewValidationError("attachments.objectKey", errortypes.ErrRequired, "Attachment content or object key is required")
	}

	return &email.Attachment{
		BusinessUnitID: msg.BusinessUnitID,
		OrganizationID: msg.OrganizationID,
		MessageID:      msg.ID,
		FileName:       fileName,
		ContentType:    contentType,
		ObjectKey:      objectKey,
		SizeBytes:      sizeBytes,
	}, nil
}

func (s *Service) providerAttachments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) ([]ProviderAttachment, error) {
	attachments, err := s.repo.ListAttachments(ctx, repositories.ListEmailAttachmentsRequest{
		MessageID:  messageID,
		TenantInfo: tenantInfo,
	})
	if err != nil || len(attachments) == 0 {
		return []ProviderAttachment{}, err
	}
	result := make([]ProviderAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		download, err := s.storage.Download(ctx, attachment.ObjectKey)
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(download.Body)
		closeErr := download.Body.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		result = append(result, ProviderAttachment{
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			Content:     content,
		})
	}
	return result, nil
}

func (s *Service) startSendWorkflow(
	ctx context.Context,
	msg *email.Message,
	html string,
	text string,
	headers map[string]string,
	openTracking bool,
) error {
	if !s.workflowStarter.Enabled() {
		return services.ErrWorkflowStarterDisabled
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
		emailjobs.SendEmailWorkflowName,
		&emailjobs.SendEmailPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: msg.OrganizationID,
				BusinessUnitID: msg.BusinessUnitID,
				Timestamp:      timeutils.NowUnix(),
			},
			MessageID:    msg.ID,
			HTML:         html,
			Text:         text,
			Headers:      headers,
			OpenTracking: openTracking,
		},
	)
	return err
}

func (s *Service) HandleProviderEvent(
	ctx context.Context,
	params HandleProviderEventParams,
) error {
	tenantInfo := params.TenantInfo
	event := params.Event
	if event.MessageID.IsNil() {
		if params.ProviderMessageID != "" {
			msg, lookupErr := s.repo.GetMessageByProviderID(
				ctx,
				repositories.GetEmailMessageByProviderIDRequest{
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
	msg, err := s.repo.GetMessage(ctx, repositories.GetEmailEntityRequest{
		ID:         event.MessageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	now := timeutils.NowUnix()
	switch event.Type {
	case email.EventTypeDelivered:
		msg.Status = email.MessageStatusDelivered
		msg.DeliveredAt = now
	case email.EventTypeOpened:
		msg.Status = email.MessageStatusOpened
	case email.EventTypeClicked:
		msg.Status = email.MessageStatusClicked
	case email.EventTypeBounced:
		msg.Status = email.MessageStatusBounced
		msg.FailedAt = now
	case email.EventTypeComplained:
		msg.Status = email.MessageStatusComplained
		msg.FailedAt = now
	case email.EventTypeFailed:
		msg.Status = email.MessageStatusFailed
		msg.FailedAt = now
	}
	if _, err = s.repo.UpdateMessage(ctx, msg); err != nil {
		return err
	}
	s.syncInvoiceAttempts(ctx, msg)
	if params.SuppressionReason != "" {
		_, err = s.repo.CreateSuppression(ctx, &email.Suppression{
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
			EmailAddress:   event.Recipient,
			Reason:         params.SuppressionReason,
			Provider:       event.Provider,
			SourceEventID:  event.ProviderEventID,
		})
	}
	return err
}

func (s *Service) ListMessages(
	ctx context.Context,
	req *repositories.ListEmailMessagesRequest,
) (*pagination.ListResult[*email.Message], error) {
	return s.repo.ListMessages(ctx, req)
}

func (s *Service) GetMessage(
	ctx context.Context,
	req repositories.GetEmailEntityRequest,
) (*email.Message, error) {
	return s.repo.GetMessage(ctx, req)
}

func (s *Service) ListSuppressions(
	ctx context.Context,
	req *repositories.ListEmailSuppressionsRequest,
) (*pagination.ListResult[*email.Suppression], error) {
	return s.repo.ListSuppressions(ctx, req)
}

func (s *Service) CreateSuppression(
	ctx context.Context,
	suppression *email.Suppression,
) (*email.Suppression, error) {
	if !strings.Contains(suppression.EmailAddress, "@") {
		return nil, errortypes.NewValidationError("emailAddress", errortypes.ErrInvalid, "Email address is invalid")
	}
	if suppression.Reason == "" {
		suppression.Reason = email.SuppressionReasonManual
	}
	return s.repo.CreateSuppression(ctx, suppression)
}

func (s *Service) DeleteSuppression(
	ctx context.Context,
	req repositories.GetEmailEntityRequest,
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
	cfg, err := s.repo.GetEmailWebhookConfig(ctx, repositories.GetEmailWebhookConfigRequest{
		IntegrationType: sender.IntegrationType(),
		Token:           token,
	})
	if err != nil {
		return pagination.TenantInfo{}, "", err
	}
	signingSecret, err := s.encryptionService.DecryptString(cfg.SigningSecret)
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
	cfg, err := s.repo.GetEmailWebhookConfig(ctx, repositories.GetEmailWebhookConfigRequest{
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
	req *services.SendEmailRequest,
) (*email.Profile, error) {
	if !req.ProfileID.IsNil() {
		return s.repo.GetProfile(ctx, repositories.GetEmailEntityRequest{
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
	return s.repo.GetAssignedProfile(ctx, req.TenantInfo, email.PurposeGeneral)
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
	s.syncInvoiceAttempts(ctx, updated)
	return updated, err
}

func (s *Service) syncInvoiceAttempts(ctx context.Context, msg *email.Message) {
	if s.invoiceRepo == nil || msg == nil || msg.ID.IsNil() {
		return
	}
	if err := s.invoiceRepo.SyncEmailAttemptsForMessage(ctx, msg.ID, pagination.TenantInfo{
		OrgID: msg.OrganizationID,
		BuID:  msg.BusinessUnitID,
	}); err != nil && s.l != nil {
		s.l.Error("failed to sync invoice email attempts", zap.Error(err), zap.String("messageId", msg.ID.String()))
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
	if err := s.auditService.LogAction(&services.LogActionParams{
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
