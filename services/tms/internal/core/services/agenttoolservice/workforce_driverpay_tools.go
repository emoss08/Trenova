package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramExpenseID             = "expenseId"
	paramDecision              = "decision"
	paramResolutionNote        = "resolutionNote"
	paramAdjustmentAmountMoney = "adjustmentAmount"
	paramAdjustmentLabel       = "adjustmentDescription"
	kindSettlementDispute      = "pay dispute"
	kindDriverExpense          = "driver expense"
	decisionApprove            = "Approve"
	decisionDeny               = "Deny"
	decisionReject             = "Reject"
	maxDisputeResolution       = 2000
	defaultPayCurrency         = "USD"
)

var (
	payDisputeFields = []string{
		fieldStatus, fieldCategory, "resolutionNote", wfFieldResolvedByID, fieldResolvedAt,
	}
	expenseFields = []string{
		fieldStatus, fieldDescription, "reviewNote", "reviewedById", fieldReviewedAt,
	}
	disputeDecisions = agenttoolschema.Source("settlementDispute.decision",
		[]string{decisionApprove, decisionDeny})
	expenseDecisions = agenttoolschema.Source("driverExpense.decision",
		[]string{decisionApprove, decisionReject})
)

type driverPayReviewer interface {
	PlanStartDisputeReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		disputeID pulid.ID,
		actor *serviceports.RequestActor,
	) (*driversettlementservice.DisputeChange, error)
	StartDisputeReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		disputeID pulid.ID,
		actor *serviceports.RequestActor,
	) (*driversettlement.Dispute, error)
	PlanResolveDispute(
		ctx context.Context,
		req *driversettlementservice.ResolveDisputeRequest,
		actor *serviceports.RequestActor,
	) (*driversettlementservice.ResolveDisputePlan, error)
	ResolveDispute(
		ctx context.Context,
		req *driversettlementservice.ResolveDisputeRequest,
		actor *serviceports.RequestActor,
	) (*driversettlement.Dispute, error)
	PlanReviewExpense(
		ctx context.Context,
		req *driversettlementservice.ReviewExpenseRequest,
		actor *serviceports.RequestActor,
	) (*driversettlementservice.ReviewExpensePlan, error)
	ReviewExpense(
		ctx context.Context,
		req *driversettlementservice.ReviewExpenseRequest,
		actor *serviceports.RequestActor,
	) (*driverpay.Expense, error)
}

var _ driverPayReviewer = (*driversettlementservice.Service)(nil)

func driverPayReviewToolProviders() []any {
	return []any{
		provideStartSettlementDisputeReviewTool,
		provideResolveSettlementDisputeTool,
		provideReviewDriverExpenseTool,
	}
}

func classifyDriverPayDecision(moneyMoves func(params map[string]any) bool) func(
	serviceports.ToolExecuteParams,
) serviceports.CallPolicy {
	return func(params serviceports.ToolExecuteParams) serviceports.CallPolicy {
		if moneyMoves(params.Params) {
			return serviceports.CallPolicy{Egress: agent.EgressMoney}
		}
		return serviceports.CallPolicy{Egress: agent.EgressDriverVisible}
	}
}

func approves(params map[string]any) bool {
	return strings.TrimSpace(optionalString(params, paramDecision)) == decisionApprove
}

func disputeMovesMoney(params map[string]any) bool {
	raw, given := params[paramAdjustmentAmountMoney]
	return approves(params) && given && raw != nil
}

func payDisputeIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceSettlementDispute, "The pay dispute",
		"get_driver_settlement's openDisputes, get_settlement_dispute or the page you are on")
}

func disputeRecord(dispute *driversettlement.Dispute) toolpreview.Record {
	return wfRecord(permission.ResourceSettlementDispute, dispute.ID,
		fmt.Sprintf("%s pay dispute", dispute.Category), dispute.Version)
}

func expenseRecord(expense *driverpay.Expense) toolpreview.Record {
	return wfRecord(permission.ResourceDriverExpense, expense.ID,
		"Expense: "+expense.Description, expense.Version)
}

func settlementCurrency(target *driversettlement.Settlement) string {
	if target == nil || target.CurrencyCode == "" {
		return defaultPayCurrency
	}
	return target.CurrencyCode
}

func settlementTargetText(target *driversettlement.Settlement) string {
	if target == nil {
		return "a new off-cycle settlement"
	}
	return "settlement " + target.SettlementNumber
}

type disputeStart struct {
	tenant    pagination.TenantInfo
	disputeID pulid.ID
}

func newStartSettlementDisputeReviewTool(reviewer driverPayReviewer) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"start_settlement_dispute_review",
		"Move an open pay dispute a driver raised to under review, so the driver and the "+
			"team see someone is looking into it. It decides nothing; "+
			"resolve_settlement_dispute does.",
		"Marks the dispute as being worked inside Trenova; the driver sees its status in "+
			"Dash, and resolving it moves it on.",
		permission.ResourceSettlementDispute,
		permission.OpUpdate,
	), map[string]any{paramDisputeID: payDisputeIDProperty()}, paramDisputeID), paramDisputeID,
		permission.ResourceSettlementDispute)
	spec.egress = agent.EgressDriverVisible
	spec.reversible = false
	spec.searchTerms = []string{
		"pay dispute", "settlement dispute", "driver complaint", "under review",
		"mark under review", "investigate dispute",
	}

	return newReportingReceivableTool(spec, receivablePlan[
		*disputeStart, *driversettlementservice.DisputeChange,
	]{
		request: func(params *serviceports.ToolExecuteParams) (*disputeStart, error) {
			id, err := requirePulid(params.Params, paramDisputeID)
			if err != nil {
				return nil, err
			}
			return &disputeStart{tenant: tenantFrom(*params), disputeID: id}, nil
		},
		plan: func(
			ctx context.Context,
			req *disputeStart,
			params *serviceports.ToolExecuteParams,
		) (*driversettlementservice.DisputeChange, error) {
			return reviewer.PlanStartDisputeReview(ctx, req.tenant, req.disputeID, params.Actor)
		},
		refused: func(*disputeStart) string { return "Would move a pay dispute to review." },
		render: func(
			_ *disputeStart,
			change *driversettlementservice.DisputeChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(disputeRecord(change.Before), change.Before,
				change.After, wfOptions(fieldStatus)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would move the pay dispute to under review.",
				recorded), nil
		},
		run: func(
			ctx context.Context,
			req *disputeStart,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := reviewer.StartDisputeReview(ctx, req.tenant, req.disputeID,
				params.Actor)
			if err != nil {
				return nil, err
			}
			return wfResult("moved to review", kindSettlementDispute, paramDisputeID,
				updated.ID, updated.WorkerID), nil
		},
	})
}

func resolvePayDisputeRequest(
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.ResolveDisputeRequest, error) {
	id, err := requirePulid(params.Params, paramDisputeID)
	if err != nil {
		return nil, err
	}
	decision, err := requireEnum(params.Params, paramDecision, disputeDecisions.Values)
	if err != nil {
		return nil, err
	}
	note, err := requireBoundedText(params.Params, paramResolutionNote, maxDisputeResolution)
	if err != nil {
		return nil, err
	}
	req := &driversettlementservice.ResolveDisputeRequest{
		TenantInfo:     tenantFrom(*params),
		DisputeID:      id,
		Approve:        decision == decisionApprove,
		ResolutionNote: note,
	}
	if raw, given := params.Params[paramAdjustmentAmountMoney]; !given || raw == nil {
		return req, nil
	}
	amount, err := requireSignedMoney(params.Params, paramAdjustmentAmountMoney)
	if err != nil {
		return nil, err
	}
	description, err := requireBoundedText(params.Params, paramAdjustmentLabel,
		maxAdjustmentDescription)
	if err != nil {
		return nil, err
	}
	payCodeID, err := optionalPulidParam(params.Params, paramPayCodeID)
	if err != nil {
		return nil, err
	}
	req.Adjustment = &driversettlementservice.AdjustmentLineInput{
		Description: description,
		AmountMinor: amount,
		PayCodeID:   payCodeID,
	}
	return req, nil
}

func newResolveSettlementDisputeTool(reviewer driverPayReviewer) serviceports.AgentTool {
	spec := targeting(personOnly(withSchema(wfSpec(
		"resolve_settlement_dispute",
		"Draft the decision on a driver's pay dispute for a person to approve. Approve or "+
			"deny it with a resolution note the driver reads, and for an approval, an "+
			"optional adjustment that pays the driver back on their open draft settlement, "+
			"or on a new off-cycle one when they have none. A denial carries no adjustment.",
		"Decides a driver's pay complaint, tells the driver, and can add pay to a "+
			"settlement, so a person approves it and it runs as them; a decided dispute "+
			"stays decided.",
		permission.ResourceSettlementDispute,
		permission.OpApprove,
	), map[string]any{
		paramDisputeID: payDisputeIDProperty(),
		paramDecision: agenttoolschema.Enum("Approve to side with the driver, Deny to "+
			"turn the dispute down.", disputeDecisions),
		paramResolutionNote: stringProperty("The decision in words the driver reads in "+
			"Dash. Required either way.", maxDisputeResolution),
		paramAdjustmentAmountMoney: amountProperty("For an approval, what to pay back as a " +
			"decimal such as 42.50; negative to take money back. Leave it out for none."),
		paramAdjustmentLabel: stringProperty("What the adjustment is for, as the driver "+
			"and payroll read it. Required with an adjustment.", maxAdjustmentDescription),
		paramPayCodeID: agenttoolschema.RecordIDText(permission.ResourcePayCode,
			"The pay code the adjustment posts under, from list_pay_codes. Leave it out for "+
				"the default account."),
	}, paramDisputeID, paramDecision, paramResolutionNote)), paramDisputeID,
		permission.ResourceSettlementDispute)
	spec.egress = agent.EgressDriverVisible
	spec.alsoEgress = []agent.EgressClass{agent.EgressMoney}
	spec.classify = classifyDriverPayDecision(disputeMovesMoney)
	spec.reversible = false
	spec.searchTerms = []string{"pay dispute", "settlement dispute", "short pay", "resolve"}
	spec.recipe = []string{
		"get_driver_settlement",
		"get_settlement_dispute",
		"resolve_settlement_dispute",
	}

	return newReportingReceivableTool(spec, receivablePlan[
		*driversettlementservice.ResolveDisputeRequest, *driversettlementservice.ResolveDisputePlan,
	]{
		request: resolvePayDisputeRequest,
		plan: func(
			ctx context.Context,
			req *driversettlementservice.ResolveDisputeRequest,
			params *serviceports.ToolExecuteParams,
		) (*driversettlementservice.ResolveDisputePlan, error) {
			return reviewer.PlanResolveDispute(ctx, req, params.Actor)
		},
		refused: func(*driversettlementservice.ResolveDisputeRequest) string {
			return "Would decide a driver's pay dispute."
		},
		render: func(
			req *driversettlementservice.ResolveDisputeRequest,
			plan *driversettlementservice.ResolveDisputePlan,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(disputeRecord(plan.Dispute.Before),
				plan.Dispute.Before, plan.Dispute.After,
				append(wfOptions(payDisputeFields...), toolpreview.Volatile(fieldResolvedAt))...)
			if err != nil {
				return nil, err
			}
			changes := []*agent.RecordChange{recorded}
			verb := "deny"
			if req.Approve {
				verb = "approve"
			}
			summary := fmt.Sprintf("Would %s the pay dispute and tell the driver.", verb)
			if req.Adjustment != nil {
				adjustment := toolpreview.Money(
					toolpreview.Record{
						Resource: permission.ResourceDriverSettlement,
						Label:    req.Adjustment.Description,
					},
					toolpreview.MoneyBlock(
						settlementCurrency(plan.AdjustmentTarget),
						agent.MoneyLine{
							Label: "Dispute adjustment",
							After: minorAmount(req.Adjustment.AmountMinor),
						},
					),
					toolpreview.SensitiveAs("amountMinor"),
				)
				adjustment.Operation = agent.PreviewOperationCreate
				changes = append(changes, adjustment)
				summary += fmt.Sprintf(" An adjustment of %s lands on %s.",
					money.FormatMinor(req.Adjustment.AmountMinor,
						settlementCurrency(plan.AdjustmentTarget)),
					settlementTargetText(plan.AdjustmentTarget))
			}
			return toolpreview.Build(summary, changes...), nil
		},
		run: func(
			ctx context.Context,
			req *driversettlementservice.ResolveDisputeRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := reviewer.ResolveDispute(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return wfResult(string(updated.Status), kindSettlementDispute, paramDisputeID,
				updated.ID, updated.WorkerID), nil
		},
	})
}

func reviewExpenseFrom(
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.ReviewExpenseRequest, error) {
	id, err := requirePulid(params.Params, paramExpenseID)
	if err != nil {
		return nil, err
	}
	decision, err := requireEnum(params.Params, paramDecision, expenseDecisions.Values)
	if err != nil {
		return nil, err
	}
	note, err := boundedText(params.Params, fieldNote, maxDisputeResolution)
	if err != nil {
		return nil, err
	}
	return &driversettlementservice.ReviewExpenseRequest{
		TenantInfo: tenantFrom(*params),
		ExpenseID:  id,
		Approve:    decision == decisionApprove,
		Note:       note,
	}, nil
}

func newReviewDriverExpenseTool(reviewer driverPayReviewer) serviceports.AgentTool {
	spec := targeting(personOnly(withSchema(wfSpec(
		"review_driver_expense",
		"Draft the decision on a driver's expense claim for a person to approve. Approve "+
			"it to reimburse it on the driver's open draft settlement, or a new off-cycle one, "+
			"or reject it with a note saying why. An "+
			"organization that requires receipts refuses an approval without one.",
		"Pays or refuses a driver's out-of-pocket expense and tells the driver, so a "+
			"person approves it and it runs as them; a reviewed expense stays reviewed.",
		permission.ResourceDriverExpense,
		permission.OpApprove,
	), map[string]any{
		paramExpenseID: agenttoolschema.RecordIDText(
			permission.ResourceDriverExpense,
			"The expense, from list_driver_expenses or the page you "+
				"are on. Never guess one.",
		),
		paramDecision: agenttoolschema.Enum("Approve to reimburse it, Reject to turn it down.",
			expenseDecisions),
		fieldNote: stringProperty("What the driver reads with the decision. Required "+
			"to reject.", maxDisputeResolution),
	}, paramExpenseID, paramDecision)), paramExpenseID, permission.ResourceDriverExpense)
	spec.egress = agent.EgressDriverVisible
	spec.alsoEgress = []agent.EgressClass{agent.EgressMoney}
	spec.classify = classifyDriverPayDecision(approves)
	spec.reversible = false
	spec.searchTerms = []string{"reimbursement", "expense", "receipt", "lumper"}

	return newReportingReceivableTool(spec, receivablePlan[
		*driversettlementservice.ReviewExpenseRequest, *driversettlementservice.ReviewExpensePlan,
	]{
		request: reviewExpenseFrom,

		plan: func(
			ctx context.Context,
			req *driversettlementservice.ReviewExpenseRequest,
			params *serviceports.ToolExecuteParams,
		) (*driversettlementservice.ReviewExpensePlan, error) {
			return reviewer.PlanReviewExpense(ctx, req, params.Actor)
		},
		refused: func(*driversettlementservice.ReviewExpenseRequest) string {
			return "Would review a driver's expense."
		},
		render: func(
			req *driversettlementservice.ReviewExpenseRequest,
			plan *driversettlementservice.ReviewExpensePlan,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(expenseRecord(plan.Before), plan.Before,
				plan.After,
				append(wfOptions(expenseFields...), toolpreview.Volatile(fieldReviewedAt))...)
			if err != nil {
				return nil, err
			}
			changes := []*agent.RecordChange{recorded}
			amount := money.FormatMinor(plan.Before.AmountMinor, plan.Before.CurrencyCode)
			summary := fmt.Sprintf("Would reject the %s expense and tell the driver why.",
				amount)
			if req.Approve {
				reimbursement := toolpreview.Money(
					toolpreview.Record{
						Resource: permission.ResourceDriverSettlement,
						Label:    plan.Before.Description,
					},
					toolpreview.MoneyBlock(plan.Before.CurrencyCode, agent.MoneyLine{
						Label: "Reimbursement",
						After: minorAmount(plan.Before.AmountMinor),
					}),
					toolpreview.SensitiveAs("amountMinor"),
				)
				reimbursement.Operation = agent.PreviewOperationCreate
				changes = append(changes, reimbursement)
				summary = fmt.Sprintf("Would reimburse the %s expense on %s and tell the "+
					"driver.", amount, settlementTargetText(plan.ReimbursementTarget))
			}
			return toolpreview.Build(summary, changes...), nil
		},
		run: func(
			ctx context.Context,
			req *driversettlementservice.ReviewExpenseRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := reviewer.ReviewExpense(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return wfResult(string(updated.Status), kindDriverExpense, paramExpenseID,
				updated.ID, updated.WorkerID), nil
		},
	})
}

func provideStartSettlementDisputeReviewTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return newStartSettlementDisputeReviewTool(s)
}

func provideResolveSettlementDisputeTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return newResolveSettlementDisputeTool(s)
}

func provideReviewDriverExpenseTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return newReviewDriverExpenseTool(s)
}
