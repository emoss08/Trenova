package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramInvoiceRunID      = "invoiceRunId"
	paramCustomerID        = "customerId"
	paramCustomerIDs       = "customerIds"
	paramPeriodStart       = "periodStart"
	paramPeriodEnd         = "periodEnd"
	paramInvoiceDate       = "invoiceDate"
	paramExclude           = "exclude"
	paramInclude           = "include"
	paramMoves             = "moves"
	paramItemID            = "itemId"
	paramTargetGroupID     = "targetGroupId"
	paramBillingQueueItems = "billingQueueItemId"
	maxRunCustomers        = 200
	maxRunEdits            = 500
	maxRunReasonChars      = 1000
	dayLayout              = "2006-01-02"
)

var errNoMembershipEdit = errors.New("name at least one shipment to exclude, include or move")

type invoiceRunKeeper interface {
	Preview(
		ctx context.Context,
		req *serviceports.PreviewInvoiceRunRequest,
		actor *serviceports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	PreviewBuild(
		ctx context.Context,
		req *serviceports.PreviewInvoiceRunRequest,
		actor *serviceports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	AdjustMembership(
		ctx context.Context,
		req *serviceports.AdjustInvoiceRunMembershipRequest,
		actor *serviceports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	PreviewMembership(
		ctx context.Context,
		req *serviceports.AdjustInvoiceRunMembershipRequest,
	) (*serviceports.InvoiceRunChangePreview, error)
	Commit(
		ctx context.Context,
		req *serviceports.CommitInvoiceRunRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CommitInvoiceRunResult, error)
	PreviewCommit(
		ctx context.Context,
		req *serviceports.CommitInvoiceRunRequest,
	) (*serviceports.InvoiceRunCommitPlan, error)
	Cancel(
		ctx context.Context,
		req *serviceports.CancelInvoiceRunRequest,
		actor *serviceports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	PreviewCancel(
		ctx context.Context,
		req *serviceports.CancelInvoiceRunRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceRunChangePreview, error)
	BillStatementNow(
		ctx context.Context,
		req *serviceports.BillStatementNowRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CommitInvoiceRunResult, error)
	PreviewBillStatement(
		ctx context.Context,
		req *serviceports.BillStatementNowRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.StatementBillPlan, error)
}

func targetInvoiceRun(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramInvoiceRunID, permission.ResourceInvoiceRun)
}

func invoiceRunIDProperty() map[string]any {
	return stringProperty("The invoice run, from list_invoice_runs or get_invoice_run. Never "+
		"guess one.", 0)
}

func dayProperty(description string) map[string]any {
	return stringProperty(description+" YYYY-MM-DD, a UTC day.", 0)
}

func requireUTCDay(params map[string]any, key string) (time.Time, error) {
	raw, err := requireString(params, key)
	if err != nil {
		return time.Time{}, err
	}
	day, err := time.Parse(dayLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parameter %q must be YYYY-MM-DD, got %q", key, raw)
	}

	return day, nil
}

func newBuildInvoiceRunTool(runs invoiceRunKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "build_invoice_run",
		description: "Build an invoice run: a reviewable proposal of the consolidated invoices " +
			"for customers' approved, uninvoiced freight over a period. Nothing is invoiced " +
			"until the run is committed with commit_invoice_run. Find the run afterwards with " +
			"list_invoice_runs, newest first.",
		resource:    permission.ResourceInvoiceRun,
		operation:   permission.OpCreate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Builds a proposal of invoices inside Trenova for a biller to review; it " +
			"invoices nothing and is discarded with cancel_invoice_run.",
		properties: map[string]any{
			paramCustomerIDs: idListProperty("The customers to bill, from list_customers or "+
				"list_open_statements.", maxRunCustomers),
			paramPeriodStart: dayProperty("The first day of the period,"),
			paramPeriodEnd:   dayProperty("The last day of the period, included,"),
			paramInvoiceDate: dayProperty("The date the invoices carry, defaulting to today;"),
		},
		required: []string{paramCustomerIDs, paramPeriodStart, paramPeriodEnd},
	}, receivablePlan[*serviceports.PreviewInvoiceRunRequest, *invoicerun.InvoiceRun]{
		request: buildRunRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.PreviewInvoiceRunRequest,
			params *serviceports.ToolExecuteParams,
		) (*invoicerun.InvoiceRun, error) {
			return runs.PreviewBuild(ctx, req, params.Actor)
		},
		refused: func(req *serviceports.PreviewInvoiceRunRequest) string {
			return fmt.Sprintf("Would build an invoice run for %s.",
				countOf(len(req.CustomerIDs), "customer"))
		},
		render: renderBuiltRun,
		run: func(
			ctx context.Context,
			req *serviceports.PreviewInvoiceRunRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := runs.Preview(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func buildRunRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.PreviewInvoiceRunRequest, error) {
	customerIDs, err := requirePulidSlice(params.Params, paramCustomerIDs, maxRunCustomers)
	if err != nil {
		return nil, err
	}
	start, err := requireUTCDay(params.Params, paramPeriodStart)
	if err != nil {
		return nil, err
	}
	end, err := requireUTCDay(params.Params, paramPeriodEnd)
	if err != nil {
		return nil, err
	}
	if end.Before(start) {
		return nil, fmt.Errorf("%s %s is before %s %s", paramPeriodEnd, end.Format(dayLayout),
			paramPeriodStart, start.Format(dayLayout))
	}

	req := &serviceports.PreviewInvoiceRunRequest{
		TenantInfo:  tenantFrom(*params),
		CustomerIDs: customerIDs,
		PeriodStart: start.Unix(),
		PeriodEnd:   end.AddDate(0, 0, 1).Unix(),
		Source:      invoicerun.SourceManual,
	}
	if optionalString(params.Params, paramInvoiceDate) != "" {
		invoiceDate, dateErr := requireUTCDay(params.Params, paramInvoiceDate)
		if dateErr != nil {
			return nil, dateErr
		}
		req.InvoiceDate = invoiceDate.Unix()
	}

	return req, nil
}

func newAdjustInvoiceRunMembershipTool(runs invoiceRunKeeper) serviceports.AgentTool {
	itemProperty := stringProperty("A shipment on the run, by the item id get_invoice_run "+
		"lists under each group.", 0)

	return newReceivableTool(receivableSpec{
		name: "adjust_invoice_run_membership",
		description: "Take shipments off an invoice run that is still a proposal, put them back, " +
			"or move them to another of the same customer's invoices on it. An excluded shipment " +
			"stays approved and rolls into the next period. Every exclusion needs a reason the " +
			"next biller can read.",
		resource:    permission.ResourceInvoiceRun,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		taintHold: "An exclusion read from what a customer sent is proposed, since it " +
			"decides what they are billed this period.",
		rationale: "Edits a proposal of invoices before anything is invoiced; each change is " +
			"undone by the opposite edit.",
		properties: map[string]any{
			paramInvoiceRunID: invoiceRunIDProperty(),
			paramExclude: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Shipments to take off the run, each with why.",
				toolschema.KeyMaxItems:    maxRunEdits,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramItemID: itemProperty,
						paramReason: stringProperty("Why it is held back.", maxRunReasonChars),
					},
					toolschema.KeyRequired:             []string{paramItemID, paramReason},
					toolschema.KeyAdditionalProperties: false,
				},
			},
			paramInclude: idListProperty("Excluded shipments to put back, by the item id "+
				"get_invoice_run lists.", maxRunEdits),
			paramMoves: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Shipments to move to another invoice of the same customer.",
				toolschema.KeyMaxItems:    maxRunEdits,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramItemID: itemProperty,
						paramTargetGroupID: stringProperty(
							"The invoice it moves to, by the group id get_invoice_run lists.", 0),
					},
					toolschema.KeyRequired:             []string{paramItemID, paramTargetGroupID},
					toolschema.KeyAdditionalProperties: false,
				},
			},
		},
		required: []string{paramInvoiceRunID},
		target:   targetInvoiceRun,
	}, receivablePlan[*serviceports.AdjustInvoiceRunMembershipRequest, *serviceports.InvoiceRunChangePreview]{
		request: membershipRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.AdjustInvoiceRunMembershipRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceRunChangePreview, error) {
			return runs.PreviewMembership(ctx, req)
		},
		refused: func(*serviceports.AdjustInvoiceRunMembershipRequest) string {
			return "Would change which shipments the invoice run bills."
		},
		render: renderMembership,
		run: func(
			ctx context.Context,
			req *serviceports.AdjustInvoiceRunMembershipRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := runs.AdjustMembership(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func membershipRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.AdjustInvoiceRunMembershipRequest, error) {
	runID, err := requirePulid(params.Params, paramInvoiceRunID)
	if err != nil {
		return nil, err
	}
	req := &serviceports.AdjustInvoiceRunMembershipRequest{
		TenantInfo: tenantFrom(*params),
		RunID:      runID,
	}

	if req.Exclude, err = readRunExclusions(params.Params); err != nil {
		return nil, err
	}
	if _, ok := params.Params[paramInclude]; ok {
		if req.Include, err = requirePulidSlice(params.Params, paramInclude, maxRunEdits); err != nil {
			return nil, err
		}
	}
	if req.Moves, err = readRunMoves(params.Params); err != nil {
		return nil, err
	}
	if len(req.Exclude)+len(req.Include)+len(req.Moves) == 0 {
		return nil, errNoMembershipEdit
	}

	return req, nil
}

func readObjects(params map[string]any, key string, limit int) ([]map[string]any, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list of objects", key)
	}
	if len(entries) > limit {
		return nil, fmt.Errorf("%s lists %d entries; at most %d", key, len(entries), limit)
	}

	objects := make([]map[string]any, 0, len(entries))
	for idx, entry := range entries {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", key, idx)
		}
		objects = append(objects, fields)
	}

	return objects, nil
}

func readRunExclusions(params map[string]any) ([]serviceports.ItemExclusion, error) {
	entries, err := readObjects(params, paramExclude, maxRunEdits)
	if err != nil {
		return nil, err
	}

	exclusions := make([]serviceports.ItemExclusion, 0, len(entries))
	for idx, fields := range entries {
		itemID, idErr := requirePulid(fields, paramItemID)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, idErr)
		}
		reason, reasonErr := requireBoundedText(fields, paramReason, maxRunReasonChars)
		if reasonErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, reasonErr)
		}
		exclusions = append(exclusions, serviceports.ItemExclusion{ItemID: itemID, Reason: reason})
	}

	return exclusions, nil
}

func readRunMoves(params map[string]any) ([]serviceports.ItemMove, error) {
	entries, err := readObjects(params, paramMoves, maxRunEdits)
	if err != nil {
		return nil, err
	}

	moves := make([]serviceports.ItemMove, 0, len(entries))
	for idx, fields := range entries {
		itemID, idErr := requirePulid(fields, paramItemID)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramMoves, idx, idErr)
		}
		groupID, groupErr := requirePulid(fields, paramTargetGroupID)
		if groupErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramMoves, idx, groupErr)
		}
		moves = append(moves, serviceports.ItemMove{ItemID: itemID, TargetGroupID: groupID})
	}

	return moves, nil
}

func commitRunRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.CommitInvoiceRunRequest, error) {
	runID, err := requirePulid(params.Params, paramInvoiceRunID)
	if err != nil {
		return nil, err
	}

	return &serviceports.CommitInvoiceRunRequest{TenantInfo: tenantFrom(*params), RunID: runID}, nil
}

func newCommitInvoiceRunTool(runs invoiceRunKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "commit_invoice_run",
		description: "Propose committing an invoice run, which turns each of its proposed " +
			"invoices into a draft invoice for the customer. A group below the customer's " +
			"minimum, or whose shipments were invoiced or re-approved elsewhere since it was " +
			"built, is skipped with a reason. A person always decides.",
		resource:    permission.ResourceInvoiceRun,
		operation:   permission.OpApprove,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Issues the invoices a run proposes and advances the customers' billing " +
			"period; only a person commits a run.",
		properties: map[string]any{paramInvoiceRunID: invoiceRunIDProperty()},
		required:   []string{paramInvoiceRunID},
		target:     targetInvoiceRun,
	}, receivablePlan[*serviceports.CommitInvoiceRunRequest, *serviceports.InvoiceRunCommitPlan]{
		request: commitRunRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.CommitInvoiceRunRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceRunCommitPlan, error) {
			return runs.PreviewCommit(ctx, req)
		},
		refused: func(*serviceports.CommitInvoiceRunRequest) string {
			return "Would commit the invoice run."
		},
		render: renderCommit,
		run: func(
			ctx context.Context,
			req *serviceports.CommitInvoiceRunRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := runs.Commit(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func newCancelInvoiceRunTool(runs invoiceRunKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "cancel_invoice_run",
		description: "Cancel an invoice run that will never be committed, such as one built " +
			"for the wrong period or customers. Its shipments stay approved for the next run. " +
			"Say why; the reason is kept on the run.",
		resource:    permission.ResourceInvoiceRun,
		operation:   permission.OpCancel,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Discards a proposal of invoices; nothing was invoiced, and a new run is " +
			"built with build_invoice_run.",
		properties: map[string]any{
			paramInvoiceRunID: invoiceRunIDProperty(),
			paramReason:       stringProperty("Why it is canceled.", maxRunReasonChars),
		},
		required: []string{paramInvoiceRunID, paramReason},
		target:   targetInvoiceRun,
	}, receivablePlan[*serviceports.CancelInvoiceRunRequest, *serviceports.InvoiceRunChangePreview]{
		request: func(params *serviceports.ToolExecuteParams) (*serviceports.CancelInvoiceRunRequest, error) {
			runID, err := requirePulid(params.Params, paramInvoiceRunID)
			if err != nil {
				return nil, err
			}
			reason, err := requireBoundedText(params.Params, paramReason, maxRunReasonChars)
			if err != nil {
				return nil, err
			}
			return &serviceports.CancelInvoiceRunRequest{
				TenantInfo: tenantFrom(*params),
				RunID:      runID,
				Reason:     reason,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *serviceports.CancelInvoiceRunRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceRunChangePreview, error) {
			return runs.PreviewCancel(ctx, req, params.Actor)
		},
		refused: func(*serviceports.CancelInvoiceRunRequest) string {
			return "Would cancel the invoice run."
		},
		render: renderCancel,
		run: func(
			ctx context.Context,
			req *serviceports.CancelInvoiceRunRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := runs.Cancel(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func newBillStatementNowTool(runs invoiceRunKeeper) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "bill_statement_now",
		description: "Propose billing a statement customer's open period now, before its cycle " +
			"closes, with the reason recorded on the run. Shipments you name in exclude stay on " +
			"the statement for the period's end. The customer's next boundary does not move. A " +
			"person always decides; read list_open_statements first.",
		resource:    permission.ResourceInvoiceRun,
		operation:   permission.OpApprove,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Invoices a customer off their agreed cycle; only a person decides to bill " +
			"early, and the reason stays on the run.",
		properties: map[string]any{
			paramCustomerID: stringProperty("The statement customer, from list_open_statements.", 0),
			paramReason: stringProperty("Why it is billed before the cycle closes.",
				maxRunReasonChars),
			paramExclude: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Shipments to leave on the statement for now.",
				toolschema.KeyMaxItems:    maxRunEdits,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramBillingQueueItems: stringProperty(
							"The shipment's billing queue item, from list_open_statements.", 0),
						paramReason: stringProperty("Why it is held back.", maxRunReasonChars),
					},
					toolschema.KeyRequired:             []string{paramBillingQueueItems, paramReason},
					toolschema.KeyAdditionalProperties: false,
				},
			},
		},
		required: []string{paramCustomerID, paramReason},
	}, receivablePlan[*serviceports.BillStatementNowRequest, *serviceports.StatementBillPlan]{
		request: billStatementRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.BillStatementNowRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.StatementBillPlan, error) {
			return runs.PreviewBillStatement(ctx, req, params.Actor)
		},
		refused: func(*serviceports.BillStatementNowRequest) string {
			return "Would bill the customer's open statement now."
		},
		render: renderStatementBill,
		run: func(
			ctx context.Context,
			req *serviceports.BillStatementNowRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := runs.BillStatementNow(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func billStatementRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.BillStatementNowRequest, error) {
	customerID, err := requirePulid(params.Params, paramCustomerID)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxRunReasonChars)
	if err != nil {
		return nil, err
	}
	entries, err := readObjects(params.Params, paramExclude, maxRunEdits)
	if err != nil {
		return nil, err
	}

	exclusions := make([]serviceports.StatementExclusion, 0, len(entries))
	seen := make(map[pulid.ID]struct{}, len(entries))
	for idx, fields := range entries {
		itemID, idErr := requirePulid(fields, paramBillingQueueItems)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, idErr)
		}
		if _, duplicate := seen[itemID]; duplicate {
			return nil, fmt.Errorf("%s[%d] repeats %s", paramExclude, idx, itemID)
		}
		seen[itemID] = struct{}{}
		why, reasonErr := requireBoundedText(fields, paramReason, maxRunReasonChars)
		if reasonErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, reasonErr)
		}
		exclusions = append(exclusions, serviceports.StatementExclusion{
			BillingQueueItemID: itemID,
			Reason:             why,
		})
	}

	return &serviceports.BillStatementNowRequest{
		TenantInfo: tenantFrom(*params),
		CustomerID: customerID,
		Reason:     reason,
		Exclude:    exclusions,
	}, nil
}
