package operationstools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (t *EmailCustomerTool) compose(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*outboundpreview.CustomerEmail, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := toolparams.RequirePulid(params.Params, changepreview.PreviewFieldShipmentID)
	if err != nil {
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

	tenant := toolparams.TenantFrom(*params)
	sp, err := t.Deps.Shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              shipmentID,
		TenantInfo:      tenant,
		ShipmentOptions: shipmentports.ShipmentOptions{IncludeCustomer: true},
	})
	if err != nil {
		return nil, err
	}

	composed := &outboundpreview.CustomerEmail{Shipment: sp, Tenant: tenant}

	// The customer update desk is woken by every arrival and every
	// departure, so a shipment crossing a yard can raise several runs in a
	// few minutes. Telling the customer once is the rule; reading back what
	// was already sent is what enforces it.
	composed.AlreadyTold, err = customerupdateservice.AlreadyTold(
		ctx, t.Deps.Comments, tenant, sp.ID, timeutils.NowUnix(),
	)
	if err != nil || composed.AlreadyTold {
		return composed, err
	}

	recipients, customerName, err := customerupdateservice.Recipients(
		ctx,
		t.Deps.Customers,
		sp,
		tenant,
	)
	if err != nil {
		return nil, err
	}
	composed.Recipients = recipients

	composed.Data = documenttemplate.AgentEmailContext{
		AgentSubject:      strings.TrimSpace(subject),
		AgentBody:         strings.TrimSpace(body),
		CustomerName:      customerName,
		ShipmentProNumber: sp.ProNumber,
	}
	customerupdateservice.Brand(ctx, t.Deps.orgRepo, t.Deps.inliner, tenant, &composed.Data)

	rendered, err := t.Deps.Templates.RenderMessage(
		ctx,
		&documenttemplateports.RenderMessageRequest{
			TenantInfo: tenant,
			Kind:       documenttemplate.KindAgentCustomerUpdateEmail,
			CustomerID: pulid.PtrOrNil(sp.CustomerID),
			Data:       composed.Data,
		},
	)
	if err != nil {
		return nil, err
	}
	composed.VersionID = rendered.VersionID

	composed.Send = &emailports.SendEmailRequest{
		TenantInfo:     tenant,
		ProfileID:      profileID,
		Purpose:        email.PurposeOperations,
		To:             recipients,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: params.IdempotencyKey,
	}

	return composed, nil
}

func (t *EmailCustomerTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return nil, err
	}

	if composed.AlreadyTold {
		preview := toolpreview.Build(fmt.Sprintf(
			"Would send nothing: the customer was already emailed about %s within the hour.",
			composed.Shipment.ProNumber,
		))
		toolpreview.Warn(preview, agent.PreviewWarningAlreadyToldCustomer,
			"This customer was already emailed about this shipment within the hour.",
			composed.Shipment.ProNumber)

		return preview, nil
	}

	sender, err := outboundpreview.ResolveEmailSender(ctx, t.Deps.Senders, composed.Send)
	if err != nil {
		return nil, err
	}

	send := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceCustomer,
			ID:       composed.Shipment.CustomerID,
			Label:    composed.Data.CustomerName,
		},
		outboundpreview.EmailMessagePreview(
			sender,
			outboundpreview.RenderedEmailPreview(composed.Send, composed.VersionID),
		),
	)

	comment := outboundpreview.CustomerUpdateComment(composed)
	recorded, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceShipmentComment,
			Label:    "Comment on " + composed.Shipment.ProNumber,
		},
		&shipment.ShipmentComment{
			Comment:    comment.Comment,
			Type:       comment.Type,
			Visibility: comment.Visibility,
		},
		toolpreview.Only("comment", "type", "visibility"),
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would email %s about %s, to %s, and record it on the shipment.",
			composed.Data.CustomerName,
			composed.Shipment.ProNumber,
			strings.Join(composed.Recipients, ", ")),
		send,
		recorded,
	)
	outboundpreview.WarnSuppressedRecipients(preview, sender)
	outboundpreview.WarnSensitiveContent(
		preview,
		composed.Data.AgentSubject,
		composed.Data.AgentBody,
	)

	return preview, nil
}
