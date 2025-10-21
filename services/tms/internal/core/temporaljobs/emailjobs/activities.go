package emailjobs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/email/providers"
	"github.com/emoss08/trenova/internal/core/services/encryption"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Logger            *zap.Logger
	ProfileRepo       repositories.EmailProfileRepository
	EncryptionService encryption.Service
	ProviderRegistry  providers.Registry
}

type Activities struct {
	logger            *zap.Logger
	profileRepo       repositories.EmailProfileRepository
	encryptionService encryption.Service
	providerRegistry  providers.Registry
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		logger:            p.Logger.With(zap.String("worker", "email")),
		profileRepo:       p.ProfileRepo,
		encryptionService: p.EncryptionService,
		providerRegistry:  p.ProviderRegistry,
	}
}

func (a *Activities) SendEmailActivity( //nolint:funlen // This is a long function
	ctx context.Context,
	payload *temporaltype.SendEmailPayload,
) (*temporaltype.EmailResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting email send activity",
		"to", payload.To,
		"subject", payload.Subject,
		"orgID", payload.OrganizationID,
	)

	if len(payload.To) == 0 {
		return nil, temporaltype.NewInvalidInputError(
			"At least one recipient is required",
			map[string]any{"to": payload.To},
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "fetching email profile")

	profile, err := a.getProfileOrDefault(ctx, payload)
	if err != nil {
		logger.Error("Failed to get email profile", "error", err)
		return nil, temporaltype.NewNonRetryableError(
			"Failed to get email profile",
			err,
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "initializing provider")

	provider, err := a.getProvider(profile)
	if err != nil {
		logger.Error("Failed to get email provider", "error", err)
		return nil, temporaltype.NewNonRetryableError(
			"Failed to initialize email provider",
			err,
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "sending email")

	providerConfig := &providers.ProviderConfig{
		Host:           profile.Host,
		Port:           profile.Port,
		Username:       profile.Username,
		AuthType:       profile.AuthType,
		EncryptionType: profile.EncryptionType,
		TimeoutSeconds: profile.TimeoutSeconds,
		MaxConnections: profile.MaxConnections,
		Metadata:       profile.Metadata,
	}

	if profile.EncryptedPassword != "" {
		decrypted, decryptedErr := a.encryptionService.Decrypt(profile.EncryptedPassword)
		if decryptedErr != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", decryptedErr)
		}
		providerConfig.Password = decrypted
	}

	if profile.EncryptedAPIKey != "" {
		decrypted, decryptedErr := a.encryptionService.Decrypt(profile.EncryptedAPIKey)
		if decryptedErr != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", decryptedErr)
		}
		providerConfig.APIKey = decrypted
	}

	var attachments []providers.Attachment
	if len(payload.Attachments) > 0 {
		activity.RecordHeartbeat(ctx, "processing attachments")
		attachments, err = a.processAttachments(ctx, payload.Attachments)
		if err != nil {
			logger.Warn("Failed to process some attachments, continuing without them",
				"error", err,
				"attachmentCount", len(payload.Attachments),
			)
		}
	}

	message := &providers.Message{
		From: providers.EmailAddress{
			Email: profile.FromAddress,
			Name:  profile.FromName,
		},
		To:          a.convertToEmailAddresses(payload.To),
		CC:          a.convertToEmailAddresses(payload.CC),
		BCC:         a.convertToEmailAddresses(payload.BCC),
		Subject:     payload.Subject,
		HTMLBody:    payload.HTMLBody,
		TextBody:    payload.TextBody,
		Headers:     make(map[string]string),
		Priority:    payload.Priority,
		Attachments: attachments,
	}

	if profile.ReplyTo != "" {
		message.ReplyTo = &providers.EmailAddress{Email: profile.ReplyTo}
	}

	result, err := provider.Send(ctx, providerConfig, message)
	if err != nil {
		logger.Error("Failed to send email",
			"error", err,
			"provider", profile.ProviderType,
		)

		appErr := temporaltype.ClassifyError(err)
		return nil, appErr.ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "email sent successfully")

	logger.Info("Email sent successfully",
		"messageID", result,
		"provider", profile.ProviderType,
	)

	return &temporaltype.EmailResult{
		MessageID:    result,
		Status:       "sent",
		ProviderType: string(profile.ProviderType),
		SentAt:       utils.NowUnix(),
	}, nil
}

func (a *Activities) RenderTemplateActivity(
	ctx context.Context,
	payload *temporaltype.SendTemplatedEmailPayload,
) (*temporaltype.SendEmailPayload, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting template render activity",
		"templateKey", payload.TemplateKey,
		"orgID", payload.OrganizationID,
	)

	activity.RecordHeartbeat(ctx, "rendering template")

	tm, err := NewTemplateManager()
	if err != nil {
		logger.Error("Failed to initialize template manager", "error", err)
		return nil, temporaltype.NewNonRetryableError(
			"Failed to initialize template manager",
			err,
		).ToTemporalError()
	}

	rendered, err := tm.RenderTemplate(payload.TemplateKey, payload.Variables)
	if err != nil {
		logger.Error("Failed to render template",
			"error", err,
			"templateKey", payload.TemplateKey,
		)
		return nil, temporaltype.NewNonRetryableError(
			fmt.Sprintf("Failed to render template %s", payload.TemplateKey),
			err,
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "template rendered successfully")

	return &temporaltype.SendEmailPayload{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		UserID:         payload.UserID,
		ProfileID:      payload.ProfileID,
		To:             payload.To,
		CC:             payload.CC,
		BCC:            payload.BCC,
		Subject:        rendered.Subject,
		HTMLBody:       rendered.HTMLBody,
		TextBody:       rendered.TextBody,
		Priority:       payload.Priority,
		Metadata:       payload.Metadata,
		Attachments:    payload.Attachments,
	}, nil
}

func (a *Activities) getProfileOrDefault(
	ctx context.Context,
	payload *temporaltype.SendEmailPayload,
) (*email.EmailProfile, error) {
	req := repositories.GetEmailProfileByIDRequest{
		OrgID:      payload.OrganizationID,
		BuID:       payload.BusinessUnitID,
		UserID:     payload.UserID,
		ProfileID:  pulid.ConvertFromPtr(payload.ProfileID),
		ExpandData: true,
	}

	if req.ProfileID.IsNotNil() {
		return a.profileRepo.Get(ctx, req)
	}

	return a.profileRepo.GetDefault(ctx, req.OrgID, req.BuID)
}

func (a *Activities) getProvider(profile *email.EmailProfile) (providers.Provider, error) {
	provider, err := a.providerRegistry.Get(profile.ProviderType)
	if err != nil {
		return nil, fmt.Errorf("provider %s not found: %w", profile.ProviderType, err)
	}

	return provider, nil
}

func (a *Activities) convertToEmailAddresses(emails []string) []providers.EmailAddress {
	addresses := make([]providers.EmailAddress, len(emails))
	for i, email := range emails {
		addresses[i] = providers.EmailAddress{Email: email}
	}
	return addresses
}

func (a *Activities) processAttachments(
	ctx context.Context,
	attachmentMetas []services.AttachmentMeta,
) ([]providers.Attachment, error) {
	if len(attachmentMetas) == 0 {
		return nil, nil
	}

	attachments := make([]providers.Attachment, 0, len(attachmentMetas))
	client := &http.Client{Timeout: 30 * time.Second}

	for _, meta := range attachmentMetas {
		attachment, err := a.downloadAttachment(ctx, client, meta)
		if err != nil {
			a.logger.Warn("Skipping attachment due to error",
				zap.String("fileName", meta.FileName),
				zap.Error(err),
			)
			continue
		}
		attachments = append(attachments, attachment)
	}

	if len(attachments) == 0 && len(attachmentMetas) > 0 {
		return nil, ErrFailedToProcessAttachments
	}

	return attachments, nil
}

func (a *Activities) downloadAttachment(
	ctx context.Context,
	client *http.Client,
	meta services.AttachmentMeta,
) (providers.Attachment, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meta.URL, http.NoBody)
	if err != nil {
		return providers.Attachment{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return providers.Attachment{}, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return providers.Attachment{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return providers.Attachment{}, fmt.Errorf("read data: %w", err)
	}

	return providers.Attachment{
		FileName:    meta.FileName,
		ContentType: meta.ContentType,
		Data:        data,
		ContentID:   meta.ContentID,
	}, nil
}
