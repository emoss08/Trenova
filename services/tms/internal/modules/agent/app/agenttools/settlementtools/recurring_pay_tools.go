package settlementtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramRecurringDescription = "description"

const paramRecurringFrequency = "frequency"

const paramRecurringStatus = "status"

const paramRecurringCap = "totalCap"

const paramRecurringStart = "startDate"

const paramRecurringEnd = "endDate"

const paramEscrowContribution = "escrowContribution"

const maxRecurringDescription = 255

const fieldRecurringFrequency = "frequency"

const fieldEscrowAccountID = "escrowAccountId"

type recurringFacts struct {
	id          pulid.ID
	version     int64
	description string
	amount      int64
	capMinor    *int64
	currency    string
}

type recurringFields struct {
	workerID    pulid.ID
	payCodeID   pulid.ID
	frequency   string
	status      string
	description string
	amount      int64
	capMinor    *int64
	start       int64
	end         *int64
}

type RecurringBook[E any] interface {
	get(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID) (*E, error)
	check(ctx context.Context, entity *E, escrow bool) error
	create(
		ctx context.Context,
		entity *E,
		escrow bool,
		actor *permissionports.RequestActor,
	) (*E, error)
	update(ctx context.Context, entity *E, actor *permissionports.RequestActor) (*E, error)
}

type recurringKind[E any] struct {
	noun        string
	createName  string
	updateName  string
	resource    permission.Resource
	listTool    string
	idParam     string
	payee       string
	direction   string
	statuses    agenttoolschema.EnumSource[string]
	frequencies agenttoolschema.EnumSource[string]
	escrow      bool
	create      func(tenant pagination.TenantInfo, fields *recurringFields) *E
	apply       func(entity *E, fields *recurringFields, given map[string]bool)
	facts       func(entity *E) recurringFacts
	sensitive   string
	createTerms []string
	updateTerms []string
}

type RecurringPayTool[E any] struct {
	kind   recurringKind[E]
	book   RecurringBook[E]
	update bool
}

func (t *RecurringPayTool[E]) Name() string {
	if t.update {
		return t.kind.updateName
	}

	return t.kind.createName
}

func (t *RecurringPayTool[E]) SearchTerms() []string {
	if t.update {
		return t.kind.updateTerms
	}

	return t.kind.createTerms
}

func (t *RecurringPayTool[E]) Description() string {
	if t.update {
		return "Propose changing a driver's " + t.kind.noun + ": its amount, cap, schedule, " +
			"dates or description, or pausing, resuming or completing it. What was already " +
			"taken or paid stays as it is. Only the values you give change. A person always " +
			"decides."
	}
	description := "Propose setting up a " + t.kind.noun + " for a driver, an amount " +
		t.kind.direction + " each settlement or once a month. It runs from a start date, " +
		"optionally until an end date or a total cap. A person always decides."
	if t.kind.escrow {
		description += " Set escrowContribution to pay it into the driver's escrow account."
	}

	return description
}

func (t *RecurringPayTool[E]) ParamSchema() map[string]any {
	properties := map[string]any{
		common.ParamPayCodeID: agenttoolschema.RecordIDText(
			permission.ResourcePayCode,
			"The "+t.kind.payee+" pay code it posts under, from list_pay_codes.",
		),
		paramRecurringDescription: common.StringProperty("What it is for, as it will read on the "+
			"driver's statement.", maxRecurringDescription),
		common.ParamDriverPayAmount: common.AmountProperty("The amount each time, as a decimal such as " +
			"25.00."),
		paramRecurringFrequency: agenttoolschema.Enum("How often it applies.", t.kind.frequencies),
		paramRecurringCap: common.AmountProperty("The total after which it stops, when there is " +
			"one."),
		paramRecurringStart: agenttoolschema.Date("The first day it applies."),
		paramRecurringEnd:   agenttoolschema.Date("The last day it applies, when it ends."),
	}
	if t.update {
		properties[t.kind.idParam] = agenttoolschema.RecordIDText(t.kind.resource,
			"The "+t.kind.noun+", from "+t.kind.listTool+". Never guess one.")
		properties[paramRecurringStatus] = agenttoolschema.Enum(
			"Paused stops it for now; Completed ends it for good.", t.kind.statuses,
		)

		return common.ObjectParams(properties, t.kind.idParam)
	}

	properties[common.ParamWorkerID] = common.WorkerProperty()
	if t.kind.escrow {
		properties[paramEscrowContribution] = map[string]any{
			toolschema.KeyType: toolschema.TypeBoolean,
			toolschema.KeyDescription: "true to pay it into the driver's active escrow " +
				"account, from open_escrow_account.",
		}
	}

	return common.ObjectParams(
		properties,
		common.ParamWorkerID,
		common.ParamPayCodeID,
		paramRecurringDescription,
		common.ParamDriverPayAmount,
		paramRecurringStart,
	)
}

func (t *RecurringPayTool[E]) Policy() agentports.ToolPolicy {
	operation := permission.OpCreate
	rationale := "Sets money taken from or added to a driver's pay every settlement; only a " +
		"person agrees it."
	if t.update {
		operation = permission.OpUpdate
		rationale = "Changes money taken from or added to a driver's pay every settlement; " +
			"only a person agrees it."
	}

	return (&common.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  t.kind.resource,
		Operation: operation,
		Rationale: rationale,
	}).Policy()
}

func (t *RecurringPayTool[E]) Target(params map[string]any) (agentports.ToolTarget, bool) {
	if !t.update {
		return agentports.ToolTarget{}, false
	}

	return common.TargetOf(params, t.kind.idParam, t.kind.resource)
}

func readRecurringFields(
	params map[string]any,
	kind recurringKindText,
	required bool,
) (*recurringFields, map[string]bool, error) {
	fields := &recurringFields{}
	given := map[string]bool{}

	var err error
	if fields.frequency, given[paramRecurringFrequency], err = common.OptionalEnum(
		params, paramRecurringFrequency, kind.frequencies,
	); err != nil {
		return nil, nil, err
	}
	if fields.status, given[paramRecurringStatus], err = common.OptionalEnum(
		params, paramRecurringStatus, kind.statuses,
	); err != nil {
		return nil, nil, err
	}
	if fields.description, err = common.BoundedString(
		params, paramRecurringDescription, maxRecurringDescription, required,
	); err != nil {
		return nil, nil, err
	}
	given[paramRecurringDescription] = fields.description != ""
	if fields.amount, given[common.ParamDriverPayAmount], err = common.OptionalMoney(
		params, common.ParamDriverPayAmount,
	); err != nil {
		return nil, nil, err
	}
	if required && !given[common.ParamDriverPayAmount] {
		return nil, nil, fmt.Errorf("parameter %q is required", common.ParamDriverPayAmount)
	}
	capMinor, hasCap, err := common.OptionalMoney(params, paramRecurringCap)
	if err != nil {
		return nil, nil, err
	}
	if hasCap {
		fields.capMinor = &capMinor
	}
	given[paramRecurringCap] = hasCap
	if fields.start, given[paramRecurringStart], err = common.OptionalDay(
		params, paramRecurringStart,
	); err != nil {
		return nil, nil, err
	}
	if required && !given[paramRecurringStart] {
		return nil, nil, fmt.Errorf("parameter %q is required", paramRecurringStart)
	}
	end, hasEnd, err := common.OptionalDay(params, paramRecurringEnd)
	if err != nil {
		return nil, nil, err
	}
	if hasEnd {
		fields.end = &end
	}
	given[paramRecurringEnd] = hasEnd
	payCode, err := common.OptionalPulidParam(params, common.ParamPayCodeID)
	if err != nil {
		return nil, nil, err
	}
	if payCode != nil {
		fields.payCodeID = *payCode
	}
	given[common.ParamPayCodeID] = payCode != nil

	return fields, given, nil
}

type recurringKindText struct {
	frequencies []string
	statuses    []string
}

type recurringPlan[E any] struct {
	before  *E
	after   *E
	escrow  bool
	refusal error
}

func (t *RecurringPayTool[E]) planned(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*recurringPlan[E], error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	text := recurringKindText{
		frequencies: t.kind.frequencies.Values,
		statuses:    t.kind.statuses.Values,
	}
	tenant := common.TenantFrom(*params)
	if !t.update {
		workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
		if err != nil {
			return nil, err
		}
		if _, err = common.RequirePulid(params.Params, common.ParamPayCodeID); err != nil {
			return nil, err
		}
		fields, _, err := readRecurringFields(params.Params, text, true)
		if err != nil {
			return nil, err
		}
		fields.workerID = workerID
		entity := t.kind.create(tenant, fields)
		escrow := t.kind.escrow && common.OptionalBool(params.Params, paramEscrowContribution)
		refusal, failure := common.RefusalOrFailure(t.book.check(ctx, entity, escrow))
		if failure != nil {
			return nil, failure
		}

		return &recurringPlan[E]{after: entity, escrow: escrow, refusal: refusal}, nil
	}

	id, err := common.RequirePulid(params.Params, t.kind.idParam)
	if err != nil {
		return nil, err
	}
	fields, given, err := readRecurringFields(params.Params, text, false)
	if err != nil {
		return nil, err
	}
	changed := false
	for _, set := range given {
		changed = changed || set
	}
	if !changed {
		return nil, common.ErrNothingToChange
	}
	before, err := t.book.get(ctx, tenant, id)
	if err != nil {
		return nil, err
	}
	after := *before
	t.kind.apply(&after, fields, given)
	refusal, failure := common.RefusalOrFailure(t.book.check(ctx, &after, false))
	if failure != nil {
		return nil, failure
	}

	return &recurringPlan[E]{before: before, after: &after, refusal: refusal}, nil
}

func (t *RecurringPayTool[E]) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *RecurringPayTool[E]) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	if err := common.RequirePersonsApproval(&params); err != nil {
		return err
	}
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}
	if t.update {
		_, err = t.book.update(ctx, plan.after, params.Actor)

		return err
	}
	_, err = t.book.create(ctx, plan.after, plan.escrow, params.Actor)

	return err
}

func (t *RecurringPayTool[E]) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	after := t.kind.facts(plan.after)
	summary := fmt.Sprintf(
		"Would set up a %s of %s for the driver: %s.",
		t.kind.noun,
		money.FormatMinor(after.amount, after.currency),
		after.description,
	)
	if t.update {
		summary = fmt.Sprintf("Would change the driver's %s %q.", t.kind.noun,
			t.kind.facts(plan.before).description)
	}
	if plan.refusal != nil {
		return common.WouldFail(toolpreview.Build(summary), plan.refusal)
	}

	fields := []string{
		common.FieldStatus, fieldRecurringFrequency, paramRecurringDescription, common.FieldRecurringStart,
		common.FieldRecurringEnd, fieldEscrowAccountID,
	}
	var change *agent.RecordChange
	if t.update {
		before := t.kind.facts(plan.before)
		change, err = toolpreview.Changed(toolpreview.Record{
			Resource: t.kind.resource,
			ID:       before.id,
			Label:    before.description,
			Version:  common.PinnedVersion(before.version),
		}, plan.before, plan.after,
			toolpreview.Only(fields...),
			toolpreview.Types(common.DriverPayDateTypes),
		)
		if err != nil {
			return nil, err
		}
		toolpreview.AttachMoney(change, recurringMoney(&before, &after),
			toolpreview.SensitiveAs(t.kind.sensitive))

		return toolpreview.Build(summary, change), nil
	}

	change, err = toolpreview.Create(
		toolpreview.Record{Resource: t.kind.resource, Label: after.description},
		plan.after,
		toolpreview.Only(append(fields, common.ParamWorkerID)...),
		toolpreview.WithRefs(map[string]permission.Resource{
			common.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(common.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, recurringMoney(nil, &after),
		toolpreview.SensitiveAs(t.kind.sensitive))
	if plan.escrow {
		summary += " It is paid into the driver's escrow account."
	}

	return toolpreview.Build(summary, change), nil
}

func recurringMoney(before, after *recurringFacts) *agent.MoneyPreview {
	amount := agent.MoneyLine{Label: "Each time", After: common.MinorAmount(after.amount)}
	if before != nil {
		amount.Before = common.MinorAmount(before.amount)
	}

	return toolpreview.MoneyBlock(after.currency, amount)
}

func deductionKind() recurringKind[driverpay.RecurringDeduction] {
	return recurringKind[driverpay.RecurringDeduction]{
		noun:        "recurring deduction",
		createName:  "create_recurring_deduction",
		updateName:  "update_recurring_deduction",
		resource:    permission.ResourceRecurringDeduction,
		listTool:    "list_recurring_deductions",
		idParam:     "deductionId",
		payee:       "deduction",
		direction:   "taken from the driver's pay",
		statuses:    common.DeductionStatuses.AsStrings(),
		frequencies: common.DeductionFrequencies.AsStrings(),
		escrow:      true,
		sensitive:   "amountMinor",
		updateTerms: []string{"lease deduction", "raise deduction", "pause deduction"},
		create: func(
			tenant pagination.TenantInfo,
			fields *recurringFields,
		) *driverpay.RecurringDeduction {
			entity := &driverpay.RecurringDeduction{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				WorkerID:       fields.workerID,
				PayCodeID:      fields.payCodeID,
				Status:         driverpay.DeductionStatusActive,
				Frequency:      driverpay.DeductionFrequencyEverySettlement,
				Description:    fields.description,
				AmountMinor:    fields.amount,
				TotalCapMinor:  fields.capMinor,
				StartDate:      fields.start,
				EndDate:        fields.end,
				CurrencyCode:   money.DefaultCurrencyCode,
			}
			if fields.frequency != "" {
				entity.Frequency = driverpay.DeductionFrequency(fields.frequency)
			}

			return entity
		},
		apply: func(
			entity *driverpay.RecurringDeduction,
			fields *recurringFields,
			given map[string]bool,
		) {
			entity.Worker = nil
			entity.PayCode = nil
			entity.EscrowAccount = nil
			applyRecurringChanges(&recurringTarget{
				payCodeID:   &entity.PayCodeID,
				description: &entity.Description,
				amount:      &entity.AmountMinor,
				capMinor:    &entity.TotalCapMinor,
				start:       &entity.StartDate,
				end:         &entity.EndDate,
			}, fields, given)
			if given[paramRecurringFrequency] {
				entity.Frequency = driverpay.DeductionFrequency(fields.frequency)
			}
			if given[paramRecurringStatus] {
				entity.Status = driverpay.DeductionStatus(fields.status)
			}
		},
		facts: func(entity *driverpay.RecurringDeduction) recurringFacts {
			return recurringFacts{
				id:          entity.ID,
				version:     entity.Version,
				description: entity.Description,
				amount:      entity.AmountMinor,
				capMinor:    entity.TotalCapMinor,
				currency:    entity.CurrencyCode,
			}
		},
	}
}

func earningKind() recurringKind[driverpay.RecurringEarning] {
	return recurringKind[driverpay.RecurringEarning]{
		noun:        "recurring earning",
		createName:  "create_recurring_earning",
		updateName:  "update_recurring_earning",
		resource:    permission.ResourceRecurringEarning,
		listTool:    "list_recurring_earnings",
		idParam:     "earningId",
		payee:       "earning",
		direction:   "added to the driver's pay",
		statuses:    common.EarningStatuses.AsStrings(),
		frequencies: common.EarningFrequencies.AsStrings(),
		sensitive:   "amountMinor",
		createTerms: []string{"bonus", "recurring bonus"},
		updateTerms: []string{"pause bonus", "stop bonus", "edit bonus"},
		create: func(
			tenant pagination.TenantInfo,
			fields *recurringFields,
		) *driverpay.RecurringEarning {
			entity := &driverpay.RecurringEarning{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				WorkerID:       fields.workerID,
				PayCodeID:      fields.payCodeID,
				Status:         driverpay.EarningStatusActive,
				Frequency:      driverpay.EarningFrequencyEverySettlement,
				Description:    fields.description,
				AmountMinor:    fields.amount,
				TotalCapMinor:  fields.capMinor,
				StartDate:      fields.start,
				EndDate:        fields.end,
				CurrencyCode:   money.DefaultCurrencyCode,
			}
			if fields.frequency != "" {
				entity.Frequency = driverpay.EarningFrequency(fields.frequency)
			}

			return entity
		},
		apply: func(
			entity *driverpay.RecurringEarning,
			fields *recurringFields,
			given map[string]bool,
		) {
			entity.Worker = nil
			entity.PayCode = nil
			applyRecurringChanges(&recurringTarget{
				payCodeID:   &entity.PayCodeID,
				description: &entity.Description,
				amount:      &entity.AmountMinor,
				capMinor:    &entity.TotalCapMinor,
				start:       &entity.StartDate,
				end:         &entity.EndDate,
			}, fields, given)
			if given[paramRecurringFrequency] {
				entity.Frequency = driverpay.EarningFrequency(fields.frequency)
			}
			if given[paramRecurringStatus] {
				entity.Status = driverpay.EarningStatus(fields.status)
			}
		},
		facts: func(entity *driverpay.RecurringEarning) recurringFacts {
			return recurringFacts{
				id:          entity.ID,
				version:     entity.Version,
				description: entity.Description,
				amount:      entity.AmountMinor,
				capMinor:    entity.TotalCapMinor,
				currency:    entity.CurrencyCode,
			}
		},
	}
}

type recurringTarget struct {
	payCodeID   *pulid.ID
	description *string
	amount      *int64
	capMinor    **int64
	start       *int64
	end         **int64
}

func applyRecurringChanges(
	target *recurringTarget,
	fields *recurringFields,
	given map[string]bool,
) {
	if given[common.ParamPayCodeID] {
		*target.payCodeID = fields.payCodeID
	}
	if given[paramRecurringDescription] {
		*target.description = fields.description
	}
	if given[common.ParamDriverPayAmount] {
		*target.amount = fields.amount
	}
	if given[paramRecurringCap] {
		*target.capMinor = fields.capMinor
	}
	if given[paramRecurringStart] {
		*target.start = fields.start
	}
	if given[paramRecurringEnd] {
		*target.end = fields.end
	}
}

type DeductionBook struct {
	pay *driverpayservice.Service
}

func (b DeductionBook) get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.RecurringDeduction, error) {
	return b.pay.GetDeduction(ctx, driverpayports.GetRecurringDeductionByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}

func (b DeductionBook) check(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	escrow bool,
) error {
	return b.pay.CheckDeduction(ctx, entity, escrow)
}

func (b DeductionBook) create(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	escrow bool,
	actor *permissionports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	return b.pay.CreateDeduction(ctx, entity, escrow, actor)
}

func (b DeductionBook) update(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	actor *permissionports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	return b.pay.UpdateDeduction(ctx, entity, actor)
}

type EarningBook struct {
	pay *driverpayservice.Service
}

func (b EarningBook) get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.RecurringEarning, error) {
	return b.pay.GetEarning(ctx, driverpayports.GetRecurringEarningByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}

func (b EarningBook) check(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	_ bool,
) error {
	return b.pay.CheckEarning(ctx, entity)
}

func (b EarningBook) create(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	_ bool,
	actor *permissionports.RequestActor,
) (*driverpay.RecurringEarning, error) {
	return b.pay.CreateEarning(ctx, entity, actor)
}

func (b EarningBook) update(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	actor *permissionports.RequestActor,
) (*driverpay.RecurringEarning, error) {
	return b.pay.UpdateEarning(ctx, entity, actor)
}

func ProvideCreateRecurringDeductionTool(s *driverpayservice.Service) agentports.AgentTool {
	return &RecurringPayTool[driverpay.RecurringDeduction]{
		kind: deductionKind(),
		book: DeductionBook{pay: s},
	}
}

func ProvideUpdateRecurringDeductionTool(s *driverpayservice.Service) agentports.AgentTool {
	return &RecurringPayTool[driverpay.RecurringDeduction]{
		kind:   deductionKind(),
		book:   DeductionBook{pay: s},
		update: true,
	}
}

func ProvideCreateRecurringEarningTool(s *driverpayservice.Service) agentports.AgentTool {
	return &RecurringPayTool[driverpay.RecurringEarning]{
		kind: earningKind(),
		book: EarningBook{pay: s},
	}
}

func ProvideUpdateRecurringEarningTool(s *driverpayservice.Service) agentports.AgentTool {
	return &RecurringPayTool[driverpay.RecurringEarning]{
		kind:   earningKind(),
		book:   EarningBook{pay: s},
		update: true,
	}
}
