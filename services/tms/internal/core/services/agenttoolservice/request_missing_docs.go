package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
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
	senders   serviceports.EmailSenderResolver
}

type requestMissingDocsParams struct {
	fx.In

	Email     serviceports.EmailService
	Templates serviceports.DocumentTemplateResolver
	OrgRepo   repositories.OrganizationRepository
	Inliner   serviceports.AssetInliner
	Senders   serviceports.EmailSenderResolver `optional:"true"`
}

func newRequestMissingDocsTool(p requestMissingDocsParams) serviceports.AgentTool {
	return &requestMissingDocsTool{
		email:     p.Email,
		templates: p.Templates,
		orgRepo:   p.OrgRepo,
		inliner:   p.Inliner,
		senders:   p.Senders,
	}
}

func (t *requestMissingDocsTool) Name() string { return "request_missing_docs" }

func (t *requestMissingDocsTool) Description() string {
	return "Request missing documentation from a party by sending them an email. Use it " +
		"when a POD, BOL or other paper a shipment needs has not arrived. Call " +
		"list_email_profiles first for profileId; the organization's template adds the " +
		"greeting, letterhead and sign-off. The email is sent and cannot be recalled."
}

func (t *requestMissingDocsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"profileId": map[string]any{
				"type": "string",
				"description": "The email profile to send from, from list_email_profiles; pick " +
					"billing for a document request when there is one.",
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

func (t *requestMissingDocsTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceCustomerCommunication,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Emails an outside party a request for paperwork in words the model wrote.",
	}
}

func (t *requestMissingDocsTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.email.Send(ctx, composed.send)

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
func (t *requestMissingDocsTool) applyBranding(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *documenttemplate.AgentEmailContext,
) {
	brandAgentEmail(ctx, t.orgRepo, t.inliner, tenantInfo, out)
}

// brandAgentEmail names the sender and inlines their logo on any email an
// agent composes.
//
// Branding is decoration: an organization that cannot be read still gets its
// message sent, unsigned, rather than not at all.
func brandAgentEmail(
	ctx context.Context,
	orgRepo repositories.OrganizationRepository,
	inliner serviceports.AssetInliner,
	tenantInfo pagination.TenantInfo,
	out *documenttemplate.AgentEmailContext,
) {
	if orgRepo == nil {
		return
	}

	org, err := orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return
	}

	out.CompanyName = org.Name

	if dataURI, logoErr := serviceports.ResolveLogoDataURI(ctx, inliner, org.LogoURL); logoErr == nil {
		out.LogoDataURI = dataURI
	}
}
