package shipmenttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
)

const (
	toolShareCustomerTrackingLink  = "shipment_share_tracking_link"
	toolRevokeCustomerTrackingLink = "shipment_revoke_tracking_link"
	paramRecipientEmail            = "recipientEmail"
	paramTrackingLinkID            = "linkId"
	ToolShipmentList               = "shipment_list"
)

type customerTrackingLinker interface {
	IssueLink(
		ctx context.Context,
		req *customertrackingports.IssueLinkRequest,
	) (*customertrackingports.IssuedLink, error)
	PreviewIssueLink(
		ctx context.Context,
		req *customertrackingports.IssueLinkRequest,
	) (*customertrackingports.LinkPreview, error)
	RevokeLink(ctx context.Context, req *customertrackingports.RevokeLinkRequest) (int, error)
	PreviewRevokeLink(
		ctx context.Context,
		req *customertrackingports.RevokeLinkRequest,
	) ([]*customertracking.Link, error)
}

type sharedLinkView struct {
	Customer       string                   `json:"customer,omitempty"`
	RecipientEmail string                   `json:"recipientEmail,omitempty"`
	ExpiresAt      int64                    `json:"expiresAt"`
	Shows          []customertracking.Field `json:"shows"`
}

// classifyShareLink tells a link that is only shared apart from one emailed to
// an address outside the organization.
func classifyShareLink(params *agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy {
	if toolparams.OptionalString(params.Params, paramRecipientEmail) != "" {
		return agenttoolports.CallPolicy{Egress: agent.EgressExternalRecipient}
	}
	return agenttoolports.CallPolicy{Egress: agent.EgressCustomerVisible}
}

func shipmentTarget(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

func NewShareCustomerTrackingLinkTool(
	tracking customertrackingports.CustomerTrackingService,
) agenttoolports.AgentTool {
	return newShareCustomerTrackingLinkTool(tracking)
}

func newShareCustomerTrackingLinkTool(tracking customerTrackingLinker) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(
		&toolspec.ReceivableSpec{
			Name:   toolShareCustomerTrackingLink,
			Recipe: []string{ToolShipmentList, toolShareCustomerTrackingLink},
			Description: "Share a link to a shipment's tracking page with the customer, and email it " +
				"when given an address. The page shows only what the customer's tracking settings " +
				"allow and needs no sign-in. Use the address the customer asked you to send it to; " +
				"never invent one.",
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressCustomerVisible,
			AlsoEgress:  []agent.EgressClass{agent.EgressExternalRecipient},
			Classify:    classifyShareLink,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Gives the customer a link to the shipment's tracking page and can email it " +
				"outside the organization; the link can be revoked.",
			Target: shipmentTarget,
			Properties: map[string]any{
				toolparams.ParamShipmentID: agenttoolschema.RecordIDText(
					permission.ResourceShipment,
					"The shipment to share, from the page, shipment_list.",
				),
				paramRecipientEmail: toolparams.StringProperty(
					"Where to email the link. Leave it out to share the link without emailing it.",
					customertracking.MaxRecipientEmailLength,
				),
			},
			Required: []string{toolparams.ParamShipmentID},
		},
		toolspec.ReceivablePlan[*customertrackingports.IssueLinkRequest, *customertrackingports.LinkPreview]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*customertrackingports.IssueLinkRequest, error) {
				shipmentID, err := toolparams.RequirePulid(
					params.Params,
					toolparams.ParamShipmentID,
				)
				if err != nil {
					return nil, err
				}
				tenant := toolparams.TenantFrom(params)
				return &customertrackingports.IssueLinkRequest{
					TenantInfo:     tenant,
					ShipmentID:     shipmentID,
					RecipientEmail: toolparams.OptionalString(params.Params, paramRecipientEmail),
					IssuedByID:     tenant.UserID,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *customertrackingports.IssueLinkRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*customertrackingports.LinkPreview, error) {
				return tracking.PreviewIssueLink(ctx, req)
			},
			Refused: func(*customertrackingports.IssueLinkRequest) string {
				return "Would share a tracking link with the customer."
			},
			Render: renderSharedLink,
			Run: func(
				ctx context.Context,
				req *customertrackingports.IssueLinkRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				return shareLink(ctx, tracking, req)
			},
		},
	)
}

func renderSharedLink(
	_ *customertrackingports.IssueLinkRequest,
	plan *customertrackingports.LinkPreview,
) (*agentpreview.ToolPreview, error) {
	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.ShipmentID,
		Label:    plan.ShipmentProNumber,
	}, &sharedLinkView{
		Customer:       plan.CustomerName,
		RecipientEmail: plan.RecipientEmail,
		ExpiresAt:      plan.ExpiresAt,
		Shows:          plan.VisibleFields,
	}, toolpreview.Volatile("expiresAt"))
	if err != nil {
		return nil, err
	}
	summary := fmt.Sprintf("Share a tracking link for %s.", plan.ShipmentProNumber)
	if plan.RecipientEmail != "" {
		summary = fmt.Sprintf("Email %s a tracking link for %s.",
			plan.RecipientEmail, plan.ShipmentProNumber)
	}
	return toolpreview.Build(summary, created), nil
}

func shareLink(
	ctx context.Context,
	tracking customerTrackingLinker,
	req *customertrackingports.IssueLinkRequest,
) (*agentrecord.ToolExecutionResult, error) {
	issued, err := tracking.IssueLink(ctx, req)
	if err != nil {
		return nil, err
	}
	result := &agentrecord.ToolExecutionResult{
		Action: "shared",
		Kind:   "customer tracking link",
		Name:   issued.URL,
		IDs: map[string]string{
			toolparams.ParamShipmentID: req.ShipmentID.String(),
			paramTrackingLinkID:        issued.Link.ID.String(),
		},
		State: "Link shared; not emailed",
	}
	switch {
	case issued.Emailed:
		result.State = "Emailed to " + req.RecipientEmail
	case issued.EmailError != "":
		result.State = "The link works, but the email could not be sent: " + issued.EmailError
	}
	return result, nil
}

type revokedLinksView struct {
	LiveLinks int      `json:"liveLinks"`
	SentTo    []string `json:"sentTo,omitempty"`
}

func NewRevokeCustomerTrackingLinkTool(
	tracking customertrackingports.CustomerTrackingService,
) agenttoolports.AgentTool {
	return newRevokeCustomerTrackingLinkTool(tracking)
}

func newRevokeCustomerTrackingLinkTool(tracking customerTrackingLinker) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(
		&toolspec.ReceivableSpec{
			Name:   toolRevokeCustomerTrackingLink,
			Recipe: []string{ToolShipmentList, toolRevokeCustomerTrackingLink},
			Description: "Stop every live customer tracking link on a shipment from working. Use " +
				"it when a link reached the wrong person or the customer asks for access to end; " +
				"share a new link afterwards if someone should keep following the load.",
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressCustomerVisible,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Rationale: "Ends the customer's access to the shipment's tracking page; a new link " +
				"can be shared later.",
			Target: shipmentTarget,
			Properties: map[string]any{
				toolparams.ParamShipmentID: agenttoolschema.RecordIDText(
					permission.ResourceShipment,
					"The shipment whose links to revoke, from the page or "+
						"shipment_list.",
				),
			},
			Required: []string{toolparams.ParamShipmentID},
		},
		toolspec.ReceivablePlan[*customertrackingports.RevokeLinkRequest, []*customertracking.Link]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*customertrackingports.RevokeLinkRequest, error) {
				shipmentID, err := toolparams.RequirePulid(
					params.Params,
					toolparams.ParamShipmentID,
				)
				if err != nil {
					return nil, err
				}
				tenant := toolparams.TenantFrom(params)
				return &customertrackingports.RevokeLinkRequest{
					TenantInfo:  tenant,
					ShipmentID:  shipmentID,
					RevokedByID: tenant.UserID,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *customertrackingports.RevokeLinkRequest,
				_ *agenttoolports.ToolExecuteParams,
			) ([]*customertracking.Link, error) {
				return tracking.PreviewRevokeLink(ctx, req)
			},
			Refused: func(*customertrackingports.RevokeLinkRequest) string {
				return "Would revoke the shipment's customer tracking links."
			},
			Render: func(
				req *customertrackingports.RevokeLinkRequest,
				links []*customertracking.Link,
			) (*agentpreview.ToolPreview, error) {
				sentTo := make([]string, 0, len(links))
				for _, link := range links {
					if link.RecipientEmail != "" {
						sentTo = append(sentTo, link.RecipientEmail)
					}
				}
				changed, err := toolpreview.Changed(toolpreview.Record{
					Resource: permission.ResourceShipment,
					ID:       req.ShipmentID,
				}, &revokedLinksView{LiveLinks: len(links), SentTo: sentTo},
					&revokedLinksView{LiveLinks: 0})
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(
					fmt.Sprintf("Revoke %d customer tracking link(s).", len(links)),
					changed,
				), nil
			},
			Run: func(
				ctx context.Context,
				req *customertrackingports.RevokeLinkRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				revoked, err := tracking.RevokeLink(ctx, req)
				if err != nil {
					return nil, err
				}
				return &agentrecord.ToolExecutionResult{
					Action: "revoked",
					Kind:   "customer tracking link",
					Name:   fmt.Sprintf("%d link(s)", revoked),
					IDs:    map[string]string{toolparams.ParamShipmentID: req.ShipmentID.String()},
				}, nil
			},
		},
	)
}
