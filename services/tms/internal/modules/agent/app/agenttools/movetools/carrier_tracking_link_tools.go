package movetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	toolIssueCarrierTrackingLink  = "issue_carrier_tracking_link"
	toolRevokeCarrierTrackingLink = "revoke_carrier_tracking_link"
	toolShipmentGet               = "shipment_get"
	toolShipmentList              = "shipment_list"
	toolGetCarrier                = "get_carrier"
	paramRecipientEmail           = "recipientEmail"
	fieldExpiresAt                = "expiresAt"
	fieldRevokedAt                = "revokedAt"
)

type carrierTrackingLinker interface {
	IssueLink(
		ctx context.Context,
		req *checkcallports.IssueTrackingLinkRequest,
	) (*checkcallports.IssuedTrackingLink, error)
	PreviewIssueLink(
		ctx context.Context,
		req *checkcallports.IssueTrackingLinkRequest,
	) (*checkcallports.TrackingLinkPreview, error)
	RevokeLink(ctx context.Context, req *checkcallports.RevokeTrackingLinkRequest) error
	PreviewRevokeLink(
		ctx context.Context,
		req *checkcallports.RevokeTrackingLinkRequest,
	) (*checkcall.TrackingLink, error)
}

type issuedLinkView struct {
	CarrierID      pulid.ID `json:"carrierId"`
	RecipientEmail string   `json:"recipientEmail"`
	ExpiresAt      int64    `json:"expiresAt"`
}

type liveLinkView struct {
	RecipientEmail string `json:"recipientEmail,omitempty"`
	ExpiresAt      int64  `json:"expiresAt"`
	LastUsedAt     *int64 `json:"lastUsedAt,omitempty"`
	RevokedAt      *int64 `json:"revokedAt,omitempty"`
}

var trackingLinkLabels = map[string]string{
	"carrierId":         "Carrier",
	paramRecipientEmail: "Sent to",
	fieldExpiresAt:      "Expires",
	"lastUsedAt":        "Last used",
	fieldRevokedAt:      "Revoked",
}

var trackingLinkTypes = map[string]assistantartifact.DisplayType{
	fieldExpiresAt: assistantartifact.DisplayDateTime,
	"lastUsedAt":   assistantartifact.DisplayDateTime,
	fieldRevokedAt: assistantartifact.DisplayDateTime,
}

func liveLinkOf(link *checkcall.TrackingLink) *liveLinkView {
	return &liveLinkView{
		RecipientEmail: link.RecipientEmail,
		ExpiresAt:      link.ExpiresAt,
		LastUsedAt:     link.LastUsedAt,
		RevokedAt:      link.RevokedAt,
	}
}

func moveTarget(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, ParamMoveID, permission.ResourceShipmentMove)
}

func moveProperty(what string) map[string]any {
	return agenttoolschema.RecordIDText(permission.ResourceShipmentMove,
		what+", from shipment_get (its moves). Only a move covered by an outside carrier "+
			"has a carrier tracking link.")
}

func revokedLinkChange(
	moveID pulid.ID,
	label string,
	live *checkcall.TrackingLink,
) (*agent.RecordChange, error) {
	after := *live
	revokedAt := int64(0)
	after.RevokedAt = &revokedAt

	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		ID:       moveID,
		Label:    label,
	}, liveLinkOf(live), liveLinkOf(&after),
		toolpreview.Labels(trackingLinkLabels),
		toolpreview.Types(trackingLinkTypes),
		toolpreview.Volatile(fieldRevokedAt),
	)
}

func ProvideIssueCarrierTrackingLinkTool(
	checkCalls checkcallports.CheckCallService,
) agentports.AgentTool {
	return newIssueCarrierTrackingLinkTool(checkCalls)
}

func newIssueCarrierTrackingLinkTool(links carrierTrackingLinker) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name: toolIssueCarrierTrackingLink,
			Recipe: []string{
				toolShipmentList,
				toolShipmentGet,
				toolGetCarrier,
				toolIssueCarrierTrackingLink,
			},
			Prerequisites: []string{toolShipmentGet},
			Description: "Email the carrier covering a brokered move a tracking link, so their " +
				"dispatcher or driver can post check calls, arrivals and departures without " +
				"signing in. Issuing replaces any live link on the move, which stops working. " +
				"Send it to the address the carrier gave you or the email get_carrier shows for " +
				"the carrier or its dispatch contact; never invent one. The link itself is never " +
				"shown to you, so it cannot be pasted anywhere else.",
			Resource:    permission.ResourceShipmentMove,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressExternalRecipient,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Emails a link outside the organization that lets the carrier post to the " +
				"load, and ends any link the carrier holds now; revoke_carrier_tracking_link " +
				"ends it.",
			Target:   moveTarget,
			Artifact: common.ShipmentRecordEntity,
			Properties: map[string]any{
				ParamMoveID: moveProperty("The move to track"),
				paramRecipientEmail: common.StringProperty(
					"Where to email the link: the carrier's dispatch address they gave or "+
						"get_carrier shows.",
					checkcall.MaxRecipientLength,
				),
			},
			Required: []string{ParamMoveID, paramRecipientEmail},
			SearchTerms: []string{
				"carrier tracking link",
				"send tracking link",
				"carrier check calls",
			},
		},
		common.ReceivablePlan[
			*checkcallports.IssueTrackingLinkRequest,
			*checkcallports.TrackingLinkPreview,
		]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*checkcallports.IssueTrackingLinkRequest, error) {
				moveID, err := common.RequirePulid(params.Params, ParamMoveID)
				if err != nil {
					return nil, err
				}
				recipient, err := common.RequireBoundedText(
					params.Params,
					paramRecipientEmail,
					checkcall.MaxRecipientLength,
				)
				if err != nil {
					return nil, err
				}
				tenant := common.TenantFrom(*params)
				return &checkcallports.IssueTrackingLinkRequest{
					TenantInfo:     tenant,
					MoveID:         moveID,
					RecipientEmail: recipient,
					IssuedByID:     tenant.UserID,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *checkcallports.IssueTrackingLinkRequest,
				_ *agentports.ToolExecuteParams,
			) (*checkcallports.TrackingLinkPreview, error) {
				return links.PreviewIssueLink(ctx, req)
			},
			Refused: func(*checkcallports.IssueTrackingLinkRequest) string {
				return "Would email the carrier a tracking link."
			},
			Render: renderIssuedLink,
			Run: func(
				ctx context.Context,
				req *checkcallports.IssueTrackingLinkRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				issued, err := links.IssueLink(ctx, req)
				if err != nil {
					return nil, err
				}
				result := &agent.ToolExecutionResult{
					Action: "issued",
					Kind:   "carrier tracking link",
					Name:   "Tracking link for " + req.RecipientEmail,
					IDs: map[string]string{
						ParamMoveID:      req.MoveID.String(),
						"trackingLinkId": issued.Link.ID.String(),
					},
					Record: &agent.RecordRef{
						EntityType: common.ShipmentRecordEntity,
						ID:         issued.Link.ShipmentID.String(),
					},
					State: "Emailed to " + req.RecipientEmail,
				}
				if !issued.Emailed {
					result.State = "The link was issued, but the email could not be sent: " +
						issued.EmailError + ". A dispatcher can issue it again from the move to " +
						"copy it."
				}
				return result, nil
			},
		},
	)
}

func renderIssuedLink(
	req *checkcallports.IssueTrackingLinkRequest,
	plan *checkcallports.TrackingLinkPreview,
) (*agent.ToolPreview, error) {
	label := "Carrier tracking link"
	if plan.ShipmentProNumber != "" {
		label += " for " + plan.ShipmentProNumber
	}
	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		ID:       req.MoveID,
		Label:    label,
	}, &issuedLinkView{
		CarrierID:      plan.CarrierID,
		RecipientEmail: plan.RecipientEmail,
		ExpiresAt:      plan.ExpiresAt,
	},
		toolpreview.Labels(trackingLinkLabels),
		toolpreview.Types(trackingLinkTypes),
		toolpreview.WithRefs(map[string]permission.Resource{
			"carrierId": permission.ResourceCarrier,
		}),
		toolpreview.Volatile(fieldExpiresAt),
	)
	if err != nil {
		return nil, err
	}
	if plan.Email != nil {
		created.Message = common.PlannedEmailPreview(plan.Email)
	}
	changes := []*agent.RecordChange{created}

	carrier := plan.CarrierName
	if carrier == "" {
		carrier = "the carrier"
	}
	summary := fmt.Sprintf("Would email %s a link to post check calls on %s, valid for 21 days.",
		plan.RecipientEmail, carrierLoad(carrier, plan.ShipmentProNumber))
	if plan.Replaces != nil {
		replaced, replaceErr := revokedLinkChange(
			req.MoveID,
			"The live tracking link",
			plan.Replaces,
		)
		if replaceErr != nil {
			return nil, replaceErr
		}
		changes = append(changes, replaced)
		summary += " The link issued before stops working."
	}

	return toolpreview.Build(summary, changes...), nil
}

func carrierLoad(carrier, proNumber string) string {
	if proNumber == "" {
		return carrier + "'s load"
	}

	return carrier + "'s load " + proNumber
}

func ProvideRevokeCarrierTrackingLinkTool(
	checkCalls checkcallports.CheckCallService,
) agentports.AgentTool {
	return newRevokeCarrierTrackingLinkTool(checkCalls)
}

func newRevokeCarrierTrackingLinkTool(links carrierTrackingLinker) agentports.AgentTool {
	return common.NewReceivableTool(
		&common.ReceivableSpec{
			Name: toolRevokeCarrierTrackingLink,
			Recipe: []string{
				toolShipmentList,
				toolShipmentGet,
				toolRevokeCarrierTrackingLink,
			},
			Prerequisites: []string{toolShipmentGet},
			Description: "Stop the live carrier tracking link on a move from working, so nobody " +
				"can post check calls through it. Use it when the link reached the wrong person, " +
				"the carrier changed, or the person asks for it to end; issue a new one with " +
				"issue_carrier_tracking_link if the carrier should keep posting.",
			Resource:    permission.ResourceShipmentMove,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressExternalRecipient,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Rationale: "Ends a link someone outside the organization holds, so the carrier's " +
				"page stops working; a new link can be issued later.",
			Target: moveTarget,
			Properties: map[string]any{
				ParamMoveID: moveProperty("The move whose link to revoke"),
			},
			Required:    []string{ParamMoveID},
			SearchTerms: []string{"revoke tracking link", "stop carrier link"},
		},
		common.ReceivablePlan[*checkcallports.RevokeTrackingLinkRequest, *checkcall.TrackingLink]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*checkcallports.RevokeTrackingLinkRequest, error) {
				moveID, err := common.RequirePulid(params.Params, ParamMoveID)
				if err != nil {
					return nil, err
				}
				return &checkcallports.RevokeTrackingLinkRequest{
					TenantInfo: common.TenantFrom(*params),
					MoveID:     moveID,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *checkcallports.RevokeTrackingLinkRequest,
				_ *agentports.ToolExecuteParams,
			) (*checkcall.TrackingLink, error) {
				live, err := links.PreviewRevokeLink(ctx, req)
				if errortypes.IsNotFoundError(err) {
					return nil, errortypes.NewBusinessError(
						"This move has no live carrier tracking link to revoke",
					)
				}
				return live, err
			},
			Refused: func(*checkcallports.RevokeTrackingLinkRequest) string {
				return "Would revoke the move's carrier tracking link."
			},
			Render: func(
				req *checkcallports.RevokeTrackingLinkRequest,
				live *checkcall.TrackingLink,
			) (*agent.ToolPreview, error) {
				changed, err := revokedLinkChange(req.MoveID, "Carrier tracking link", live)
				if err != nil {
					return nil, err
				}
				summary := "Would revoke the move's carrier tracking link."
				if live.RecipientEmail != "" {
					summary = fmt.Sprintf("Would revoke the carrier tracking link sent to %s.",
						live.RecipientEmail)
				}
				return toolpreview.Build(summary, changed), nil
			},
			Run: func(
				ctx context.Context,
				req *checkcallports.RevokeTrackingLinkRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				if err := links.RevokeLink(ctx, req); err != nil {
					return nil, err
				}
				return &agent.ToolExecutionResult{
					Action: "revoked",
					Kind:   "carrier tracking link",
					IDs:    map[string]string{ParamMoveID: req.MoveID.String()},
				}, nil
			},
		},
	)
}
