package qualificationtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// RequestMissingDocsTool asks an outside party for paperwork by email. The
// email is composed and sent by requestemail.Sender, the same sender a
// person's own request from the Watchtower uses.
type RequestMissingDocsTool struct {
	requests *requestemail.Sender
}

func NewRequestMissingDocsTool(requests *requestemail.Sender) agenttoolports.AgentTool {
	return &RequestMissingDocsTool{requests: requests}
}

func (t *RequestMissingDocsTool) Name() string { return "shipment_request_documents" }

func (t *RequestMissingDocsTool) Description() string {
	return "Request missing documentation from a party by sending them an email. Use it " +
		"when a POD, BOL or other paper a shipment needs has not arrived. Call " +
		"inbox_list_email_profiles first for profileId; the organization's template adds the " +
		"greeting, letterhead and sign-off. The email is sent and cannot be recalled."
}

func (t *RequestMissingDocsTool) Prerequisites() []string {
	return []string{"inbox_list_email_profiles"}
}

func (t *RequestMissingDocsTool) SearchTerms() []string {
	return []string{"missing pod", "missing proof of delivery", "ask for paperwork", "missing bol"}
}

func (t *RequestMissingDocsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"profileId": agenttoolschema.RecordIDText(
				permission.ResourceEmailProfile,
				"The email profile to send from, from inbox_list_email_profiles; pick "+
					"billing for a document request when there is one.",
			),
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

func (t *RequestMissingDocsTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceCustomerCommunication,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Emails an outside party a request for paperwork in words the model wrote.",
	}
}

func (t *RequestMissingDocsTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	composed, err := t.compose(ctx, params)
	if err != nil {
		return err
	}

	return t.requests.Send(ctx, composed)
}
