package agenttoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
)

//nolint:gosec // G101: parameter names; "credit" matches the credential pattern, not a secret
const (
	paramCustomerPaymentID       = "customerPaymentId"
	paramAccountingDate          = "accountingDate"
	paramReason                  = "reason"
	paramCreditMemoID            = "creditMemoId"
	paramCreditMemoApplicationID = "creditMemoApplicationId"
	maxReceivableReasonChars     = 1000
)

var errApplicationsRequired = errors.New("applications must name at least one invoice")

type receivablesKeeper interface {
	ApplyUnapplied(
		ctx context.Context,
		req *serviceports.ApplyCustomerPaymentRequest,
		actor *serviceports.RequestActor,
	) (*customerpayment.Payment, error)
	Reverse(
		ctx context.Context,
		req *serviceports.ReverseCustomerPaymentRequest,
		actor *serviceports.RequestActor,
	) (*customerpayment.Payment, error)
	ApplyCreditMemo(
		ctx context.Context,
		req *serviceports.ApplyCreditMemoRequest,
		actor *serviceports.RequestActor,
	) ([]*customerpayment.CreditMemoApplication, error)
	UnapplyCreditMemoApplication(
		ctx context.Context,
		req *serviceports.UnapplyCreditMemoApplicationRequest,
		actor *serviceports.RequestActor,
	) (*customerpayment.CreditMemoApplication, error)
	PreviewApplyUnapplied(
		ctx context.Context,
		req *serviceports.ApplyCustomerPaymentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CustomerPaymentChangePreview, error)
	PreviewReverse(
		ctx context.Context,
		req *serviceports.ReverseCustomerPaymentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CustomerPaymentChangePreview, error)
	PreviewApplyCreditMemo(
		ctx context.Context,
		req *serviceports.ApplyCreditMemoRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CreditMemoApplicationPreview, error)
	PreviewUnapplyCreditMemoApplication(
		ctx context.Context,
		req *serviceports.UnapplyCreditMemoApplicationRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CreditMemoApplicationPreview, error)
}

func targetCustomerPayment(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramCustomerPaymentID, permission.ResourceCustomerPayment)
}

func accountingDateProperty() map[string]any {
	return stringProperty("The day it is booked to the ledger, YYYY-MM-DD, usually today; it "+
		"must fall in an open fiscal period.", 0)
}

func customerPaymentIDProperty() map[string]any {
	return stringProperty("The posted payment, from list_customer_payments. Never guess one.", 0)
}

func applicationsProperty(description string, withShortPay bool) map[string]any {
	properties := map[string]any{
		paramInvoiceID: stringProperty(
			"An open posted invoice or debit memo of the same customer, from list_invoices "+
				"or list_ar_open_items.", 0),
		paramAmount: stringProperty("Applied to this invoice, as a decimal such as 250.00.", 0),
	}
	if withShortPay {
		properties[paramShortPayAmount] = stringProperty(
			"Written off on this invoice as a short pay, if the customer will not pay it.", 0)
	}

	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMinItems:    1,
		toolschema.KeyMaxItems:    maxPaymentApplications,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType:                 toolschema.TypeObject,
			toolschema.KeyProperties:           properties,
			toolschema.KeyRequired:             []string{paramInvoiceID, paramAmount},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func receivableMoneySpec(spec *receivableSpec) *receivableSpec {
	spec.egress = agent.EgressMoney
	spec.defaultTier = agent.TierPropose
	spec.maxTier = agent.TierPropose
	spec.personOnly = true

	return spec
}

func newApplyCustomerPaymentTool(payments receivablesKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableMoneySpec(&receivableSpec{
		name: "apply_customer_payment",
		description: "Propose applying a posted payment's unapplied cash to that customer's open " +
			"invoices. Name each invoice and the amount it takes, and any short pay to write off; " +
			"together they cannot exceed what is unapplied. It books a ledger entry, so a person " +
			"always decides. Find the payment with list_customer_payments and the invoices with " +
			"list_ar_open_items.",
		resource:  permission.ResourceCustomerPayment,
		operation: permission.OpUpdate,
		rationale: "Moves a customer's cash onto their invoices and books the entry that says " +
			"so; only a person applies cash.",
		properties: map[string]any{
			paramCustomerPaymentID: customerPaymentIDProperty(),
			paramAccountingDate:    accountingDateProperty(),
			paramApplications: applicationsProperty(
				"The invoices the unapplied cash pays and how much goes to each.", true),
		},
		required: []string{paramCustomerPaymentID, paramAccountingDate, paramApplications},
		target:   targetCustomerPayment,
	}), receivablePlan[*serviceports.ApplyCustomerPaymentRequest, *serviceports.CustomerPaymentChangePreview]{
		request: applyPaymentRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ApplyCustomerPaymentRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.CustomerPaymentChangePreview, error) {
			return payments.PreviewApplyUnapplied(ctx, req, params.Actor)
		},
		refused: func(req *serviceports.ApplyCustomerPaymentRequest) string {
			return fmt.Sprintf("Would apply the payment's unapplied cash to %s.",
				countOf(len(req.Applications), "invoice"))
		},
		render: renderApplyPayment,
		run: func(
			ctx context.Context,
			req *serviceports.ApplyCustomerPaymentRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.ApplyUnapplied(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func applyPaymentRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ApplyCustomerPaymentRequest, error) {
	paymentID, err := requirePulid(params.Params, paramCustomerPaymentID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := requireDay(params.Params, paramAccountingDate)
	if err != nil {
		return nil, err
	}
	applications, _, err := readPaymentApplications(params.Params, paramApplications)
	if err != nil {
		return nil, err
	}
	if len(applications) == 0 {
		return nil, errApplicationsRequired
	}

	return &serviceports.ApplyCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: accountingDate,
		Applications:   applications,
		TenantInfo:     tenantFrom(*params),
	}, nil
}

func newReverseCustomerPaymentTool(payments receivablesKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableMoneySpec(&receivableSpec{
		name: "reverse_customer_payment",
		description: "Propose reversing a posted customer payment that bounced, was charged back " +
			"or was recorded in error. Every invoice it paid is reopened by what it applied and a " +
			"reversing ledger entry takes the cash back out. It cannot be undone, so a person " +
			"always decides; say why.",
		resource:  permission.ResourceCustomerPayment,
		operation: permission.OpUpdate,
		rationale: "Takes cash back off the books and reopens the invoices it paid; only a " +
			"person reverses a payment.",
		properties: map[string]any{
			paramCustomerPaymentID: customerPaymentIDProperty(),
			paramAccountingDate:    accountingDateProperty(),
			paramReason: stringProperty(
				"Why it is reversed, such as the bank's return reason.", maxReceivableReasonChars),
		},
		required: []string{paramCustomerPaymentID, paramAccountingDate, paramReason},
		target:   targetCustomerPayment,
	}), receivablePlan[*serviceports.ReverseCustomerPaymentRequest, *serviceports.CustomerPaymentChangePreview]{
		request: reversePaymentRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ReverseCustomerPaymentRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.CustomerPaymentChangePreview, error) {
			return payments.PreviewReverse(ctx, req, params.Actor)
		},
		refused: func(*serviceports.ReverseCustomerPaymentRequest) string {
			return "Would reverse the payment."
		},
		render: renderReversePayment,
		run: func(
			ctx context.Context,
			req *serviceports.ReverseCustomerPaymentRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.Reverse(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func reversePaymentRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ReverseCustomerPaymentRequest, error) {
	paymentID, err := requirePulid(params.Params, paramCustomerPaymentID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := requireDay(params.Params, paramAccountingDate)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxReceivableReasonChars)
	if err != nil {
		return nil, err
	}

	return &serviceports.ReverseCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: accountingDate,
		Reason:         reason,
		TenantInfo:     tenantFrom(*params),
	}, nil
}

func newApplyCreditMemoTool(payments receivablesKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableMoneySpec(&receivableSpec{
		name: "apply_credit_memo",
		description: "Propose settling a customer's open invoices with a posted credit memo of " +
			"theirs. Name each invoice and the amount of credit it takes; together they cannot " +
			"exceed what is left of the memo, nor any invoice its open balance. A person always " +
			"decides. Find the memo with list_invoices (billType CreditMemo).",
		resource:  permission.ResourceCustomerPayment,
		operation: permission.OpCreate,
		rationale: "Uses a customer's credit to settle what they owe; only a person applies " +
			"credit.",
		properties: map[string]any{
			paramCreditMemoID: stringProperty(
				"The posted credit memo, from list_invoices or get_invoice.", 0),
			paramAccountingDate: accountingDateProperty(),
			paramApplications: applicationsProperty(
				"The invoices the credit settles and how much goes to each.", false),
		},
		required: []string{paramCreditMemoID, paramAccountingDate, paramApplications},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramCreditMemoID, permission.ResourceInvoice)
		},
	}), receivablePlan[*serviceports.ApplyCreditMemoRequest, *serviceports.CreditMemoApplicationPreview]{
		request: applyCreditRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ApplyCreditMemoRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.CreditMemoApplicationPreview, error) {
			return payments.PreviewApplyCreditMemo(ctx, req, params.Actor)
		},
		refused: func(req *serviceports.ApplyCreditMemoRequest) string {
			return fmt.Sprintf("Would apply the credit memo to %s.",
				countOf(len(req.Applications), "invoice"))
		},
		render: renderApplyCredit,
		run: func(
			ctx context.Context,
			req *serviceports.ApplyCreditMemoRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.ApplyCreditMemo(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func applyCreditRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ApplyCreditMemoRequest, error) {
	memoID, err := requirePulid(params.Params, paramCreditMemoID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := requireDay(params.Params, paramAccountingDate)
	if err != nil {
		return nil, err
	}
	read, _, err := readPaymentApplications(params.Params, paramApplications)
	if err != nil {
		return nil, err
	}
	if len(read) == 0 {
		return nil, errApplicationsRequired
	}
	applications := make([]*serviceports.CreditMemoApplicationInput, 0, len(read))
	for idx, application := range read {
		if application.ShortPayAmountMinor > 0 {
			return nil, fmt.Errorf("%s[%d]: a credit memo writes nothing off; leave out %s",
				paramApplications, idx, paramShortPayAmount)
		}
		applications = append(applications, &serviceports.CreditMemoApplicationInput{
			InvoiceID:          application.InvoiceID,
			AppliedAmountMinor: application.AppliedAmountMinor,
		})
	}

	return &serviceports.ApplyCreditMemoRequest{
		CreditMemoID:   memoID,
		AccountingDate: accountingDate,
		Applications:   applications,
		TenantInfo:     tenantFrom(*params),
	}, nil
}

func newUnapplyCreditMemoTool(payments receivablesKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableMoneySpec(&receivableSpec{
		name: "unapply_credit_memo",
		description: "Propose taking back one credit memo application: the invoice it settled " +
			"is reopened by that amount and the credit is available again. Use it when credit " +
			"went to the wrong invoice. A person always decides; say why.",
		resource:  permission.ResourceCustomerPayment,
		operation: permission.OpUpdate,
		rationale: "Reopens an invoice's balance and restores a customer's credit; only a " +
			"person moves credit.",
		properties: map[string]any{
			paramCreditMemoApplicationID: stringProperty(
				"The application, from list_credit_memo_applications. Never guess one.", 0),
			paramReason: stringProperty("Why it is taken back.", maxReceivableReasonChars),
		},
		required: []string{paramCreditMemoApplicationID, paramReason},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(
				params, paramCreditMemoApplicationID, serviceports.RecordCreditMemoApplication,
			)
		},
	}), receivablePlan[*serviceports.UnapplyCreditMemoApplicationRequest, *serviceports.CreditMemoApplicationPreview]{
		request: unapplyCreditRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.UnapplyCreditMemoApplicationRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.CreditMemoApplicationPreview, error) {
			return payments.PreviewUnapplyCreditMemoApplication(ctx, req, params.Actor)
		},
		refused: func(*serviceports.UnapplyCreditMemoApplicationRequest) string {
			return "Would take back the credit memo application."
		},
		render: renderUnapplyCredit,
		run: func(
			ctx context.Context,
			req *serviceports.UnapplyCreditMemoApplicationRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.UnapplyCreditMemoApplication(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func unapplyCreditRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.UnapplyCreditMemoApplicationRequest, error) {
	applicationID, err := requirePulid(params.Params, paramCreditMemoApplicationID)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxReceivableReasonChars)
	if err != nil {
		return nil, err
	}

	return &serviceports.UnapplyCreditMemoApplicationRequest{
		ApplicationID: applicationID,
		Reason:        reason,
		TenantInfo:    tenantFrom(*params),
	}, nil
}
