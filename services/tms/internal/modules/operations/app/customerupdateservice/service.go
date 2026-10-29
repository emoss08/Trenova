package customerupdateservice

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/portaltemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
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

	Email         emailports.EmailService
	Templates     documenttemplateports.DocumentTemplateResolver
	Organizations tenantports.OrganizationRepository
	Inliner       documenttemplateports.AssetInliner
	Customers     customerports.CustomerRepository
	Shipments     shipmentports.ShipmentRepository
	Comments      shipmentcommentports.ShipmentCommentService
}

type Service struct {
	email         emailports.EmailService
	templates     documenttemplateports.DocumentTemplateResolver
	organizations tenantports.OrganizationRepository
	inliner       documenttemplateports.AssetInliner
	customers     customerports.CustomerRepository
	shipments     shipmentports.ShipmentRepository
	comments      shipmentcommentports.ShipmentCommentService
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
	actor *permissionports.RequestActor,
) error {
	if err := req.validate(); err != nil {
		return err
	}

	sp, err := s.shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              req.ShipmentID,
		TenantInfo:      req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{IncludeCustomer: true},
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
	data := portaltemplate.AgentEmailContext{
		AgentSubject:      "Delivery update for shipment " + sp.ProNumber,
		AgentBody:         body,
		CustomerName:      customerName,
		ShipmentProNumber: sp.ProNumber,
	}
	Brand(ctx, s.organizations, s.inliner, req.TenantInfo, &data)

	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: req.TenantInfo,
		Kind:       documenttemplate.KindAgentCustomerUpdateEmail,
		CustomerID: pulid.PtrOrNil(sp.CustomerID),
		Data:       data,
	})
	if err != nil {
		return err
	}

	if _, err = s.email.Send(ctx, &emailports.SendEmailRequest{
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
