package invoicetools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// ErrInvoiceNeedsAPerson is a post_invoice or send_invoice call that did
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
			common.ParamInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice, description),
		},
		toolschema.KeyRequired:             []string{common.ParamInvoiceID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func invoiceRequest(
	tool agentports.AgentTool,
	params *agentports.ToolExecuteParams,
) (*invoiceports.PostInvoiceRequest, error) {
	if err := common.GuardPreview(tool, params); err != nil {
		return nil, err
	}

	invoiceID, err := common.RequirePulid(params.Params, common.ParamInvoiceID)
	if err != nil {
		return nil, err
	}

	return &invoiceports.PostInvoiceRequest{
		InvoiceID:  invoiceID,
		TenantInfo: common.TenantFrom(*params),
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

func NewPostInvoiceTool(invoices invoicePoster) agentports.AgentTool {
	return &PostInvoiceTool{invoices: invoices}
}

func (t *PostInvoiceTool) Name() string { return "post_invoice" }

func (t *PostInvoiceTool) Description() string {
	return "Propose posting a draft invoice, usually the one approve_billing_queue_item just " +
		"made. Posting books it to the ledger, marks its shipments invoiced, settles its " +
		"billing queue item and queues it for the accounting system and, where the customer " +
		"takes it, an EDI 210. It cannot be undone except by voiding, so a person always " +
		"decides; propose it once the draft's lines and totals are right. For more than one " +
		"invoice, propose post_invoices once with all of them instead."
}

func (t *PostInvoiceTool) ParamSchema() map[string]any {
	return invoiceSchema("The draft invoice, from approve_billing_queue_item's result, " +
		"get_billing_queue_item or list_invoices.")
}

func (t *PostInvoiceTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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
		Artifact:      common.InvoiceRecordEntity,
		Rationale: "Books a receivable to the ledger and queues it for the accounting system " +
			"and the customer's EDI; only a person posts, and hands-off posting is the " +
			"billing-control auto-post setting.",
	}
}

func (t *PostInvoiceTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamInvoiceID, permission.ResourceInvoice)
}

func (t *PostInvoiceTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
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
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := common.GuardExecute(t, params); err != nil {
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
func postedInvoiceResult(posted *invoice.Invoice) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: "posted", Kind: common.InvoiceRecordEntity}
	if posted == nil || posted.ID.IsNil() {
		return result
	}
	result.Name = posted.Number
	result.IDs = map[string]string{common.ParamInvoiceID: posted.ID.String()}
	result.Record = &agent.RecordRef{EntityType: common.InvoiceRecordEntity, ID: posted.ID.String()}

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

func NewSendInvoiceTool(invoices invoiceSender) agentports.AgentTool {
	return &SendInvoiceTool{invoices: invoices}
}

func (t *SendInvoiceTool) Name() string { return "send_invoice" }

func (t *SendInvoiceTool) Description() string {
	return "Propose emailing an invoice to the customer. The recipients, subject, wording " +
		"and attachments come from the organization and the customer's billing profile; " +
		"nothing is chosen by you. A person always decides. Propose it after post_invoice when the " +
		"customer is not sent invoices automatically. For more than one invoice, propose " +
		"send_invoices once with all of them instead."
}

func (t *SendInvoiceTool) ParamSchema() map[string]any {
	return invoiceSchema("The invoice, from list_invoices or post_invoice's proposal.")
}

func (t *SendInvoiceTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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

func (t *SendInvoiceTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamInvoiceID, permission.ResourceInvoice)
}

// sendPlan is the send as PlanSend lays it out, and why it cannot go when it
// cannot.
type sendPlan struct {
	plan    *invoiceports.InvoiceSendPlan
	refusal error
}

func (t *SendInvoiceTool) plan(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
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
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	send, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return send.refusal
}

func (t *SendInvoiceTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
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
		TenantInfo: common.TenantFrom(params),
	}, params.Actor)

	return err
}
