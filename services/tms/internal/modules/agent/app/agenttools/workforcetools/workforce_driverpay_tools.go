package workforcetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramExpenseID = "expenseId"

const paramDecision = "decision"

const paramResolutionNote = "resolutionNote"

const paramAdjustmentAmountMoney = "adjustmentAmount"

const paramAdjustmentLabel = "adjustmentDescription"

const kindSettlementDispute = "pay dispute"

const kindDriverExpense = "driver expense"

const decisionApprove = "Approve"

const decisionDeny = "Deny"

const decisionReject = "Reject"

const maxDisputeResolution = 2000

const defaultPayCurrency = "USD"

var payDisputeFields = []string{
	common.FieldStatus,
	common.FieldCategory,
	"resolutionNote",
	wfFieldResolvedByID,
	common.FieldResolvedAt,
}

var expenseFields = []string{
	common.FieldStatus,
	common.FieldDescription,
	"reviewNote",
	"reviewedById",
	common.FieldReviewedAt,
}

var disputeDecisions = agenttoolschema.Source("settlementDispute.decision",
	[]string{decisionApprove, decisionDeny})

var expenseDecisions = agenttoolschema.Source("driverExpense.decision",
	[]string{decisionApprove, decisionReject})

type DriverPayReviewer interface {
	PlanStartDisputeReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		disputeID pulid.ID,
		actor *permissionports.RequestActor,
	) (*driversettlementservice.DisputeChange, error)
	StartDisputeReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		disputeID pulid.ID,
		actor *permissionports.RequestActor,
	) (*driversettlement.Dispute, error)
	PlanResolveDispute(
		ctx context.Context,
		req *driversettlementservice.ResolveDisputeRequest,
		actor *permissionports.RequestActor,
	) (*driversettlementservice.ResolveDisputePlan, error)
	ResolveDispute(
		ctx context.Context,
		req *driversettlementservice.ResolveDisputeRequest,
		actor *permissionports.RequestActor,
	) (*driversettlement.Dispute, error)
	PlanReviewExpense(
		ctx context.Context,
		req *driversettlementservice.ReviewExpenseRequest,
		actor *permissionports.RequestActor,
	) (*driversettlementservice.ReviewExpensePlan, error)
	ReviewExpense(
		ctx context.Context,
		req *driversettlementservice.ReviewExpenseRequest,
		actor *permissionports.RequestActor,
	) (*driverpay.Expense, error)
}

func DriverPayReviewToolProviders() []any {
	return []any{
		provideStartSettlementDisputeReviewTool,
		provideResolveSettlementDisputeTool,
		provideReviewDriverExpenseTool,
	}
}

func classifyDriverPayDecision(moneyMoves func(params map[string]any) bool) func(
	agenttoolports.ToolExecuteParams,
) agenttoolports.CallPolicy {
	return func(params agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy {
		if moneyMoves(params.Params) {
			return agenttoolports.CallPolicy{Egress: agent.EgressMoney}
		}
		return agenttoolports.CallPolicy{Egress: agent.EgressDriverVisible}
	}
}

func approves(params map[string]any) bool {
	return strings.TrimSpace(common.OptionalString(params, paramDecision)) == decisionApprove
}

func disputeMovesMoney(params map[string]any) bool {
	raw, given := params[paramAdjustmentAmountMoney]
	return approves(params) && given && raw != nil
}

func payDisputeIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceSettlementDispute, "The pay dispute",
		"settlement_get_driver's disputes, settlement_get_dispute or the page you are on")
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

func newStartSettlementDisputeReviewTool(reviewer DriverPayReviewer) agenttoolports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"settlement_start_dispute_review",
		"Move an open pay dispute a driver raised to under review, so the driver and the "+
			"team see someone is looking into it. It decides nothing; "+
			"settlement_resolve_dispute does.",
		"Marks the dispute as being worked inside Trenova; the driver sees its status in "+
			"Dash, and resolving it moves it on.",
		permission.ResourceSettlementDispute,
		permission.OpUpdate,
	), map[string]any{common.ParamDisputeID: payDisputeIDProperty()}, common.ParamDisputeID), common.ParamDisputeID,
		permission.ResourceSettlementDispute)
	spec.Egress = agent.EgressDriverVisible
	spec.Reversible = false
	spec.SearchTerms = []string{
		"pay dispute", "settlement dispute", "driver complaint", "under review",
		"mark under review", "investigate dispute",
	}

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*disputeStart, *driversettlementservice.DisputeChange,
	]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*disputeStart, error) {
			id, err := common.RequirePulid(params.Params, common.ParamDisputeID)
			if err != nil {
				return nil, err
			}
			return &disputeStart{tenant: common.TenantFrom(*params), disputeID: id}, nil
		},
		Plan: func(
			ctx context.Context,
			req *disputeStart,
			params *agenttoolports.ToolExecuteParams,
		) (*driversettlementservice.DisputeChange, error) {
			return reviewer.PlanStartDisputeReview(ctx, req.tenant, req.disputeID, params.Actor)
		},
		Refused: func(*disputeStart) string { return "Would move a pay dispute to review." },
		Render: func(
			_ *disputeStart,
			change *driversettlementservice.DisputeChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(disputeRecord(change.Before), change.Before,
				change.After, wfOptions(common.FieldStatus)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would move the pay dispute to under review.",
				recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *disputeStart,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := reviewer.StartDisputeReview(ctx, req.tenant, req.disputeID,
				params.Actor)
			if err != nil {
				return nil, err
			}
			return wfResult("moved to review", kindSettlementDispute, common.ParamDisputeID,
				updated.ID, updated.WorkerID), nil
		},
	})
}

func resolvePayDisputeRequest(
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.ResolveDisputeRequest, error) {
	id, err := common.RequirePulid(params.Params, common.ParamDisputeID)
	if err != nil {
		return nil, err
	}
	decision, err := common.RequireEnum(params.Params, paramDecision, disputeDecisions.Values)
	if err != nil {
		return nil, err
	}
	note, err := common.RequireBoundedText(params.Params, paramResolutionNote, maxDisputeResolution)
	if err != nil {
		return nil, err
	}
	req := &driversettlementservice.ResolveDisputeRequest{
		TenantInfo:     common.TenantFrom(*params),
		DisputeID:      id,
		Approve:        decision == decisionApprove,
		ResolutionNote: note,
	}
	if raw, given := params.Params[paramAdjustmentAmountMoney]; !given || raw == nil {
		return req, nil
	}
	amount, err := common.RequireSignedMoney(params.Params, paramAdjustmentAmountMoney)
	if err != nil {
		return nil, err
	}
	description, err := common.RequireBoundedText(params.Params, paramAdjustmentLabel,
		common.MaxAdjustmentDescription)
	if err != nil {
		return nil, err
	}
	payCodeID, err := common.OptionalPulidParam(params.Params, common.ParamPayCodeID)
	if err != nil {
		return nil, err
	}
	req.Adjustment = &driversettlementservice.AdjustmentLineInput{
		Description: description,
		AmountMinor: amount,
		PayCodeID:   payCodeID,
		ByAgent:     true,
	}
	return req, nil
}

func newResolveSettlementDisputeTool(reviewer DriverPayReviewer) agenttoolports.AgentTool {
	spec := targeting(personOnly(withSchema(wfSpec(
		"settlement_resolve_dispute",
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
		common.ParamDisputeID: payDisputeIDProperty(),
		paramDecision: agenttoolschema.Enum("Approve to side with the driver, Deny to "+
			"turn the dispute down.", disputeDecisions),
		paramResolutionNote: common.StringProperty("The decision in words the driver reads in "+
			"Dash. Required either way.", maxDisputeResolution),
		paramAdjustmentAmountMoney: common.AmountProperty(
			"For an approval, what to pay back as a " +
				"decimal such as 42.50; negative to take money back. Leave it out for none.",
		),
		paramAdjustmentLabel: common.StringProperty("What the adjustment is for, as the driver "+
			"and payroll read it. Required with an adjustment.", common.MaxAdjustmentDescription),
		common.ParamPayCodeID: agenttoolschema.RecordIDText(
			permission.ResourcePayCode,
			"The pay code the adjustment posts under, from settlement_list_pay_codes. Leave it out for "+
				"the default account.",
		),
	}, common.ParamDisputeID, paramDecision, paramResolutionNote)), common.ParamDisputeID,
		permission.ResourceSettlementDispute)
	spec.Egress = agent.EgressDriverVisible
	spec.AlsoEgress = []agent.EgressClass{agent.EgressMoney}
	spec.Classify = classifyDriverPayDecision(disputeMovesMoney)
	spec.Reversible = false
	spec.SearchTerms = []string{"pay dispute", "settlement dispute", "short pay", "resolve"}
	spec.Recipe = []string{
		"settlement_get_driver",
		"settlement_get_dispute",
		"settlement_resolve_dispute",
	}

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*driversettlementservice.ResolveDisputeRequest, *driversettlementservice.ResolveDisputePlan,
	]{
		Request: resolvePayDisputeRequest,
		Plan: func(
			ctx context.Context,
			req *driversettlementservice.ResolveDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*driversettlementservice.ResolveDisputePlan, error) {
			return reviewer.PlanResolveDispute(ctx, req, params.Actor)
		},
		Refused: func(*driversettlementservice.ResolveDisputeRequest) string {
			return "Would decide a driver's pay dispute."
		},
		Render: renderSettlementDisputeResolution,
		Run: func(
			ctx context.Context,
			req *driversettlementservice.ResolveDisputeRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := reviewer.ResolveDispute(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return wfResult(string(updated.Status), kindSettlementDispute, common.ParamDisputeID,
				updated.ID, updated.WorkerID), nil
		},
	})
}

func renderSettlementDisputeResolution(
	req *driversettlementservice.ResolveDisputeRequest,
	plan *driversettlementservice.ResolveDisputePlan,
) (*agent.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		disputeRecord(plan.Dispute.Before),
		plan.Dispute.Before,
		plan.Dispute.After,
		append(
			wfOptions(payDisputeFields...),
			toolpreview.Volatile(common.FieldResolvedAt),
		)...)
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
					After: common.MinorAmount(req.Adjustment.AmountMinor),
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
}

func reviewExpenseFrom(
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.ReviewExpenseRequest, error) {
	id, err := common.RequirePulid(params.Params, paramExpenseID)
	if err != nil {
		return nil, err
	}
	decision, err := common.RequireEnum(params.Params, paramDecision, expenseDecisions.Values)
	if err != nil {
		return nil, err
	}
	note, err := common.BoundedText(params.Params, common.FieldNote, maxDisputeResolution)
	if err != nil {
		return nil, err
	}
	return &driversettlementservice.ReviewExpenseRequest{
		TenantInfo: common.TenantFrom(*params),
		ExpenseID:  id,
		Approve:    decision == decisionApprove,
		Note:       note,
	}, nil
}

func newReviewDriverExpenseTool(reviewer DriverPayReviewer) agenttoolports.AgentTool {
	spec := targeting(personOnly(withSchema(wfSpec(
		"settlement_review_driver_expense",
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
			"The expense, from settlement_list_driver_expenses or the page you "+
				"are on. Never guess one.",
		),
		paramDecision: agenttoolschema.Enum("Approve to reimburse it, Reject to turn it down.",
			expenseDecisions),
		common.FieldNote: common.StringProperty(
			"What the driver reads with the decision. Required "+
				"to reject.",
			maxDisputeResolution,
		),
	}, paramExpenseID, paramDecision)), paramExpenseID, permission.ResourceDriverExpense)
	spec.Egress = agent.EgressDriverVisible
	spec.AlsoEgress = []agent.EgressClass{agent.EgressMoney}
	spec.Classify = classifyDriverPayDecision(approves)
	spec.Reversible = false
	spec.SearchTerms = []string{"reimbursement", "expense", "receipt", "lumper"}

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*driversettlementservice.ReviewExpenseRequest, *driversettlementservice.ReviewExpensePlan,
	]{
		Request: reviewExpenseFrom,

		Plan: func(
			ctx context.Context,
			req *driversettlementservice.ReviewExpenseRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*driversettlementservice.ReviewExpensePlan, error) {
			return reviewer.PlanReviewExpense(ctx, req, params.Actor)
		},
		Refused: func(*driversettlementservice.ReviewExpenseRequest) string {
			return "Would review a driver's expense."
		},
		Render: func(
			req *driversettlementservice.ReviewExpenseRequest,
			plan *driversettlementservice.ReviewExpensePlan,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(
				expenseRecord(plan.Before),
				plan.Before,
				plan.After,
				append(
					wfOptions(expenseFields...),
					toolpreview.Volatile(common.FieldReviewedAt),
				)...)
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
						After: common.MinorAmount(plan.Before.AmountMinor),
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
		Run: func(
			ctx context.Context,
			req *driversettlementservice.ReviewExpenseRequest,
			params *agenttoolports.ToolExecuteParams,
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
) agenttoolports.AgentTool {
	return newStartSettlementDisputeReviewTool(s)
}

func provideResolveSettlementDisputeTool(
	s *driversettlementservice.Service,
) agenttoolports.AgentTool {
	return newResolveSettlementDisputeTool(s)
}

func provideReviewDriverExpenseTool(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return newReviewDriverExpenseTool(s)
}
