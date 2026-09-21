package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxResolutionNoteChars = 500
	maxPaymentApplications = 50
	defaultPaymentMethod   = customerpayment.MethodACH
)

// bankReceiptMatcher is the slice of the bank receipt service the write
// tools use: read one receipt, and match it to a payment.
type bankReceiptMatcher interface {
	Get(ctx context.Context, req *serviceports.GetBankReceiptRequest) (*bankreceipt.BankReceipt, error)
	Match(
		ctx context.Context,
		req *serviceports.MatchBankReceiptRequest,
		actor *serviceports.RequestActor,
	) (*bankreceipt.BankReceipt, error)
}

type customerPaymentReader interface {
	Get(ctx context.Context, req *serviceports.GetCustomerPaymentRequest) (*customerpayment.Payment, error)
}

type customerPaymentPoster interface {
	PostAndApply(
		ctx context.Context,
		req *serviceports.PostCustomerPaymentRequest,
		actor *serviceports.RequestActor,
	) (*customerpayment.Payment, error)
}

type workItemResolver interface {
	Get(ctx context.Context, req *serviceports.GetBankReceiptWorkItemRequest) (*bankreceiptworkitem.WorkItem, error)
	Resolve(
		ctx context.Context,
		req *serviceports.ResolveBankReceiptWorkItemRequest,
		actor *serviceports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
	Dismiss(
		ctx context.Context,
		req *serviceports.DismissBankReceiptWorkItemRequest,
		actor *serviceports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
}

// matchBankReceiptTool records that a bank receipt is a posted customer
// payment. The amounts must agree to the cent; the service refuses anything
// else, and the preview says so before anyone approves it.
type matchBankReceiptTool struct {
	receipts bankReceiptMatcher
	payments customerPaymentReader
}

func newMatchBankReceiptTool(
	receipts bankReceiptMatcher,
	payments customerPaymentReader,
) serviceports.AgentTool {
	return &matchBankReceiptTool{receipts: receipts, payments: payments}
}

func (t *matchBankReceiptTool) Name() string { return "match_bank_receipt" }

func (t *matchBankReceiptTool) Description() string {
	return "Match a bank receipt to the posted customer payment it represents. The " +
		"payment's amount must equal the receipt's exactly; the reference and date " +
		"should agree too. Matching closes the receipt's reconciliation work item. " +
		"Use the candidate from get_bank_receipt, or a payment found with " +
		"list_customer_payments. When no payment has been recorded yet, use " +
		"post_customer_payment with bankReceiptId instead."
}

func (t *matchBankReceiptTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bankReceiptId":     map[string]any{"type": "string", "description": "The bank receipt's id."},
			"customerPaymentId": map[string]any{"type": "string", "description": "The posted customer payment's id."},
		},
		"required":             []string{"bankReceiptId", "customerPaymentId"},
		"additionalProperties": false,
	}
}

func (t *matchBankReceiptTool) Reversible() bool { return false }

func (t *matchBankReceiptTool) PermissionResource() permission.Resource {
	return permission.ResourceBankReceipt
}

func (t *matchBankReceiptTool) PermissionOperation() permission.Operation { return permission.OpUpdate }

func (t *matchBankReceiptTool) RequiresIdempotencyKey() bool { return false }

func (t *matchBankReceiptTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *matchBankReceiptTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	id, err := requirePulid(params, "bankReceiptId")
	if err != nil {
		return serviceports.ToolTarget{}, false
	}

	return serviceports.ToolTarget{Resource: permission.ResourceBankReceipt, ID: id}, true
}

func (t *matchBankReceiptTool) Execute(ctx context.Context, params serviceports.ToolExecuteParams) error {
	receiptID, paymentID, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.receipts.Match(ctx, &serviceports.MatchBankReceiptRequest{
		ReceiptID:  receiptID,
		PaymentID:  paymentID,
		TenantInfo: tenantFrom(params),
	}, params.Actor)

	return err
}

func (t *matchBankReceiptTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	receiptID, paymentID, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	tenant := tenantFrom(params)
	receipt, err := t.receipts.Get(ctx, &serviceports.GetBankReceiptRequest{ReceiptID: receiptID, TenantInfo: tenant})
	if err != nil {
		return nil, err
	}
	if receipt.Status == bankreceipt.StatusMatched {
		return nil, fmt.Errorf("bank receipt %s is already matched", receiptID)
	}

	payment, err := t.payments.Get(ctx, &serviceports.GetCustomerPaymentRequest{PaymentID: paymentID, TenantInfo: tenant})
	if err != nil {
		return nil, err
	}
	if payment.Status != customerpayment.StatusPosted {
		return nil, fmt.Errorf("customer payment %s is %s, and only a posted payment can be matched", paymentID, payment.Status)
	}
	if payment.AmountMinor != receipt.AmountMinor {
		return nil, fmt.Errorf(
			"the receipt is %s and the payment is %s; a match needs the same amount",
			money.DecimalFromMinor(receipt.AmountMinor).StringFixed(2),
			money.DecimalFromMinor(payment.AmountMinor).StringFixed(2),
		)
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would match bank receipt %s for %s to customer payment %s.",
			receiptID, money.DecimalFromMinor(receipt.AmountMinor).StringFixed(2), paymentID,
		),
		Changes: []agent.FieldChange{
			{Field: "status", From: string(receipt.Status), To: string(bankreceipt.StatusMatched)},
			{Field: "matchedCustomerPaymentId", From: "", To: paymentID.String()},
			{Field: "workItem", From: "active", To: string(bankreceiptworkitem.ResolutionMatchedToPayment)},
		},
		Previewed: true,
	}, nil
}

func (t *matchBankReceiptTool) arguments(params serviceports.ToolExecuteParams) (pulid.ID, pulid.ID, error) {
	if err := guardExecute(t, params); err != nil {
		return "", "", err
	}

	receiptID, err := requirePulid(params.Params, "bankReceiptId")
	if err != nil {
		return "", "", err
	}

	paymentID, err := requirePulid(params.Params, "customerPaymentId")
	if err != nil {
		return "", "", err
	}

	return receiptID, paymentID, nil
}

// postCustomerPaymentTool records a customer payment and applies it to
// invoices, and when the money is a bank receipt, matches that receipt to
// the new payment in the same step, so the receipt leaves the
// reconciliation queue with the payment that explains it.
type postCustomerPaymentTool struct {
	payments    customerPaymentPoster
	receipts    bankReceiptMatcher
	permissions serviceports.PermissionEngine
}

func newPostCustomerPaymentTool(
	payments customerPaymentPoster,
	receipts bankReceiptMatcher,
	permissions serviceports.PermissionEngine,
) serviceports.AgentTool {
	return &postCustomerPaymentTool{payments: payments, receipts: receipts, permissions: permissions}
}

func (t *postCustomerPaymentTool) Name() string { return "post_customer_payment" }

func (t *postCustomerPaymentTool) Description() string {
	return "Record a customer payment and apply it to that customer's invoices. Give the " +
		"customer, the amount, the date and the method, and the invoices it pays with the " +
		"amount applied to each; anything not applied stays on the customer's account as " +
		"unapplied cash, which is right when you cannot tell which invoice a remainder " +
		"pays. Give bankReceiptId when the money is an unmatched bank receipt: the amount " +
		"must equal the receipt's, and the receipt is matched to the new payment. Only " +
		"post against a customer you identified from the reference, the memo or the " +
		"invoices, never from the amount alone."
}

func (t *postCustomerPaymentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": map[string]any{"type": "string", "description": "The paying customer's id."},
			"amount": map[string]any{
				"type":        "string",
				"description": "The payment amount as a decimal string, such as 1250.00.",
			},
			"paymentDate": map[string]any{
				"type":        "string",
				"description": "The day the money arrived, YYYY-MM-DD; the receipt date for a bank receipt.",
			},
			"paymentMethod": map[string]any{
				"type":        "string",
				"enum":        paymentMethodNames(),
				"description": "How it was paid. Defaults to ACH.",
			},
			"referenceNumber": map[string]any{
				"type":        "string",
				"description": "The bank's reference, so the payment can be found again.",
			},
			"memo": map[string]any{"type": "string"},
			"applications": map[string]any{
				"type":        "array",
				"description": "The invoices this payment pays and how much of it goes to each.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"invoiceId": map[string]any{"type": "string"},
						"amount": map[string]any{
							"type":        "string",
							"description": "Applied to this invoice, as a decimal string.",
						},
						"shortPayAmount": map[string]any{
							"type":        "string",
							"description": "Written off on this invoice as a short pay, if any.",
						},
					},
					"required":             []string{"invoiceId", "amount"},
					"additionalProperties": false,
				},
			},
			"bankReceiptId": map[string]any{
				"type":        "string",
				"description": "The unmatched bank receipt this payment records, to match it in the same step.",
			},
		},
		"required":             []string{"customerId", "amount", "paymentDate"},
		"additionalProperties": false,
	}
}

func (t *postCustomerPaymentTool) Reversible() bool { return true }

func (t *postCustomerPaymentTool) PermissionResource() permission.Resource {
	return permission.ResourceCustomerPayment
}

func (t *postCustomerPaymentTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *postCustomerPaymentTool) RequiresIdempotencyKey() bool { return false }

func (t *postCustomerPaymentTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *postCustomerPaymentTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	id, err := requirePulid(params, "bankReceiptId")
	if err != nil {
		return serviceports.ToolTarget{}, false
	}

	return serviceports.ToolTarget{Resource: permission.ResourceBankReceipt, ID: id}, true
}

type postPaymentArgs struct {
	request   *serviceports.PostCustomerPaymentRequest
	receiptID pulid.ID
}

func (t *postCustomerPaymentTool) Execute(ctx context.Context, params serviceports.ToolExecuteParams) error {
	args, err := t.arguments(ctx, params)
	if err != nil {
		return err
	}

	payment, err := t.payments.PostAndApply(ctx, args.request, params.Actor)
	if err != nil {
		return err
	}
	if args.receiptID.IsNil() {
		return nil
	}

	_, err = t.receipts.Match(ctx, &serviceports.MatchBankReceiptRequest{
		ReceiptID:  args.receiptID,
		PaymentID:  payment.ID,
		TenantInfo: args.request.TenantInfo,
	}, params.Actor)
	if err != nil {
		return fmt.Errorf(
			"the payment %s was posted, but the bank receipt could not be matched to it: %w",
			payment.ID, err,
		)
	}

	return nil
}

func (t *postCustomerPaymentTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	args, err := t.arguments(ctx, params)
	if err != nil {
		return nil, err
	}

	req := args.request
	applied := int64(0)
	changes := make([]agent.FieldChange, 0, len(req.Applications)+3)
	changes = append(changes,
		agent.FieldChange{Field: "customer " + req.CustomerID.String(), To: "payment of " + money.DecimalFromMinor(req.AmountMinor).StringFixed(2)},
	)
	for _, application := range req.Applications {
		applied += application.AppliedAmountMinor
		change := agent.FieldChange{
			Field: "invoice " + application.InvoiceID.String(),
			To:    "applied " + money.DecimalFromMinor(application.AppliedAmountMinor).StringFixed(2),
		}
		if application.ShortPayAmountMinor > 0 {
			change.To += ", short pay " + money.DecimalFromMinor(application.ShortPayAmountMinor).StringFixed(2)
		}
		changes = append(changes, change)
	}
	if unapplied := req.AmountMinor - applied; unapplied > 0 {
		changes = append(changes, agent.FieldChange{Field: "unapplied cash", To: money.DecimalFromMinor(unapplied).StringFixed(2)})
	}
	summary := fmt.Sprintf(
		"Would record a %s %s payment from customer %s applied to %s.",
		money.DecimalFromMinor(req.AmountMinor).StringFixed(2), req.PaymentMethod, req.CustomerID,
		countOf(len(req.Applications), "invoice"),
	)
	if args.receiptID.IsNotNil() {
		summary += fmt.Sprintf(" Bank receipt %s would be matched to it.", args.receiptID)
		changes = append(changes, agent.FieldChange{
			Field: "bank receipt " + args.receiptID.String(),
			From:  string(bankreceipt.StatusException),
			To:    string(bankreceipt.StatusMatched),
		})
	}

	return &agent.ToolSimulation{Summary: summary, Changes: changes, Previewed: true}, nil
}

func (t *postCustomerPaymentTool) arguments(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (postPaymentArgs, error) {
	if err := guardExecute(t, params); err != nil {
		return postPaymentArgs{}, err
	}

	customerID, err := requirePulid(params.Params, "customerId")
	if err != nil {
		return postPaymentArgs{}, err
	}
	amount, err := requireMoney(params.Params, "amount")
	if err != nil {
		return postPaymentArgs{}, err
	}
	paymentDate, err := requireDay(params.Params, "paymentDate")
	if err != nil {
		return postPaymentArgs{}, err
	}

	method := defaultPaymentMethod
	if raw := optionalString(params.Params, "paymentMethod"); raw != "" {
		method = customerpayment.Method(raw)
		if !method.IsValid() {
			return postPaymentArgs{}, fmt.Errorf("paymentMethod %q is not one of %s", raw, strings.Join(paymentMethodNames(), ", "))
		}
	}

	applications, err := paymentApplications(params.Params, amount)
	if err != nil {
		return postPaymentArgs{}, err
	}

	tenant := tenantFrom(params)
	args := postPaymentArgs{request: &serviceports.PostCustomerPaymentRequest{
		CustomerID:      customerID,
		PaymentDate:     paymentDate,
		AccountingDate:  paymentDate,
		AmountMinor:     amount,
		PaymentMethod:   method,
		ReferenceNumber: strings.TrimSpace(optionalString(params.Params, "referenceNumber")),
		Memo:            strings.TrimSpace(optionalString(params.Params, "memo")),
		Applications:    applications,
		TenantInfo:      tenant,
	}}

	receiptID, present, err := optionalPulid(params.Params, "bankReceiptId")
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
	receipt, err := t.receipts.Get(ctx, &serviceports.GetBankReceiptRequest{ReceiptID: receiptID, TenantInfo: tenant})
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

	return args, nil
}

func (t *postCustomerPaymentTool) authorizeMatch(ctx context.Context, actor *serviceports.RequestActor) error {
	if t.permissions == nil {
		return fmt.Errorf("matching the bank receipt could not be authorized; post the payment without bankReceiptId and match it with match_bank_receipt")
	}

	result, err := t.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
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
		return fmt.Errorf("the person you are working for may not match bank receipts; post the payment without bankReceiptId")
	}

	return nil
}

func paymentApplications(
	params map[string]any,
	amount int64,
) ([]*serviceports.CustomerPaymentApplicationInput, error) {
	raw, ok := params["applications"]
	if !ok || raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("applications must be a list of {invoiceId, amount}")
	}
	if len(entries) > maxPaymentApplications {
		return nil, fmt.Errorf("applications lists %d invoices; at most %d can be applied in one payment", len(entries), maxPaymentApplications)
	}

	applications := make([]*serviceports.CustomerPaymentApplicationInput, 0, len(entries))
	seen := make(map[pulid.ID]struct{}, len(entries))
	total := int64(0)
	for idx, entry := range entries {
		fields, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("applications[%d] must be an object with invoiceId and amount", idx)
		}
		invoiceID, err := requirePulid(fields, "invoiceId")
		if err != nil {
			return nil, fmt.Errorf("applications[%d]: %w", idx, err)
		}
		if _, duplicate := seen[invoiceID]; duplicate {
			return nil, fmt.Errorf("applications[%d] repeats invoice %s", idx, invoiceID)
		}
		seen[invoiceID] = struct{}{}
		applied, err := requireMoney(fields, "amount")
		if err != nil {
			return nil, fmt.Errorf("applications[%d]: %w", idx, err)
		}
		shortPay, _, err := optionalMoney(fields, "shortPayAmount")
		if err != nil {
			return nil, fmt.Errorf("applications[%d]: %w", idx, err)
		}
		total += applied
		applications = append(applications, &serviceports.CustomerPaymentApplicationInput{
			InvoiceID:           invoiceID,
			AppliedAmountMinor:  applied,
			ShortPayAmountMinor: shortPay,
		})
	}
	if total > amount {
		return nil, fmt.Errorf(
			"the applications total %s, more than the %s payment",
			money.DecimalFromMinor(total).StringFixed(2), money.DecimalFromMinor(amount).StringFixed(2),
		)
	}

	return applications, nil
}

func paymentMethodNames() []string {
	return []string{
		string(customerpayment.MethodACH),
		string(customerpayment.MethodCheck),
		string(customerpayment.MethodWire),
		string(customerpayment.MethodCard),
		string(customerpayment.MethodCash),
		string(customerpayment.MethodOther),
	}
}

// resolveBankReceiptWorkItemTool closes a reconciliation queue entry
// without a match: the receipt is not a customer payment, or it needs
// somebody outside the system to say what it is.
type resolveBankReceiptWorkItemTool struct {
	items workItemResolver
}

func newResolveBankReceiptWorkItemTool(items workItemResolver) serviceports.AgentTool {
	return &resolveBankReceiptWorkItemTool{items: items}
}

func (t *resolveBankReceiptWorkItemTool) Name() string { return "resolve_bank_receipt_work_item" }

func (t *resolveBankReceiptWorkItemTool) Description() string {
	return "Close a bank receipt's reconciliation work item without matching it. Use " +
		"MarkedFalsePositive when the receipt is not a customer payment at all, such as " +
		"a transfer, a refund or interest, and RequiresExternalFollowUp when it is a " +
		"payment but the customer or the invoices cannot be identified from the records " +
		"and a person must ask the payer. The note is what that person reads. To match " +
		"the receipt instead, use match_bank_receipt or post_customer_payment."
}

func (t *resolveBankReceiptWorkItemTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workItemId": map[string]any{"type": "string", "description": "The work item's id, from get_bank_receipt."},
			"resolution": map[string]any{
				"type": "string",
				"enum": []string{
					string(bankreceiptworkitem.ResolutionRequiresExternalFollowUp),
					string(bankreceiptworkitem.ResolutionMarkedFalsePositive),
				},
			},
			"note": map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("Why, and what a person should do next, at most %d characters.", maxResolutionNoteChars),
			},
		},
		"required":             []string{"workItemId", "resolution", "note"},
		"additionalProperties": false,
	}
}

func (t *resolveBankReceiptWorkItemTool) Reversible() bool { return false }

func (t *resolveBankReceiptWorkItemTool) PermissionResource() permission.Resource {
	return permission.ResourceBankReceiptWorkItem
}

func (t *resolveBankReceiptWorkItemTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *resolveBankReceiptWorkItemTool) RequiresIdempotencyKey() bool { return false }

func (t *resolveBankReceiptWorkItemTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *resolveBankReceiptWorkItemTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	id, err := requirePulid(params, "workItemId")
	if err != nil {
		return serviceports.ToolTarget{}, false
	}

	return serviceports.ToolTarget{Resource: permission.ResourceBankReceiptWorkItem, ID: id}, true
}

type resolveWorkItemArgs struct {
	id         pulid.ID
	resolution bankreceiptworkitem.ResolutionType
	note       string
}

func (t *resolveBankReceiptWorkItemTool) Execute(ctx context.Context, params serviceports.ToolExecuteParams) error {
	args, err := t.arguments(params)
	if err != nil {
		return err
	}

	tenant := tenantFrom(params)
	if args.resolution == bankreceiptworkitem.ResolutionMarkedFalsePositive {
		_, err = t.items.Dismiss(ctx, &serviceports.DismissBankReceiptWorkItemRequest{
			WorkItemID:     args.id,
			ResolutionNote: args.note,
			TenantInfo:     tenant,
		}, params.Actor)

		return err
	}

	_, err = t.items.Resolve(ctx, &serviceports.ResolveBankReceiptWorkItemRequest{
		WorkItemID:     args.id,
		ResolutionType: args.resolution,
		ResolutionNote: args.note,
		TenantInfo:     tenant,
	}, params.Actor)

	return err
}

func (t *resolveBankReceiptWorkItemTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	args, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	item, err := t.items.Get(ctx, &serviceports.GetBankReceiptWorkItemRequest{
		WorkItemID: args.id,
		TenantInfo: tenantFrom(params),
	})
	if err != nil {
		return nil, err
	}
	if !item.Status.IsActive() {
		return nil, fmt.Errorf("work item %s is already %s", args.id, item.Status)
	}

	to := bankreceiptworkitem.StatusResolved
	if args.resolution == bankreceiptworkitem.ResolutionMarkedFalsePositive {
		to = bankreceiptworkitem.StatusDismissed
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf("Would close work item %s as %s.", args.id, args.resolution),
		Changes: []agent.FieldChange{
			{Field: "status", From: string(item.Status), To: string(to)},
			{Field: "resolutionType", From: string(item.ResolutionType), To: string(args.resolution)},
			{Field: "resolutionNote", From: item.ResolutionNote, To: args.note},
		},
		Previewed: true,
	}, nil
}

func (t *resolveBankReceiptWorkItemTool) arguments(params serviceports.ToolExecuteParams) (resolveWorkItemArgs, error) {
	if err := guardExecute(t, params); err != nil {
		return resolveWorkItemArgs{}, err
	}

	id, err := requirePulid(params.Params, "workItemId")
	if err != nil {
		return resolveWorkItemArgs{}, err
	}

	resolution := bankreceiptworkitem.ResolutionType(optionalString(params.Params, "resolution"))
	switch resolution {
	case bankreceiptworkitem.ResolutionRequiresExternalFollowUp, bankreceiptworkitem.ResolutionMarkedFalsePositive:
	default:
		return resolveWorkItemArgs{}, fmt.Errorf(
			"resolution must be %s or %s; to match the receipt use match_bank_receipt",
			bankreceiptworkitem.ResolutionRequiresExternalFollowUp,
			bankreceiptworkitem.ResolutionMarkedFalsePositive,
		)
	}

	note, err := requireString(params.Params, "note")
	if err != nil {
		return resolveWorkItemArgs{}, err
	}
	note = strings.TrimSpace(note)
	if len(note) > maxResolutionNoteChars {
		return resolveWorkItemArgs{}, fmt.Errorf("note must be at most %d characters", maxResolutionNoteChars)
	}

	return resolveWorkItemArgs{id: id, resolution: resolution, note: note}, nil
}
