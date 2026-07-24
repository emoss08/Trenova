package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

type requestMissingDocsTool struct {
	email serviceports.EmailService
}

func newRequestMissingDocsTool(emailSvc serviceports.EmailService) serviceports.AgentTool {
	return &requestMissingDocsTool{email: emailSvc}
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
			"subject": map[string]any{"type": "string"},
			"body":    map[string]any{"type": "string"},
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

	_, err = t.email.Send(ctx, &serviceports.SendEmailRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
		ProfileID:      profileID,
		Purpose:        email.PurposeBilling,
		To:             to,
		Subject:        subject,
		Text:           body,
		IdempotencyKey: params.IdempotencyKey,
	})

	return err
}
