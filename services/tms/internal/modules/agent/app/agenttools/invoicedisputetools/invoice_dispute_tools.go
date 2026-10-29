package invoicedisputetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

const paramDisputeReasonCode = "reasonCode"

const paramDisputedAmount = "disputedAmount"

const fieldDisputeNotes = "notes"

const paramDisputeResolution = "resolution"

const paramResolutionAdjustmentID = "resolutionAdjustmentId"

const paramResolutionNotes = "resolutionNotes"

const maxDisputeNoteChars = 2000

var disputeReasonCodes = agenttoolschema.Source(
	"invoice.disputeReasonCode",
	[]invoicedispute.DisputeReasonCode{
		invoicedispute.DisputeReasonRateDiscrepancy,
		invoicedispute.DisputeReasonAccessorialDisputed,
		invoicedispute.DisputeReasonServiceFailure,
		invoicedispute.DisputeReasonDuplicateBilling,
		invoicedispute.DisputeReasonWrongBillTo,
		invoicedispute.DisputeReasonMissingDocumentation,
		invoicedispute.DisputeReasonOther,
	},
)

var disputeResolutions = agenttoolschema.Source(
	"invoice.disputeResolution",
	[]invoicedispute.DisputeResolution{
		invoicedispute.DisputeResolutionCreditIssued,
		invoicedispute.DisputeResolutionInvoiceUpheld,
		invoicedispute.DisputeResolutionRebilled,
		invoicedispute.DisputeResolutionWrittenOff,
		invoicedispute.DisputeResolutionCustomerWithdrew,
	},
)

type invoiceDisputer interface {
	Open(
		ctx context.Context,
		req *invoiceports.OpenInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoicedispute.InvoiceDispute, error)
	Resolve(
		ctx context.Context,
		req *invoiceports.ResolveInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoicedispute.InvoiceDispute, error)
	Withdraw(
		ctx context.Context,
		req *invoiceports.WithdrawInvoiceDisputeRequest,
		actor *permissionports.RequestActor,
	) (*invoicedispute.InvoiceDispute, error)
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

func targetDispute(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamDisputeID, permission.ResourceInvoiceDispute)
}

func disputeIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceInvoiceDispute, "The open dispute",
		"invoice_list_disputes")
}

func NewOpenInvoiceDisputeTool(disputes invoiceDisputer) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name:   "invoice_open_dispute",
		Recipe: []string{toolparams.ToolInvoiceGet, "invoice_open_dispute"},
		Description: "Open a dispute on a posted invoice the customer is withholding payment on. " +
			"The invoice is marked Disputed until the case is resolved or withdrawn. The disputed " +
			"amount cannot exceed the open balance, and an invoice holds one open case at a time. " +
			"Record the customer's reason as they gave it; read invoice_get first.",
		Resource:    permission.ResourceInvoiceDispute,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		TaintHold: "A dispute opened from what a customer wrote is proposed, since the notes " +
			"and the amount are what collections acts on.",
		Rationale: "Marks an invoice Disputed inside Trenova with the customer's reason; it moves " +
			"no money and is withdrawn by invoice_withdraw_dispute.",
		Properties: map[string]any{
			toolparams.ParamInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The posted invoice, from invoice_list or invoice_get."),
			paramDisputeReasonCode: agenttoolschema.Enum(
				"Why the customer is withholding payment.", disputeReasonCodes),
			paramDisputedAmount: toolparams.StringProperty(
				"How much of the invoice the customer disputes, as a decimal such as 125.00.", 0),
			toolparams.ParamNote: toolparams.StringProperty(
				"What the customer said, in a sentence collections can act on.",
				maxDisputeNoteChars),
		},
		Required: []string{toolparams.ParamInvoiceID, paramDisputeReasonCode, paramDisputedAmount},
		Target:   toolparams.TargetInvoice,
	}, toolspec.ReceivablePlan[*invoiceports.OpenInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		Request: openDisputeRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.OpenInvoiceDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewOpen(ctx, req, params.Actor)
		},
		Refused: func(*invoiceports.OpenInvoiceDisputeRequest) string {
			return "Would open a dispute on the invoice."
		},
		Render: func(
			_ *invoiceports.OpenInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agentpreview.ToolPreview, error) {
			return disputePreview(plan, "Would open a dispute on %[3]s for %[2]s (%[1]s).")
		},
		Run: func(
			ctx context.Context,
			req *invoiceports.OpenInvoiceDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := disputes.Open(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func openDisputeRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.OpenInvoiceDisputeRequest, error) {
	invoiceID, err := toolparams.RequirePulid(params.Params, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireEnum(
		params.Params,
		paramDisputeReasonCode,
		disputeReasonCodes.Values,
	)
	if err != nil {
		return nil, err
	}
	amount, err := toolparams.RequireAmount(params.Params, paramDisputedAmount)
	if err != nil {
		return nil, err
	}
	notes, err := toolparams.BoundedText(params.Params, toolparams.ParamNote, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.OpenInvoiceDisputeRequest{
		InvoiceID:      invoiceID,
		TenantInfo:     toolparams.TenantFrom(params),
		ReasonCode:     reason,
		DisputedAmount: amount,
		Notes:          notes,
	}, nil
}

func NewResolveInvoiceDisputeTool(disputes invoiceDisputer) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_resolve_dispute",
		Recipe: []string{
			"invoice_list_disputes",
			"invoice_list_adjustments",
			"invoice_resolve_dispute",
		},
		Description: "Propose closing an open invoice dispute with its outcome. A credit or a " +
			"write-off must name the executed adjustment that settled it, from " +
			"invoice_list_adjustments; an upheld invoice, a rebill or a customer who dropped it " +
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
			toolparams.ParamDisputeID: disputeIDProperty(),
			paramDisputeResolution: agenttoolschema.Enum(
				"How the dispute ended.", disputeResolutions),
			paramResolutionAdjustmentID: agenttoolschema.KindID(
				"The executed adjustment that credited or wrote off the amount, from "+
					"invoice_list_adjustments. Required for CreditIssued and WrittenOff.",
				recordkind.InvoiceAdjustment),
			paramResolutionNotes: toolparams.StringProperty(
				"What was agreed with the customer.", maxDisputeNoteChars),
		},
		Required: []string{toolparams.ParamDisputeID, paramDisputeResolution},
		Target:   targetDispute,
	}, toolspec.ReceivablePlan[*invoiceports.ResolveInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		Request: resolveDisputeRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.ResolveInvoiceDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewResolve(ctx, req, params.Actor)
		},
		Refused: func(req *invoiceports.ResolveInvoiceDisputeRequest) string {
			return "Would resolve the dispute as " + string(req.Resolution) + "."
		},
		Render: func(
			_ *invoiceports.ResolveInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agentpreview.ToolPreview, error) {
			return disputePreview(plan, "Would resolve the dispute on %[3]s for %[2]s (%[1]s) as "+
				string(plan.After.Resolution)+".")
		},
		Run: func(
			ctx context.Context,
			req *invoiceports.ResolveInvoiceDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := disputes.Resolve(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func resolveDisputeRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.ResolveInvoiceDisputeRequest, error) {
	disputeID, err := toolparams.RequirePulid(params.Params, toolparams.ParamDisputeID)
	if err != nil {
		return nil, err
	}
	resolution, err := toolparams.RequireEnum(
		params.Params,
		paramDisputeResolution,
		disputeResolutions.Values,
	)
	if err != nil {
		return nil, err
	}
	adjustmentID, _, err := toolparams.OptionalPulid(params.Params, paramResolutionAdjustmentID)
	if err != nil {
		return nil, err
	}
	notes, err := toolparams.BoundedText(params.Params, paramResolutionNotes, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.ResolveInvoiceDisputeRequest{
		DisputeID:              disputeID,
		TenantInfo:             toolparams.TenantFrom(params),
		Resolution:             resolution,
		ResolutionAdjustmentID: adjustmentID,
		ResolutionNotes:        notes,
	}, nil
}

func NewWithdrawInvoiceDisputeTool(disputes invoiceDisputer) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_withdraw_dispute",
		Description: "Withdraw an open invoice dispute the customer no longer pursues, without " +
			"recording an outcome. The invoice's Disputed flag clears and it goes back to " +
			"collections as it was. Use invoice_resolve_dispute instead when there is an outcome " +
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
			"a new case can be opened with invoice_open_dispute.",
		Properties: map[string]any{
			toolparams.ParamDisputeID: disputeIDProperty(),
			toolparams.ParamNote: toolparams.StringProperty(
				"Why the customer dropped it.", maxDisputeNoteChars),
		},
		Required: []string{toolparams.ParamDisputeID},
		Target:   targetDispute,
	}, toolspec.ReceivablePlan[*invoiceports.WithdrawInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		Request: withdrawDisputeRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.WithdrawInvoiceDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewWithdraw(ctx, req, params.Actor)
		},
		Refused: func(*invoiceports.WithdrawInvoiceDisputeRequest) string {
			return "Would withdraw the dispute."
		},
		Render: func(
			_ *invoiceports.WithdrawInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agentpreview.ToolPreview, error) {
			return disputePreview(plan, "Would withdraw the dispute on %[3]s for %[2]s (%[1]s).")
		},
		Run: func(
			ctx context.Context,
			req *invoiceports.WithdrawInvoiceDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := disputes.Withdraw(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func withdrawDisputeRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.WithdrawInvoiceDisputeRequest, error) {
	disputeID, err := toolparams.RequirePulid(params.Params, toolparams.ParamDisputeID)
	if err != nil {
		return nil, err
	}
	notes, err := toolparams.BoundedText(params.Params, toolparams.ParamNote, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.WithdrawInvoiceDisputeRequest{
		DisputeID:  disputeID,
		TenantInfo: toolparams.TenantFrom(params),
		Notes:      notes,
	}, nil
}
