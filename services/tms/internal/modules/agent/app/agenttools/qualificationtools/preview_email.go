package qualificationtools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

// compose reads the call and renders the request as it would go out: to the
// addresses the call names, in the organization's letterhead, with no
// fallback when the template fails. The optional parameters are read
// leniently: a malformed name must not stop a document request going out,
// because the request's own words already say what is being asked for.
func (t *RequestMissingDocsTool) compose(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*requestemail.Composed, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
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
	var documents []string
	if dErr := toolparams.DecodeParam(params.Params, "requestedDocuments", &documents); dErr != nil {
		documents = nil
	}

	return t.requests.Compose(ctx, &requestemail.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
		Kind:               documenttemplate.KindAgentRequestMissingDocsEmail,
		ProfileID:          profileID,
		To:                 to,
		Subject:            subject,
		Body:               body,
		RecipientName:      toolparams.OptionalString(params.Params, "customerName"),
		ProNumber:          toolparams.OptionalString(params.Params, "shipmentProNumber"),
		RequestedDocuments: documents,
		IdempotencyKey:     params.IdempotencyKey,
	})
}

func (t *RequestMissingDocsTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	composed, err := t.compose(ctx, params)
	if err != nil {
		return nil, err
	}

	sender, err := t.requests.From(ctx, composed)
	if err != nil {
		return nil, err
	}

	label := stringutils.FirstNonEmpty(
		composed.Data.CustomerName,
		strings.Join(composed.Send.To, ", "),
	)
	send := toolpreview.Send(
		toolpreview.Record{Resource: permission.ResourceCustomerCommunication, Label: label},
		outboundpreview.EmailMessagePreview(
			sender,
			outboundpreview.RenderedEmailPreview(composed.Send, composed.VersionID),
		),
	)

	summary := "Would email " + strings.Join(composed.Send.To, ", ") +
		" asking for the missing paperwork"
	if len(composed.Data.RequestedDocuments) > 0 {
		summary += ": " + strings.Join(composed.Data.RequestedDocuments, ", ")
	}
	if composed.Data.ShipmentProNumber != "" {
		summary += " for " + composed.Data.ShipmentProNumber
	}

	preview := toolpreview.Build(summary+". It cannot be recalled once sent.", send)
	outboundpreview.WarnSuppressedRecipients(preview, sender)
	outboundpreview.WarnSensitiveContent(
		preview,
		composed.Data.AgentSubject,
		composed.Data.AgentBody,
	)

	return preview, nil
}
