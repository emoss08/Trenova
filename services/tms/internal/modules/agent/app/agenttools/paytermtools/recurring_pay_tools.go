package paytermtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
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
	Get(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID) (*E, error)
	Check(ctx context.Context, entity *E, escrow bool) error
	Create(
		ctx context.Context,
		entity *E,
		escrow bool,
		actor *permissionports.RequestActor,
	) (*E, error)
	Update(ctx context.Context, entity *E, actor *permissionports.RequestActor) (*E, error)
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
		toolparams.ParamPayCodeID: agenttoolschema.RecordIDText(
			permission.ResourcePayCode,
			"The "+t.kind.payee+" pay code it posts under, from settlement_list_pay_codes.",
		),
		paramRecurringDescription: toolparams.StringProperty(
			"What it is for, as it will read on the "+
				"driver's statement.",
			maxRecurringDescription,
		),
		settlementpreview.ParamDriverPayAmount: toolparams.AmountProperty(
			"The amount each time, as a decimal such as " +
				"25.00.",
		),
		paramRecurringFrequency: agenttoolschema.Enum("How often it applies.", t.kind.frequencies),
		paramRecurringCap: toolparams.AmountProperty(
			"The total after which it stops, when there is " +
				"one.",
		),
		paramRecurringStart: agenttoolschema.Date("The first day it applies."),
		paramRecurringEnd:   agenttoolschema.Date("The last day it applies, when it ends."),
	}
	if t.update {
		properties[t.kind.idParam] = agenttoolschema.RecordIDText(t.kind.resource,
			"The "+t.kind.noun+", from "+t.kind.listTool+". Never guess one.")
		properties[paramRecurringStatus] = agenttoolschema.Enum(
			"Paused stops it for now; Completed ends it for good.", t.kind.statuses,
		)

		return toolparams.ObjectParams(properties, t.kind.idParam)
	}

	properties[toolparams.ParamWorkerID] = toolparams.WorkerProperty()
	if t.kind.escrow {
		properties[paramEscrowContribution] = map[string]any{
			toolschema.KeyType: toolschema.TypeBoolean,
			toolschema.KeyDescription: "true to pay it into the driver's active escrow " +
				"account, from settlement_open_escrow_account.",
		}
	}

	return toolparams.ObjectParams(
		properties,
		toolparams.ParamWorkerID,
		toolparams.ParamPayCodeID,
		paramRecurringDescription,
		settlementpreview.ParamDriverPayAmount,
		paramRecurringStart,
	)
}

func (t *RecurringPayTool[E]) Policy() agenttoolports.ToolPolicy {
	operation := permission.OpCreate
	rationale := "Sets money taken from or added to a driver's pay every settlement; only a " +
		"person agrees it."
	if t.update {
		operation = permission.OpUpdate
		rationale = "Changes money taken from or added to a driver's pay every settlement; " +
			"only a person agrees it."
	}

	return (&settlementpreview.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  t.kind.resource,
		Operation: operation,
		Rationale: rationale,
	}).Policy()
}

func (t *RecurringPayTool[E]) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	if !t.update {
		return agenttoolports.ToolTarget{}, false
	}

	return toolparams.TargetOf(params, t.kind.idParam, t.kind.resource)
}

func readRecurringFields(
	params map[string]any,
	kind recurringKindText,
	required bool,
) (*recurringFields, map[string]bool, error) {
	fields := &recurringFields{}
	given := map[string]bool{}

	var err error
	if fields.frequency, given[paramRecurringFrequency], err = toolparams.OptionalEnum(
		params, paramRecurringFrequency, kind.frequencies,
	); err != nil {
		return nil, nil, err
	}
	if fields.status, given[paramRecurringStatus], err = toolparams.OptionalEnum(
		params, paramRecurringStatus, kind.statuses,
	); err != nil {
		return nil, nil, err
	}
	if fields.description, err = toolparams.BoundedString(
		params, paramRecurringDescription, maxRecurringDescription, required,
	); err != nil {
		return nil, nil, err
	}
	given[paramRecurringDescription] = fields.description != ""
	if fields.amount, given[settlementpreview.ParamDriverPayAmount], err = toolparams.OptionalMoney(
		params, settlementpreview.ParamDriverPayAmount,
	); err != nil {
		return nil, nil, err
	}
	if required && !given[settlementpreview.ParamDriverPayAmount] {
		return nil, nil, fmt.Errorf(
			"parameter %q is required",
			settlementpreview.ParamDriverPayAmount,
		)
	}
	capMinor, hasCap, err := toolparams.OptionalMoney(params, paramRecurringCap)
	if err != nil {
		return nil, nil, err
	}
	if hasCap {
		fields.capMinor = &capMinor
	}
	given[paramRecurringCap] = hasCap
	if fields.start, given[paramRecurringStart], err = toolparams.OptionalDay(
		params, paramRecurringStart,
	); err != nil {
		return nil, nil, err
	}
	if required && !given[paramRecurringStart] {
		return nil, nil, fmt.Errorf("parameter %q is required", paramRecurringStart)
	}
	end, hasEnd, err := toolparams.OptionalDay(params, paramRecurringEnd)
	if err != nil {
		return nil, nil, err
	}
	if hasEnd {
		fields.end = &end
	}
	given[paramRecurringEnd] = hasEnd
	payCode, err := toolparams.OptionalPulidParam(params, toolparams.ParamPayCodeID)
	if err != nil {
		return nil, nil, err
	}
	if payCode != nil {
		fields.payCodeID = *payCode
	}
	given[toolparams.ParamPayCodeID] = payCode != nil

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
	params *agenttoolports.ToolExecuteParams,
) (*recurringPlan[E], error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	text := recurringKindText{
		frequencies: t.kind.frequencies.Values,
		statuses:    t.kind.statuses.Values,
	}
	tenant := toolparams.TenantFrom(params)
	if !t.update {
		workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
		if err != nil {
			return nil, err
		}
		if _, err = toolparams.RequirePulid(params.Params, toolparams.ParamPayCodeID); err != nil {
			return nil, err
		}
		fields, _, err := readRecurringFields(params.Params, text, true)
		if err != nil {
			return nil, err
		}
		fields.workerID = workerID
		entity := t.kind.create(tenant, fields)
		escrow := t.kind.escrow && toolparams.OptionalBool(params.Params, paramEscrowContribution)
		refusal, failure := changepreview.RefusalOrFailure(t.book.Check(ctx, entity, escrow))
		if failure != nil {
			return nil, failure
		}

		return &recurringPlan[E]{after: entity, escrow: escrow, refusal: refusal}, nil
	}

	id, err := toolparams.RequirePulid(params.Params, t.kind.idParam)
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
		return nil, toolparams.ErrNothingToChange
	}
	before, err := t.book.Get(ctx, tenant, id)
	if err != nil {
		return nil, err
	}
	after := *before
	t.kind.apply(&after, fields, given)
	refusal, failure := changepreview.RefusalOrFailure(t.book.Check(ctx, &after, false))
	if failure != nil {
		return nil, failure
	}

	return &recurringPlan[E]{before: before, after: &after, refusal: refusal}, nil
}

func (t *RecurringPayTool[E]) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *RecurringPayTool[E]) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, &params); err != nil {
		return err
	}
	if err := settlementpreview.RequirePersonsApproval(&params); err != nil {
		return err
	}
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}
	if t.update {
		_, err = t.book.Update(ctx, plan.after, params.Actor)

		return err
	}
	_, err = t.book.Create(ctx, plan.after, plan.escrow, params.Actor)

	return err
}

func (t *RecurringPayTool[E]) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
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
		return changepreview.WouldFail(toolpreview.Build(summary), plan.refusal)
	}

	fields := []string{
		changepreview.FieldStatus,
		fieldRecurringFrequency,
		paramRecurringDescription,
		changepreview.FieldRecurringStart,
		changepreview.FieldRecurringEnd,
		fieldEscrowAccountID,
	}
	var change *agentpreview.RecordChange
	if t.update {
		before := t.kind.facts(plan.before)
		change, err = toolpreview.Changed(toolpreview.Record{
			Resource: t.kind.resource,
			ID:       before.id,
			Label:    before.description,
			Version:  changepreview.PinnedVersion(before.version),
		}, plan.before, plan.after,
			toolpreview.Only(fields...),
			toolpreview.Types(settlementpreview.DriverPayDateTypes),
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
		toolpreview.Only(append(fields, toolparams.ParamWorkerID)...),
		toolpreview.WithRefs(map[string]permission.Resource{
			toolparams.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(settlementpreview.DriverPayDateTypes),
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

func recurringMoney(before, after *recurringFacts) *agentpreview.MoneyPreview {
	amount := agentpreview.MoneyLine{
		Label: "Each time",
		After: changepreview.MinorAmount(after.amount),
	}
	if before != nil {
		amount.Before = changepreview.MinorAmount(before.amount)
	}

	return toolpreview.MoneyBlock(after.currency, amount)
}

func deductionKind() recurringKind[recurringpay.RecurringDeduction] {
	return recurringKind[recurringpay.RecurringDeduction]{
		noun:        "recurring deduction",
		createName:  "settlement_create_recurring_deduction",
		updateName:  "settlement_update_recurring_deduction",
		resource:    permission.ResourceRecurringDeduction,
		listTool:    "settlement_list_recurring_deductions",
		idParam:     "deductionId",
		payee:       "deduction",
		direction:   "taken from the driver's pay",
		statuses:    settlementpreview.DeductionStatuses.AsStrings(),
		frequencies: settlementpreview.DeductionFrequencies.AsStrings(),
		escrow:      true,
		sensitive:   "amountMinor",
		updateTerms: []string{"lease deduction", "raise deduction", "pause deduction"},
		create: func(
			tenant pagination.TenantInfo,
			fields *recurringFields,
		) *recurringpay.RecurringDeduction {
			entity := &recurringpay.RecurringDeduction{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				WorkerID:       fields.workerID,
				PayCodeID:      fields.payCodeID,
				Status:         recurringpay.DeductionStatusActive,
				Frequency:      recurringpay.DeductionFrequencyEverySettlement,
				Description:    fields.description,
				AmountMinor:    fields.amount,
				TotalCapMinor:  fields.capMinor,
				StartDate:      fields.start,
				EndDate:        fields.end,
				CurrencyCode:   money.DefaultCurrencyCode,
			}
			if fields.frequency != "" {
				entity.Frequency = recurringpay.DeductionFrequency(fields.frequency)
			}

			return entity
		},
		apply: func(
			entity *recurringpay.RecurringDeduction,
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
				entity.Frequency = recurringpay.DeductionFrequency(fields.frequency)
			}
			if given[paramRecurringStatus] {
				entity.Status = recurringpay.DeductionStatus(fields.status)
			}
		},
		facts: func(entity *recurringpay.RecurringDeduction) recurringFacts {
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

func earningKind() recurringKind[recurringpay.RecurringEarning] {
	return recurringKind[recurringpay.RecurringEarning]{
		noun:        "recurring earning",
		createName:  "settlement_create_recurring_earning",
		updateName:  "settlement_update_recurring_earning",
		resource:    permission.ResourceRecurringEarning,
		listTool:    "settlement_list_recurring_earnings",
		idParam:     "earningId",
		payee:       "earning",
		direction:   "added to the driver's pay",
		statuses:    settlementpreview.EarningStatuses.AsStrings(),
		frequencies: settlementpreview.EarningFrequencies.AsStrings(),
		sensitive:   "amountMinor",
		createTerms: []string{"bonus", "recurring bonus"},
		updateTerms: []string{"pause bonus", "stop bonus", "edit bonus"},
		create: func(
			tenant pagination.TenantInfo,
			fields *recurringFields,
		) *recurringpay.RecurringEarning {
			entity := &recurringpay.RecurringEarning{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				WorkerID:       fields.workerID,
				PayCodeID:      fields.payCodeID,
				Status:         recurringpay.EarningStatusActive,
				Frequency:      recurringpay.EarningFrequencyEverySettlement,
				Description:    fields.description,
				AmountMinor:    fields.amount,
				TotalCapMinor:  fields.capMinor,
				StartDate:      fields.start,
				EndDate:        fields.end,
				CurrencyCode:   money.DefaultCurrencyCode,
			}
			if fields.frequency != "" {
				entity.Frequency = recurringpay.EarningFrequency(fields.frequency)
			}

			return entity
		},
		apply: func(
			entity *recurringpay.RecurringEarning,
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
				entity.Frequency = recurringpay.EarningFrequency(fields.frequency)
			}
			if given[paramRecurringStatus] {
				entity.Status = recurringpay.EarningStatus(fields.status)
			}
		},
		facts: func(entity *recurringpay.RecurringEarning) recurringFacts {
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
	if given[toolparams.ParamPayCodeID] {
		*target.payCodeID = fields.payCodeID
	}
	if given[paramRecurringDescription] {
		*target.description = fields.description
	}
	if given[settlementpreview.ParamDriverPayAmount] {
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

func (b DeductionBook) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*recurringpay.RecurringDeduction, error) {
	return b.pay.GetDeduction(ctx, driverpayports.GetRecurringDeductionByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}

func (b DeductionBook) Check(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
	escrow bool,
) error {
	return b.pay.CheckDeduction(ctx, entity, escrow)
}

func (b DeductionBook) Create(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
	escrow bool,
	actor *permissionports.RequestActor,
) (*recurringpay.RecurringDeduction, error) {
	return b.pay.CreateDeduction(ctx, entity, escrow, actor)
}

func (b DeductionBook) Update(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
	actor *permissionports.RequestActor,
) (*recurringpay.RecurringDeduction, error) {
	return b.pay.UpdateDeduction(ctx, entity, actor)
}

type EarningBook struct {
	pay *driverpayservice.Service
}

func (b EarningBook) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*recurringpay.RecurringEarning, error) {
	return b.pay.GetEarning(ctx, driverpayports.GetRecurringEarningByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}

func (b EarningBook) Check(
	ctx context.Context,
	entity *recurringpay.RecurringEarning,
	_ bool,
) error {
	return b.pay.CheckEarning(ctx, entity)
}

func (b EarningBook) Create(
	ctx context.Context,
	entity *recurringpay.RecurringEarning,
	_ bool,
	actor *permissionports.RequestActor,
) (*recurringpay.RecurringEarning, error) {
	return b.pay.CreateEarning(ctx, entity, actor)
}

func (b EarningBook) Update(
	ctx context.Context,
	entity *recurringpay.RecurringEarning,
	actor *permissionports.RequestActor,
) (*recurringpay.RecurringEarning, error) {
	return b.pay.UpdateEarning(ctx, entity, actor)
}

func ProvideCreateRecurringDeductionTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &RecurringPayTool[recurringpay.RecurringDeduction]{
		kind: deductionKind(),
		book: DeductionBook{pay: s},
	}
}

func ProvideUpdateRecurringDeductionTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &RecurringPayTool[recurringpay.RecurringDeduction]{
		kind:   deductionKind(),
		book:   DeductionBook{pay: s},
		update: true,
	}
}

func ProvideCreateRecurringEarningTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &RecurringPayTool[recurringpay.RecurringEarning]{
		kind: earningKind(),
		book: EarningBook{pay: s},
	}
}

func ProvideUpdateRecurringEarningTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &RecurringPayTool[recurringpay.RecurringEarning]{
		kind:   earningKind(),
		book:   EarningBook{pay: s},
		update: true,
	}
}
