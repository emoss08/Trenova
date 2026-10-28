package settlementtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const paramEscrowAccountID = "accountId"

const paramEscrowTarget = "targetAmount"

const paramEscrowInterestRate = "annualInterestRate"

const paramEscrowOpenedDate = "openedDate"

const paramEscrowDescription = "description"

const paramEscrowOccurredDate = "occurredDate"

const maxEscrowDescription = 255

const escrowSourcesTool = "settlement_list_escrow_accounts"

const escrowBalanceMark = "balanceMinor"

const escrowTargetMark = "targetAmountMinor"

const fieldAnnualInterestRate = "annualInterestRate"

const fieldClosedDate = "closedDate"

type escrowBook interface {
	GetEscrowAccount(
		ctx context.Context,
		req driverpayports.GetEscrowAccountByIDRequest,
	) (*driverpay.EscrowAccount, error)
	PlanOpenEscrowAccount(ctx context.Context, entity *driverpay.EscrowAccount, now int64) error
	OpenEscrowAccount(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
		actor *permissionports.RequestActor,
	) (*driverpay.EscrowAccount, error)
	PlanUpdateEscrowAccount(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
	) (*driverpay.EscrowAccount, error)
	UpdateEscrowAccount(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
		actor *permissionports.RequestActor,
	) (*driverpay.EscrowAccount, error)
	AdjustEscrowAccount(
		ctx context.Context,
		req *driverpayservice.EscrowAdjustmentRequest,
		actor *permissionports.RequestActor,
	) (*driverpay.EscrowAccount, error)
	CloseEscrowAccount(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		accountID pulid.ID,
		actor *permissionports.RequestActor,
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
		Version:  changepreview.PinnedVersion(account.Version),
	}
}

func escrowAccountProperty() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceEscrowAccount,
		"The escrow account, from "+escrowSourcesTool+". Never guess one.",
	)
}

func interestRateProperty() map[string]any {
	return toolparams.AmountProperty("The annual interest rate paid on the balance, as a percent " +
		"such as 2.5.")
}

func readInterestRate(params map[string]any) (decimal.Decimal, bool, error) {
	rate, ok, err := toolparams.OptionalDecimal(params, paramEscrowInterestRate)
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
	params *agenttoolports.ToolExecuteParams,
) (*driverpay.EscrowAccount, error) {
	accountID, err := toolparams.RequirePulid(params.Params, paramEscrowAccountID)
	if err != nil {
		return nil, err
	}

	return book.GetEscrowAccount(ctx, driverpayports.GetEscrowAccountByIDRequest{
		ID:         accountID,
		TenantInfo: toolparams.TenantFrom(*params),
	})
}

func escrowBalanceMoney(before, after *driverpay.EscrowAccount) *agent.MoneyPreview {
	return toolpreview.MoneyBlock(after.CurrencyCode, agent.MoneyLine{
		Label:  "Balance",
		Before: changepreview.MinorAmount(before.BalanceMinor),
		After:  changepreview.MinorAmount(after.BalanceMinor),
	})
}

type OpenEscrowAccountTool struct {
	escrow escrowBook
}

func ProvideOpenEscrowAccountTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &OpenEscrowAccountTool{escrow: s}
}

func (t *OpenEscrowAccountTool) Name() string { return "settlement_open_escrow_account" }

func (t *OpenEscrowAccountTool) Description() string {
	return "Propose opening an owner-operator's escrow account under their lease, with the " +
		"target balance and the interest it earns. Contributions come from a recurring " +
		"deduction linked to it. These are lease terms, so a person always decides; a " +
		"driver has one active account."
}

func (t *OpenEscrowAccountTool) ParamSchema() map[string]any {
	return toolparams.ObjectParams(map[string]any{
		toolparams.ParamWorkerID: toolparams.WorkerProperty(),
		paramEscrowTarget: toolparams.AmountProperty(
			"The balance the lease requires, as a decimal " +
				"such as 2500.00.",
		),
		paramEscrowInterestRate: interestRateProperty(),
		paramEscrowOpenedDate:   agenttoolschema.Date("The day it opens; leave it out for today."),
	}, toolparams.ParamWorkerID, paramEscrowTarget)
}

func (t *OpenEscrowAccountTool) Policy() agenttoolports.ToolPolicy {
	return (&settlementpreview.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  permission.ResourceEscrowAccount,
		Operation: permission.OpCreate,
		Rationale: "Sets escrow terms under an owner-operator's lease that their pay is then " +
			"withheld against; only a person agrees them.",
	}).Policy()
}

func (t *OpenEscrowAccountTool) entity(
	params *agenttoolports.ToolExecuteParams,
) (*driverpay.EscrowAccount, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	target, err := toolparams.RequireMoney(params.Params, paramEscrowTarget)
	if err != nil {
		return nil, err
	}
	rate, _, err := readInterestRate(params.Params)
	if err != nil {
		return nil, err
	}
	opened, _, err := toolparams.OptionalDay(params.Params, paramEscrowOpenedDate)
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

func (t *OpenEscrowAccountTool) planned(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*escrowOpening, error) {
	entity, err := t.entity(params)
	if err != nil {
		return nil, err
	}

	refusal, failure := changepreview.RefusalOrFailure(
		t.escrow.PlanOpenEscrowAccount(ctx, entity, timeutils.NowUnix()),
	)
	if failure != nil {
		return nil, failure
	}

	return &escrowOpening{entity: entity, refusal: refusal}, nil
}

func (t *OpenEscrowAccountTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *OpenEscrowAccountTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if err := settlementpreview.RequirePersonsApproval(&params); err != nil {
		return err
	}
	entity, err := t.entity(&params)
	if err != nil {
		return err
	}
	_, err = t.escrow.OpenEscrowAccount(ctx, entity, params.Actor)

	return err
}

func (t *OpenEscrowAccountTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		return changepreview.WouldFail(toolpreview.Build(summary), refusal)
	}

	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceEscrowAccount, Label: "New escrow account"},
		entity,
		toolpreview.Only(toolparams.ParamWorkerID, changepreview.FieldStatus, paramEscrowOpenedDate,
			fieldAnnualInterestRate),
		toolpreview.Volatile(paramEscrowOpenedDate),
		toolpreview.WithRefs(map[string]permission.Resource{
			toolparams.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(settlementpreview.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		entity.CurrencyCode,
		agent.MoneyLine{
			Label: "Target",
			After: changepreview.MinorAmount(entity.TargetAmountMinor),
		},
	), toolpreview.SensitiveAs(escrowTargetMark))

	return toolpreview.Build(summary+" Its balance starts at zero.", change), nil
}

type UpdateEscrowAccountTool struct {
	escrow escrowBook
}

func ProvideUpdateEscrowAccountTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &UpdateEscrowAccountTool{escrow: s}
}

func (t *UpdateEscrowAccountTool) Name() string { return "settlement_update_escrow_account" }

func (t *UpdateEscrowAccountTool) Description() string {
	return "Propose changing an escrow account's target balance or the interest it earns, " +
		"as an amended lease sets them. The balance itself changes only through " +
		"contributions and settlement_adjust_escrow_account. A person always decides."
}

func (t *UpdateEscrowAccountTool) SearchTerms() []string {
	return []string{"target balance", "escrow interest", "amended lease"}
}

func (t *UpdateEscrowAccountTool) ParamSchema() map[string]any {
	return toolparams.ObjectParams(map[string]any{
		paramEscrowAccountID:    escrowAccountProperty(),
		paramEscrowTarget:       toolparams.AmountProperty("The new target balance, as a decimal."),
		paramEscrowInterestRate: interestRateProperty(),
	}, paramEscrowAccountID)
}

func (t *UpdateEscrowAccountTool) Policy() agenttoolports.ToolPolicy {
	return (&settlementpreview.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  permission.ResourceEscrowAccount,
		Operation: permission.OpUpdate,
		Rationale: "Changes the escrow terms of an owner-operator's lease; only a person " +
			"agrees them.",
	}).Policy()
}

func (t *UpdateEscrowAccountTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramEscrowAccountID, permission.ResourceEscrowAccount)
}

type escrowUpdate struct {
	before  *driverpay.EscrowAccount
	after   *driverpay.EscrowAccount
	refusal error
}

func (t *UpdateEscrowAccountTool) planned(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*escrowUpdate, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	target, hasTarget, err := toolparams.OptionalMoney(params.Params, paramEscrowTarget)
	if err != nil {
		return nil, err
	}
	rate, hasRate, err := readInterestRate(params.Params)
	if err != nil {
		return nil, err
	}
	if !hasTarget && !hasRate {
		return nil, toolparams.ErrNothingToChange
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
	refusal, failure := changepreview.RefusalOrFailure(planErr)
	if failure != nil {
		return nil, failure
	}

	return &escrowUpdate{before: before, after: &after, refusal: refusal}, nil
}

func (t *UpdateEscrowAccountTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *UpdateEscrowAccountTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if err := settlementpreview.RequirePersonsApproval(&params); err != nil {
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

func (t *UpdateEscrowAccountTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would change the terms of " + escrowRecord(plan.before).Label + "."
	if plan.refusal != nil {
		return changepreview.WouldFail(toolpreview.Build(summary), plan.refusal)
	}
	change, err := toolpreview.Changed(escrowRecord(plan.before), plan.before, plan.after,
		toolpreview.Only(fieldAnnualInterestRate))
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(plan.after.CurrencyCode,
		agent.MoneyLine{
			Label:  "Target",
			Before: changepreview.MinorAmount(plan.before.TargetAmountMinor),
			After:  changepreview.MinorAmount(plan.after.TargetAmountMinor),
		},
	), toolpreview.SensitiveAs(escrowTargetMark))

	return toolpreview.Build(summary, change), nil
}

type AdjustEscrowAccountTool struct {
	escrow escrowBook
}

func ProvideAdjustEscrowAccountTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &AdjustEscrowAccountTool{escrow: s}
}

func (t *AdjustEscrowAccountTool) Name() string { return "settlement_adjust_escrow_account" }

func (t *AdjustEscrowAccountTool) Description() string {
	return "Propose a manual adjustment to an owner-operator's escrow balance, with what it " +
		"is for. Use it for a deposit the driver made directly or money drawn to cover a " +
		"lease obligation. A positive amount adds, a negative one draws; the balance cannot " +
		"go below zero. A person always decides."
}

func (t *AdjustEscrowAccountTool) SearchTerms() []string {
	return []string{"escrow debit", "manual escrow adjustment"}
}

func (t *AdjustEscrowAccountTool) ParamSchema() map[string]any {
	return toolparams.ObjectParams(map[string]any{
		paramEscrowAccountID: escrowAccountProperty(),
		settlementpreview.ParamDriverPayAmount: toolparams.AmountProperty(
			"The amount as a decimal; negative to draw " +
				"from the balance. Never zero.",
		),
		paramEscrowDescription: toolparams.StringProperty(
			"What the adjustment is for, as it will read "+
				"on the escrow ledger.",
			maxEscrowDescription,
		),
		paramEscrowOccurredDate: agenttoolschema.Date(
			"The day it happened; leave it out for today.",
		),
	}, paramEscrowAccountID, settlementpreview.ParamDriverPayAmount, paramEscrowDescription)
}

func (t *AdjustEscrowAccountTool) Policy() agenttoolports.ToolPolicy {
	return (&settlementpreview.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  permission.ResourceEscrowAccount,
		Operation: permission.OpUpdate,
		Rationale: "Moves money held in escrow for an owner-operator; only a person moves it.",
	}).Policy()
}

func (t *AdjustEscrowAccountTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramEscrowAccountID, permission.ResourceEscrowAccount)
}

func (t *AdjustEscrowAccountTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*driverpayservice.EscrowAdjustmentRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	accountID, err := toolparams.RequirePulid(params.Params, paramEscrowAccountID)
	if err != nil {
		return nil, err
	}
	amount, err := toolparams.RequireSignedMoney(
		params.Params,
		settlementpreview.ParamDriverPayAmount,
	)
	if err != nil {
		return nil, err
	}
	description, err := toolparams.BoundedString(
		params.Params,
		paramEscrowDescription,
		maxEscrowDescription,
		true,
	)
	if err != nil {
		return nil, err
	}
	occurred, hasOccurred, err := toolparams.OptionalDay(params.Params, paramEscrowOccurredDate)
	if err != nil {
		return nil, err
	}
	if !hasOccurred {
		occurred = timeutils.NowUnix()
	}

	return &driverpayservice.EscrowAdjustmentRequest{
		TenantInfo:   toolparams.TenantFrom(*params),
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

func (t *AdjustEscrowAccountTool) planned(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*driverpayservice.EscrowAdjustmentRequest, *escrowMovement, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, nil, err
	}
	before, err := t.escrow.GetEscrowAccount(ctx, driverpayports.GetEscrowAccountByIDRequest{
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

func (t *AdjustEscrowAccountTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *AdjustEscrowAccountTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if err := settlementpreview.RequirePersonsApproval(&params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.escrow.AdjustEscrowAccount(ctx, req, params.Actor)

	return err
}

func (t *AdjustEscrowAccountTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		return changepreview.WouldFail(toolpreview.Build(summary), plan.refusal)
	}

	change := toolpreview.Money(
		escrowRecord(plan.before),
		escrowBalanceMoney(plan.before, plan.after),
		toolpreview.SensitiveAs(escrowBalanceMark),
	)

	return toolpreview.Build(summary, change), nil
}

type CloseEscrowAccountTool struct {
	escrow escrowBook
}

func ProvideCloseEscrowAccountTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &CloseEscrowAccountTool{escrow: s}
}

func (t *CloseEscrowAccountTool) Name() string { return "settlement_close_escrow_account" }

func (t *CloseEscrowAccountTool) Description() string {
	return "Propose closing an owner-operator's escrow account when their lease ends, " +
		"refunding whatever balance remains to them. Settle any lease obligations with " +
		"settlement_adjust_escrow_account first. It cannot be reopened, so a person always decides."
}

func (t *CloseEscrowAccountTool) ParamSchema() map[string]any {
	return toolparams.ObjectParams(map[string]any{
		paramEscrowAccountID: escrowAccountProperty(),
	}, paramEscrowAccountID)
}

func (t *CloseEscrowAccountTool) Policy() agenttoolports.ToolPolicy {
	return (&settlementpreview.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  permission.ResourceEscrowAccount,
		Operation: permission.OpClose,
		Rationale: "Refunds an owner-operator's escrow balance and closes the account for " +
			"good; only a person closes it.",
	}).Policy()
}

func (t *CloseEscrowAccountTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramEscrowAccountID, permission.ResourceEscrowAccount)
}

func (t *CloseEscrowAccountTool) planned(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*escrowMovement, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
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

func (t *CloseEscrowAccountTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *CloseEscrowAccountTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if err := settlementpreview.RequirePersonsApproval(&params); err != nil {
		return err
	}
	accountID, err := toolparams.RequirePulid(params.Params, paramEscrowAccountID)
	if err != nil {
		return err
	}
	_, err = t.escrow.CloseEscrowAccount(
		ctx,
		toolparams.TenantFrom(params),
		accountID,
		params.Actor,
	)

	return err
}

func (t *CloseEscrowAccountTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would close " + escrowRecord(plan.before).Label + "."
	if plan.refusal != nil {
		return changepreview.WouldFail(toolpreview.Build(summary), plan.refusal)
	}
	if plan.moved != nil {
		summary += fmt.Sprintf(" The remaining %s is refunded to the driver.",
			money.FormatMinor(-plan.moved.AmountMinor, plan.before.CurrencyCode))
	}

	change, err := toolpreview.Changed(escrowRecord(plan.before), plan.before, plan.after,
		toolpreview.Only(changepreview.FieldStatus, fieldClosedDate),
		toolpreview.Volatile(fieldClosedDate),
		toolpreview.Types(settlementpreview.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, escrowBalanceMoney(plan.before, plan.after),
		toolpreview.SensitiveAs(escrowBalanceMark))

	return toolpreview.Build(summary, change), nil
}
