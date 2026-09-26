package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	paramEscrowAccountID    = "accountId"
	paramEscrowTarget       = "targetAmount"
	paramEscrowInterestRate = "annualInterestRate"
	paramEscrowOpenedDate   = "openedDate"
	paramEscrowDescription  = "description"
	paramEscrowOccurredDate = "occurredDate"
	maxEscrowDescription    = 255
	escrowSourcesTool       = "list_escrow_accounts"
	escrowBalanceMark       = "balanceMinor"
	escrowTargetMark        = "targetAmountMinor"
	fieldAnnualInterestRate = "annualInterestRate"
	fieldClosedDate         = "closedDate"
)

type escrowBook interface {
	GetEscrowAccount(
		ctx context.Context,
		req repositories.GetEscrowAccountByIDRequest,
	) (*driverpay.EscrowAccount, error)
	PlanOpenEscrowAccount(ctx context.Context, entity *driverpay.EscrowAccount, now int64) error
	OpenEscrowAccount(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
		actor *serviceports.RequestActor,
	) (*driverpay.EscrowAccount, error)
	PlanUpdateEscrowAccount(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
	) (*driverpay.EscrowAccount, error)
	UpdateEscrowAccount(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
		actor *serviceports.RequestActor,
	) (*driverpay.EscrowAccount, error)
	AdjustEscrowAccount(
		ctx context.Context,
		req *driverpayservice.EscrowAdjustmentRequest,
		actor *serviceports.RequestActor,
	) (*driverpay.EscrowAccount, error)
	CloseEscrowAccount(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		accountID pulid.ID,
		actor *serviceports.RequestActor,
	) (*driverpay.EscrowAccount, error)
}

func escrowRecord(account *driverpay.EscrowAccount) toolpreview.Record {
	label := "Escrow account"
	if account.Worker != nil {
		label = "Escrow account for " + account.Worker.FullName()
	}

	return toolpreview.Record{
		Resource: permission.ResourceEscrowAccount,
		ID:       account.ID,
		Label:    label,
		Version:  pinnedVersion(account.Version),
	}
}

func escrowAccountProperty() map[string]any {
	return idProperty("The escrow account, from " + escrowSourcesTool + ". Never guess one.")
}

func interestRateProperty() map[string]any {
	return amountProperty("The annual interest rate paid on the balance, as a percent " +
		"such as 2.5.")
}

func readInterestRate(params map[string]any) (decimal.Decimal, bool, error) {
	rate, ok, err := optionalDecimal(params, paramEscrowInterestRate)
	if err != nil || !ok {
		return rate, ok, err
	}
	if rate.IsNegative() || rate.GreaterThan(decimal.NewFromInt(100)) {
		return rate, false, fmt.Errorf("%s must be between 0 and 100", paramEscrowInterestRate)
	}

	return rate, true, nil
}

func loadEscrow(
	ctx context.Context,
	book escrowBook,
	params *serviceports.ToolExecuteParams,
) (*driverpay.EscrowAccount, error) {
	accountID, err := requirePulid(params.Params, paramEscrowAccountID)
	if err != nil {
		return nil, err
	}

	return book.GetEscrowAccount(ctx, repositories.GetEscrowAccountByIDRequest{
		ID:         accountID,
		TenantInfo: tenantFrom(*params),
	})
}

func escrowBalanceMoney(before, after *driverpay.EscrowAccount) *agent.MoneyPreview {
	return toolpreview.MoneyBlock(after.CurrencyCode, agent.MoneyLine{
		Label:  "Balance",
		Before: minorAmount(before.BalanceMinor),
		After:  minorAmount(after.BalanceMinor),
	})
}

type openEscrowAccountTool struct {
	escrow escrowBook
}

var (
	_ serviceports.ToolPreviewer = (*openEscrowAccountTool)(nil)
	_ serviceports.ToolValidator = (*openEscrowAccountTool)(nil)
)

func provideOpenEscrowAccountTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &openEscrowAccountTool{escrow: s}
}

func (t *openEscrowAccountTool) Name() string { return "open_escrow_account" }

func (t *openEscrowAccountTool) Description() string {
	return "Propose opening an owner-operator's escrow account under their lease, with the " +
		"target balance and the interest it earns. Contributions come from a recurring " +
		"deduction linked to it. These are lease terms, so a person always decides; a " +
		"driver has one active account."
}

func (t *openEscrowAccountTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramWorkerID: workerProperty(),
		paramEscrowTarget: amountProperty("The balance the lease requires, as a decimal " +
			"such as 2500.00."),
		paramEscrowInterestRate: interestRateProperty(),
		paramEscrowOpenedDate:   dateProperty("The day it opens; leave it out for today."),
	}, paramWorkerID, paramEscrowTarget)
}

func (t *openEscrowAccountTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  permission.ResourceEscrowAccount,
		operation: permission.OpCreate,
		rationale: "Sets escrow terms under an owner-operator's lease that their pay is then " +
			"withheld against; only a person agrees them.",
	}).policy()
}

func (t *openEscrowAccountTool) entity(
	params *serviceports.ToolExecuteParams,
) (*driverpay.EscrowAccount, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	target, err := requireMoney(params.Params, paramEscrowTarget)
	if err != nil {
		return nil, err
	}
	rate, _, err := readInterestRate(params.Params)
	if err != nil {
		return nil, err
	}
	opened, _, err := optionalDay(params.Params, paramEscrowOpenedDate)
	if err != nil {
		return nil, err
	}

	return &driverpay.EscrowAccount{
		OrganizationID:     params.OrganizationID,
		BusinessUnitID:     params.BusinessUnitID,
		WorkerID:           workerID,
		TargetAmountMinor:  target,
		AnnualInterestRate: rate,
		OpenedDate:         opened,
		CurrencyCode:       money.DefaultCurrencyCode,
	}, nil
}

type escrowOpening struct {
	entity  *driverpay.EscrowAccount
	refusal error
}

func (t *openEscrowAccountTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*escrowOpening, error) {
	entity, err := t.entity(params)
	if err != nil {
		return nil, err
	}

	refusal, failure := refusalOrFailure(
		t.escrow.PlanOpenEscrowAccount(ctx, entity, timeutils.NowUnix()),
	)
	if failure != nil {
		return nil, failure
	}

	return &escrowOpening{entity: entity, refusal: refusal}, nil
}

func (t *openEscrowAccountTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *openEscrowAccountTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if err := requirePersonsApproval(&params); err != nil {
		return err
	}
	entity, err := t.entity(&params)
	if err != nil {
		return err
	}
	_, err = t.escrow.OpenEscrowAccount(ctx, entity, params.Actor)

	return err
}

func (t *openEscrowAccountTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}
	entity, refusal := plan.entity, plan.refusal

	summary := fmt.Sprintf(
		"Would open an escrow account with a %s target, earning %s%% a year.",
		money.FormatMinor(entity.TargetAmountMinor, entity.CurrencyCode),
		entity.AnnualInterestRate.String(),
	)
	if refusal != nil {
		return wouldFail(toolpreview.Build(summary), refusal)
	}

	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceEscrowAccount, Label: "New escrow account"},
		entity,
		toolpreview.Only(paramWorkerID, fieldStatus, paramEscrowOpenedDate,
			fieldAnnualInterestRate),
		toolpreview.Volatile(paramEscrowOpenedDate),
		toolpreview.WithRefs(map[string]permission.Resource{
			paramWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(driverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(entity.CurrencyCode,
		agent.MoneyLine{Label: "Target", After: minorAmount(entity.TargetAmountMinor)},
	), toolpreview.SensitiveAs(escrowTargetMark))

	return toolpreview.Build(summary+" Its balance starts at zero.", change), nil
}

type updateEscrowAccountTool struct {
	escrow escrowBook
}

var (
	_ serviceports.ToolPreviewer = (*updateEscrowAccountTool)(nil)
	_ serviceports.ToolValidator = (*updateEscrowAccountTool)(nil)
	_ serviceports.TargetedTool  = (*updateEscrowAccountTool)(nil)
)

func provideUpdateEscrowAccountTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &updateEscrowAccountTool{escrow: s}
}

func (t *updateEscrowAccountTool) Name() string { return "update_escrow_account" }

func (t *updateEscrowAccountTool) Description() string {
	return "Propose changing an escrow account's target balance or the interest it earns, " +
		"as an amended lease sets them. The balance itself changes only through " +
		"contributions and adjust_escrow_account. A person always decides."
}

func (t *updateEscrowAccountTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramEscrowAccountID:    escrowAccountProperty(),
		paramEscrowTarget:       amountProperty("The new target balance, as a decimal."),
		paramEscrowInterestRate: interestRateProperty(),
	}, paramEscrowAccountID)
}

func (t *updateEscrowAccountTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  permission.ResourceEscrowAccount,
		operation: permission.OpUpdate,
		rationale: "Changes the escrow terms of an owner-operator's lease; only a person " +
			"agrees them.",
	}).policy()
}

func (t *updateEscrowAccountTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramEscrowAccountID, permission.ResourceEscrowAccount)
}

type escrowUpdate struct {
	before  *driverpay.EscrowAccount
	after   *driverpay.EscrowAccount
	refusal error
}

func (t *updateEscrowAccountTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*escrowUpdate, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	target, hasTarget, err := optionalMoney(params.Params, paramEscrowTarget)
	if err != nil {
		return nil, err
	}
	rate, hasRate, err := readInterestRate(params.Params)
	if err != nil {
		return nil, err
	}
	if !hasTarget && !hasRate {
		return nil, errNothingToChange
	}
	before, err := loadEscrow(ctx, t.escrow, params)
	if err != nil {
		return nil, err
	}

	after := *before
	after.Transactions = nil
	if hasTarget {
		after.TargetAmountMinor = target
	}
	if hasRate {
		after.AnnualInterestRate = rate
	}
	_, planErr := t.escrow.PlanUpdateEscrowAccount(ctx, &after)
	refusal, failure := refusalOrFailure(planErr)
	if failure != nil {
		return nil, failure
	}

	return &escrowUpdate{before: before, after: &after, refusal: refusal}, nil
}

func (t *updateEscrowAccountTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *updateEscrowAccountTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if err := requirePersonsApproval(&params); err != nil {
		return err
	}
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}
	if plan.refusal != nil {
		return plan.refusal
	}
	_, err = t.escrow.UpdateEscrowAccount(ctx, plan.after, params.Actor)

	return err
}

func (t *updateEscrowAccountTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would change the terms of " + escrowRecord(plan.before).Label + "."
	if plan.refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.refusal)
	}
	change, err := toolpreview.Changed(escrowRecord(plan.before), plan.before, plan.after,
		toolpreview.Only(fieldAnnualInterestRate))
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(plan.after.CurrencyCode,
		agent.MoneyLine{
			Label:  "Target",
			Before: minorAmount(plan.before.TargetAmountMinor),
			After:  minorAmount(plan.after.TargetAmountMinor),
		},
	), toolpreview.SensitiveAs(escrowTargetMark))

	return toolpreview.Build(summary, change), nil
}

type adjustEscrowAccountTool struct {
	escrow escrowBook
}

var (
	_ serviceports.ToolPreviewer = (*adjustEscrowAccountTool)(nil)
	_ serviceports.ToolValidator = (*adjustEscrowAccountTool)(nil)
	_ serviceports.TargetedTool  = (*adjustEscrowAccountTool)(nil)
)

func provideAdjustEscrowAccountTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &adjustEscrowAccountTool{escrow: s}
}

func (t *adjustEscrowAccountTool) Name() string { return "adjust_escrow_account" }

func (t *adjustEscrowAccountTool) Description() string {
	return "Propose a manual adjustment to an owner-operator's escrow balance, with what it " +
		"is for. Use it for a deposit the driver made directly or money drawn to cover a " +
		"lease obligation. A positive amount adds, a negative one draws; the balance cannot " +
		"go below zero. A person always decides."
}

func (t *adjustEscrowAccountTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramEscrowAccountID: escrowAccountProperty(),
		paramDriverPayAmount: amountProperty("The amount as a decimal; negative to draw " +
			"from the balance. Never zero."),
		paramEscrowDescription: stringProperty("What the adjustment is for, as it will read "+
			"on the escrow ledger.", maxEscrowDescription),
		paramEscrowOccurredDate: dateProperty("The day it happened; leave it out for today."),
	}, paramEscrowAccountID, paramDriverPayAmount, paramEscrowDescription)
}

func (t *adjustEscrowAccountTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  permission.ResourceEscrowAccount,
		operation: permission.OpUpdate,
		rationale: "Moves money held in escrow for an owner-operator; only a person moves it.",
	}).policy()
}

func (t *adjustEscrowAccountTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramEscrowAccountID, permission.ResourceEscrowAccount)
}

func (t *adjustEscrowAccountTool) request(
	params *serviceports.ToolExecuteParams,
) (*driverpayservice.EscrowAdjustmentRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	accountID, err := requirePulid(params.Params, paramEscrowAccountID)
	if err != nil {
		return nil, err
	}
	amount, err := requireSignedMoney(params.Params, paramDriverPayAmount)
	if err != nil {
		return nil, err
	}
	description, err := boundedString(
		params.Params,
		paramEscrowDescription,
		maxEscrowDescription,
		true,
	)
	if err != nil {
		return nil, err
	}
	occurred, hasOccurred, err := optionalDay(params.Params, paramEscrowOccurredDate)
	if err != nil {
		return nil, err
	}
	if !hasOccurred {
		occurred = timeutils.NowUnix()
	}

	return &driverpayservice.EscrowAdjustmentRequest{
		TenantInfo:   tenantFrom(*params),
		AccountID:    accountID,
		AmountMinor:  amount,
		Description:  description,
		OccurredDate: occurred,
	}, nil
}

type escrowMovement struct {
	before  *driverpay.EscrowAccount
	after   *driverpay.EscrowAccount
	moved   *driverpay.EscrowTransaction
	refusal error
}

func (t *adjustEscrowAccountTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*driverpayservice.EscrowAdjustmentRequest, *escrowMovement, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, nil, err
	}
	before, err := t.escrow.GetEscrowAccount(ctx, repositories.GetEscrowAccountByIDRequest{
		ID:         req.AccountID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	after := *before
	after.Transactions = nil
	moved, refusal := driverpayservice.PlanEscrowAdjustment(&after, req, params.Actor.UserID)
	if refusal == nil {
		driverpayservice.StampEscrowTransaction(&after, moved)
		after.BalanceMinor = moved.BalanceAfterMinor
	}

	return req, &escrowMovement{
		before:  before,
		after:   &after,
		moved:   moved,
		refusal: refusal,
	}, nil
}

func (t *adjustEscrowAccountTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *adjustEscrowAccountTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if err := requirePersonsApproval(&params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.escrow.AdjustEscrowAccount(ctx, req, params.Actor)

	return err
}

func (t *adjustEscrowAccountTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would adjust %s by %s: %s.",
		escrowRecord(plan.before).Label,
		money.FormatMinor(req.AmountMinor, plan.before.CurrencyCode),
		req.Description,
	)
	if plan.refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.refusal)
	}

	change := toolpreview.Money(
		escrowRecord(plan.before),
		escrowBalanceMoney(plan.before, plan.after),
		toolpreview.SensitiveAs(escrowBalanceMark),
	)

	return toolpreview.Build(summary, change), nil
}

type closeEscrowAccountTool struct {
	escrow escrowBook
}

var (
	_ serviceports.ToolPreviewer = (*closeEscrowAccountTool)(nil)
	_ serviceports.ToolValidator = (*closeEscrowAccountTool)(nil)
	_ serviceports.TargetedTool  = (*closeEscrowAccountTool)(nil)
)

func provideCloseEscrowAccountTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &closeEscrowAccountTool{escrow: s}
}

func (t *closeEscrowAccountTool) Name() string { return "close_escrow_account" }

func (t *closeEscrowAccountTool) Description() string {
	return "Propose closing an owner-operator's escrow account when their lease ends, " +
		"refunding whatever balance remains to them. Settle any lease obligations with " +
		"adjust_escrow_account first. It cannot be reopened, so a person always decides."
}

func (t *closeEscrowAccountTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramEscrowAccountID: escrowAccountProperty(),
	}, paramEscrowAccountID)
}

func (t *closeEscrowAccountTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  permission.ResourceEscrowAccount,
		operation: permission.OpClose,
		rationale: "Refunds an owner-operator's escrow balance and closes the account for " +
			"good; only a person closes it.",
	}).policy()
}

func (t *closeEscrowAccountTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramEscrowAccountID, permission.ResourceEscrowAccount)
}

func (t *closeEscrowAccountTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*escrowMovement, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	before, err := loadEscrow(ctx, t.escrow, params)
	if err != nil {
		return nil, err
	}

	after := *before
	after.Transactions = nil
	now := timeutils.NowUnix()
	refund, refusal := driverpayservice.PlanCloseEscrowAccount(&after, params.Actor.UserID, now)
	if refusal == nil {
		if refund != nil {
			driverpayservice.StampEscrowTransaction(&after, refund)
			after.BalanceMinor = refund.BalanceAfterMinor
		}
		driverpayservice.CloseEscrowAccount(&after, now)
	}

	return &escrowMovement{before: before, after: &after, moved: refund, refusal: refusal}, nil
}

func (t *closeEscrowAccountTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *closeEscrowAccountTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if err := requirePersonsApproval(&params); err != nil {
		return err
	}
	accountID, err := requirePulid(params.Params, paramEscrowAccountID)
	if err != nil {
		return err
	}
	_, err = t.escrow.CloseEscrowAccount(ctx, tenantFrom(params), accountID, params.Actor)

	return err
}

func (t *closeEscrowAccountTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would close " + escrowRecord(plan.before).Label + "."
	if plan.refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.refusal)
	}
	if plan.moved != nil {
		summary += fmt.Sprintf(" The remaining %s is refunded to the driver.",
			money.FormatMinor(-plan.moved.AmountMinor, plan.before.CurrencyCode))
	}

	change, err := toolpreview.Changed(escrowRecord(plan.before), plan.before, plan.after,
		toolpreview.Only(fieldStatus, fieldClosedDate),
		toolpreview.Volatile(fieldClosedDate),
		toolpreview.Types(driverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, escrowBalanceMoney(plan.before, plan.after),
		toolpreview.SensitiveAs(escrowBalanceMark))

	return toolpreview.Build(summary, change), nil
}
