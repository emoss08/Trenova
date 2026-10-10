package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramInvoiceIDs           = "invoiceIds"
	paramBillingQueueItemIDs  = "billingQueueItemIds"
	nounInvoice               = "invoice"
	nounInvoices              = "invoices"
	nounBillingQueueItem      = "billing queue item"
	nounBillingQueueItems     = "billing queue items"
	bulkApprovalDescription   = "Propose approving several billing queue items in one proposal the person approves together. On your own, propose only items you reviewed and found clean; when the person asks for them, propose those items and say in reviewNotes what you could not confirm. Use it instead of approve_billing_queue_item whenever more than one item is ready. Each item is approved exactly as approve_billing_queue_item would: charges that match the agreement, the documents the customer requires and no detention charge waiting on approval. Approval creates each item's draft invoice, or puts it on a statement customer's statement, and the auto-post setting may post it; a person always decides and may untick items. An item the queue would refuse is reported, not approved, and the rest still go. Up to 50 items."
	bulkPostDescription       = "Propose posting several draft invoices in one proposal the person approves together. Use it instead of post_invoice whenever more than one invoice is ready, rather than one proposal per invoice. Each invoice is posted exactly as post_invoice would: booked to the ledger, its shipments marked invoiced, its billing queue item settled and queued for the accounting system and, where the customer takes it, an EDI 210. It cannot be undone except by voiding, so a person always decides and may untick invoices. An invoice posting would refuse is reported, not posted, and the rest still go. Up to 50 invoices."
	bulkSendDescription       = "Propose emailing several invoices to their customers in one proposal the person approves together. Use it instead of send_invoice whenever more than one invoice goes out. Each invoice goes exactly as send_invoice would send it: the recipients, subject, wording and attachments come from the organization and each customer's billing profile, and nothing is chosen by you. The person approving sees every email, may untick invoices, and is who sends them. An invoice that cannot go is reported, not sent, and the rest still go. Up to 50 invoices."
	bulkInvoiceIDsDescription = "The invoices, by id from list_invoices, get_invoices or the proposals that made them. Never guess one."
)

type invoiceNumberReader interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
}

func invoiceNumbers(reader invoiceNumberReader) recordLabels {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) map[pulid.ID]string {
		if reader == nil {
			return nil
		}
		invoices, err := reader.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
			TenantInfo: tenant,
			InvoiceIDs: ids,
		})
		if err != nil {
			return nil
		}
		labels := make(map[pulid.ID]string, len(invoices))
		for _, entity := range invoices {
			labels[entity.ID] = invoiceLabel(entity)
		}

		return labels
	}
}

func queueItemNumbers(billing billingQueueDecider) recordLabels {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) map[pulid.ID]string {
		if billing == nil {
			return nil
		}
		labels := make(map[pulid.ID]string, len(ids))
		for _, id := range ids {
			item, err := billing.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
				ItemID:     id,
				TenantInfo: tenant,
			})
			if err == nil && item != nil {
				labels[id] = item.Number
			}
		}

		return labels
	}
}

func bulkIDsSchema(batch *recordBatch, description string, extra map[string]any) map[string]any {
	properties := map[string]any{batch.param: batch.property(description)}
	for name, property := range extra {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             []string{batch.param},
		toolschema.KeyAdditionalProperties: false,
	}
}

type postInvoicesTool struct {
	single *postInvoiceTool
	batch  *recordBatch
}

var (
	_ serviceports.ToolPreviewer      = (*postInvoicesTool)(nil)
	_ serviceports.ToolValidator      = (*postInvoicesTool)(nil)
	_ serviceports.ToolResultReporter = (*postInvoicesTool)(nil)
)

func newPostInvoicesTool(
	invoices invoicePoster,
	numbers invoiceNumberReader,
) serviceports.AgentTool {
	single := &postInvoiceTool{invoices: invoices}

	return &postInvoicesTool{
		single: single,
		batch: &recordBatch{
			param:       paramInvoiceIDs,
			singleParam: paramInvoiceID,
			resource:    permission.ResourceInvoice,
			noun:        nounInvoice,
			nouns:       nounInvoices,
			verb:        "post",
			past:        pastPosted,
			unchanged:   "is already posted",
			single:      single,
			needsPerson: ErrInvoiceNeedsAPerson,
			labels:      invoiceNumbers(numbers),
		},
	}
}

func (t *postInvoicesTool) Name() string { return "post_invoices" }

func (t *postInvoicesTool) Recipe() []string {
	return []string{"list_invoices", "get_invoices", "post_invoices"}
}

func (t *postInvoicesTool) BatchOf() string { return t.single.Name() }

func (t *postInvoicesTool) Description() string { return bulkPostDescription }

func (t *postInvoicesTool) ParamSchema() map[string]any {
	return bulkIDsSchema(t.batch, bulkInvoiceIDsDescription, nil)
}

func (t *postInvoicesTool) Policy() serviceports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = invoiceRecordEntity
	policy.Rationale = "Books several receivables to the ledger at once and queues each for " +
		"the accounting system and the customer's EDI; only a person posts, invoice by " +
		"invoice exactly as post_invoice, and may untick any of them."

	return policy
}

func (t *postInvoicesTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *postInvoicesTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *postInvoicesTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *postInvoicesTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

type sendInvoicesTool struct {
	single *sendInvoiceTool
	batch  *recordBatch
}

var (
	_ serviceports.ToolPreviewer      = (*sendInvoicesTool)(nil)
	_ serviceports.ToolValidator      = (*sendInvoicesTool)(nil)
	_ serviceports.ToolResultReporter = (*sendInvoicesTool)(nil)
)

func newSendInvoicesTool(
	invoices invoiceSender,
	numbers invoiceNumberReader,
) serviceports.AgentTool {
	single := &sendInvoiceTool{invoices: invoices}

	return &sendInvoicesTool{
		single: single,
		batch: &recordBatch{
			param:       paramInvoiceIDs,
			singleParam: paramInvoiceID,
			resource:    permission.ResourceInvoice,
			noun:        nounInvoice,
			nouns:       nounInvoices,
			verb:        "email",
			past:        "sent",
			single:      single,
			needsPerson: ErrInvoiceNeedsAPerson,
			labels:      invoiceNumbers(numbers),
		},
	}
}

func (t *sendInvoicesTool) Name() string { return "send_invoices" }

func (t *sendInvoicesTool) BatchOf() string { return t.single.Name() }

func (t *sendInvoicesTool) Description() string { return bulkSendDescription }

func (t *sendInvoicesTool) ParamSchema() map[string]any {
	return bulkIDsSchema(t.batch, bulkInvoiceIDsDescription, nil)
}

func (t *sendInvoicesTool) Policy() serviceports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = invoiceRecordEntity
	policy.Rationale = "Emails several customers their invoices at once; only a person sends " +
		"them, is named as who sent each, and sees every recipient before approving."

	return policy
}

func (t *sendInvoicesTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *sendInvoicesTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *sendInvoicesTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *sendInvoicesTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

type approveBillingQueueItemsTool struct {
	single *approveBillingQueueItemTool
	batch  *recordBatch
}

var (
	_ serviceports.ToolPreviewer      = (*approveBillingQueueItemsTool)(nil)
	_ serviceports.ToolValidator      = (*approveBillingQueueItemsTool)(nil)
	_ serviceports.ToolResultReporter = (*approveBillingQueueItemsTool)(nil)
)

func newApproveBillingQueueItemsTool(
	billing billingQueueDecider,
	invoices approvalInvoicePlanner,
) serviceports.AgentTool {
	single, _ := newApproveBillingQueueItemTool(billing, invoices).(*approveBillingQueueItemTool)

	return &approveBillingQueueItemsTool{
		single: single,
		batch: &recordBatch{
			param:       paramBillingQueueItemIDs,
			singleParam: paramBillingQueueItemID,
			resource:    permission.ResourceBillingQueue,
			kinds:       []permission.RecordKind{permission.KindBillingQueueItem},
			noun:        nounBillingQueueItem,
			nouns:       nounBillingQueueItems,
			verb:        verbApprove,
			past:        pastApproved,
			shared:      []string{paramReviewNotes},
			single:      single,
			needsPerson: ErrDecisionNeedsAPerson,
			labels:      queueItemNumbers(billing),
		},
	}
}

func (t *approveBillingQueueItemsTool) Name() string { return "approve_billing_queue_items" }

func (t *approveBillingQueueItemsTool) Recipe() []string {
	return []string{
		"list_billing_queue_items",
		"get_billing_queue_items",
		"approve_billing_queue_items",
	}
}

func (t *approveBillingQueueItemsTool) BatchOf() string { return t.single.Name() }

func (t *approveBillingQueueItemsTool) Description() string { return bulkApprovalDescription }

func (t *approveBillingQueueItemsTool) ParamSchema() map[string]any {
	return bulkIDsSchema(t.batch,
		"The billing queue items, by id from list_billing_queue_items or "+
			"get_billing_queue_item. Never guess one.",
		map[string]any{
			paramReviewNotes: map[string]any{
				toolschema.KeyType:      toolschema.TypeString,
				toolschema.KeyMaxLength: maxDecisionNoteChars,
				toolschema.KeyDescription: "What you checked, kept on each item as its " +
					"review notes.",
			},
		})
}

func (t *approveBillingQueueItemsTool) Policy() serviceports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Rationale = "Approving creates the invoices customers are billed on, so only a " +
		"person approves, item by item exactly as approve_billing_queue_item, and may " +
		"untick any of them."

	return policy
}

func (t *approveBillingQueueItemsTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *approveBillingQueueItemsTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *approveBillingQueueItemsTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *approveBillingQueueItemsTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}
