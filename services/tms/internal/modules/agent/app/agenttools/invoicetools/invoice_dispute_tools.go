package invoicetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

const paramDisputeReasonCode = "reasonCode"

const paramDisputedAmount = "disputedAmount"

const paramDisputeNotes = "notes"

const paramDisputeResolution = "resolution"

const paramResolutionAdjustmentID = "resolutionAdjustmentId"

const paramResolutionNotes = "resolutionNotes"

const maxDisputeNoteChars = 2000

var disputeReasonCodes = agenttoolschema.Source(
	"invoice.disputeReasonCode",
	[]invoice.DisputeReasonCode{
		invoice.DisputeReasonRateDiscrepancy,
		invoice.DisputeReasonAccessorialDisputed,
		invoice.DisputeReasonServiceFailure,
		invoice.DisputeReasonDuplicateBilling,
		invoice.DisputeReasonWrongBillTo,
		invoice.DisputeReasonMissingDocumentation,
		invoice.DisputeReasonOther,
	},
)

var disputeResolutions = agenttoolschema.Source(
	"invoice.disputeResolution",
	[]invoice.DisputeResolution{
		invoice.DisputeResolutionCreditIssued,
		invoice.DisputeResolutionInvoiceUpheld,
		invoice.DisputeResolutionRebilled,
		invoice.DisputeResolutionWrittenOff,
		invoice.DisputeResolutionCustomerWithdrew,
	},
)

type invoiceDisputer interface {
	Open(
		ctx context.Context,
		req *invoiceports.OpenInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoice.InvoiceDispute, error)
	Resolve(
		ctx context.Context,
		req *invoiceports.ResolveInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoice.InvoiceDispute, error)
	Withdraw(
		ctx context.Context,
		req *invoiceports.WithdrawInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoice.InvoiceDispute, error)
	PreviewOpen(
		ctx context.Context,
		req *invoiceports.OpenInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceDisputePreview, error)
	PreviewResolve(
		ctx context.Context,
		req *invoiceports.ResolveInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceDisputePreview, error)
	PreviewWithdraw(
		ctx context.Context,
		req *invoiceports.WithdrawInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceDisputePreview, error)
}

func targetDispute(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamDisputeID, permission.ResourceInvoiceDispute)
}

func disputeIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceInvoiceDispute, "The open dispute",
		"list_invoice_disputes")
}

func NewOpenInvoiceDisputeTool(disputes invoiceDisputer) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
		Name:   "open_invoice_dispute",
		Recipe: []string{"get_invoice", "open_invoice_dispute"},
		Description: "Open a dispute on a posted invoice the customer is withholding payment on. " +
			"The invoice is marked Disputed until the case is resolved or withdrawn. The disputed " +
			"amount cannot exceed the open balance, and an invoice holds one open case at a time. " +
			"Record the customer's reason as they gave it; read get_invoice first.",
		Resource:    permission.ResourceInvoiceDispute,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		TaintHold: "A dispute opened from what a customer wrote is proposed, since the notes " +
			"and the amount are what collections acts on.",
		Rationale: "Marks an invoice Disputed inside Trenova with the customer's reason; it moves " +
			"no money and is withdrawn by withdraw_invoice_dispute.",
		Properties: map[string]any{
			common.ParamInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The posted invoice, from list_invoices or get_invoice."),
			paramDisputeReasonCode: agenttoolschema.Enum(
				"Why the customer is withholding payment.", disputeReasonCodes),
			paramDisputedAmount: common.StringProperty(
				"How much of the invoice the customer disputes, as a decimal such as 125.00.", 0),
			paramDisputeNotes: common.StringProperty(
				"What the customer said, in a sentence collections can act on.",
				maxDisputeNoteChars),
		},
		Required: []string{common.ParamInvoiceID, paramDisputeReasonCode, paramDisputedAmount},
		Target:   common.TargetInvoice,
	}, common.ReceivablePlan[*invoiceports.OpenInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		Request: openDisputeRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.OpenInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewOpen(ctx, req, params.Actor)
		},
		Refused: func(*invoiceports.OpenInvoiceDisputeRequest) string {
			return "Would open a dispute on the invoice."
		},
		Render: func(
			_ *invoiceports.OpenInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agent.ToolPreview, error) {
			return disputePreview(plan, "Would open a dispute on %[3]s for %[2]s (%[1]s).")
		},
		Run: func(
			ctx context.Context,
			req *invoiceports.OpenInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := disputes.Open(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func openDisputeRequest(
	params *agentports.ToolExecuteParams,
) (*invoiceports.OpenInvoiceDisputeRequest, error) {
	invoiceID, err := common.RequirePulid(params.Params, common.ParamInvoiceID)
	if err != nil {
		return nil, err
	}
	reason, err := common.RequireEnum(
		params.Params,
		paramDisputeReasonCode,
		disputeReasonCodes.Values,
	)
	if err != nil {
		return nil, err
	}
	amount, err := common.RequireAmount(params.Params, paramDisputedAmount)
	if err != nil {
		return nil, err
	}
	notes, err := common.BoundedText(params.Params, paramDisputeNotes, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.OpenInvoiceDisputeRequest{
		InvoiceID:      invoiceID,
		TenantInfo:     common.TenantFrom(*params),
		ReasonCode:     reason,
		DisputedAmount: amount,
		Notes:          notes,
	}, nil
}

func NewResolveInvoiceDisputeTool(disputes invoiceDisputer) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
		Name: "resolve_invoice_dispute",
		Recipe: []string{
			"list_invoice_disputes",
			"list_invoice_adjustments",
			"resolve_invoice_dispute",
		},
		Description: "Propose closing an open invoice dispute with its outcome. A credit or a " +
			"write-off must name the executed adjustment that settled it, from " +
			"list_invoice_adjustments; an upheld invoice, a rebill or a customer who dropped it " +
			"need none. Resolving clears the invoice's Disputed flag. A person always decides.",
		Resource:    permission.ResourceInvoiceDispute,
		Operation:   permission.OpApprove,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Records the outcome of a customer's dispute and releases the invoice back " +
			"to collections; the outcome is a biller's call, so the agent proposes it.",
		Properties: map[string]any{
			common.ParamDisputeID: disputeIDProperty(),
			paramDisputeResolution: agenttoolschema.Enum(
				"How the dispute ended.", disputeResolutions),
			paramResolutionAdjustmentID: agenttoolschema.KindID(
				"The executed adjustment that credited or wrote off the amount, from "+
					"list_invoice_adjustments. Required for CreditIssued and WrittenOff.",
				permission.KindInvoiceAdjustment),
			paramResolutionNotes: common.StringProperty(
				"What was agreed with the customer.", maxDisputeNoteChars),
		},
		Required: []string{common.ParamDisputeID, paramDisputeResolution},
		Target:   targetDispute,
	}, common.ReceivablePlan[*invoiceports.ResolveInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		Request: resolveDisputeRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.ResolveInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewResolve(ctx, req, params.Actor)
		},
		Refused: func(req *invoiceports.ResolveInvoiceDisputeRequest) string {
			return "Would resolve the dispute as " + string(req.Resolution) + "."
		},
		Render: func(
			_ *invoiceports.ResolveInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agent.ToolPreview, error) {
			return disputePreview(plan, "Would resolve the dispute on %[3]s for %[2]s (%[1]s) as "+
				string(plan.After.Resolution)+".")
		},
		Run: func(
			ctx context.Context,
			req *invoiceports.ResolveInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := disputes.Resolve(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func resolveDisputeRequest(
	params *agentports.ToolExecuteParams,
) (*invoiceports.ResolveInvoiceDisputeRequest, error) {
	disputeID, err := common.RequirePulid(params.Params, common.ParamDisputeID)
	if err != nil {
		return nil, err
	}
	resolution, err := common.RequireEnum(
		params.Params,
		paramDisputeResolution,
		disputeResolutions.Values,
	)
	if err != nil {
		return nil, err
	}
	adjustmentID, _, err := common.OptionalPulid(params.Params, paramResolutionAdjustmentID)
	if err != nil {
		return nil, err
	}
	notes, err := common.BoundedText(params.Params, paramResolutionNotes, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.ResolveInvoiceDisputeRequest{
		DisputeID:              disputeID,
		TenantInfo:             common.TenantFrom(*params),
		Resolution:             resolution,
		ResolutionAdjustmentID: adjustmentID,
		ResolutionNotes:        notes,
	}, nil
}

func NewWithdrawInvoiceDisputeTool(disputes invoiceDisputer) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
		Name: "withdraw_invoice_dispute",
		Description: "Withdraw an open invoice dispute the customer no longer pursues, without " +
			"recording an outcome. The invoice's Disputed flag clears and it goes back to " +
			"collections as it was. Use resolve_invoice_dispute instead when there is an outcome " +
			"to record.",
		Resource:    permission.ResourceInvoiceDispute,
		Operation:   permission.OpCancel,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		TaintHold: "A withdrawal read from what a customer wrote is proposed, since it sends " +
			"the invoice back to collections.",
		Rationale: "Closes a dispute case inside Trenova with no outcome; it moves no money and " +
			"a new case can be opened with open_invoice_dispute.",
		Properties: map[string]any{
			common.ParamDisputeID: disputeIDProperty(),
			paramDisputeNotes: common.StringProperty(
				"Why the customer dropped it.", maxDisputeNoteChars),
		},
		Required: []string{common.ParamDisputeID},
		Target:   targetDispute,
	}, common.ReceivablePlan[*invoiceports.WithdrawInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		Request: withdrawDisputeRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.WithdrawInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewWithdraw(ctx, req, params.Actor)
		},
		Refused: func(*invoiceports.WithdrawInvoiceDisputeRequest) string {
			return "Would withdraw the dispute."
		},
		Render: func(
			_ *invoiceports.WithdrawInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agent.ToolPreview, error) {
			return disputePreview(plan, "Would withdraw the dispute on %[3]s for %[2]s (%[1]s).")
		},
		Run: func(
			ctx context.Context,
			req *invoiceports.WithdrawInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := disputes.Withdraw(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func withdrawDisputeRequest(
	params *agentports.ToolExecuteParams,
) (*invoiceports.WithdrawInvoiceDisputeRequest, error) {
	disputeID, err := common.RequirePulid(params.Params, common.ParamDisputeID)
	if err != nil {
		return nil, err
	}
	notes, err := common.BoundedText(params.Params, paramDisputeNotes, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.WithdrawInvoiceDisputeRequest{
		DisputeID:  disputeID,
		TenantInfo: common.TenantFrom(*params),
		Notes:      notes,
	}, nil
}
