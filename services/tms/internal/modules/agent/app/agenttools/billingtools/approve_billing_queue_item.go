package billingtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
)

// approveBillingQueueItemTool proposes approving an item, which creates its
// draft invoice. The agent reviews and proposes; the person who approves the
// proposal is who approves the item, and nothing approves one unattended but
// the organization's own auto-approve rule.
type ApproveBillingQueueItemTool struct {
	*BillingQueueDecisionTool

	invoices common.ApprovalInvoicePlanner
}

func NewApproveBillingQueueItemTool(
	billing common.BillingQueueDecider,
	invoices common.ApprovalInvoicePlanner,
) agenttoolports.AgentTool {
	return &ApproveBillingQueueItemTool{
		invoices: invoices,
		BillingQueueDecisionTool: &BillingQueueDecisionTool{
			billing: billing,
			decision: queueDecision{
				name: "billing_approve_queue_item",
				description: "Propose approving a billing queue item. On your own, propose it only " +
					"for an item you reviewed and found clean: charges that match the agreement, the " +
					"documents the customer requires, and no detention charge waiting on approval. When " +
					"the person asks you to approve it, propose it, and say in reviewNotes and your reply " +
					"what you could not confirm; they decide with that in front of them. An item still " +
					"waiting for review is taken into review by the approval itself. Approval creates the " +
					"item's draft invoice, or puts it on a statement customer's statement, and the " +
					"organization's auto-post setting may post it. Once approved, propose invoice_post " +
					"for the draft. For more than one item, propose billing_approve_queue_item once " +
					"with all of them instead.",
				status:     billingqueue.StatusApproved,
				personOnly: true,
				egress:     agent.EgressMoney,
				defaultTo:  agent.TierPropose,
				maxTier:    agent.TierPropose,
				rationale: "Approving creates the invoice a customer is billed on, so only a person " +
					"approves; the agent proposes it with what it checked.",
				properties: map[string]any{
					common.ParamReviewNotes: map[string]any{
						toolschema.KeyType:        toolschema.TypeString,
						toolschema.KeyMaxLength:   common.MaxDecisionNoteChars,
						toolschema.KeyDescription: "What you checked, kept on the item as its review notes.",
					},
				},
				fill: fillApproval,
			},
		},
	}
}

func (t *ApproveBillingQueueItemTool) Policy() agenttoolports.ToolPolicy {
	policy := t.BillingQueueDecisionTool.Policy()
	policy.Artifact = common.InvoiceRecordEntity

	return policy
}

func (t *ApproveBillingQueueItemTool) draftRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.CreateInvoiceFromBillingQueueRequest, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return &invoiceports.CreateInvoiceFromBillingQueueRequest{
		BillingQueueItemID: req.ItemID,
		TenantInfo:         req.TenantInfo,
		DeferToStatement:   true,
	}, nil
}

func (t *ApproveBillingQueueItemTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, item, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would approve billing queue item %s (now %s)", item.Number, item.Status)
	if !plan.Accepted() {
		return plan.PlanPreview(summary + "."), nil
	}

	draftReq, err := t.draftRequest(&params)
	if err != nil {
		return nil, err
	}
	draft, err := t.invoices.PreviewApprovalInvoice(ctx, draftReq)
	if err != nil {
		return common.WarnRefusal(plan.PlanPreview(summary+"."), err)
	}

	switch {
	case draft.DeferredToStatement:
		return plan.PlanPreview(
			summary + "; the customer is billed on a statement, so the charge " +
				"goes onto their next statement and no invoice is made now.",
		), nil
	case draft.Invoice == nil:
		return plan.PlanPreview(summary + "."), nil
	case draft.Invoice.ID.IsNotNil():
		return plan.PlanPreview(fmt.Sprintf(
			"%s; invoice %s already exists for it and is kept.",
			summary,
			draft.Invoice.Number,
		)), nil
	}

	invoiceChange, err := common.DraftInvoiceChange(draft.Invoice)
	if err != nil {
		return nil, err
	}
	summary += fmt.Sprintf(
		", creating draft invoice %s for %s",
		draft.Invoice.Number,
		money.FormatMinor(draft.Invoice.TotalAmountMinor, draft.Invoice.CurrencyCode),
	)
	if draft.AutoPost {
		summary += "; the billing-control auto-post setting would then post it on its own"
	}

	return plan.PlanPreview(summary+".", invoiceChange), nil
}

// ExecuteWithResult approves the item as the person who approved the proposal
// and names the invoice approval made, which invoice_post takes next.
func (t *ApproveBillingQueueItemTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := t.BillingQueueDecisionTool.Execute(ctx, params); err != nil {
		return nil, err
	}

	draftReq, err := t.draftRequest(&params)
	if err != nil {
		return nil, err
	}
	made, err := t.invoices.PreviewApprovalInvoice(ctx, draftReq)
	if err != nil || made == nil || made.Invoice == nil || made.Invoice.ID.IsNil() {
		//nolint:nilerr // the approval is made; not finding its invoice only costs the pointer
		return &agent.ToolExecutionResult{
			Action: common.PastApproved,
			Kind:   "billing queue item",
		}, nil
	}

	return &agent.ToolExecutionResult{
		Action: common.ResultCreated,
		Kind:   common.InvoiceRecordEntity,
		Name:   made.Invoice.Number,
		IDs:    map[string]string{common.ParamInvoiceID: made.Invoice.ID.String()},
		Record: &agent.RecordRef{
			EntityType: common.InvoiceRecordEntity,
			ID:         made.Invoice.ID.String(),
		},
	}, nil
}

func (t *ApproveBillingQueueItemTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}
