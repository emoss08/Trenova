package billingissuetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingqueuepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueuereviewservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolReleaseBillingQueueItem  = "billing_release_queue_item"
	toolResolveBillingQueueIssue = "billing_resolve_issue"
	toolUndoBillingQueueIssue    = "billing_undo_issue"
	toolListBillingQueueItems    = "billing_list_queue_items"
	paramIssueID                 = "issueId"
	paramOptionKey               = "optionKey"
	checkStateOpen               = "Open"
	checkStateSettled            = "Settled"
	maxOptionKeyChars            = 30
)

type queueItemReader interface {
	GetByID(
		ctx context.Context,
		req *billingqueueports.GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
}

type queueReviewer interface {
	Release(
		ctx context.Context,
		req *billingqueueports.BillingQueueItemRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	ResolveIssue(
		ctx context.Context,
		req *billingqueueports.ResolveBillingQueueIssueRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	UndoIssue(
		ctx context.Context,
		req *billingqueueports.UndoBillingQueueIssueRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	PlanResolveIssue(
		ctx context.Context,
		req *billingqueueports.ResolveBillingQueueIssueRequest,
	) (*billingqueueports.IssueSettlementPlan, error)
	PlanUndoIssue(
		ctx context.Context,
		req *billingqueueports.UndoBillingQueueIssueRequest,
	) (*billingqueueports.IssueSettlementPlan, error)
}

func queueItemProperty() map[string]any {
	return agenttoolschema.OfKinds(map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The billing queue item, from billing_list_queue_items, " +
			"billing_get_queue_item or this run's subject. Never guess one.",
	}, permission.KindBillingQueueItem)
}

func issueProperty() map[string]any {
	return toolparams.StringProperty("The check, by the id billing_get_queue_item lists under the "+
		"item's issues. Never guess one.", 0)
}

func queueTarget(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		billingqueuepreview.ParamBillingQueueItemID,
		permission.ResourceBillingQueue,
	)
}

type releasePlan struct {
	before *billingqueue.BillingQueueItem
	after  *billingqueue.BillingQueueItem
}

func NewReleaseBillingQueueItemTool(
	items billingqueuepreview.BillingQueueDecider,
	review billingqueueports.BillingQueueReviewService,
) agenttoolports.AgentTool {
	return newReleaseBillingQueueItemTool(items, review)
}

func newReleaseBillingQueueItemTool(
	items queueItemReader,
	review queueReviewer,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name: toolReleaseBillingQueueItem,
			Recipe: []string{
				toolListBillingQueueItems,
				billingqueuepreview.ToolGetBillingQueueItem,
				toolReleaseBillingQueueItem,
			},
			Description: "Take a billing queue item off hold and back to the status the hold " +
				"found it in, the counterpart of billing_hold_queue_item. A hold was put on for " +
				"a reason: read the item's review notes with billing_get_queue_item and release " +
				"it only when the record shows that what it waited on has arrived, never because " +
				"nothing looks wrong.",
			Resource:    permission.ResourceBillingQueue,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			TaintHold: "What the hold waited on is what outside text would claim has arrived, " +
				"so a run that has read it proposes the release rather than making it.",
			Rationale: "Moves an item back into the review queue inside Trenova; it creates no " +
				"money and billing_hold_queue_item puts it back on hold.",
			Target:   queueTarget,
			Artifact: billingqueuepreview.BillingQueueRecordEntity,
			Properties: map[string]any{
				billingqueuepreview.ParamBillingQueueItemID: queueItemProperty(),
			},
			Required:    []string{billingqueuepreview.ParamBillingQueueItemID},
			SearchTerms: []string{"release hold", "take off hold", "unhold billing"},
		},
		toolspec.ReceivablePlan[*billingqueueports.BillingQueueItemRequest, *releasePlan]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*billingqueueports.BillingQueueItemRequest, error) {
				itemID, err := toolparams.RequirePulid(
					params.Params,
					billingqueuepreview.ParamBillingQueueItemID,
				)
				if err != nil {
					return nil, err
				}
				return &billingqueueports.BillingQueueItemRequest{
					ItemID:     itemID,
					TenantInfo: toolparams.TenantFrom(*params),
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *billingqueueports.BillingQueueItemRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*releasePlan, error) {
				return planRelease(ctx, items, req, params.Actor)
			},
			Refused: func(*billingqueueports.BillingQueueItemRequest) string {
				return "Would take the billing queue item off hold."
			},
			Render: func(
				_ *billingqueueports.BillingQueueItemRequest,
				plan *releasePlan,
			) (*agentpreview.ToolPreview, error) {
				changed, err := toolpreview.Changed(
					billingqueuepreview.QueueRecord(plan.before),
					plan.before,
					plan.after,
					toolpreview.Only(
						changepreview.FieldStatus,
						"holdReasonCode",
						billingqueuepreview.FieldReviewStartedAt,
					),
					toolpreview.Volatile(billingqueuepreview.DecisionVolatile...),
					toolpreview.WithRefs(billingqueuepreview.DecisionRefs),
				)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would take billing queue item %s off hold and back to %s.",
					plan.before.Number,
					plan.after.Status,
				), changed), nil
			},
			Run: func(
				ctx context.Context,
				req *billingqueueports.BillingQueueItemRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				released, err := review.Release(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return queueItemResult("released", released), nil
			},
		},
	)
}

func planRelease(
	ctx context.Context,
	items queueItemReader,
	req *billingqueueports.BillingQueueItemRequest,
	actor *permissionports.RequestActor,
) (*releasePlan, error) {
	item, err := items.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	target, err := billingqueuereviewservice.ReleaseTarget(item)
	if err != nil {
		return nil, err
	}

	after := *item
	statusReq := &billingqueueports.UpdateBillingQueueStatusRequest{
		ItemID:     req.ItemID,
		NewStatus:  target,
		TenantInfo: req.TenantInfo,
	}
	plan := billingqueueservice.PlanStatusChange
	if actor.IsAgent() && !billingqueueservice.AgentMayMoveTo(target) {
		plan = billingqueueservice.PlanTransition
	}
	if err = plan(&after, statusReq, actor, timeutils.NowUnix()); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	billingqueueservice.CheckStatusFields(&after, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &releasePlan{before: item, after: &after}, nil
}

func queueItemResult(
	action string,
	item *billingqueue.BillingQueueItem,
) *agentrecord.ToolExecutionResult {
	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   "billing queue item",
		Name:   item.Number,
		IDs:    map[string]string{billingqueuepreview.ParamBillingQueueItemID: item.ID.String()},
		Record: &agentrecord.RecordRef{
			EntityType: billingqueuepreview.BillingQueueRecordEntity,
			ID:         item.ID.String(),
		},
		State: string(item.Status),
	}
}

type queueCheckView struct {
	Check      string `json:"check"`
	State      string `json:"state"`
	Resolution string `json:"resolution,omitempty"`
}

type queueChargeView struct {
	Method string `json:"method"`
	Amount string `json:"amount"`
	Unit   int16  `json:"unit"`
}

var queueCheckLabels = map[string]string{
	"check":      "Check",
	"state":      "State",
	"resolution": "Settled as",
	"method":     "Method",
	"amount":     "Amount",
	"unit":       "Units",
}

func checkViewOf(issue *billingqueue.Issue) *queueCheckView {
	view := &queueCheckView{Check: issue.Summary, State: checkStateOpen}
	if issue.Settled() {
		view.State = checkStateSettled
		view.Resolution = issue.ResolutionText
	}

	return view
}

func chargeViewOf(charge *shipment.AdditionalCharge) *queueChargeView {
	return &queueChargeView{
		Method: string(charge.Method),
		Amount: charge.Amount.StringFixed(2),
		Unit:   charge.Unit,
	}
}

func chargeLabel(item *billingqueue.BillingQueueItem, chargeID pulid.ID) string {
	if item.Review != nil && item.Review.Charges != nil && chargeID.IsNotNil() {
		for _, line := range item.Review.Charges.Lines {
			if line != nil && line.AdditionalChargeID == chargeID && line.Label != "" {
				return line.Label
			}
		}
	}

	return "Accessorial charge"
}

func checkRecord(plan *billingqueueports.IssueSettlementPlan) toolpreview.Record {
	record := billingqueuepreview.QueueRecord(plan.Item)
	record.Label = plan.Item.Number + " · " + plan.Issue.Summary

	return record
}

func chargeChange(plan *billingqueueports.IssueSettlementPlan) (*agentpreview.RecordChange, error) {
	before, after := plan.ChargeBefore, plan.ChargeAfter
	if before == nil && after == nil {
		return nil, nil //nolint:nilnil // the choice changes no charge
	}

	chargeID := pulid.Nil
	if before != nil {
		chargeID = before.ID
	}
	record := toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.Item.ShipmentID,
		Label: chargeLabel(
			plan.Item,
			chargeID,
		) + " on " + billingqueuepreview.ShipmentLabel(
			plan.Item,
		),
	}
	opts := []toolpreview.Option{toolpreview.Labels(queueCheckLabels)}
	switch {
	case after == nil:
		return toolpreview.Delete(record, chargeViewOf(before), opts...)
	case before == nil:
		return toolpreview.Create(record, chargeViewOf(after), opts...)
	default:
		return toolpreview.Changed(record, chargeViewOf(before), chargeViewOf(after), opts...)
	}
}

func chargeClause(plan *billingqueueports.IssueSettlementPlan) string {
	before, after := plan.ChargeBefore, plan.ChargeAfter
	switch {
	case before != nil && after == nil:
		return fmt.Sprintf(", removing the %s from the shipment",
			strings.ToLower(chargeLabel(plan.Item, before.ID)))
	case before == nil && after != nil:
		return fmt.Sprintf(", putting the removed charge back on the shipment at %s",
			after.Amount.StringFixed(2))
	case before != nil && after != nil:
		return fmt.Sprintf(", changing the %s from %s to %s",
			strings.ToLower(chargeLabel(plan.Item, before.ID)),
			before.Amount.StringFixed(2),
			after.Amount.StringFixed(2))
	default:
		return ""
	}
}

func classifyIssueOption(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // Classify passes params by value
) agenttoolports.CallPolicy {
	if strings.TrimSpace(toolparams.OptionalString(params.Params, paramOptionKey)) ==
		billingqueue.OptionKeyRequest {
		return agenttoolports.CallPolicy{Egress: agent.EgressDriverVisible}
	}

	return agenttoolports.CallPolicy{Egress: agent.EgressMoney}
}

func NewResolveBillingQueueIssueTool(
	review billingqueueports.BillingQueueReviewService,
) agenttoolports.AgentTool {
	return newResolveBillingQueueIssueTool(review)
}

func newResolveBillingQueueIssueTool(review queueReviewer) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name: toolResolveBillingQueueIssue,
			Recipe: []string{
				billingqueuepreview.ToolGetBillingQueueItem,
				toolResolveBillingQueueIssue,
			},
			Prerequisites: []string{billingqueuepreview.ToolGetBillingQueueItem},
			Description: "Propose settling one of a billing queue item's flagged checks with " +
				"one of the choices that check offers. A choice bills a charge as it stands, " +
				"takes it off, reprices it, bills without the paperwork, or asks the driver for " +
				"a signed POD. " +
				"billing_get_queue_item lists each check's issueId and its options with their " +
				"optionKey; use them exactly. Settling can change what the customer is charged, " +
				"so a person always decides, and asking the driver settles nothing until the " +
				"paperwork arrives.",
			Resource:    permission.ResourceBillingQueue,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressMoney,
			AlsoEgress:  []agent.EgressClass{agent.EgressDriverVisible},
			Classify:    classifyIssueOption,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierPropose,
			PersonOnly:  true,
			Reversible:  true,
			Rationale: "Decides what a customer is billed for a flagged charge, or messages " +
				"the driver for paperwork, so only a person settles it; billing_undo_issue " +
				"takes it back.",
			Target:   queueTarget,
			Artifact: billingqueuepreview.BillingQueueRecordEntity,
			Properties: map[string]any{
				billingqueuepreview.ParamBillingQueueItemID: queueItemProperty(),
				paramIssueID: issueProperty(),
				paramOptionKey: toolparams.StringProperty(
					"The optionKey of the choice, exactly as billing_get_queue_item lists it "+
						"under the check.",
					maxOptionKeyChars,
				),
			},
			Required: []string{
				billingqueuepreview.ParamBillingQueueItemID,
				paramIssueID,
				paramOptionKey,
			},
			SearchTerms: []string{
				"settle check",
				"billing issue",
				"flagged charge",
				"resolve issue",
			},
		},
		toolspec.ReceivablePlan[
			*billingqueueports.ResolveBillingQueueIssueRequest,
			*billingqueueports.IssueSettlementPlan,
		]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*billingqueueports.ResolveBillingQueueIssueRequest, error) {
				itemID, issueID, err := queueIssueIDs(params)
				if err != nil {
					return nil, err
				}
				optionKey, err := toolparams.RequireBoundedText(
					params.Params,
					paramOptionKey,
					maxOptionKeyChars,
				)
				if err != nil {
					return nil, err
				}
				return &billingqueueports.ResolveBillingQueueIssueRequest{
					ItemID:     itemID,
					IssueID:    issueID,
					OptionKey:  optionKey,
					TenantInfo: toolparams.TenantFrom(*params),
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *billingqueueports.ResolveBillingQueueIssueRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*billingqueueports.IssueSettlementPlan, error) {
				return review.PlanResolveIssue(ctx, req)
			},
			Refused: func(*billingqueueports.ResolveBillingQueueIssueRequest) string {
				return "Would settle a flagged check on the billing queue item."
			},
			Render: renderResolveIssue,
			Run: func(
				ctx context.Context,
				req *billingqueueports.ResolveBillingQueueIssueRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				item, err := review.ResolveIssue(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return queueItemResult("settled", item), nil
			},
		},
	)
}

func queueIssueIDs(params *agenttoolports.ToolExecuteParams) (itemID, issueID pulid.ID, err error) {
	itemID, err = toolparams.RequirePulid(
		params.Params,
		billingqueuepreview.ParamBillingQueueItemID,
	)
	if err != nil {
		return pulid.Nil, pulid.Nil, err
	}
	if issueID, err = toolparams.RequirePulid(params.Params, paramIssueID); err != nil {
		return pulid.Nil, pulid.Nil, err
	}

	return itemID, issueID, nil
}

func renderResolveIssue(
	_ *billingqueueports.ResolveBillingQueueIssueRequest,
	plan *billingqueueports.IssueSettlementPlan,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, 2)
	if plan.Option.Effect.Kind == billingqueue.EffectRequest {
		summary := fmt.Sprintf("Would ask the driver on billing queue item %s's shipment for a "+
			"signed POD; the check stays open until the paperwork arrives.", plan.Item.Number)
		if !plan.AsksDriver {
			return toolpreview.Build(summary), nil
		}
		changes = append(changes, toolpreview.Send(toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       plan.Item.ShipmentID,
			Label:    driverLabel(plan.DriverName),
		}, &agentpreview.MessagePreview{
			Channel: agentpreview.MessageChannelDash,
			Subject: plan.DriverMessageTitle,
			Body:    plan.DriverMessage,
		}))
		return toolpreview.Build(summary, changes...), nil
	}

	settled := *plan.Issue
	key := plan.Option.Key
	settled.ResolutionKey = &key
	settled.ResolutionText = plan.Option.Done
	check, err := toolpreview.Changed(
		checkRecord(plan),
		checkViewOf(plan.Issue),
		checkViewOf(&settled),
		toolpreview.Labels(queueCheckLabels),
	)
	if err != nil {
		return nil, err
	}
	changes = append(changes, check)
	charge, err := chargeChange(plan)
	if err != nil {
		return nil, err
	}
	if charge != nil {
		changes = append(changes, charge)
	}

	summary := fmt.Sprintf("Would settle %q on billing queue item %s as %q%s.",
		plan.Issue.Summary, plan.Item.Number, plan.Option.Label, chargeClause(plan))

	return toolpreview.Build(summary, changes...), nil
}

func driverLabel(name string) string {
	if name == "" {
		return "Message to the driver"
	}

	return "Message to " + name
}

func NewUndoBillingQueueIssueTool(
	review billingqueueports.BillingQueueReviewService,
) agenttoolports.AgentTool {
	return newUndoBillingQueueIssueTool(review)
}

func newUndoBillingQueueIssueTool(review queueReviewer) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name: toolUndoBillingQueueIssue,
			Recipe: []string{
				billingqueuepreview.ToolGetBillingQueueItem,
				toolUndoBillingQueueIssue,
			},
			Prerequisites: []string{billingqueuepreview.ToolGetBillingQueueItem},
			Description: "Propose taking back how one of a billing queue item's checks was " +
				"settled. The check is open again and any charge the settlement removed or " +
				"repriced is put back as it was. billing_get_queue_item lists each check's " +
				"issueId and whether its settlement can be undone; only a settled check on an " +
				"item still in review can be. It can change what the customer is charged, so a " +
				"person always decides.",
			Resource:    permission.ResourceBillingQueue,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressMoney,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierPropose,
			PersonOnly:  true,
			Reversible:  true,
			Rationale: "Puts back a charge a settlement removed or repriced, which changes what " +
				"the customer is billed, so only a person decides.",
			Target:   queueTarget,
			Artifact: billingqueuepreview.BillingQueueRecordEntity,
			Properties: map[string]any{
				billingqueuepreview.ParamBillingQueueItemID: queueItemProperty(),
				paramIssueID: issueProperty(),
			},
			Required:    []string{billingqueuepreview.ParamBillingQueueItemID, paramIssueID},
			SearchTerms: []string{"undo settlement", "reopen check", "undo billing issue"},
		},
		toolspec.ReceivablePlan[
			*billingqueueports.UndoBillingQueueIssueRequest,
			*billingqueueports.IssueSettlementPlan,
		]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*billingqueueports.UndoBillingQueueIssueRequest, error) {
				itemID, issueID, err := queueIssueIDs(params)
				if err != nil {
					return nil, err
				}
				return &billingqueueports.UndoBillingQueueIssueRequest{
					ItemID:     itemID,
					IssueID:    issueID,
					TenantInfo: toolparams.TenantFrom(*params),
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *billingqueueports.UndoBillingQueueIssueRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*billingqueueports.IssueSettlementPlan, error) {
				return review.PlanUndoIssue(ctx, req)
			},
			Refused: func(*billingqueueports.UndoBillingQueueIssueRequest) string {
				return "Would undo how a check on the billing queue item was settled."
			},
			Render: renderUndoIssue,
			Run: func(
				ctx context.Context,
				req *billingqueueports.UndoBillingQueueIssueRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				item, err := review.UndoIssue(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return queueItemResult("reopened", item), nil
			},
		},
	)
}

func renderUndoIssue(
	_ *billingqueueports.UndoBillingQueueIssueRequest,
	plan *billingqueueports.IssueSettlementPlan,
) (*agentpreview.ToolPreview, error) {
	reopened := *plan.Issue
	reopened.ResolutionKey = nil
	reopened.ResolutionText = ""
	check, err := toolpreview.Changed(
		checkRecord(plan),
		checkViewOf(plan.Issue),
		checkViewOf(&reopened),
		toolpreview.Labels(queueCheckLabels),
	)
	if err != nil {
		return nil, err
	}
	changes := []*agentpreview.RecordChange{check}
	charge, err := chargeChange(plan)
	if err != nil {
		return nil, err
	}
	if charge != nil {
		changes = append(changes, charge)
	}

	return toolpreview.Build(fmt.Sprintf("Would reopen %q on billing queue item %s, undoing %q%s.",
		plan.Issue.Summary, plan.Item.Number, plan.Issue.ResolutionText, chargeClause(plan)),
		changes...), nil
}
