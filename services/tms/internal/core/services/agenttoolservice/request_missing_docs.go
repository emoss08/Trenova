package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type requestMissingDocsTool struct {
	email     serviceports.EmailService
	templates serviceports.DocumentTemplateResolver
	orgRepo   repositories.OrganizationRepository
	inliner   serviceports.AssetInliner
}

type requestMissingDocsParams struct {
	fx.In

	Email     serviceports.EmailService
	Templates serviceports.DocumentTemplateResolver
	OrgRepo   repositories.OrganizationRepository
	Inliner   serviceports.AssetInliner
}

func newRequestMissingDocsTool(p requestMissingDocsParams) serviceports.AgentTool {
	return &requestMissingDocsTool{
		email:     p.Email,
		templates: p.Templates,
		orgRepo:   p.OrgRepo,
		inliner:   p.Inliner,
	}
}

func (t *requestMissingDocsTool) Name() string { return "request_missing_docs" }

func (t *requestMissingDocsTool) Description() string {
	return "Request missing documentation from a party by sending them an email."
}

func (t *requestMissingDocsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"profileId": map[string]any{
				"type":        "string",
				"description": "The email profile id to send from.",
			},
			"to": map[string]any{
				"type":        "array",
				"description": "The recipient email addresses.",
				"items":       map[string]any{"type": "string"},
			},
			"subject": map[string]any{
				"type":        "string",
				"description": "The subject line. The organization's template wraps it; do not add a company prefix.",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "The request in plain prose. The organization's template supplies the greeting, letterhead, and sign-off, so do not write them here.",
			},
			"customerName": map[string]any{
				"type":        "string",
				"description": "Who is being asked, so the template can address them by name.",
			},
			"shipmentProNumber": map[string]any{
				"type":        "string",
				"description": "The shipment the documents belong to, so the recipient can find it.",
			},
			"requestedDocuments": map[string]any{
				"type":        "array",
				"description": "The documents being asked for, one per entry. The template renders them as a list, so leave them out of the body prose.",
				"items":       map[string]any{"type": "string"},
			},
		},
		"required":             []string{"profileId", "to", "subject", "body"},
		"additionalProperties": false,
	}
}

func (t *requestMissingDocsTool) Reversible() bool { return false }

func (t *requestMissingDocsTool) PermissionResource() permission.Resource {
	return permission.ResourceBillingQueue
}

func (t *requestMissingDocsTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *requestMissingDocsTool) RequiresIdempotencyKey() bool { return true }

func (t *requestMissingDocsTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *requestMissingDocsTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	profileID, err := requirePulid(params.Params, "profileId")
	if err != nil {
		return err
	}

	subject, err := requireString(params.Params, "subject")
	if err != nil {
		return err
	}

	body, err := requireString(params.Params, "body")
	if err != nil {
		return err
	}

	var to []string
	if err = decodeParam(params.Params, "to", &to); err != nil {
		return err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: params.OrganizationID,
		BuID:  params.BusinessUnitID,
	}

	// The agent's prose is wrapped by the organization's template rather than
	// being the whole message. That inversion is the point: an email that goes out
	// under a carrier's name should look like the carrier wrote it, and an
	// organization should be able to add a signature or a payment-terms footer to
	// everything an agent sends without the agent knowing about it.
	//
	// It renders with no fallback. A template a carrier authored badly must not be
	// silently replaced on a message being sent under their letterhead — the tool
	// fails and the run surfaces it.
	rendered, err := t.templates.RenderMessage(ctx, &serviceports.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindAgentRequestMissingDocsEmail,
		Data:       t.agentEmailContext(ctx, tenantInfo, subject, body, params.Params),
	})
	if err != nil {
		return err
	}

	_, err = t.email.Send(ctx, &serviceports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		ProfileID:      profileID,
		Purpose:        email.PurposeBilling,
		To:             to,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: params.IdempotencyKey,
	})

	return err
}

// agentEmailContext assembles what the template wraps the agent's prose in.
//
// The optional parameters are read leniently: a malformed customer name must not
// stop a document request going out, because the agent's own words already say
// what is being asked for.
func (t *requestMissingDocsTool) agentEmailContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subject, body string,
	raw map[string]any,
) documenttemplate.AgentEmailContext {
	out := documenttemplate.AgentEmailContext{
		AgentSubject: subject,
		AgentBody:    body,
	}

	out.CustomerName = optionalString(raw, "customerName")
	out.ShipmentProNumber = optionalString(raw, "shipmentProNumber")

	var documents []string
	if err := decodeParam(raw, "requestedDocuments", &documents); err == nil {
		out.RequestedDocuments = documents
	}

	t.applyBranding(ctx, tenantInfo, &out)

	return out
}

// applyBranding names the sender and inlines their logo.
//
// Branding is decoration: an organization that cannot be read still gets its
// document request sent, unsigned, rather than not at all.
func (t *requestMissingDocsTool) applyBranding(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *documenttemplate.AgentEmailContext,
) {
	if t.orgRepo == nil {
		return
	}

	org, err := t.orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return
	}

	out.CompanyName = org.Name

	if dataURI, logoErr := serviceports.ResolveLogoDataURI(
		ctx, t.inliner, org.LogoURL,
	); logoErr == nil {
		out.LogoDataURI = dataURI
	}
}
