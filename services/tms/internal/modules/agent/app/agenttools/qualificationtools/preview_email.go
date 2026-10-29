package qualificationtools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/portaltemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// docsRequest is a request for paperwork as it would go out: to the
// addresses the call names, in the organization's letterhead, with no
// fallback when the template fails.
type docsRequest struct {
	data      portaltemplate.AgentEmailContext
	send      *emailports.SendEmailRequest
	versionID *pulid.ID
}

func (t *RequestMissingDocsTool) compose(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*docsRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	profileID, err := toolparams.OptionalID(params.Params, "profileId")
	if err != nil {
		return nil, err
	}
	subject, err := toolparams.RequireString(params.Params, changepreview.PreviewFieldSubject)
	if err != nil {
		return nil, err
	}
	body, err := toolparams.RequireString(params.Params, "body")
	if err != nil {
		return nil, err
	}

	var to []string
	if err = toolparams.DecodeParam(params.Params, "to", &to); err != nil {
		return nil, err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: params.OrganizationID,
		BuID:  params.BusinessUnitID,
	}

	// The agent's prose is wrapped by the organization's template rather than
	// being the whole message, and it renders with no fallback: a template a
	// carrier authored badly must not be silently replaced on a message sent
	// under their letterhead.
	data := t.agentEmailContext(ctx, tenantInfo, subject, body, params.Params)
	rendered, err := t.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindAgentRequestMissingDocsEmail,
		Data:       data,
	})
	if err != nil {
		return nil, err
	}

	return &docsRequest{
		data:      data,
		versionID: rendered.VersionID,
		send: &emailports.SendEmailRequest{
			TenantInfo:     tenantInfo,
			ProfileID:      profileID,
			Purpose:        email.PurposeBilling,
			To:             to,
			Subject:        rendered.Subject,
			HTML:           rendered.HTML,
			Text:           rendered.Text,
			IdempotencyKey: params.IdempotencyKey,
		},
	}, nil
}

func (t *RequestMissingDocsTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return nil, err
	}

	sender, err := outboundpreview.ResolveEmailSender(ctx, t.senders, composed.send)
	if err != nil {
		return nil, err
	}

	label := stringutils.FirstNonEmpty(
		composed.data.CustomerName,
		strings.Join(composed.send.To, ", "),
	)
	send := toolpreview.Send(
		toolpreview.Record{Resource: permission.ResourceCustomerCommunication, Label: label},
		outboundpreview.EmailMessagePreview(
			sender,
			outboundpreview.RenderedEmailPreview(composed.send, composed.versionID),
		),
	)

	summary := "Would email " + strings.Join(composed.send.To, ", ") +
		" asking for the missing paperwork"
	if len(composed.data.RequestedDocuments) > 0 {
		summary += ": " + strings.Join(composed.data.RequestedDocuments, ", ")
	}
	if composed.data.ShipmentProNumber != "" {
		summary += " for " + composed.data.ShipmentProNumber
	}

	preview := toolpreview.Build(summary+". It cannot be recalled once sent.", send)
	outboundpreview.WarnSuppressedRecipients(preview, sender)
	outboundpreview.WarnSensitiveContent(
		preview,
		composed.data.AgentSubject,
		composed.data.AgentBody,
	)

	return preview, nil
}
