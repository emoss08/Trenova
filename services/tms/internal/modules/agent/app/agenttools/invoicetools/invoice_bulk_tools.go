package invoicetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingqueuepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/recordbatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramInvoiceIDs = "invoiceIds"

const nounInvoice = "invoice"

const nounInvoices = "invoices"

const toolInvoicePost = "invoice_post"

const toolInvoiceSend = "invoice_send"

const bulkApprovalDescription = "Propose approving billing queue items, one or up to 50 in one proposal the person decides together. On your own, propose only items you reviewed and found clean: charges that match the agreement, the documents the customer requires and no detention charge waiting on approval. When the person asks for items, propose those and say in reviewNotes what you could not confirm. Approval creates each item's draft invoice, or puts it on a statement customer's statement, and the auto-post setting may post it; the person may untick items. Name every ready item in one call; an item the queue would refuse is reported, not approved, and the rest still go."

const bulkPostDescription = "Propose posting draft invoices, one or up to 50 in one proposal the person decides together, usually the drafts approving billing queue items just made. Posting books each to the ledger, marks its shipments invoiced, settles its billing queue item and queues it for the accounting system and, where the customer takes it, an EDI 210. It cannot be undone except by voiding, so propose it once you have checked the drafts; the person may untick invoices. Name every ready invoice in one call; one posting would refuse is reported, not posted, and the rest still go."

const bulkSendDescription = "Propose emailing invoices to their customers, one or up to 50 in one proposal the person decides together. The recipients, subject, wording and attachments come from the organization and each customer's billing profile; nothing is chosen by you. Propose it after posting when the customer is not sent invoices automatically. The person approving sees every email, may untick invoices, and is who sends them. Name every invoice going out in one call; one that cannot go is reported, not sent, and the rest still go."

const bulkInvoiceIDsDescription = "The invoices, by id from invoice_list, invoice_get or the proposals that made them. Never guess one."

type invoiceNumberReader interface {
	GetByIDs(
		ctx context.Context,
		req invoiceports.GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
}

func invoiceNumbers(reader invoiceNumberReader) recordbatch.RecordLabels {
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
			labels[entity.ID] = invoicepreview.InvoiceLabel(entity)
		}

		return labels
	}
}

type PostInvoicesTool struct {
	single *PostInvoiceTool
	batch  *recordbatch.RecordBatch
}

func NewPostInvoicesTool(
	invoices invoicePoster,
	numbers invoiceNumberReader,
) agenttoolports.AgentTool {
	single := &PostInvoiceTool{invoices: invoices}

	return &PostInvoicesTool{
		single: single,
		batch: &recordbatch.RecordBatch{
			Param:       paramInvoiceIDs,
			SingleParam: toolparams.ParamInvoiceID,
			Resource:    permission.ResourceInvoice,
			Noun:        nounInvoice,
			Nouns:       nounInvoices,
			Verb:        "post",
			Past:        changepreview.PastPosted,
			Unchanged:   "is already posted",
			Single:      single,
			NeedsPerson: ErrInvoiceNeedsAPerson,
			Labels:      invoiceNumbers(numbers),
		},
	}
}

func (t *PostInvoicesTool) Name() string { return toolInvoicePost }

func (t *PostInvoicesTool) Recipe() []string {
	return []string{"invoice_list", toolparams.ToolInvoiceGet, toolInvoicePost}
}

func (t *PostInvoicesTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return t.batch.Target(params)
}

func (t *PostInvoicesTool) SearchTerms() []string { return []string{nounInvoices} }

func (t *PostInvoicesTool) Description() string { return bulkPostDescription }

func (t *PostInvoicesTool) ParamSchema() map[string]any {
	return recordbatch.BulkIDsSchema(t.batch, bulkInvoiceIDsDescription, nil)
}

func (t *PostInvoicesTool) Policy() agenttoolports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = changepreview.InvoiceRecordEntity
	policy.Rationale = "Books receivables to the ledger and queues each for the accounting " +
		"system and the customer's EDI; only a person posts, and may untick any of them, " +
		"and hands-off posting is the billing-control auto-post setting."

	return policy
}

func (t *PostInvoicesTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *PostInvoicesTool) SplitRecords(params map[string]any) ([]map[string]any, error) {
	return t.batch.Split(params)
}

func (t *PostInvoicesTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *PostInvoicesTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *PostInvoicesTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params)
}

type SendInvoicesTool struct {
	single *SendInvoiceTool
	batch  *recordbatch.RecordBatch
}

func NewSendInvoicesTool(
	invoices invoiceSender,
	numbers invoiceNumberReader,
) agenttoolports.AgentTool {
	single := &SendInvoiceTool{invoices: invoices}

	return &SendInvoicesTool{
		single: single,
		batch: &recordbatch.RecordBatch{
			Param:       paramInvoiceIDs,
			SingleParam: toolparams.ParamInvoiceID,
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

func (t *SendInvoicesTool) Name() string { return toolInvoiceSend }

func (t *SendInvoicesTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return t.batch.Target(params)
}

func (t *SendInvoicesTool) SearchTerms() []string { return []string{nounInvoices} }

func (t *SendInvoicesTool) Description() string { return bulkSendDescription }

func (t *SendInvoicesTool) ParamSchema() map[string]any {
	return recordbatch.BulkIDsSchema(t.batch, bulkInvoiceIDsDescription, nil)
}

func (t *SendInvoicesTool) Policy() agenttoolports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = changepreview.InvoiceRecordEntity
	policy.Rationale = "Emails customers their invoices; only a person sends them, is named " +
		"as who sent each, and sees every recipient before approving."

	return policy
}

func (t *SendInvoicesTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *SendInvoicesTool) SplitRecords(params map[string]any) ([]map[string]any, error) {
	return t.batch.Split(params)
}

func (t *SendInvoicesTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *SendInvoicesTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *SendInvoicesTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params)
}

type ApproveBillingQueueItemsTool struct {
	single *billingtools.ApproveBillingQueueItemTool
	batch  *recordbatch.RecordBatch
}

func NewApproveBillingQueueItemsTool(
	billing billingqueuepreview.BillingQueueDecider,
	invoices billingqueuepreview.ApprovalInvoicePlanner,
) agenttoolports.AgentTool {
	single, _ := billingtools.NewApproveBillingQueueItemTool(billing, invoices).(*billingtools.ApproveBillingQueueItemTool)

	return &ApproveBillingQueueItemsTool{
		single: single,
		batch: &recordbatch.RecordBatch{
			Param:       billingqueuepreview.ParamBillingQueueItemIDs,
			SingleParam: billingqueuepreview.ParamBillingQueueItemID,
			Resource:    permission.ResourceBillingQueue,
			Kinds:       []recordkind.Kind{recordkind.BillingQueueItem},
			Noun:        billingqueuepreview.NounBillingQueueItem,
			Nouns:       billingqueuepreview.NounBillingQueueItems,
			Verb:        changepreview.VerbApprove,
			Past:        changepreview.PastApproved,
			Shared:      []string{billingqueuepreview.ParamReviewNotes},
			Single:      single,
			NeedsPerson: billingqueuepreview.ErrDecisionNeedsAPerson,
			Labels:      billingqueuepreview.QueueItemNumbers(billing),
		},
	}
}

func (t *ApproveBillingQueueItemsTool) Name() string { return "billing_approve_queue_item" }

func (t *ApproveBillingQueueItemsTool) Recipe() []string {
	return []string{
		"billing_list_queue_items",
		"billing_get_queue_item",
		"billing_approve_queue_item",
	}
}

func (t *ApproveBillingQueueItemsTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return t.batch.Target(params)
}

func (t *ApproveBillingQueueItemsTool) Description() string { return bulkApprovalDescription }

func (t *ApproveBillingQueueItemsTool) ParamSchema() map[string]any {
	return recordbatch.BulkIDsSchema(t.batch,
		"The billing queue items, by id from billing_list_queue_items or "+
			"billing_get_queue_item. Never guess one.",
		map[string]any{
			billingqueuepreview.ParamReviewNotes: map[string]any{
				toolschema.KeyType:      toolschema.TypeString,
				toolschema.KeyMaxLength: billingqueuepreview.MaxDecisionNoteChars,
				toolschema.KeyDescription: "What you checked, kept on each item as its " +
					"review notes.",
			},
		})
}

func (t *ApproveBillingQueueItemsTool) Policy() agenttoolports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Rationale = "Approving creates the invoices customers are billed on, so only a " +
		"person approves, and may untick any of them."

	return policy
}

func (t *ApproveBillingQueueItemsTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *ApproveBillingQueueItemsTool) SplitRecords(
	params map[string]any,
) ([]map[string]any, error) {
	return t.batch.Split(params)
}

func (t *ApproveBillingQueueItemsTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *ApproveBillingQueueItemsTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *ApproveBillingQueueItemsTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params)
}
