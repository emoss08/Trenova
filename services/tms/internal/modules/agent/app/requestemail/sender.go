package requestemail

import (
	"context"
	"strings"

	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/portaltemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Params struct {
	fx.In

	Email     emailports.EmailService
	Templates documenttemplateports.DocumentTemplateResolver
	OrgRepo   tenantports.OrganizationRepository
	Inliner   documenttemplateports.AssetInliner
	Senders   emailports.EmailSenderResolver
}

// Sender sends an outside party a request the organization composed, for
// paperwork or for a load's status, wrapped in the organization's own
// template for the kind. It renders with no fallback: a template the
// organization authored badly must not be silently replaced on a message
// sent under their letterhead. An agent's tool and a person's own request
// send through it, so both go out looking the same.
type Sender struct {
	email     emailports.EmailService
	templates documenttemplateports.DocumentTemplateResolver
	orgRepo   tenantports.OrganizationRepository
	inliner   documenttemplateports.AssetInliner
	senders   emailports.EmailSenderResolver
}

func New(p Params) *Sender { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Sender{
		email:     p.Email,
		templates: p.Templates,
		orgRepo:   p.OrgRepo,
		inliner:   p.Inliner,
		senders:   p.Senders,
	}
}

// Request is one request email before it is rendered.
type Request struct {
	TenantInfo pagination.TenantInfo
	// Kind is the template it is wrapped in: the missing documents request
	// or the load status request.
	Kind documenttemplate.Kind
	// ProfileID is the email profile to send from; nil lets the mailer pick
	// the billing profile, then the general one.
	ProfileID pulid.ID
	To        []string
	Subject   string
	Body      string
	// RecipientName is who the greeting addresses.
	RecipientName string
	ProNumber     string
	// RequestedDocuments lists what a documents request asks for.
	RequestedDocuments []string
	IdempotencyKey     string
}

// Composed is a request as it would go out.
type Composed struct {
	Data      portaltemplate.AgentEmailContext
	Send      *emailports.SendEmailRequest
	VersionID *pulid.ID
}

// Compose renders the request in the organization's template.
func (s *Sender) Compose(ctx context.Context, req *Request) (*Composed, error) {
	data := portaltemplate.AgentEmailContext{
		AgentSubject:       req.Subject,
		AgentBody:          req.Body,
		CustomerName:       strings.TrimSpace(req.RecipientName),
		ShipmentProNumber:  strings.TrimSpace(req.ProNumber),
		RequestedDocuments: req.RequestedDocuments,
	}
	customerupdateservice.Brand(ctx, s.orgRepo, s.inliner, req.TenantInfo, &data)

	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		Data:       data,
	})
	if err != nil {
		return nil, err
	}

	return &Composed{
		Data:      data,
		VersionID: rendered.VersionID,
		Send: &emailports.SendEmailRequest{
			TenantInfo:     req.TenantInfo,
			ProfileID:      req.ProfileID,
			Purpose:        email.PurposeBilling,
			To:             req.To,
			Subject:        rendered.Subject,
			HTML:           rendered.HTML,
			Text:           rendered.Text,
			IdempotencyKey: req.IdempotencyKey,
		},
	}, nil
}

// From resolves the profile the request would be sent from and which of its
// recipients would be refused. It fails, in words for a person, when the
// organization has no email profile to send from.
func (s *Sender) From(ctx context.Context, composed *Composed) (*emailports.EmailSender, error) {
	if s.senders == nil {
		return nil, nil //nolint:nilnil // a deployment without a resolver names no sender
	}

	return s.senders.ResolveSender(ctx, composed.Send)
}

// CanSend reports whether the organization has an email profile a request
// can go out from.
func (s *Sender) CanSend(ctx context.Context, tenantInfo pagination.TenantInfo) bool {
	if s.senders == nil {
		return false
	}
	_, err := s.senders.ResolveSender(ctx, &emailports.SendEmailRequest{
		TenantInfo: tenantInfo,
		Purpose:    email.PurposeBilling,
	})

	return err == nil
}

// Send sends a composed request.
func (s *Sender) Send(ctx context.Context, composed *Composed) error {
	_, err := s.email.Send(ctx, composed.Send)

	return err
}
