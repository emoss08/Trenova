package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

const (
	paramDisputeID              = "disputeId"
	paramDisputeReasonCode      = "reasonCode"
	paramDisputedAmount         = "disputedAmount"
	paramDisputeNotes           = "notes"
	paramDisputeResolution      = "resolution"
	paramResolutionAdjustmentID = "resolutionAdjustmentId"
	paramResolutionNotes        = "resolutionNotes"
	maxDisputeNoteChars         = 2000
)

var (
	disputeReasonCodes = agenttoolschema.Source(
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
	disputeResolutions = agenttoolschema.Source(
		"invoice.disputeResolution",
		[]invoice.DisputeResolution{
			invoice.DisputeResolutionCreditIssued,
			invoice.DisputeResolutionInvoiceUpheld,
			invoice.DisputeResolutionRebilled,
			invoice.DisputeResolutionWrittenOff,
			invoice.DisputeResolutionCustomerWithdrew,
		},
	)
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
	return targetOf(params, paramDisputeID, permission.ResourceInvoiceDispute)
}

func disputeIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceInvoiceDispute, "The open dispute",
		"list_invoice_disputes")
}

func newOpenInvoiceDisputeTool(disputes invoiceDisputer) agentports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name:   "open_invoice_dispute",
		recipe: []string{"get_invoice", "open_invoice_dispute"},
		description: "Open a dispute on a posted invoice the customer is withholding payment on. " +
			"The invoice is marked Disputed until the case is resolved or withdrawn. The disputed " +
			"amount cannot exceed the open balance, and an invoice holds one open case at a time. " +
			"Record the customer's reason as they gave it; read get_invoice first.",
		resource:    permission.ResourceInvoiceDispute,
		operation:   permission.OpCreate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		taintHold: "A dispute opened from what a customer wrote is proposed, since the notes " +
			"and the amount are what collections acts on.",
		rationale: "Marks an invoice Disputed inside Trenova with the customer's reason; it moves " +
			"no money and is withdrawn by withdraw_invoice_dispute.",
		properties: map[string]any{
			paramInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The posted invoice, from list_invoices or get_invoice."),
			paramDisputeReasonCode: agenttoolschema.Enum(
				"Why the customer is withholding payment.", disputeReasonCodes),
			paramDisputedAmount: stringProperty(
				"How much of the invoice the customer disputes, as a decimal such as 125.00.", 0),
			paramDisputeNotes: stringProperty(
				"What the customer said, in a sentence collections can act on.",
				maxDisputeNoteChars),
		},
		required: []string{paramInvoiceID, paramDisputeReasonCode, paramDisputedAmount},
		target:   targetInvoice,
	}, receivablePlan[*invoiceports.OpenInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		request: openDisputeRequest,
		plan: func(
			ctx context.Context,
			req *invoiceports.OpenInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewOpen(ctx, req, params.Actor)
		},
		refused: func(*invoiceports.OpenInvoiceDisputeRequest) string {
			return "Would open a dispute on the invoice."
		},
		render: func(
			_ *invoiceports.OpenInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agent.ToolPreview, error) {
			return disputePreview(plan, "Would open a dispute on %[3]s for %[2]s (%[1]s).")
		},
		run: func(
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
	invoiceID, err := requirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	reason, err := requireEnum(params.Params, paramDisputeReasonCode, disputeReasonCodes.Values)
	if err != nil {
		return nil, err
	}
	amount, err := requireAmount(params.Params, paramDisputedAmount)
	if err != nil {
		return nil, err
	}
	notes, err := boundedText(params.Params, paramDisputeNotes, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.OpenInvoiceDisputeRequest{
		InvoiceID:      invoiceID,
		TenantInfo:     tenantFrom(*params),
		ReasonCode:     reason,
		DisputedAmount: amount,
		Notes:          notes,
	}, nil
}

func newResolveInvoiceDisputeTool(disputes invoiceDisputer) agentports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "resolve_invoice_dispute",
		recipe: []string{
			"list_invoice_disputes",
			"list_invoice_adjustments",
			"resolve_invoice_dispute",
		},
		description: "Propose closing an open invoice dispute with its outcome. A credit or a " +
			"write-off must name the executed adjustment that settled it, from " +
			"list_invoice_adjustments; an upheld invoice, a rebill or a customer who dropped it " +
			"need none. Resolving clears the invoice's Disputed flag. A person always decides.",
		resource:    permission.ResourceInvoiceDispute,
		operation:   permission.OpApprove,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Records the outcome of a customer's dispute and releases the invoice back " +
			"to collections; the outcome is a biller's call, so the agent proposes it.",
		properties: map[string]any{
			paramDisputeID: disputeIDProperty(),
			paramDisputeResolution: agenttoolschema.Enum(
				"How the dispute ended.", disputeResolutions),
			paramResolutionAdjustmentID: agenttoolschema.KindID(
				"The executed adjustment that credited or wrote off the amount, from "+
					"list_invoice_adjustments. Required for CreditIssued and WrittenOff.",
				permission.KindInvoiceAdjustment),
			paramResolutionNotes: stringProperty(
				"What was agreed with the customer.", maxDisputeNoteChars),
		},
		required: []string{paramDisputeID, paramDisputeResolution},
		target:   targetDispute,
	}, receivablePlan[*invoiceports.ResolveInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		request: resolveDisputeRequest,
		plan: func(
			ctx context.Context,
			req *invoiceports.ResolveInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewResolve(ctx, req, params.Actor)
		},
		refused: func(req *invoiceports.ResolveInvoiceDisputeRequest) string {
			return "Would resolve the dispute as " + string(req.Resolution) + "."
		},
		render: func(
			_ *invoiceports.ResolveInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agent.ToolPreview, error) {
			return disputePreview(plan, "Would resolve the dispute on %[3]s for %[2]s (%[1]s) as "+
				string(plan.After.Resolution)+".")
		},
		run: func(
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
	disputeID, err := requirePulid(params.Params, paramDisputeID)
	if err != nil {
		return nil, err
	}
	resolution, err := requireEnum(params.Params, paramDisputeResolution, disputeResolutions.Values)
	if err != nil {
		return nil, err
	}
	adjustmentID, _, err := optionalPulid(params.Params, paramResolutionAdjustmentID)
	if err != nil {
		return nil, err
	}
	notes, err := boundedText(params.Params, paramResolutionNotes, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.ResolveInvoiceDisputeRequest{
		DisputeID:              disputeID,
		TenantInfo:             tenantFrom(*params),
		Resolution:             resolution,
		ResolutionAdjustmentID: adjustmentID,
		ResolutionNotes:        notes,
	}, nil
}

func newWithdrawInvoiceDisputeTool(disputes invoiceDisputer) agentports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "withdraw_invoice_dispute",
		description: "Withdraw an open invoice dispute the customer no longer pursues, without " +
			"recording an outcome. The invoice's Disputed flag clears and it goes back to " +
			"collections as it was. Use resolve_invoice_dispute instead when there is an outcome " +
			"to record.",
		resource:    permission.ResourceInvoiceDispute,
		operation:   permission.OpCancel,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		taintHold: "A withdrawal read from what a customer wrote is proposed, since it sends " +
			"the invoice back to collections.",
		rationale: "Closes a dispute case inside Trenova with no outcome; it moves no money and " +
			"a new case can be opened with open_invoice_dispute.",
		properties: map[string]any{
			paramDisputeID: disputeIDProperty(),
			paramDisputeNotes: stringProperty(
				"Why the customer dropped it.", maxDisputeNoteChars),
		},
		required: []string{paramDisputeID},
		target:   targetDispute,
	}, receivablePlan[*invoiceports.WithdrawInvoiceDisputeRequest, *invoiceports.InvoiceDisputePreview]{
		request: withdrawDisputeRequest,
		plan: func(
			ctx context.Context,
			req *invoiceports.WithdrawInvoiceDisputeRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceDisputePreview, error) {
			return disputes.PreviewWithdraw(ctx, req, params.Actor)
		},
		refused: func(*invoiceports.WithdrawInvoiceDisputeRequest) string {
			return "Would withdraw the dispute."
		},
		render: func(
			_ *invoiceports.WithdrawInvoiceDisputeRequest,
			plan *invoiceports.InvoiceDisputePreview,
		) (*agent.ToolPreview, error) {
			return disputePreview(plan, "Would withdraw the dispute on %[3]s for %[2]s (%[1]s).")
		},
		run: func(
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
	disputeID, err := requirePulid(params.Params, paramDisputeID)
	if err != nil {
		return nil, err
	}
	notes, err := boundedText(params.Params, paramDisputeNotes, maxDisputeNoteChars)
	if err != nil {
		return nil, err
	}

	return &invoiceports.WithdrawInvoiceDisputeRequest{
		DisputeID:  disputeID,
		TenantInfo: tenantFrom(*params),
		Notes:      notes,
	}, nil
}
