package receivabletools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const ParamCustomerPaymentID = "customerPaymentId"

const paramCreditMemoID = "creditMemoId"

//nolint:gosec // G101: a parameter name; "credit" matches the credential pattern, not a secret
const ParamCreditMemoApplicationID = "creditMemoApplicationId"

const maxReceivableReasonChars = 1000

var errApplicationsRequired = errors.New("applications must name at least one invoice")

type receivablesKeeper interface {
	ApplyUnapplied(
		ctx context.Context,
		req *customerpaymentports.ApplyCustomerPaymentRequest,
		actor *permissionports.RequestActor,
	) (*customerpayment.Payment, error)
	Reverse(
		ctx context.Context,
		req *customerpaymentports.ReverseCustomerPaymentRequest,
		actor *permissionports.RequestActor,
	) (*customerpayment.Payment, error)
	ApplyCreditMemo(
		ctx context.Context,
		req *customerpaymentports.ApplyCreditMemoRequest,
		actor *permissionports.RequestActor,
	) ([]*customerpayment.CreditMemoApplication, error)
	UnapplyCreditMemoApplication(
		ctx context.Context,
		req *customerpaymentports.UnapplyCreditMemoApplicationRequest,
		actor *permissionports.RequestActor,
	) (*customerpayment.CreditMemoApplication, error)
	PreviewApplyUnapplied(
		ctx context.Context,
		req *customerpaymentports.ApplyCustomerPaymentRequest,
		actor *permissionports.RequestActor,
	) (*customerpaymentports.CustomerPaymentChangePreview, error)
	PreviewReverse(
		ctx context.Context,
		req *customerpaymentports.ReverseCustomerPaymentRequest,
		actor *permissionports.RequestActor,
	) (*customerpaymentports.CustomerPaymentChangePreview, error)
	PreviewApplyCreditMemo(
		ctx context.Context,
		req *customerpaymentports.ApplyCreditMemoRequest,
		actor *permissionports.RequestActor,
	) (*customerpaymentports.CreditMemoApplicationPreview, error)
	PreviewUnapplyCreditMemoApplication(
		ctx context.Context,
		req *customerpaymentports.UnapplyCreditMemoApplicationRequest,
		actor *permissionports.RequestActor,
	) (*customerpaymentports.CreditMemoApplicationPreview, error)
}

func targetCustomerPayment(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, ParamCustomerPaymentID, permission.ResourceCustomerPayment)
}

func accountingDateProperty() map[string]any {
	return agenttoolschema.Date("The day it is booked to the ledger, usually today; it must " +
		"fall in an open fiscal period.")
}

func customerPaymentIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceCustomerPayment, "The posted payment",
		"receivable_list_customer_payments")
}

func applicationsProperty(description string, withShortPay bool) map[string]any {
	properties := map[string]any{
		common.ParamInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
			"An open posted invoice or debit memo of the same customer, from invoice_list "+
				"or receivable_list_open_items."),
		common.ParamAmount: common.StringProperty(
			"Applied to this invoice, as a decimal such as 250.00.",
			0,
		),
	}
	if withShortPay {
		properties[common.ParamShortPayAmount] = common.StringProperty(
			"Written off on this invoice as a short pay, if the customer will not pay it.", 0)
	}

	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMinItems:    1,
		toolschema.KeyMaxItems:    common.MaxPaymentApplications,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType:                 toolschema.TypeObject,
			toolschema.KeyProperties:           properties,
			toolschema.KeyRequired:             []string{common.ParamInvoiceID, common.ParamAmount},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func receivableMoneySpec(spec *common.ReceivableSpec) *common.ReceivableSpec {
	spec.Egress = agent.EgressMoney
	spec.DefaultTier = agent.TierPropose
	spec.MaxTier = agent.TierPropose
	spec.PersonOnly = true

	return spec
}

func NewApplyCustomerPaymentTool(payments receivablesKeeper) agenttoolports.AgentTool {
	return common.NewReceivableTool(receivableMoneySpec(&common.ReceivableSpec{
		Name: "receivable_apply_customer_payment",
		Recipe: []string{
			"receivable_list_customer_payments",
			"receivable_list_open_items",
			"receivable_apply_customer_payment",
		},
		SearchTerms: []string{"unapplied cash", "unapplied payment"},
		Description: "Propose applying a posted payment's unapplied cash to that customer's open " +
			"invoices. Name each invoice and the amount it takes, and any short pay to write off; " +
			"together they cannot exceed what is unapplied. It books a ledger entry, so a person " +
			"always decides. Find the payment with receivable_list_customer_payments and the invoices with " +
			"receivable_list_open_items.",
		Resource:  permission.ResourceCustomerPayment,
		Operation: permission.OpUpdate,
		Rationale: "Moves a customer's cash onto their invoices and books the entry that says " +
			"so; only a person applies cash.",
		Properties: map[string]any{
			ParamCustomerPaymentID:     customerPaymentIDProperty(),
			common.ParamAccountingDate: accountingDateProperty(),
			common.ParamApplications: applicationsProperty(
				"The invoices the unapplied cash pays and how much goes to each.", true),
		},
		Required: []string{
			ParamCustomerPaymentID,
			common.ParamAccountingDate,
			common.ParamApplications,
		},
		Target: targetCustomerPayment,
	}), common.ReceivablePlan[*customerpaymentports.ApplyCustomerPaymentRequest, *customerpaymentports.CustomerPaymentChangePreview]{
		Request: applyPaymentRequest,
		Plan: func(
			ctx context.Context,
			req *customerpaymentports.ApplyCustomerPaymentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*customerpaymentports.CustomerPaymentChangePreview, error) {
			return payments.PreviewApplyUnapplied(ctx, req, params.Actor)
		},
		Refused: func(req *customerpaymentports.ApplyCustomerPaymentRequest) string {
			return fmt.Sprintf("Would apply the payment's unapplied cash to %s.",
				common.CountOf(len(req.Applications), "invoice"))
		},
		Render: invoicetools.RenderApplyPayment,
		Run: func(
			ctx context.Context,
			req *customerpaymentports.ApplyCustomerPaymentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.ApplyUnapplied(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func applyPaymentRequest(
	params *agenttoolports.ToolExecuteParams,
) (*customerpaymentports.ApplyCustomerPaymentRequest, error) {
	paymentID, err := common.RequirePulid(params.Params, ParamCustomerPaymentID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := common.RequireDay(params.Params, common.ParamAccountingDate)
	if err != nil {
		return nil, err
	}
	applications, _, err := common.ReadPaymentApplications(params.Params, common.ParamApplications)
	if err != nil {
		return nil, err
	}
	if len(applications) == 0 {
		return nil, errApplicationsRequired
	}

	return &customerpaymentports.ApplyCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: accountingDate,
		Applications:   applications,
		TenantInfo:     common.TenantFrom(*params),
	}, nil
}

func NewReverseCustomerPaymentTool(payments receivablesKeeper) agenttoolports.AgentTool {
	return common.NewReceivableTool(receivableMoneySpec(&common.ReceivableSpec{
		Name: "receivable_reverse_customer_payment",
		Description: "Propose reversing a posted customer payment that bounced, was charged back " +
			"or was recorded in error. Every invoice it paid is reopened by what it applied and a " +
			"reversing ledger entry takes the cash back out. It cannot be undone, so a person " +
			"always decides; say why.",
		Resource:  permission.ResourceCustomerPayment,
		Operation: permission.OpUpdate,
		Rationale: "Takes cash back off the books and reopens the invoices it paid; only a " +
			"person reverses a payment.",
		Properties: map[string]any{
			ParamCustomerPaymentID:     customerPaymentIDProperty(),
			common.ParamAccountingDate: accountingDateProperty(),
			common.ParamReason: common.StringProperty(
				"Why it is reversed, such as the bank's return reason.", maxReceivableReasonChars),
		},
		Required: []string{ParamCustomerPaymentID, common.ParamAccountingDate, common.ParamReason},
		Target:   targetCustomerPayment,
	}), common.ReceivablePlan[*customerpaymentports.ReverseCustomerPaymentRequest, *customerpaymentports.CustomerPaymentChangePreview]{
		Request: reversePaymentRequest,
		Plan: func(
			ctx context.Context,
			req *customerpaymentports.ReverseCustomerPaymentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*customerpaymentports.CustomerPaymentChangePreview, error) {
			return payments.PreviewReverse(ctx, req, params.Actor)
		},
		Refused: func(*customerpaymentports.ReverseCustomerPaymentRequest) string {
			return "Would reverse the payment."
		},
		Render: invoicetools.RenderReversePayment,
		Run: func(
			ctx context.Context,
			req *customerpaymentports.ReverseCustomerPaymentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.Reverse(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func reversePaymentRequest(
	params *agenttoolports.ToolExecuteParams,
) (*customerpaymentports.ReverseCustomerPaymentRequest, error) {
	paymentID, err := common.RequirePulid(params.Params, ParamCustomerPaymentID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := common.RequireDay(params.Params, common.ParamAccountingDate)
	if err != nil {
		return nil, err
	}
	reason, err := common.RequireBoundedText(
		params.Params,
		common.ParamReason,
		maxReceivableReasonChars,
	)
	if err != nil {
		return nil, err
	}

	return &customerpaymentports.ReverseCustomerPaymentRequest{
		PaymentID:      paymentID,
		AccountingDate: accountingDate,
		Reason:         reason,
		TenantInfo:     common.TenantFrom(*params),
	}, nil
}

func NewApplyCreditMemoTool(payments receivablesKeeper) agenttoolports.AgentTool {
	return common.NewReceivableTool(receivableMoneySpec(&common.ReceivableSpec{
		Name:   "receivable_apply_credit_memo",
		Recipe: []string{"receivable_list_open_items", "receivable_apply_credit_memo"},
		Description: "Propose settling a customer's open invoices with a posted credit memo of " +
			"theirs. Name each invoice and the amount of credit it takes; together they cannot " +
			"exceed what is left of the memo, nor any invoice its open balance. A person always " +
			"decides. Find the memo with invoice_list (billType CreditMemo).",
		Resource:  permission.ResourceCustomerPayment,
		Operation: permission.OpCreate,
		Rationale: "Uses a customer's credit to settle what they owe; only a person applies " +
			"credit.",
		Properties: map[string]any{
			paramCreditMemoID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The posted credit memo, from invoice_list or invoice_get."),
			common.ParamAccountingDate: accountingDateProperty(),
			common.ParamApplications: applicationsProperty(
				"The invoices the credit settles and how much goes to each.", false),
		},
		Required: []string{paramCreditMemoID, common.ParamAccountingDate, common.ParamApplications},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return common.TargetOf(params, paramCreditMemoID, permission.ResourceInvoice)
		},
	}), common.ReceivablePlan[*customerpaymentports.ApplyCreditMemoRequest, *customerpaymentports.CreditMemoApplicationPreview]{
		Request: applyCreditRequest,
		Plan: func(
			ctx context.Context,
			req *customerpaymentports.ApplyCreditMemoRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*customerpaymentports.CreditMemoApplicationPreview, error) {
			return payments.PreviewApplyCreditMemo(ctx, req, params.Actor)
		},
		Refused: func(req *customerpaymentports.ApplyCreditMemoRequest) string {
			return fmt.Sprintf("Would apply the credit memo to %s.",
				common.CountOf(len(req.Applications), "invoice"))
		},
		Render: common.RenderApplyCredit,
		Run: func(
			ctx context.Context,
			req *customerpaymentports.ApplyCreditMemoRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.ApplyCreditMemo(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func applyCreditRequest(
	params *agenttoolports.ToolExecuteParams,
) (*customerpaymentports.ApplyCreditMemoRequest, error) {
	memoID, err := common.RequirePulid(params.Params, paramCreditMemoID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := common.RequireDay(params.Params, common.ParamAccountingDate)
	if err != nil {
		return nil, err
	}
	read, _, err := common.ReadPaymentApplications(params.Params, common.ParamApplications)
	if err != nil {
		return nil, err
	}
	if len(read) == 0 {
		return nil, errApplicationsRequired
	}
	applications := make([]*customerpaymentports.CreditMemoApplicationInput, 0, len(read))
	for idx, application := range read {
		if application.ShortPayAmountMinor > 0 {
			return nil, fmt.Errorf("%s[%d]: a credit memo writes nothing off; leave out %s",
				common.ParamApplications, idx, common.ParamShortPayAmount)
		}
		applications = append(applications, &customerpaymentports.CreditMemoApplicationInput{
			InvoiceID:          application.InvoiceID,
			AppliedAmountMinor: application.AppliedAmountMinor,
		})
	}

	return &customerpaymentports.ApplyCreditMemoRequest{
		CreditMemoID:   memoID,
		AccountingDate: accountingDate,
		Applications:   applications,
		TenantInfo:     common.TenantFrom(*params),
	}, nil
}

func NewUnapplyCreditMemoTool(payments receivablesKeeper) agenttoolports.AgentTool {
	return common.NewReceivableTool(receivableMoneySpec(&common.ReceivableSpec{
		Name: "receivable_unapply_credit_memo",
		Description: "Propose taking back one credit memo application: the invoice it settled " +
			"is reopened by that amount and the credit is available again. Use it when credit " +
			"went to the wrong invoice. A person always decides; say why.",
		Resource:  permission.ResourceCustomerPayment,
		Operation: permission.OpUpdate,
		Rationale: "Reopens an invoice's balance and restores a customer's credit; only a " +
			"person moves credit.",
		Properties: map[string]any{
			ParamCreditMemoApplicationID: agenttoolschema.KindID(
				"The application, from receivable_list_credit_memo_applications. Never guess one.",
				permission.KindCreditMemoApplication),
			common.ParamReason: common.StringProperty(
				"Why it is taken back.",
				maxReceivableReasonChars,
			),
		},
		Required: []string{ParamCreditMemoApplicationID, common.ParamReason},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return common.TargetOf(
				params, ParamCreditMemoApplicationID, agenttoolports.RecordCreditMemoApplication,
			)
		},
	}), common.ReceivablePlan[*customerpaymentports.UnapplyCreditMemoApplicationRequest, *customerpaymentports.CreditMemoApplicationPreview]{
		Request: unapplyCreditRequest,
		Plan: func(
			ctx context.Context,
			req *customerpaymentports.UnapplyCreditMemoApplicationRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*customerpaymentports.CreditMemoApplicationPreview, error) {
			return payments.PreviewUnapplyCreditMemoApplication(ctx, req, params.Actor)
		},
		Refused: func(*customerpaymentports.UnapplyCreditMemoApplicationRequest) string {
			return "Would take back the credit memo application."
		},
		Render: common.RenderUnapplyCredit,
		Run: func(
			ctx context.Context,
			req *customerpaymentports.UnapplyCreditMemoApplicationRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := payments.UnapplyCreditMemoApplication(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func unapplyCreditRequest(
	params *agenttoolports.ToolExecuteParams,
) (*customerpaymentports.UnapplyCreditMemoApplicationRequest, error) {
	applicationID, err := common.RequirePulid(params.Params, ParamCreditMemoApplicationID)
	if err != nil {
		return nil, err
	}
	reason, err := common.RequireBoundedText(
		params.Params,
		common.ParamReason,
		maxReceivableReasonChars,
	)
	if err != nil {
		return nil, err
	}

	return &customerpaymentports.UnapplyCreditMemoApplicationRequest{
		ApplicationID: applicationID,
		Reason:        reason,
		TenantInfo:    common.TenantFrom(*params),
	}, nil
}
