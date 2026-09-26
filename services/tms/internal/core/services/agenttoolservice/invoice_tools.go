package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoiceservice"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const paramInvoiceID = "invoiceId"

var (
	// ErrInvoiceNeedsAPerson is a post_invoice or send_invoice call that did
	// not come from a proposal a person approved. Both tools stop at Propose;
	// this is the same rule where the invoice is posted or sent.
	ErrInvoiceNeedsAPerson = errors.New(
		"an invoice is posted or sent only once a person approves the proposal",
	)
	// ErrInvoiceNeedsLinks is a send whose attachments are over the email
	// provider's limit, which go as signed download links. Those are built
	// against the address the person's browser used, which an agent has not
	// got, so the send is refused rather than sent with links that go
	// nowhere.
	ErrInvoiceNeedsLinks = errors.New(
		"some attachments are over the email provider's limit and would go as download " +
			"links; send this invoice from its page",
	)
)

type invoicePoster interface {
	Post(
		ctx context.Context,
		req *serviceports.PostInvoiceRequest,
		actor *serviceports.RequestActor,
	) (*invoice.Invoice, error)
	PreviewPost(
		ctx context.Context,
		req *serviceports.PostInvoiceRequest,
		actor *serviceports.RequestActor,
	) (*invoiceservice.PostPreview, error)
}

type invoiceSender interface {
	PlanSend(
		ctx context.Context,
		req *serviceports.InvoiceSendPlanRequest,
	) (*serviceports.InvoiceSendPlan, error)
	Send(
		ctx context.Context,
		req *serviceports.InvoiceSendRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceSendResult, error)
}

func invoiceSchema(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramInvoiceID: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: description,
			},
		},
		toolschema.KeyRequired:             []string{paramInvoiceID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func invoiceRequest(
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
) (*serviceports.PostInvoiceRequest, error) {
	if err := guardPreview(tool, params); err != nil {
		return nil, err
	}

	invoiceID, err := requirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}

	return &serviceports.PostInvoiceRequest{
		InvoiceID:  invoiceID,
		TenantInfo: tenantFrom(*params),
	}, nil
}

// postInvoiceTool proposes posting a draft invoice: it books the receivable,
// invoices the shipments, settles the queue item and queues the invoice for
// the accounting system and, where the customer takes it, EDI. A person
// always decides; hands-off posting is only the billing-control auto-post
// setting.
type postInvoiceTool struct {
	invoices invoicePoster
}

var (
	_ serviceports.ToolPreviewer = (*postInvoiceTool)(nil)
	_ serviceports.ToolValidator = (*postInvoiceTool)(nil)
	_ serviceports.TargetedTool  = (*postInvoiceTool)(nil)
)

func newPostInvoiceTool(invoices invoicePoster) serviceports.AgentTool {
	return &postInvoiceTool{invoices: invoices}
}

func (t *postInvoiceTool) Name() string { return "post_invoice" }

func (t *postInvoiceTool) Description() string {
	return "Propose posting a draft invoice, usually the one approve_billing_queue_item just " +
		"made. Posting books it to the ledger, marks its shipments invoiced, settles its " +
		"billing queue item and queues it for the accounting system and, where the customer " +
		"takes it, an EDI 210. It cannot be undone except by voiding, so a person always " +
		"decides; propose it once the draft's lines and totals are right."
}

func (t *postInvoiceTool) ParamSchema() map[string]any {
	return invoiceSchema("The draft invoice, from approve_billing_queue_item's result, " +
		"get_billing_queue_item or list_invoices.")
}

func (t *postInvoiceTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceInvoice,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Books a receivable to the ledger and queues it for the accounting system " +
			"and the customer's EDI; only a person posts, and hands-off posting is the " +
			"billing-control auto-post setting.",
	}
}

func (t *postInvoiceTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramInvoiceID, permission.ResourceInvoice)
}

func (t *postInvoiceTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := invoiceRequest(t, &params)
	if err != nil {
		return err
	}

	preview, err := t.invoices.PreviewPost(ctx, req, params.Actor)
	if err != nil {
		return err
	}
	if preview.Refusal != nil {
		return preview.Refusal
	}
	if preview.AlreadyPosted {
		return fmt.Errorf("invoice %s is already posted", preview.Before.Number)
	}

	return nil
}

func (t *postInvoiceTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrInvoiceNeedsAPerson
	}

	req, err := invoiceRequest(t, &params)
	if err != nil {
		return err
	}

	_, err = t.invoices.Post(ctx, req, params.Actor)

	return err
}

// sendInvoiceTool proposes emailing a posted invoice to the customer, to the
// recipients, with the wording and attachments the organization configured.
type sendInvoiceTool struct {
	invoices invoiceSender
}

var (
	_ serviceports.ToolPreviewer = (*sendInvoiceTool)(nil)
	_ serviceports.ToolValidator = (*sendInvoiceTool)(nil)
	_ serviceports.TargetedTool  = (*sendInvoiceTool)(nil)
)

func newSendInvoiceTool(invoices invoiceSender) serviceports.AgentTool {
	return &sendInvoiceTool{invoices: invoices}
}

func (t *sendInvoiceTool) Name() string { return "send_invoice" }

func (t *sendInvoiceTool) Description() string {
	return "Propose emailing an invoice to the customer, to the recipients, subject, wording " +
		"and attachments the organization and the customer's billing profile set; nothing is " +
		"chosen by you. A person always decides. Propose it after post_invoice when the " +
		"customer is not sent invoices automatically."
}

func (t *sendInvoiceTool) ParamSchema() map[string]any {
	return invoiceSchema("The invoice, from list_invoices or post_invoice's proposal.")
}

func (t *sendInvoiceTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceInvoice,
		Operation:     permission.OpSubmit,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Emails the customer their invoice; only a person sends it, to the " +
			"recipients the customer's billing profile names.",
	}
}

func (t *sendInvoiceTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramInvoiceID, permission.ResourceInvoice)
}

// sendPlan is the send as PlanSend lays it out, and why it cannot go when it
// cannot.
type sendPlan struct {
	plan    *serviceports.InvoiceSendPlan
	refusal error
}

func (t *sendInvoiceTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (sendPlan, error) {
	req, err := invoiceRequest(t, params)
	if err != nil {
		return sendPlan{}, err
	}

	plan, err := t.invoices.PlanSend(ctx, &serviceports.InvoiceSendPlanRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return sendPlan{}, err
	}

	return sendPlan{plan: plan, refusal: sendRefusal(plan)}, nil
}

func sendRefusal(plan *serviceports.InvoiceSendPlan) error {
	if len(plan.Errors) > 0 {
		return errors.New(strings.Join(plan.Errors, "; "))
	}
	for _, part := range plan.Parts {
		if part != nil && len(part.Links) > 0 {
			return ErrInvoiceNeedsLinks
		}
	}

	return nil
}

func (t *sendInvoiceTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	send, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return send.refusal
}

func (t *sendInvoiceTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrInvoiceNeedsAPerson
	}

	send, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	if send.refusal != nil {
		return send.refusal
	}

	_, err = t.invoices.Send(ctx, &serviceports.InvoiceSendRequest{
		InvoiceID:  send.plan.InvoiceID,
		TenantInfo: tenantFrom(params),
	}, params.Actor)

	return err
}
