package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
)

// invoiceRecordEntity is the record-link registry's name for an invoice.
const invoiceRecordEntity = "invoice"

type approvalInvoicePlanner interface {
	PreviewApprovalInvoice(
		ctx context.Context,
		req *serviceports.CreateInvoiceFromBillingQueueRequest,
	) (*serviceports.CreateInvoiceFromBillingQueueResult, error)
}

// approveBillingQueueItemTool proposes approving an item, which creates its
// draft invoice. The agent reviews and proposes; the person who approves the
// proposal is who approves the item, and nothing approves one unattended but
// the organization's own auto-approve rule.
type approveBillingQueueItemTool struct {
	*billingQueueDecisionTool

	invoices approvalInvoicePlanner
}

var (
	_ serviceports.ToolPreviewer      = (*approveBillingQueueItemTool)(nil)
	_ serviceports.ToolResultReporter = (*approveBillingQueueItemTool)(nil)
)

func newApproveBillingQueueItemTool(
	billing billingQueueDecider,
	invoices approvalInvoicePlanner,
) serviceports.AgentTool {
	return &approveBillingQueueItemTool{
		invoices: invoices,
		billingQueueDecisionTool: &billingQueueDecisionTool{
			billing: billing,
			decision: queueDecision{
				name: "approve_billing_queue_item",
				description: "Propose approving a billing queue item you have reviewed and found " +
					"clean. Clean means charges that match the agreement, the documents the customer " +
					"requires, and no detention charge waiting on approval. Approval creates the item's draft " +
					"invoice, or puts it on a statement customer's statement, and the organization's " +
					"auto-post setting may post it; a person always decides. Say in reviewNotes what " +
					"you checked. Once approved, propose post_invoice for the draft.",
				status:     billingqueue.StatusApproved,
				personOnly: true,
				egress:     agent.EgressMoney,
				defaultTo:  agent.TierPropose,
				maxTier:    agent.TierPropose,
				rationale: "Approving creates the invoice a customer is billed on, so only a person " +
					"approves; the agent proposes it with what it checked.",
				properties: map[string]any{
					paramReviewNotes: map[string]any{
						toolschema.KeyType:        toolschema.TypeString,
						toolschema.KeyMaxLength:   maxDecisionNoteChars,
						toolschema.KeyDescription: "What you checked, kept on the item as its review notes.",
					},
				},
				fill: fillApproval,
			},
		},
	}
}

func (t *approveBillingQueueItemTool) Policy() serviceports.ToolPolicy {
	policy := t.billingQueueDecisionTool.Policy()
	policy.Artifact = invoiceRecordEntity

	return policy
}

func (t *approveBillingQueueItemTool) draftRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.CreateInvoiceFromBillingQueueRequest, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return &serviceports.CreateInvoiceFromBillingQueueRequest{
		BillingQueueItemID: req.ItemID,
		TenantInfo:         req.TenantInfo,
		DeferToStatement:   true,
	}, nil
}

func (t *approveBillingQueueItemTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, item, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would approve billing queue item %s (now %s)", item.Number, item.Status)
	if !plan.accepted() {
		return plan.preview(summary + "."), nil
	}

	draftReq, err := t.draftRequest(&params)
	if err != nil {
		return nil, err
	}
	draft, err := t.invoices.PreviewApprovalInvoice(ctx, draftReq)
	if err != nil {
		return warnRefusal(plan.preview(summary+"."), err)
	}

	switch {
	case draft.DeferredToStatement:
		return plan.preview(summary + "; the customer is billed on a statement, so the charge " +
			"goes onto their next statement and no invoice is made now."), nil
	case draft.Invoice == nil:
		return plan.preview(summary + "."), nil
	case draft.Invoice.ID.IsNotNil():
		return plan.preview(fmt.Sprintf(
			"%s; invoice %s already exists for it and is kept.",
			summary,
			draft.Invoice.Number,
		)), nil
	}

	invoiceChange, err := draftInvoiceChange(draft.Invoice)
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

	return plan.preview(summary+".", invoiceChange), nil
}

// draftInvoiceChange is the draft approval creates, with its totals.
func draftInvoiceChange(draft *invoice.Invoice) (*agent.RecordChange, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceInvoice, Label: invoiceLabel(draft)},
		draft,
		toolpreview.Only(
			"number",
			fieldStatus,
			"billType",
			"billToName",
			"invoiceDate",
			"dueDate",
			"paymentTerm",
			"currencyCode",
		),
	)
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		draft.CurrencyCode,
		agent.MoneyLine{Label: moneyLineFreight, After: knownAmount(draft.SubtotalAmount)},
		agent.MoneyLine{Label: moneyLineAccessorials, After: knownAmount(draft.OtherAmount)},
	), toolpreview.SensitiveAs("totalAmount", "subtotalAmount", "otherAmount"))

	return change, nil
}

// ExecuteWithResult approves the item as the person who approved the proposal
// and names the invoice approval made, which post_invoice takes next.
func (t *approveBillingQueueItemTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := t.billingQueueDecisionTool.Execute(ctx, params); err != nil {
		return nil, err
	}

	draftReq, err := t.draftRequest(&params)
	if err != nil {
		return nil, err
	}
	made, err := t.invoices.PreviewApprovalInvoice(ctx, draftReq)
	if err != nil || made == nil || made.Invoice == nil || made.Invoice.ID.IsNil() {
		//nolint:nilerr // the approval is made; not finding its invoice only costs the pointer
		return &agent.ToolExecutionResult{Action: "approved", Kind: "billing queue item"}, nil
	}

	return &agent.ToolExecutionResult{
		Action: resultCreated,
		Kind:   invoiceRecordEntity,
		Name:   made.Invoice.Number,
		IDs:    map[string]string{paramInvoiceID: made.Invoice.ID.String()},
		Record: &agent.RecordRef{EntityType: invoiceRecordEntity, ID: made.Invoice.ID.String()},
	}, nil
}

func (t *approveBillingQueueItemTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}
