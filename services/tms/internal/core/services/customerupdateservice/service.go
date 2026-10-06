package customerupdateservice

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	MaxMessageLength = 2000
	metadataSentBy   = "sentBy"
)

type Params struct {
	fx.In

	Email         services.EmailService
	Templates     services.DocumentTemplateResolver
	Organizations repositories.OrganizationRepository
	Inliner       services.AssetInliner
	Customers     repositories.CustomerRepository
	Shipments     repositories.ShipmentRepository
	Comments      services.ShipmentCommentService
}

type Service struct {
	email         services.EmailService
	templates     services.DocumentTemplateResolver
	organizations repositories.OrganizationRepository
	inliner       services.AssetInliner
	customers     repositories.CustomerRepository
	shipments     repositories.ShipmentRepository
	comments      services.ShipmentCommentService
	now           func() int64
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		email:         p.Email,
		templates:     p.Templates,
		organizations: p.Organizations,
		inliner:       p.Inliner,
		customers:     p.Customers,
		shipments:     p.Shipments,
		comments:      p.Comments,
		now:           timeutils.NowUnix,
	}
}

type NotifyDelayRequest struct {
	TenantInfo     pagination.TenantInfo
	ShipmentID     pulid.ID
	Message        string
	IdempotencyKey string
}

func (r *NotifyDelayRequest) validate() error {
	multiErr := errortypes.NewMultiError()
	if r.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment is required")
	}
	message := strings.TrimSpace(r.Message)
	switch {
	case message == "":
		multiErr.Add("message", errortypes.ErrRequired, "Message is required")
	case utf8.RuneCountInString(message) > MaxMessageLength:
		multiErr.Add(
			"message",
			errortypes.ErrInvalid,
			fmt.Sprintf("Message must be at most %d characters", MaxMessageLength),
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) NotifyDelay(
	ctx context.Context,
	req *NotifyDelayRequest,
	actor *services.RequestActor,
) error {
	if err := req.validate(); err != nil {
		return err
	}

	sp, err := s.shipments.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              req.ShipmentID,
		TenantInfo:      req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{IncludeCustomer: true},
	})
	if err != nil {
		return err
	}

	told, err := AlreadyTold(ctx, s.comments, req.TenantInfo, sp.ID, s.now())
	if err != nil {
		return err
	}
	if told {
		return errortypes.NewBusinessError(ErrAlreadyTold.Error())
	}

	recipients, customerName, err := Recipients(ctx, s.customers, sp, req.TenantInfo)
	if err != nil {
		return errortypes.NewBusinessError(err.Error())
	}

	body := strings.TrimSpace(req.Message)
	data := documenttemplate.AgentEmailContext{
		AgentSubject:      "Delivery update for shipment " + sp.ProNumber,
		AgentBody:         body,
		CustomerName:      customerName,
		ShipmentProNumber: sp.ProNumber,
	}
	Brand(ctx, s.organizations, s.inliner, req.TenantInfo, &data)

	rendered, err := s.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo: req.TenantInfo,
		Kind:       documenttemplate.KindAgentCustomerUpdateEmail,
		CustomerID: pulid.PtrOrNil(sp.CustomerID),
		Data:       data,
	})
	if err != nil {
		return err
	}

	if _, err = s.email.Send(ctx, &services.SendEmailRequest{
		TenantInfo:     req.TenantInfo,
		Purpose:        email.PurposeOperations,
		To:             recipients,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: req.IdempotencyKey,
	}); err != nil {
		return err
	}

	sentBy := pulid.Nil
	if actor != nil {
		sentBy = actor.UserID
	}
	if _, err = s.comments.CreateSystem(ctx, Comment(&CommentParams{
		Tenant:     req.TenantInfo,
		ShipmentID: sp.ID,
		Recipients: recipients,
		Subject:    data.AgentSubject,
		Body:       body,
		Origin:     shipment.CommentOriginBoard,
		Source:     SourceDelayNotice,
		Extra:      map[string]any{metadataSentBy: sentBy.String()},
	})); err != nil {
		return fmt.Errorf("the email was sent but could not be recorded on the shipment: %w", err)
	}

	return nil
}
