package email

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/email/providers"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"

	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger           *zap.Logger
	TemporalClient   client.Client
	ProviderRegistry providers.Registry
}

type Service struct {
	l                *zap.Logger
	temporalClient   client.Client
	providerRegistry providers.Registry
}

func NewService(p ServiceParams) services.EmailService {
	return &Service{
		l:                p.Logger.With(zap.String("service", "email")),
		temporalClient:   p.TemporalClient,
		providerRegistry: p.ProviderRegistry,
	}
}

func (s *Service) SendEmail(
	ctx context.Context,
	req *services.SendEmailRequest,
) error {
	log := s.l.With(
		zap.String("operation", "send_email"),
		zap.String("org_id", req.OrganizationID.String()),
	)

	if err := req.Validate(); err != nil {
		return err
	}

	payload := &temporaltype.SendEmailPayload{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		UserID:         req.UserID,
		ProfileID:      req.ProfileID,
		To:             req.To,
		CC:             req.CC,
		BCC:            req.BCC,
		Subject:        req.Subject,
		HTMLBody:       req.HTMLBody,
		TextBody:       req.TextBody,
		Priority:       req.Priority,
		Metadata:       req.Metadata,
		Attachments:    req.Attachments,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("email-%s-%d", pulid.MustNew("eml_"), len(req.To)),
		TaskQueue: temporaltype.EmailTaskQueue,
	}

	_, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "SendEmailWorkflow", payload)
	if err != nil {
		log.Error("failed to start email workflow", zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) SendSystemEmail(
	ctx context.Context,
	req *services.SendSystemEmailRequest,
) error {
	log := s.l.With(
		zap.String("operation", "send_system_email"),
		zap.Any("req", req),
	)

	payload := &temporaltype.SendTemplatedEmailPayload{
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
		UserID:         req.UserID,
		TemplateKey:    req.TemplateKey,
		To:             req.To,
		Variables:      req.Variables,
		Priority:       email.PriorityMedium,
	}
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("email-template-%s-%s", req.TemplateKey, pulid.MustNew("eml_")),
		TaskQueue: temporaltype.EmailTaskQueue,
	}

	_, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"SendTemplatedEmailWorkflow",
		payload,
	)
	if err != nil {
		log.Error("failed to start templated email workflow", zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) TestConnection(
	ctx context.Context,
	req *services.TestConnectionRequest,
) (success bool, err error) {
	log := s.l.With(
		zap.String("operation", "TestConnection"),
		zap.String("provider_type", string(req.ProviderType)),
		zap.Any("req", req),
	)
	provider, err := s.providerRegistry.Get(req.ProviderType)
	if err != nil {
		log.Error("failed to get provider", zap.Error(err))
		return false, err
	}

	config := &providers.ProviderConfig{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		APIKey:   req.APIKey,
	}

	if err = provider.TestConnection(ctx, config); err != nil {
		log.Error("failed to test connection", zap.Error(err))
		return false, err
	}

	return true, nil
}
