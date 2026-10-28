package invoicetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxResolutionNoteChars = 500

const defaultPaymentMethod = customerpayment.MethodACH

// bankReceiptMatcher is the slice of the bank receipt service the write
// tools use: read one receipt, and match it to a payment.
type bankReceiptMatcher interface {
	Get(
		ctx context.Context,
		req *bankreceiptports.GetBankReceiptRequest,
	) (*bankreceipt.BankReceipt, error)
	Match(
		ctx context.Context,
		req *bankreceiptports.MatchBankReceiptRequest,
		actor *permissionports.RequestActor,
	) (*bankreceipt.BankReceipt, error)
	PreviewMatch(
		ctx context.Context,
		req *bankreceiptports.MatchBankReceiptRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptservice.MatchPreview, error)
	PreviewMatchPayment(
		ctx context.Context,
		req *bankreceiptservice.PreviewMatchPaymentRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptservice.MatchPreview, error)
	PaymentsOfReceiptAmount(
		ctx context.Context,
		req *bankreceiptports.ReceiptAmountPaymentsRequest,
	) ([]*customerpayment.Payment, error)
}

type customerPaymentPoster interface {
	PostAndApply(
		ctx context.Context,
		req *customerpaymentports.PostCustomerPaymentRequest,
		actor *permissionports.RequestActor,
	) (*customerpayment.Payment, error)
	PreviewPostAndApply(
		ctx context.Context,
		req *customerpaymentports.PostCustomerPaymentRequest,
		actor *permissionports.RequestActor,
	) (*customerpaymentports.CustomerPaymentPostPreview, error)
}

type workItemResolver interface {
	Get(
		ctx context.Context,
		req *bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
	) (*bankreceiptworkitem.WorkItem, error)
	Resolve(
		ctx context.Context,
		req *bankreceiptworkitemports.ResolveBankReceiptWorkItemRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
	Dismiss(
		ctx context.Context,
		req *bankreceiptworkitemports.DismissBankReceiptWorkItemRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
}

// matchBankReceiptTool records that a bank receipt is a posted customer
// payment. The amounts must agree to the cent; the service refuses anything
// else, and the preview says so before anyone approves it.
type MatchBankReceiptTool struct {
	receipts bankReceiptMatcher
}

func NewMatchBankReceiptTool(receipts bankReceiptMatcher) agenttoolports.AgentTool {
	return &MatchBankReceiptTool{receipts: receipts}
}

func (t *MatchBankReceiptTool) Name() string { return "receivable_match_bank_receipt" }

func (t *MatchBankReceiptTool) Recipe() []string {
	return []string{
		"receivable_get_bank_receipt",
		"receivable_list_customer_payments",
		"receivable_match_bank_receipt",
	}
}

func (t *MatchBankReceiptTool) Description() string {
	return "Match a bank receipt to the posted customer payment it represents. The " +
		"payment's amount must equal the receipt's exactly; the reference and date " +
		"should agree too. Matching closes the receipt's reconciliation work item. " +
		"Use the candidate from receivable_get_bank_receipt, or a payment found with " +
		"receivable_list_customer_payments. When no payment has been recorded yet, use " +
		"receivable_post_customer_payment with bankReceiptId instead."
}

func (t *MatchBankReceiptTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bankReceiptId": agenttoolschema.RecordIDText(
				permission.ResourceBankReceipt,
				"The bank receipt, from receivable_list_bank_receipt_exceptions or the page.",
			),
			"customerPaymentId": agenttoolschema.RecordIDText(
				permission.ResourceCustomerPayment,
				"The posted payment, from receivable_get_bank_receipt's suggestions or "+
					"receivable_list_customer_payments.",
			),
		},
		"required":             []string{"bankReceiptId", "customerPaymentId"},
		"additionalProperties": false,
	}
}

func (t *MatchBankReceiptTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:        t.Name(),
		Kind:        agent.ToolKindAction,
		Resource:    permission.ResourceBankReceipt,
		Operation:   permission.OpUpdate,
		Scope:       agent.ToolScopeTenant,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Egress:      []agent.EgressClass{agent.EgressMoney},
		Condition: &agenttoolports.TierCondition{
			Description: matchReceiptBoundText,
			Bounds:      t.withinBounds,
			Vouches:     []agent.TaintSource{agent.TaintSourceBankReceipt},
		},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Matches a bank receipt to a posted payment, closing its reconciliation.",
	}
}

func (t *MatchBankReceiptTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	id, err := toolparams.RequirePulid(params, "bankReceiptId")
	if err != nil {
		return agenttoolports.ToolTarget{}, false
	}

	return agenttoolports.ToolTarget{Resource: permission.ResourceBankReceipt, ID: id}, true
}

func (t *MatchBankReceiptTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(params)
	if err != nil {
		return err
	}

	_, err = t.receipts.Match(ctx, request, params.Actor)

	return err
}

func (t *MatchBankReceiptTool) request(
	params agenttoolports.ToolExecuteParams,
) (*bankreceiptports.MatchBankReceiptRequest, error) {
	receiptID, paymentID, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	return &bankreceiptports.MatchBankReceiptRequest{
		ReceiptID:  receiptID,
		PaymentID:  paymentID,
		TenantInfo: toolparams.TenantFrom(params),
	}, nil
}

func (t *MatchBankReceiptTool) arguments(
	params agenttoolports.ToolExecuteParams,
) (pulid.ID, pulid.ID, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return "", "", err
	}

	receiptID, err := toolparams.RequirePulid(params.Params, "bankReceiptId")
	if err != nil {
		return "", "", err
	}

	paymentID, err := toolparams.RequirePulid(params.Params, "customerPaymentId")
	if err != nil {
		return "", "", err
	}

	return receiptID, paymentID, nil
}

// postCustomerPaymentTool records a customer payment and applies it to
// invoices, and when the money is a bank receipt, matches that receipt to
// the new payment in the same step, so the receipt leaves the
// reconciliation queue with the payment that explains it.
type PostCustomerPaymentTool struct {
	payments    customerPaymentPoster
	receipts    bankReceiptMatcher
	permissions permissionports.PermissionEngine
}

func NewPostCustomerPaymentTool(
	payments customerPaymentPoster,
	receipts bankReceiptMatcher,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentTool {
	return &PostCustomerPaymentTool{
		payments:    payments,
		receipts:    receipts,
		permissions: permissions,
	}
}

func (t *PostCustomerPaymentTool) Name() string { return "receivable_post_customer_payment" }

func (t *PostCustomerPaymentTool) Recipe() []string {
	return []string{
		"receivable_get_bank_receipt",
		"receivable_list_customer_payments",
		"invoice_list",
		"receivable_post_customer_payment",
	}
}

func (t *PostCustomerPaymentTool) SearchTerms() []string {
	return []string{"received payment", "customer paid", "payment against invoices"}
}

func (t *PostCustomerPaymentTool) Description() string {
	return "Record a customer payment and apply it to that customer's invoices. Give the " +
		"customer, the amount, the date and the method, and the invoices it pays with the " +
		"amount applied to each; anything not applied stays on the customer's account as " +
		"unapplied cash, which is right when you cannot tell which invoice a remainder " +
		"pays. Give bankReceiptId when the money is an unmatched bank receipt: the amount " +
		"must equal the receipt's, and the receipt is matched to the new payment. Only " +
		"post against a customer you identified from the reference, the memo or the " +
		"invoices, never from the amount alone."
}

func (t *PostCustomerPaymentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": agenttoolschema.RecordIDText(
				permission.ResourceCustomer,
				"The paying customer, from customer_list or receivable_get_bank_receipt's "+
					"suggestions.",
			),
			toolparams.ParamAmount: map[string]any{
				"type":        "string",
				"description": "The payment amount as a decimal string, such as 1250.00.",
			},
			"paymentDate": agenttoolschema.Date(
				"The day the money arrived; the receipt date for a bank receipt.",
			),
			"paymentMethod": agenttoolschema.Enum(
				"How it was paid. Defaults to ACH.", customerPaymentMethods,
			),
			"referenceNumber": map[string]any{
				"type":        "string",
				"description": "The bank's reference, so the payment can be found again.",
			},
			"memo": map[string]any{
				"type": "string",
				"description": "A note kept on the payment, such as the remittance text or why a " +
					"remainder was left unapplied.",
			},
			"applications": map[string]any{
				"type":        "array",
				"description": "The invoices this payment pays and how much of it goes to each.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"invoiceId": agenttoolschema.RecordIDText(
							permission.ResourceInvoice,
							"An open invoice of this customer, from invoice_list.",
						),
						toolparams.ParamAmount: map[string]any{
							"type":        "string",
							"description": "Applied to this invoice, as a decimal string.",
						},
						"shortPayAmount": map[string]any{
							"type":        "string",
							"description": "Written off on this invoice as a short pay, if any.",
						},
					},
					"required":             []string{"invoiceId", toolparams.ParamAmount},
					"additionalProperties": false,
				},
			},
			"bankReceiptId": agenttoolschema.RecordIDText(
				permission.ResourceBankReceipt,
				"The unmatched bank receipt this payment records, from "+
					"receivable_list_bank_receipt_exceptions or receivable_get_bank_receipt, to match it in the same "+
					"step.",
			),
		},
		"required":             []string{"customerId", toolparams.ParamAmount, "paymentDate"},
		"additionalProperties": false,
	}
}

func (t *PostCustomerPaymentTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:        t.Name(),
		Kind:        agent.ToolKindAction,
		Resource:    permission.ResourceCustomerPayment,
		Operation:   permission.OpCreate,
		Scope:       agent.ToolScopeTenant,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Egress:      []agent.EgressClass{agent.EgressMoney},
		Condition: &agenttoolports.TierCondition{
			Description: postPaymentBoundText,
			Bounds:      t.withinBounds,
			Vouches:     []agent.TaintSource{agent.TaintSourceBankReceipt},
		},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Records a customer payment and applies it to invoices.",
		Artifact:      changepreview.CustomerRecordEntity,
	}
}

func (t *PostCustomerPaymentTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	id, err := toolparams.RequirePulid(params, "bankReceiptId")
	if err != nil {
		return agenttoolports.ToolTarget{}, false
	}

	return agenttoolports.ToolTarget{Resource: permission.ResourceBankReceipt, ID: id}, true
}

type postPaymentArgs struct {
	request   *customerpaymentports.PostCustomerPaymentRequest
	receiptID pulid.ID
	receipt   *bankreceipt.BankReceipt
}

func (t *PostCustomerPaymentTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// ExecuteWithResult posts the payment and matches the receipt. Once the post
// has gone through the write has moved money, so a receipt that will not match
// is reported in the result rather than failing the write: a failure would
// say nothing moved, and a retry would post the payment again.
func (t *PostCustomerPaymentTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	args, err := t.arguments(ctx, params)
	if err != nil {
		return nil, err
	}

	payment, err := t.payments.PostAndApply(ctx, args.request, params.Actor)
	if err != nil {
		return nil, err
	}
	result := &agent.ToolExecutionResult{
		Action: "posted",
		Kind:   "customer payment",
		Name:   payment.ReferenceNumber,
		IDs: map[string]string{
			"customerPaymentId": payment.ID.String(),
			"customerId":        payment.CustomerID.String(),
		},
		Record: &agent.RecordRef{
			EntityType: changepreview.CustomerRecordEntity,
			ID:         payment.CustomerID.String(),
		},
		State: fmt.Sprintf(
			"Posted, %s %s",
			money.DecimalFromMinor(payment.AmountMinor).StringFixed(2),
			payment.CurrencyCode,
		),
	}
	if args.receiptID.IsNil() {
		return result, nil
	}

	_, err = t.receipts.Match(ctx, &bankreceiptports.MatchBankReceiptRequest{
		ReceiptID:  args.receiptID,
		PaymentID:  payment.ID,
		TenantInfo: args.request.TenantInfo,
	}, params.Actor)
	if err != nil {
		result.State += "; the bank receipt was not matched to it"
		result.Total = 1
		result.Failed = []agent.ExecutionItemFail{{
			ID:     args.receiptID.String(),
			Label:  "bank receipt",
			Reason: fmt.Sprintf("could not be matched to payment %s: %s", payment.ID, err),
		}}

		return result, nil
	}
	result.State += ", matched to its bank receipt"

	return result, nil
}

func (t *PostCustomerPaymentTool) arguments(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) (postPaymentArgs, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return postPaymentArgs{}, err
	}

	customerID, err := toolparams.RequirePulid(params.Params, "customerId")
	if err != nil {
		return postPaymentArgs{}, err
	}
	amount, err := toolparams.RequireMoney(params.Params, toolparams.ParamAmount)
	if err != nil {
		return postPaymentArgs{}, err
	}
	paymentDate, err := toolparams.RequireDay(params.Params, "paymentDate")
	if err != nil {
		return postPaymentArgs{}, err
	}

	method := defaultPaymentMethod
	if raw := toolparams.OptionalString(params.Params, "paymentMethod"); raw != "" {
		method = customerpayment.Method(raw)
		if !method.IsValid() {
			return postPaymentArgs{}, fmt.Errorf(
				"paymentMethod %q is not one of %s",
				raw,
				strings.Join(customerPaymentMethods.Names(), ", "),
			)
		}
	}

	applications, err := paymentApplications(params.Params, amount)
	if err != nil {
		return postPaymentArgs{}, err
	}

	tenant := toolparams.TenantFrom(params)
	args := postPaymentArgs{request: &customerpaymentports.PostCustomerPaymentRequest{
		CustomerID:     customerID,
		PaymentDate:    paymentDate,
		AccountingDate: paymentDate,
		AmountMinor:    amount,
		PaymentMethod:  method,
		ReferenceNumber: strings.TrimSpace(
			toolparams.OptionalString(params.Params, "referenceNumber"),
		),
		Memo:         strings.TrimSpace(toolparams.OptionalString(params.Params, "memo")),
		Applications: applications,
		TenantInfo:   tenant,
	}}

	receiptID, present, err := toolparams.OptionalPulid(params.Params, "bankReceiptId")
	if err != nil {
		return postPaymentArgs{}, err
	}
	if !present {
		return args, nil
	}

	// Matching is a second write on a second resource, so the actor's right
	// to it is checked here rather than assumed from the right to post.
	if err = t.authorizeMatch(ctx, params.Actor); err != nil {
		return postPaymentArgs{}, err
	}
	receipt, err := t.receipts.Get(
		ctx,
		&bankreceiptports.GetBankReceiptRequest{ReceiptID: receiptID, TenantInfo: tenant},
	)
	if err != nil {
		return postPaymentArgs{}, err
	}
	if receipt.Status == bankreceipt.StatusMatched {
		return postPaymentArgs{}, fmt.Errorf("bank receipt %s is already matched", receiptID)
	}
	if receipt.AmountMinor != amount {
		return postPaymentArgs{}, fmt.Errorf(
			"the receipt is %s and the payment would be %s; a receipt is matched only by a payment of the same amount",
			money.DecimalFromMinor(receipt.AmountMinor).StringFixed(2),
			money.DecimalFromMinor(amount).StringFixed(2),
		)
	}
	if args.request.ReferenceNumber == "" {
		args.request.ReferenceNumber = receipt.ReferenceNumber
	}
	args.receiptID = receiptID
	args.receipt = receipt

	return args, nil
}

func (t *PostCustomerPaymentTool) authorizeMatch(
	ctx context.Context,
	actor *permissionports.RequestActor,
) error {
	if t.permissions == nil {
		return fmt.Errorf(
			"matching the bank receipt could not be authorized; post the payment without bankReceiptId and match it with receivable_match_bank_receipt",
		)
	}

	result, err := t.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceBankReceipt.String(),
		Operation:      permission.OpUpdate,
	})
	if err != nil {
		return fmt.Errorf("authorize bank receipt match: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf(
			"the person you are working for may not match bank receipts; post the payment without bankReceiptId",
		)
	}

	return nil
}

func paymentApplications(
	params map[string]any,
	amount int64,
) ([]*customerpaymentports.CustomerPaymentApplicationInput, error) {
	applications, total, err := invoicepreview.ReadPaymentApplications(
		params,
		invoicepreview.ParamApplications,
	)
	if err != nil {
		return nil, err
	}
	if total > amount {
		return nil, fmt.Errorf(
			"the applications total %s, more than the %s payment",
			money.DecimalFromMinor(total).
				StringFixed(2),
			money.DecimalFromMinor(amount).StringFixed(2),
		)
	}

	return applications, nil
}

var customerPaymentMethods = agenttoolschema.Source(
	"customerPayment.method",
	[]customerpayment.Method{
		customerpayment.MethodACH,
		customerpayment.MethodCheck,
		customerpayment.MethodWire,
		customerpayment.MethodCard,
		customerpayment.MethodCash,
		customerpayment.MethodOther,
	},
)

var bankReceiptResolutions = agenttoolschema.Source(
	"bankReceiptWorkItem.resolution",
	[]bankreceiptworkitem.ResolutionType{
		bankreceiptworkitem.ResolutionRequiresExternalFollowUp,
		bankreceiptworkitem.ResolutionMarkedFalsePositive,
	},
)

// resolveBankReceiptWorkItemTool closes a reconciliation queue entry
// without a match: the receipt is not a customer payment, or it needs
// somebody outside the system to say what it is.
type ResolveBankReceiptWorkItemTool struct {
	items workItemResolver
}

func NewResolveBankReceiptWorkItemTool(items workItemResolver) agenttoolports.AgentTool {
	return &ResolveBankReceiptWorkItemTool{items: items}
}

func (t *ResolveBankReceiptWorkItemTool) Name() string {
	return "receivable_resolve_bank_receipt_work_item"
}

func (t *ResolveBankReceiptWorkItemTool) Description() string {
	return "Close a bank receipt's reconciliation work item without matching it. Use " +
		"MarkedFalsePositive when the receipt is not a customer payment at all, such as " +
		"a transfer, a refund or interest, and RequiresExternalFollowUp when it is a " +
		"payment but the customer or the invoices cannot be identified from the records " +
		"and a person must ask the payer. The note is what that person reads. To match " +
		"the receipt instead, use receivable_match_bank_receipt or receivable_post_customer_payment."
}

func (t *ResolveBankReceiptWorkItemTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workItemId": agenttoolschema.RecordIDText(
				permission.ResourceBankReceiptWorkItem,
				"The work item's id, from receivable_get_bank_receipt.",
			),
			changepreview.FieldResolution: agenttoolschema.Enum(
				"MarkedFalsePositive when the receipt is not a customer payment; "+
					"RequiresExternalFollowUp when a person must ask the payer.",
				bankReceiptResolutions,
			),
			"note": map[string]any{
				"type": "string",
				"description": fmt.Sprintf(
					"Why, and what a person should do next, at most %d characters.",
					maxResolutionNoteChars,
				),
			},
		},
		"required":             []string{"workItemId", "resolution", "note"},
		"additionalProperties": false,
	}
}

func (t *ResolveBankReceiptWorkItemTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceBankReceiptWorkItem,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Closes a reconciliation work item without moving money; the note is read " +
			"inside the organization.",
	}
}

func (t *ResolveBankReceiptWorkItemTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	id, err := toolparams.RequirePulid(params, "workItemId")
	if err != nil {
		return agenttoolports.ToolTarget{}, false
	}

	return agenttoolports.ToolTarget{Resource: permission.ResourceBankReceiptWorkItem, ID: id}, true
}

type resolveWorkItemArgs struct {
	id         pulid.ID
	resolution bankreceiptworkitem.ResolutionType
	note       string
}

func (t *ResolveBankReceiptWorkItemTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	args, err := t.arguments(params)
	if err != nil {
		return err
	}

	tenant := toolparams.TenantFrom(params)
	if args.dismisses() {
		_, err = t.items.Dismiss(ctx, args.dismissal(tenant), params.Actor)

		return err
	}

	_, err = t.items.Resolve(ctx, args.resolved(tenant), params.Actor)

	return err
}

func (a resolveWorkItemArgs) dismisses() bool {
	return a.resolution == bankreceiptworkitem.ResolutionMarkedFalsePositive
}

func (a resolveWorkItemArgs) dismissal(
	tenant pagination.TenantInfo,
) *bankreceiptworkitemports.DismissBankReceiptWorkItemRequest {
	return &bankreceiptworkitemports.DismissBankReceiptWorkItemRequest{
		WorkItemID:     a.id,
		ResolutionNote: a.note,
		TenantInfo:     tenant,
	}
}

func (a resolveWorkItemArgs) resolved(
	tenant pagination.TenantInfo,
) *bankreceiptworkitemports.ResolveBankReceiptWorkItemRequest {
	return &bankreceiptworkitemports.ResolveBankReceiptWorkItemRequest{
		WorkItemID:     a.id,
		ResolutionType: a.resolution,
		ResolutionNote: a.note,
		TenantInfo:     tenant,
	}
}

func (t *ResolveBankReceiptWorkItemTool) arguments(
	params agenttoolports.ToolExecuteParams,
) (resolveWorkItemArgs, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return resolveWorkItemArgs{}, err
	}

	id, err := toolparams.RequirePulid(params.Params, "workItemId")
	if err != nil {
		return resolveWorkItemArgs{}, err
	}

	resolution := bankreceiptworkitem.ResolutionType(
		toolparams.OptionalString(params.Params, "resolution"),
	)
	switch resolution {
	case bankreceiptworkitem.ResolutionRequiresExternalFollowUp,
		bankreceiptworkitem.ResolutionMarkedFalsePositive:
	default:
		return resolveWorkItemArgs{}, fmt.Errorf(
			"resolution must be %s or %s; to match the receipt use receivable_match_bank_receipt",
			bankreceiptworkitem.ResolutionRequiresExternalFollowUp,
			bankreceiptworkitem.ResolutionMarkedFalsePositive,
		)
	}

	note, err := toolparams.RequireString(params.Params, "note")
	if err != nil {
		return resolveWorkItemArgs{}, err
	}
	note = strings.TrimSpace(note)
	if len(note) > maxResolutionNoteChars {
		return resolveWorkItemArgs{}, fmt.Errorf(
			"note must be at most %d characters",
			maxResolutionNoteChars,
		)
	}

	return resolveWorkItemArgs{id: id, resolution: resolution, note: note}, nil
}
