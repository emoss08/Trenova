package invoicetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramInvoiceIDs = "invoiceIds"

const nounInvoice = "invoice"

const nounInvoices = "invoices"

const bulkApprovalDescription = "Propose approving several billing queue items in one proposal the person approves together. On your own, propose only items you reviewed and found clean; when the person asks for them, propose those items and say in reviewNotes what you could not confirm. Use it instead of approve_billing_queue_item whenever more than one item is ready. Each item is approved exactly as approve_billing_queue_item would: charges that match the agreement, the documents the customer requires and no detention charge waiting on approval. Approval creates each item's draft invoice, or puts it on a statement customer's statement, and the auto-post setting may post it; a person always decides and may untick items. An item the queue would refuse is reported, not approved, and the rest still go. Up to 50 items."

const bulkPostDescription = "Propose posting several draft invoices in one proposal the person approves together. Use it instead of post_invoice whenever more than one invoice is ready, rather than one proposal per invoice. Each invoice is posted exactly as post_invoice would: booked to the ledger, its shipments marked invoiced, its billing queue item settled and queued for the accounting system and, where the customer takes it, an EDI 210. It cannot be undone except by voiding, so a person always decides and may untick invoices. An invoice posting would refuse is reported, not posted, and the rest still go. Up to 50 invoices."

const bulkSendDescription = "Propose emailing several invoices to their customers in one proposal the person approves together. Use it instead of send_invoice whenever more than one invoice goes out. Each invoice goes exactly as send_invoice would send it: the recipients, subject, wording and attachments come from the organization and each customer's billing profile, and nothing is chosen by you. The person approving sees every email, may untick invoices, and is who sends them. An invoice that cannot go is reported, not sent, and the rest still go. Up to 50 invoices."

const bulkInvoiceIDsDescription = "The invoices, by id from list_invoices, get_invoices or the proposals that made them. Never guess one."

type invoiceNumberReader interface {
	GetByIDs(
		ctx context.Context,
		req invoiceports.GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
}

func invoiceNumbers(reader invoiceNumberReader) common.RecordLabels {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
	) map[pulid.ID]string {
		if reader == nil {
			return nil
		}
		invoices, err := reader.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
			TenantInfo: tenant,
			InvoiceIDs: ids,
		})
		if err != nil {
			return nil
		}
		labels := make(map[pulid.ID]string, len(invoices))
		for _, entity := range invoices {
			labels[entity.ID] = common.InvoiceLabel(entity)
		}

		return labels
	}
}

type PostInvoicesTool struct {
	single *PostInvoiceTool
	batch  *common.RecordBatch
}

func NewPostInvoicesTool(
	invoices invoicePoster,
	numbers invoiceNumberReader,
) agentports.AgentTool {
	single := &PostInvoiceTool{invoices: invoices}

	return &PostInvoicesTool{
		single: single,
		batch: &common.RecordBatch{
			Param:       paramInvoiceIDs,
			SingleParam: common.ParamInvoiceID,
			Resource:    permission.ResourceInvoice,
			Noun:        nounInvoice,
			Nouns:       nounInvoices,
			Verb:        "post",
			Past:        common.PastPosted,
			Unchanged:   "is already posted",
			Single:      single,
			NeedsPerson: ErrInvoiceNeedsAPerson,
			Labels:      invoiceNumbers(numbers),
		},
	}
}

func (t *PostInvoicesTool) Name() string { return "post_invoices" }

func (t *PostInvoicesTool) Recipe() []string {
	return []string{"list_invoices", "get_invoices", "post_invoices"}
}

func (t *PostInvoicesTool) BatchOf() string { return t.single.Name() }

func (t *PostInvoicesTool) Description() string { return bulkPostDescription }

func (t *PostInvoicesTool) ParamSchema() map[string]any {
	return common.BulkIDsSchema(t.batch, bulkInvoiceIDsDescription, nil)
}

func (t *PostInvoicesTool) Policy() agentports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = common.InvoiceRecordEntity
	policy.Rationale = "Books several receivables to the ledger at once and queues each for " +
		"the accounting system and the customer's EDI; only a person posts, invoice by " +
		"invoice exactly as post_invoice, and may untick any of them."

	return policy
}

func (t *PostInvoicesTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *PostInvoicesTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *PostInvoicesTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *PostInvoicesTool) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

type SendInvoicesTool struct {
	single *SendInvoiceTool
	batch  *common.RecordBatch
}

func NewSendInvoicesTool(
	invoices invoiceSender,
	numbers invoiceNumberReader,
) agentports.AgentTool {
	single := &SendInvoiceTool{invoices: invoices}

	return &SendInvoicesTool{
		single: single,
		batch: &common.RecordBatch{
			Param:       paramInvoiceIDs,
			SingleParam: common.ParamInvoiceID,
			Resource:    permission.ResourceInvoice,
			Noun:        nounInvoice,
			Nouns:       nounInvoices,
			Verb:        "email",
			Past:        "sent",
			Single:      single,
			NeedsPerson: ErrInvoiceNeedsAPerson,
			Labels:      invoiceNumbers(numbers),
		},
	}
}

func (t *SendInvoicesTool) Name() string { return "send_invoices" }

func (t *SendInvoicesTool) BatchOf() string { return t.single.Name() }

func (t *SendInvoicesTool) Description() string { return bulkSendDescription }

func (t *SendInvoicesTool) ParamSchema() map[string]any {
	return common.BulkIDsSchema(t.batch, bulkInvoiceIDsDescription, nil)
}

func (t *SendInvoicesTool) Policy() agentports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = common.InvoiceRecordEntity
	policy.Rationale = "Emails several customers their invoices at once; only a person sends " +
		"them, is named as who sent each, and sees every recipient before approving."

	return policy
}

func (t *SendInvoicesTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *SendInvoicesTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *SendInvoicesTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *SendInvoicesTool) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

type ApproveBillingQueueItemsTool struct {
	single *billingtools.ApproveBillingQueueItemTool
	batch  *common.RecordBatch
}

func NewApproveBillingQueueItemsTool(
	billing common.BillingQueueDecider,
	invoices common.ApprovalInvoicePlanner,
) agentports.AgentTool {
	single, _ := billingtools.NewApproveBillingQueueItemTool(billing, invoices).(*billingtools.ApproveBillingQueueItemTool)

	return &ApproveBillingQueueItemsTool{
		single: single,
		batch: &common.RecordBatch{
			Param:       common.ParamBillingQueueItemIDs,
			SingleParam: common.ParamBillingQueueItemID,
			Resource:    permission.ResourceBillingQueue,
			Kinds:       []permission.RecordKind{permission.KindBillingQueueItem},
			Noun:        common.NounBillingQueueItem,
			Nouns:       common.NounBillingQueueItems,
			Verb:        common.VerbApprove,
			Past:        common.PastApproved,
			Shared:      []string{common.ParamReviewNotes},
			Single:      single,
			NeedsPerson: common.ErrDecisionNeedsAPerson,
			Labels:      common.QueueItemNumbers(billing),
		},
	}
}

func (t *ApproveBillingQueueItemsTool) Name() string { return "approve_billing_queue_items" }

func (t *ApproveBillingQueueItemsTool) Recipe() []string {
	return []string{
		"list_billing_queue_items",
		"get_billing_queue_items",
		"approve_billing_queue_items",
	}
}

func (t *ApproveBillingQueueItemsTool) BatchOf() string { return t.single.Name() }

func (t *ApproveBillingQueueItemsTool) Description() string { return bulkApprovalDescription }

func (t *ApproveBillingQueueItemsTool) ParamSchema() map[string]any {
	return common.BulkIDsSchema(t.batch,
		"The billing queue items, by id from list_billing_queue_items or "+
			"get_billing_queue_item. Never guess one.",
		map[string]any{
			common.ParamReviewNotes: map[string]any{
				toolschema.KeyType:      toolschema.TypeString,
				toolschema.KeyMaxLength: common.MaxDecisionNoteChars,
				toolschema.KeyDescription: "What you checked, kept on each item as its " +
					"review notes.",
			},
		})
}

func (t *ApproveBillingQueueItemsTool) Policy() agentports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Rationale = "Approving creates the invoices customers are billed on, so only a " +
		"person approves, item by item exactly as approve_billing_queue_item, and may " +
		"untick any of them."

	return policy
}

func (t *ApproveBillingQueueItemsTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *ApproveBillingQueueItemsTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *ApproveBillingQueueItemsTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *ApproveBillingQueueItemsTool) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}
