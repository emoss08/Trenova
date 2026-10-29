package invoicetools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// ErrInvoiceNeedsAPerson is a invoice_post or invoice_send call that did
// not come from a proposal a person approved. Both tools stop at Propose;
// this is the same rule where the invoice is posted or sent.
var ErrInvoiceNeedsAPerson = errors.New(
	"an invoice is posted or sent only once a person approves the proposal",
)

// ErrInvoiceNeedsLinks is a send whose attachments are over the email
// provider's limit, which go as signed download links. Those are built
// against the address the person's browser used, which an agent has not
// got, so the send is refused rather than sent with links that go
// nowhere.
var ErrInvoiceNeedsLinks = errors.New(
	"some attachments are over the email provider's limit and would go as download " +
		"links; send this invoice from its page",
)

type invoicePoster interface {
	Post(
		ctx context.Context,
		req *invoiceports.PostInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
	PreviewPost(
		ctx context.Context,
		req *invoiceports.PostInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*invoiceservice.PostPreview, error)
}

type invoiceSender interface {
	PlanSend(
		ctx context.Context,
		req *invoiceports.InvoiceSendPlanRequest,
	) (*invoiceports.InvoiceSendPlan, error)
	Send(
		ctx context.Context,
		req *invoiceports.InvoiceSendRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceSendResult, error)
}

func invoiceSchema(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			toolparams.ParamInvoiceID: agenttoolschema.RecordIDText(
				permission.ResourceInvoice,
				description,
			),
		},
		toolschema.KeyRequired:             []string{toolparams.ParamInvoiceID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func invoiceRequest(
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.PostInvoiceRequest, error) {
	if err := toolguard.GuardPreview(tool, params); err != nil {
		return nil, err
	}

	invoiceID, err := toolparams.RequirePulid(params.Params, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, err
	}

	return &invoiceports.PostInvoiceRequest{
		InvoiceID:  invoiceID,
		TenantInfo: toolparams.TenantFrom(*params),
	}, nil
}

// postInvoiceTool proposes posting a draft invoice: it books the receivable,
// invoices the shipments, settles the queue item and queues the invoice for
// the accounting system and, where the customer takes it, EDI. A person
// always decides; hands-off posting is only the billing-control auto-post
// setting.
type PostInvoiceTool struct {
	invoices invoicePoster
}

func NewPostInvoiceTool(invoices invoicePoster) agenttoolports.AgentTool {
	return &PostInvoiceTool{invoices: invoices}
}

func (t *PostInvoiceTool) Name() string { return toolInvoicePost }

func (t *PostInvoiceTool) Description() string {
	return "Propose posting a draft invoice, usually the one billing_approve_queue_item just " +
		"made. Posting books it to the ledger, marks its shipments invoiced, settles its " +
		"billing queue item and queues it for the accounting system and, where the customer " +
		"takes it, an EDI 210. It cannot be undone except by voiding, so a person always " +
		"decides; propose it once the draft's lines and totals are right. For more than one " +
		"invoice, propose invoice_post once with all of them instead."
}

func (t *PostInvoiceTool) ParamSchema() map[string]any {
	return invoiceSchema("The draft invoice, from billing_approve_queue_item's result, " +
		"billing_get_queue_item or invoice_list.")
}

func (t *PostInvoiceTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceInvoice,
		Operation:     permission.OpPost,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Artifact:      changepreview.InvoiceRecordEntity,
		Rationale: "Books a receivable to the ledger and queues it for the accounting system " +
			"and the customer's EDI; only a person posts, and hands-off posting is the " +
			"billing-control auto-post setting.",
	}
}

func (t *PostInvoiceTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamInvoiceID, permission.ResourceInvoice)
}

func (t *PostInvoiceTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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

func (t *PostInvoiceTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
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

// ExecuteWithResult posts the invoice and says where it stands after, so the
// agent can tell the person it is posted, for how much and to whom, rather
// than sending them to look.
func (t *PostInvoiceTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, ErrInvoiceNeedsAPerson
	}

	req, err := invoiceRequest(t, &params)
	if err != nil {
		return nil, err
	}
	posted, err := t.invoices.Post(ctx, req, params.Actor)
	if err != nil {
		return nil, err
	}

	return postedInvoiceResult(posted), nil
}

// postedInvoiceResult names a posted invoice and where it stands.
func postedInvoiceResult(posted *invoice.Invoice) *agentrecord.ToolExecutionResult {
	result := &agentrecord.ToolExecutionResult{
		Action: "posted",
		Kind:   changepreview.InvoiceRecordEntity,
	}
	if posted == nil || posted.ID.IsNil() {
		return result
	}
	result.Name = posted.Number
	result.IDs = map[string]string{toolparams.ParamInvoiceID: posted.ID.String()}
	result.Record = &agentrecord.RecordRef{
		EntityType: changepreview.InvoiceRecordEntity,
		ID:         posted.ID.String(),
	}

	state := string(posted.Status) + ", " + posted.TotalAmount.StringFixed(2) + " " +
		posted.CurrencyCode
	if posted.BillToName != "" {
		state += " to " + posted.BillToName
	}
	if posted.SettlementStatus != "" {
		state += ", " + strings.ToLower(string(posted.SettlementStatus))
	}
	result.State = state

	return result
}

// sendInvoiceTool proposes emailing a posted invoice to the customer, to the
// recipients, with the wording and attachments the organization configured.
type SendInvoiceTool struct {
	invoices invoiceSender
}

func NewSendInvoiceTool(invoices invoiceSender) agenttoolports.AgentTool {
	return &SendInvoiceTool{invoices: invoices}
}

func (t *SendInvoiceTool) Name() string { return toolInvoiceSend }

func (t *SendInvoiceTool) Description() string {
	return "Propose emailing an invoice to the customer. The recipients, subject, wording " +
		"and attachments come from the organization and the customer's billing profile; " +
		"nothing is chosen by you. A person always decides. Propose it after invoice_post when the " +
		"customer is not sent invoices automatically. For more than one invoice, propose " +
		"invoice_send once with all of them instead."
}

func (t *SendInvoiceTool) ParamSchema() map[string]any {
	return invoiceSchema("The invoice, from invoice_list or invoice_post's proposal.")
}

func (t *SendInvoiceTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceInvoice,
		Operation:     permission.OpSubmit,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Emails the customer their invoice; only a person sends it, to the " +
			"recipients the customer's billing profile names.",
	}
}

func (t *SendInvoiceTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamInvoiceID, permission.ResourceInvoice)
}

// sendPlan is the send as PlanSend lays it out, and why it cannot go when it
// cannot.
type sendPlan struct {
	plan    *invoiceports.InvoiceSendPlan
	refusal error
}

func (t *SendInvoiceTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (sendPlan, error) {
	req, err := invoiceRequest(t, params)
	if err != nil {
		return sendPlan{}, err
	}

	plan, err := t.invoices.PlanSend(ctx, &invoiceports.InvoiceSendPlanRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return sendPlan{}, err
	}

	return sendPlan{plan: plan, refusal: sendRefusal(plan)}, nil
}

func sendRefusal(plan *invoiceports.InvoiceSendPlan) error {
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

func (t *SendInvoiceTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	send, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return send.refusal
}

func (t *SendInvoiceTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
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

	_, err = t.invoices.Send(ctx, &invoiceports.InvoiceSendRequest{
		InvoiceID:  send.plan.InvoiceID,
		TenantInfo: toolparams.TenantFrom(params),
	}, params.Actor)

	return err
}
