package invoiceruntools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamInvoiceRunID = "invoiceRunId"

const paramExclude = "exclude"

const paramInclude = "include"

const paramMoves = "moves"

const paramTargetGroupID = "targetGroupId"

const paramBillingQueueItems = "billingQueueItemId"

const maxRunCustomers = 200

const maxRunEdits = 500

const maxRunReasonChars = 1000

var errNoMembershipEdit = errors.New("name at least one shipment to exclude, include or move")

type invoiceRunKeeper interface {
	Preview(
		ctx context.Context,
		req *invoicerunports.PreviewInvoiceRunRequest,
		actor *permissionports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	PreviewBuild(
		ctx context.Context,
		req *invoicerunports.PreviewInvoiceRunRequest,
		actor *permissionports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	AdjustMembership(
		ctx context.Context,
		req *invoicerunports.AdjustInvoiceRunMembershipRequest,
		actor *permissionports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	PreviewMembership(
		ctx context.Context,
		req *invoicerunports.AdjustInvoiceRunMembershipRequest,
	) (*invoicerunports.InvoiceRunChangePreview, error)
	Commit(
		ctx context.Context,
		req *invoicerunports.CommitInvoiceRunRequest,
		actor *permissionports.RequestActor,
	) (*invoicerunports.CommitInvoiceRunResult, error)
	PreviewCommit(
		ctx context.Context,
		req *invoicerunports.CommitInvoiceRunRequest,
	) (*invoicerunports.InvoiceRunCommitPlan, error)
	Cancel(
		ctx context.Context,
		req *invoicerunports.CancelInvoiceRunRequest,
		actor *permissionports.RequestActor,
	) (*invoicerun.InvoiceRun, error)
	PreviewCancel(
		ctx context.Context,
		req *invoicerunports.CancelInvoiceRunRequest,
		actor *permissionports.RequestActor,
	) (*invoicerunports.InvoiceRunChangePreview, error)
	BillStatementNow(
		ctx context.Context,
		req *invoicerunports.BillStatementNowRequest,
		actor *permissionports.RequestActor,
	) (*invoicerunports.CommitInvoiceRunResult, error)
	PreviewBillStatement(
		ctx context.Context,
		req *invoicerunports.BillStatementNowRequest,
		actor *permissionports.RequestActor,
	) (*invoicerunports.StatementBillPlan, error)
}

func targetInvoiceRun(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, ParamInvoiceRunID, permission.ResourceInvoiceRun)
}

func invoiceRunIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceInvoiceRun, "The invoice run",
		"invoice_list_runs or invoice_get_run")
}

func NewBuildInvoiceRunTool(runs invoiceRunKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_build_run",
		Description: "Build an invoice run: a reviewable proposal of the consolidated invoices " +
			"for customers' approved, uninvoiced freight over a period. Nothing is invoiced " +
			"until the run is committed with invoice_commit_run. Find the run afterwards with " +
			"invoice_list_runs, newest first.",
		Resource:    permission.ResourceInvoiceRun,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Builds a proposal of invoices inside Trenova for a biller to review; it " +
			"invoices nothing and is discarded with invoice_cancel_run.",
		Properties: map[string]any{
			toolparams.ParamCustomerIDs: agenttoolschema.RecordIDs(
				permission.ResourceCustomer,
				"The customers to bill, from customer_list or "+
					"invoice_list_open_statements.",
				maxRunCustomers,
			),
			toolparams.ParamPeriodStart: agenttoolschema.Date("The first day of the period."),
			toolparams.ParamPeriodEnd: agenttoolschema.Date(
				"The last day of the period, included.",
			),
			toolparams.ParamInvoiceDate: agenttoolschema.Date(
				"The date the invoices carry, defaulting to today.",
			),
		},
		Required: []string{
			toolparams.ParamCustomerIDs,
			toolparams.ParamPeriodStart,
			toolparams.ParamPeriodEnd,
		},
		SearchTerms:   []string{"statement billing", "consolidated billing", "bill the period"},
		Prerequisites: []string{"invoice_list_open_statements", "customer_list"},
		Recipe: []string{
			"invoice_list_open_statements",
			"invoice_build_run",
			"invoice_get_run",
			"invoice_commit_run",
		},
	}, toolspec.ReceivablePlan[*invoicerunports.PreviewInvoiceRunRequest, *invoicerun.InvoiceRun]{
		Request: buildRunRequest,
		Plan: func(
			ctx context.Context,
			req *invoicerunports.PreviewInvoiceRunRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoicerun.InvoiceRun, error) {
			return runs.PreviewBuild(ctx, req, params.Actor)
		},
		Refused: func(req *invoicerunports.PreviewInvoiceRunRequest) string {
			return fmt.Sprintf("Would build an invoice run for %s.",
				changepreview.CountOf(len(req.CustomerIDs), "customer"))
		},
		Render: RenderBuiltRun,
		Run: func(
			ctx context.Context,
			req *invoicerunports.PreviewInvoiceRunRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := runs.Preview(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func buildRunRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoicerunports.PreviewInvoiceRunRequest, error) {
	customerIDs, err := toolparams.RequirePulidSlice(
		params.Params,
		toolparams.ParamCustomerIDs,
		maxRunCustomers,
	)
	if err != nil {
		return nil, err
	}
	start, err := toolparams.RequireUTCDay(params.Params, toolparams.ParamPeriodStart)
	if err != nil {
		return nil, err
	}
	end, err := toolparams.RequireUTCDay(params.Params, toolparams.ParamPeriodEnd)
	if err != nil {
		return nil, err
	}
	if end.Before(start) {
		return nil, fmt.Errorf(
			"%s %s is before %s %s",
			toolparams.ParamPeriodEnd,
			end.Format(toolparams.DayLayout),
			toolparams.ParamPeriodStart,
			start.Format(toolparams.DayLayout),
		)
	}

	req := &invoicerunports.PreviewInvoiceRunRequest{
		TenantInfo:  toolparams.TenantFrom(params),
		CustomerIDs: customerIDs,
		PeriodStart: start.Unix(),
		PeriodEnd:   end.AddDate(0, 0, 1).Unix(),
		Source:      invoicerun.SourceManual,
	}
	if toolparams.OptionalString(params.Params, toolparams.ParamInvoiceDate) != "" {
		invoiceDate, dateErr := toolparams.RequireUTCDay(params.Params, toolparams.ParamInvoiceDate)
		if dateErr != nil {
			return nil, dateErr
		}
		req.InvoiceDate = invoiceDate.Unix()
	}

	return req, nil
}

func NewAdjustInvoiceRunMembershipTool(runs invoiceRunKeeper) agenttoolports.AgentTool {
	itemProperty := agenttoolschema.KindID("A shipment on the run, by the item id "+
		"invoice_get_run lists under each group.", recordkind.InvoiceRunItem)

	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_adjust_run",
		SearchTerms: []string{
			"exclude shipment",
			"remove from invoice run",
			"drop from invoice run",
		},
		Description: "Take shipments off an invoice run that is still a proposal, put them back, " +
			"or move them to another of the same customer's invoices on it. An excluded shipment " +
			"stays approved and rolls into the next period. Every exclusion needs a reason the " +
			"next biller can read.",
		Resource:    permission.ResourceInvoiceRun,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		TaintHold: "An exclusion read from what a customer sent is proposed, since it " +
			"decides what they are billed this period.",
		Rationale: "Edits a proposal of invoices before anything is invoiced; each change is " +
			"undone by the opposite edit.",
		Properties: map[string]any{
			ParamInvoiceRunID: invoiceRunIDProperty(),
			paramExclude: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Shipments to take off the run, each with why.",
				toolschema.KeyMaxItems:    maxRunEdits,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						toolparams.ParamItemID: itemProperty,
						toolparams.ParamReason: toolparams.StringProperty(
							"Why it is held back.",
							maxRunReasonChars,
						),
					},
					toolschema.KeyRequired: []string{
						toolparams.ParamItemID,
						toolparams.ParamReason,
					},
					toolschema.KeyAdditionalProperties: false,
				},
			},
			paramInclude: agenttoolschema.KindIDs("Excluded shipments to put back, by the item "+
				"id invoice_get_run lists.", maxRunEdits, recordkind.InvoiceRunItem),
			paramMoves: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Shipments to move to another invoice of the same customer.",
				toolschema.KeyMaxItems:    maxRunEdits,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						toolparams.ParamItemID: itemProperty,
						paramTargetGroupID: agenttoolschema.KindID(
							"The invoice it moves to, by the group id invoice_get_run lists.",
							recordkind.InvoiceRunGroup),
					},
					toolschema.KeyRequired: []string{
						toolparams.ParamItemID,
						paramTargetGroupID,
					},
					toolschema.KeyAdditionalProperties: false,
				},
			},
		},
		Required: []string{ParamInvoiceRunID},
		Target:   targetInvoiceRun,
	}, toolspec.ReceivablePlan[*invoicerunports.AdjustInvoiceRunMembershipRequest, *invoicerunports.InvoiceRunChangePreview]{
		Request: membershipRequest,
		Plan: func(
			ctx context.Context,
			req *invoicerunports.AdjustInvoiceRunMembershipRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*invoicerunports.InvoiceRunChangePreview, error) {
			return runs.PreviewMembership(ctx, req)
		},
		Refused: func(*invoicerunports.AdjustInvoiceRunMembershipRequest) string {
			return "Would change which shipments the invoice run bills."
		},
		Render: RenderMembership,
		Run: func(
			ctx context.Context,
			req *invoicerunports.AdjustInvoiceRunMembershipRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := runs.AdjustMembership(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func membershipRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoicerunports.AdjustInvoiceRunMembershipRequest, error) {
	runID, err := toolparams.RequirePulid(params.Params, ParamInvoiceRunID)
	if err != nil {
		return nil, err
	}
	req := &invoicerunports.AdjustInvoiceRunMembershipRequest{
		TenantInfo: toolparams.TenantFrom(params),
		RunID:      runID,
	}

	if req.Exclude, err = readRunExclusions(params.Params); err != nil {
		return nil, err
	}
	if _, ok := params.Params[paramInclude]; ok {
		if req.Include, err = toolparams.RequirePulidSlice(
			params.Params,
			paramInclude,
			maxRunEdits,
		); err != nil {
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

func readRunExclusions(params map[string]any) ([]invoicerunports.ItemExclusion, error) {
	entries, err := toolparams.ReadObjects(params, paramExclude, maxRunEdits)
	if err != nil {
		return nil, err
	}

	exclusions := make([]invoicerunports.ItemExclusion, 0, len(entries))
	for idx, fields := range entries {
		itemID, idErr := toolparams.RequirePulid(fields, toolparams.ParamItemID)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, idErr)
		}
		reason, reasonErr := toolparams.RequireBoundedText(
			fields,
			toolparams.ParamReason,
			maxRunReasonChars,
		)
		if reasonErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, reasonErr)
		}
		exclusions = append(
			exclusions,
			invoicerunports.ItemExclusion{ItemID: itemID, Reason: reason},
		)
	}

	return exclusions, nil
}

func readRunMoves(params map[string]any) ([]invoicerunports.ItemMove, error) {
	entries, err := toolparams.ReadObjects(params, paramMoves, maxRunEdits)
	if err != nil {
		return nil, err
	}

	moves := make([]invoicerunports.ItemMove, 0, len(entries))
	for idx, fields := range entries {
		itemID, idErr := toolparams.RequirePulid(fields, toolparams.ParamItemID)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramMoves, idx, idErr)
		}
		groupID, groupErr := toolparams.RequirePulid(fields, paramTargetGroupID)
		if groupErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramMoves, idx, groupErr)
		}
		moves = append(moves, invoicerunports.ItemMove{ItemID: itemID, TargetGroupID: groupID})
	}

	return moves, nil
}

func commitRunRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoicerunports.CommitInvoiceRunRequest, error) {
	runID, err := toolparams.RequirePulid(params.Params, ParamInvoiceRunID)
	if err != nil {
		return nil, err
	}

	return &invoicerunports.CommitInvoiceRunRequest{
		TenantInfo: toolparams.TenantFrom(params),
		RunID:      runID,
	}, nil
}

func NewCommitInvoiceRunTool(runs invoiceRunKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_commit_run",
		Description: "Propose committing an invoice run, which turns each of its proposed " +
			"invoices into a draft invoice for the customer. A group below the customer's " +
			"minimum, or whose shipments were invoiced or re-approved elsewhere since it was " +
			"built, is skipped with a reason. A person always decides.",
		Resource:    permission.ResourceInvoiceRun,
		Operation:   permission.OpApprove,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Issues the invoices a run proposes and advances the customers' billing " +
			"period; only a person commits a run.",
		Properties: map[string]any{ParamInvoiceRunID: invoiceRunIDProperty()},
		Required:   []string{ParamInvoiceRunID},
		Target:     targetInvoiceRun,
	}, toolspec.ReceivablePlan[*invoicerunports.CommitInvoiceRunRequest, *invoicerunports.InvoiceRunCommitPlan]{
		Request: commitRunRequest,
		Plan: func(
			ctx context.Context,
			req *invoicerunports.CommitInvoiceRunRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*invoicerunports.InvoiceRunCommitPlan, error) {
			return runs.PreviewCommit(ctx, req)
		},
		Refused: func(*invoicerunports.CommitInvoiceRunRequest) string {
			return "Would commit the invoice run."
		},
		Render: RenderCommit,
		Run: func(
			ctx context.Context,
			req *invoicerunports.CommitInvoiceRunRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := runs.Commit(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func NewCancelInvoiceRunTool(runs invoiceRunKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_cancel_run",
		Description: "Cancel an invoice run that will never be committed, such as one built " +
			"for the wrong period or customers. Its shipments stay approved for the next run. " +
			"Say why; the reason is kept on the run.",
		Resource:    permission.ResourceInvoiceRun,
		Operation:   permission.OpCancel,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Rationale: "Discards a proposal of invoices; nothing was invoiced, and a new run is " +
			"built with invoice_build_run.",
		Properties: map[string]any{
			ParamInvoiceRunID: invoiceRunIDProperty(),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why it is canceled.",
				maxRunReasonChars,
			),
		},
		Required: []string{ParamInvoiceRunID, toolparams.ParamReason},
		Target:   targetInvoiceRun,
	}, toolspec.ReceivablePlan[*invoicerunports.CancelInvoiceRunRequest, *invoicerunports.InvoiceRunChangePreview]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*invoicerunports.CancelInvoiceRunRequest, error) {
			runID, err := toolparams.RequirePulid(params.Params, ParamInvoiceRunID)
			if err != nil {
				return nil, err
			}
			reason, err := toolparams.RequireBoundedText(
				params.Params,
				toolparams.ParamReason,
				maxRunReasonChars,
			)
			if err != nil {
				return nil, err
			}
			return &invoicerunports.CancelInvoiceRunRequest{
				TenantInfo: toolparams.TenantFrom(params),
				RunID:      runID,
				Reason:     reason,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *invoicerunports.CancelInvoiceRunRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoicerunports.InvoiceRunChangePreview, error) {
			return runs.PreviewCancel(ctx, req, params.Actor)
		},
		Refused: func(*invoicerunports.CancelInvoiceRunRequest) string {
			return "Would cancel the invoice run."
		},
		Render: invoicepreview.RenderCancel,
		Run: func(
			ctx context.Context,
			req *invoicerunports.CancelInvoiceRunRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := runs.Cancel(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func NewBillStatementNowTool(runs invoiceRunKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_bill_statement",
		Description: "Propose billing a statement customer's open period now, before its cycle " +
			"closes, with the reason recorded on the run. Shipments you name in exclude stay on " +
			"the statement for the period's end. The customer's next boundary does not move. A " +
			"person always decides; read invoice_list_open_statements first.",
		Resource:    permission.ResourceInvoiceRun,
		Operation:   permission.OpApprove,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Invoices a customer off their agreed cycle; only a person decides to bill " +
			"early, and the reason stays on the run.",
		Properties: map[string]any{
			toolparams.ParamCustomerID: agenttoolschema.RecordIDText(
				permission.ResourceCustomer,
				"The statement customer, from invoice_list_open_statements.",
			),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why it is billed before the cycle closes.",
				maxRunReasonChars,
			),
			paramExclude: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Shipments to leave on the statement for now.",
				toolschema.KeyMaxItems:    maxRunEdits,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramBillingQueueItems: agenttoolschema.KindID(
							"The shipment's billing queue item, from invoice_list_open_statements.",
							recordkind.BillingQueueItem),
						toolparams.ParamReason: toolparams.StringProperty(
							"Why it is held back.",
							maxRunReasonChars,
						),
					},
					toolschema.KeyRequired: []string{
						paramBillingQueueItems,
						toolparams.ParamReason,
					},
					toolschema.KeyAdditionalProperties: false,
				},
			},
		},
		Required: []string{toolparams.ParamCustomerID, toolparams.ParamReason},
	}, toolspec.ReceivablePlan[*invoicerunports.BillStatementNowRequest, *invoicerunports.StatementBillPlan]{
		Request: billStatementRequest,
		Plan: func(
			ctx context.Context,
			req *invoicerunports.BillStatementNowRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoicerunports.StatementBillPlan, error) {
			return runs.PreviewBillStatement(ctx, req, params.Actor)
		},
		Refused: func(*invoicerunports.BillStatementNowRequest) string {
			return "Would bill the customer's open statement now."
		},
		Render: RenderStatementBill,
		Run: func(
			ctx context.Context,
			req *invoicerunports.BillStatementNowRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := runs.BillStatementNow(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func billStatementRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoicerunports.BillStatementNowRequest, error) {
	customerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamCustomerID)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(
		params.Params,
		toolparams.ParamReason,
		maxRunReasonChars,
	)
	if err != nil {
		return nil, err
	}
	entries, err := toolparams.ReadObjects(params.Params, paramExclude, maxRunEdits)
	if err != nil {
		return nil, err
	}

	exclusions := make([]invoicerunports.StatementExclusion, 0, len(entries))
	seen := make(map[pulid.ID]struct{}, len(entries))
	for idx, fields := range entries {
		itemID, idErr := toolparams.RequirePulid(fields, paramBillingQueueItems)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, idErr)
		}
		if _, duplicate := seen[itemID]; duplicate {
			return nil, fmt.Errorf("%s[%d] repeats %s", paramExclude, idx, itemID)
		}
		seen[itemID] = struct{}{}
		why, reasonErr := toolparams.RequireBoundedText(
			fields,
			toolparams.ParamReason,
			maxRunReasonChars,
		)
		if reasonErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramExclude, idx, reasonErr)
		}
		exclusions = append(exclusions, invoicerunports.StatementExclusion{
			BillingQueueItemID: itemID,
			Reason:             why,
		})
	}

	return &invoicerunports.BillStatementNowRequest{
		TenantInfo: toolparams.TenantFrom(params),
		CustomerID: customerID,
		Reason:     reason,
		Exclude:    exclusions,
	}, nil
}
