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
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramRecurringDescription = "description"
	paramRecurringFrequency   = "frequency"
	paramRecurringStatus      = "status"
	paramRecurringCap         = "totalCap"
	paramRecurringStart       = "startDate"
	paramRecurringEnd         = "endDate"
	paramEscrowContribution   = "escrowContribution"
	maxRecurringDescription   = 255
	fieldRecurringFrequency   = "frequency"
	fieldRecurringStart       = "startDate"
	fieldRecurringEnd         = "endDate"
	fieldEscrowAccountID      = "escrowAccountId"
)

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

type recurringBook[E any] interface {
	get(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID) (*E, error)
	check(ctx context.Context, entity *E, escrow bool) error
	create(
		ctx context.Context,
		entity *E,
		escrow bool,
		actor *serviceports.RequestActor,
	) (*E, error)
	update(ctx context.Context, entity *E, actor *serviceports.RequestActor) (*E, error)
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
	statuses    []string
	frequencies []string
	escrow      bool
	create      func(tenant pagination.TenantInfo, fields *recurringFields) *E
	apply       func(entity *E, fields *recurringFields, given map[string]bool)
	facts       func(entity *E) recurringFacts
	sensitive   string
}

type recurringPayTool[E any] struct {
	kind   recurringKind[E]
	book   recurringBook[E]
	update bool
}

var (
	_ serviceports.ToolPreviewer = (*recurringPayTool[driverpay.RecurringDeduction])(nil)
	_ serviceports.ToolValidator = (*recurringPayTool[driverpay.RecurringDeduction])(nil)
	_ serviceports.TargetedTool  = (*recurringPayTool[driverpay.RecurringDeduction])(nil)
)

func (t *recurringPayTool[E]) Name() string {
	if t.update {
		return t.kind.updateName
	}

	return t.kind.createName
}

func (t *recurringPayTool[E]) Description() string {
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

func (t *recurringPayTool[E]) ParamSchema() map[string]any {
	properties := map[string]any{
		paramPayCodeID: idProperty("The " + t.kind.payee + " pay code it posts under, from " +
			"list_pay_codes."),
		paramRecurringDescription: stringProperty("What it is for, as it will read on the "+
			"driver's statement.", maxRecurringDescription),
		paramDriverPayAmount: amountProperty("The amount each time, as a decimal such as " +
			"25.00."),
		paramRecurringFrequency: map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyEnum:        t.kind.frequencies,
			toolschema.KeyDescription: "How often it applies.",
		},
		paramRecurringCap: amountProperty("The total after which it stops, when there is " +
			"one."),
		paramRecurringStart: dateProperty("The first day it applies."),
		paramRecurringEnd:   dateProperty("The last day it applies, when it ends."),
	}
	if t.update {
		properties[t.kind.idParam] = idProperty("The " + t.kind.noun + ", from " +
			t.kind.listTool + ". Never guess one.")
		properties[paramRecurringStatus] = map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyEnum:        t.kind.statuses,
			toolschema.KeyDescription: "Paused stops it for now; Completed ends it for good.",
		}

		return objectParams(properties, t.kind.idParam)
	}

	properties[paramWorkerID] = workerProperty()
	if t.kind.escrow {
		properties[paramEscrowContribution] = map[string]any{
			toolschema.KeyType: toolschema.TypeBoolean,
			toolschema.KeyDescription: "true to pay it into the driver's active escrow " +
				"account, from open_escrow_account.",
		}
	}

	return objectParams(
		properties,
		paramWorkerID,
		paramPayCodeID,
		paramRecurringDescription,
		paramDriverPayAmount,
		paramRecurringStart,
	)
}

func (t *recurringPayTool[E]) Policy() serviceports.ToolPolicy {
	operation := permission.OpCreate
	rationale := "Sets money taken from or added to a driver's pay every settlement; only a " +
		"person agrees it."
	if t.update {
		operation = permission.OpUpdate
		rationale = "Changes money taken from or added to a driver's pay every settlement; " +
			"only a person agrees it."
	}

	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  t.kind.resource,
		operation: operation,
		rationale: rationale,
	}).policy()
}

func (t *recurringPayTool[E]) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	if !t.update {
		return serviceports.ToolTarget{}, false
	}

	return targetOf(params, t.kind.idParam, t.kind.resource)
}

func readRecurringFields(
	params map[string]any,
	kind recurringKindText,
	required bool,
) (*recurringFields, map[string]bool, error) {
	fields := &recurringFields{}
	given := map[string]bool{}

	var err error
	if fields.frequency, given[paramRecurringFrequency], err = optionalEnum(
		params, paramRecurringFrequency, kind.frequencies,
	); err != nil {
		return nil, nil, err
	}
	if fields.status, given[paramRecurringStatus], err = optionalEnum(
		params, paramRecurringStatus, kind.statuses,
	); err != nil {
		return nil, nil, err
	}
	if fields.description, err = boundedString(
		params, paramRecurringDescription, maxRecurringDescription, required,
	); err != nil {
		return nil, nil, err
	}
	given[paramRecurringDescription] = fields.description != ""
	if fields.amount, given[paramDriverPayAmount], err = optionalMoney(
		params, paramDriverPayAmount,
	); err != nil {
		return nil, nil, err
	}
	if required && !given[paramDriverPayAmount] {
		return nil, nil, fmt.Errorf("parameter %q is required", paramDriverPayAmount)
	}
	capMinor, hasCap, err := optionalMoney(params, paramRecurringCap)
	if err != nil {
		return nil, nil, err
	}
	if hasCap {
		fields.capMinor = &capMinor
	}
	given[paramRecurringCap] = hasCap
	if fields.start, given[paramRecurringStart], err = optionalDay(
		params, paramRecurringStart,
	); err != nil {
		return nil, nil, err
	}
	if required && !given[paramRecurringStart] {
		return nil, nil, fmt.Errorf("parameter %q is required", paramRecurringStart)
	}
	end, hasEnd, err := optionalDay(params, paramRecurringEnd)
	if err != nil {
		return nil, nil, err
	}
	if hasEnd {
		fields.end = &end
	}
	given[paramRecurringEnd] = hasEnd
	payCode, err := optionalPulidParam(params, paramPayCodeID)
	if err != nil {
		return nil, nil, err
	}
	if payCode != nil {
		fields.payCodeID = *payCode
	}
	given[paramPayCodeID] = payCode != nil

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

func (t *recurringPayTool[E]) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*recurringPlan[E], error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	text := recurringKindText{frequencies: t.kind.frequencies, statuses: t.kind.statuses}
	tenant := tenantFrom(*params)
	if !t.update {
		workerID, err := requirePulid(params.Params, paramWorkerID)
		if err != nil {
			return nil, err
		}
		if _, err = requirePulid(params.Params, paramPayCodeID); err != nil {
			return nil, err
		}
		fields, _, err := readRecurringFields(params.Params, text, true)
		if err != nil {
			return nil, err
		}
		fields.workerID = workerID
		entity := t.kind.create(tenant, fields)
		escrow := t.kind.escrow && optionalBool(params.Params, paramEscrowContribution)
		refusal, failure := refusalOrFailure(t.book.check(ctx, entity, escrow))
		if failure != nil {
			return nil, failure
		}

		return &recurringPlan[E]{after: entity, escrow: escrow, refusal: refusal}, nil
	}

	id, err := requirePulid(params.Params, t.kind.idParam)
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
		return nil, errNothingToChange
	}
	before, err := t.book.get(ctx, tenant, id)
	if err != nil {
		return nil, err
	}
	after := *before
	t.kind.apply(&after, fields, given)
	refusal, failure := refusalOrFailure(t.book.check(ctx, &after, false))
	if failure != nil {
		return nil, failure
	}

	return &recurringPlan[E]{before: before, after: &after, refusal: refusal}, nil
}

func (t *recurringPayTool[E]) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *recurringPayTool[E]) Execute(
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
	if t.update {
		_, err = t.book.update(ctx, plan.after, params.Actor)

		return err
	}
	_, err = t.book.create(ctx, plan.after, plan.escrow, params.Actor)

	return err
}

func (t *recurringPayTool[E]) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		return wouldFail(toolpreview.Build(summary), plan.refusal)
	}

	fields := []string{
		fieldStatus, fieldRecurringFrequency, paramRecurringDescription, fieldRecurringStart,
		fieldRecurringEnd, fieldEscrowAccountID,
	}
	var change *agent.RecordChange
	if t.update {
		before := t.kind.facts(plan.before)
		change, err = toolpreview.Changed(toolpreview.Record{
			Resource: t.kind.resource,
			ID:       before.id,
			Label:    before.description,
			Version:  pinnedVersion(before.version),
		}, plan.before, plan.after,
			toolpreview.Only(fields...),
			toolpreview.Types(driverPayDateTypes),
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
		toolpreview.Only(append(fields, paramWorkerID)...),
		toolpreview.WithRefs(map[string]permission.Resource{
			paramWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(driverPayDateTypes),
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
	amount := agent.MoneyLine{Label: "Each time", After: minorAmount(after.amount)}
	if before != nil {
		amount.Before = minorAmount(before.amount)
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
		statuses:    enumNames(deductionStatuses),
		frequencies: enumNames(deductionFrequencies),
		escrow:      true,
		sensitive:   "amountMinor",
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
		statuses:    enumNames(earningStatuses),
		frequencies: enumNames(earningFrequencies),
		sensitive:   "amountMinor",
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
	if given[paramPayCodeID] {
		*target.payCodeID = fields.payCodeID
	}
	if given[paramRecurringDescription] {
		*target.description = fields.description
	}
	if given[paramDriverPayAmount] {
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

var (
	_ recurringBook[driverpay.RecurringDeduction] = deductionBook{}
	_ recurringBook[driverpay.RecurringEarning]   = earningBook{}
)

type deductionBook struct {
	pay *driverpayservice.Service
}

func (b deductionBook) get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.RecurringDeduction, error) {
	return b.pay.GetDeduction(ctx, repositories.GetRecurringDeductionByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}

func (b deductionBook) check(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	escrow bool,
) error {
	return b.pay.CheckDeduction(ctx, entity, escrow)
}

func (b deductionBook) create(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	escrow bool,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	return b.pay.CreateDeduction(ctx, entity, escrow, actor)
}

func (b deductionBook) update(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringDeduction, error) {
	return b.pay.UpdateDeduction(ctx, entity, actor)
}

type earningBook struct {
	pay *driverpayservice.Service
}

func (b earningBook) get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.RecurringEarning, error) {
	return b.pay.GetEarning(ctx, repositories.GetRecurringEarningByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
}

func (b earningBook) check(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	_ bool,
) error {
	return b.pay.CheckEarning(ctx, entity)
}

func (b earningBook) create(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	_ bool,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringEarning, error) {
	return b.pay.CreateEarning(ctx, entity, actor)
}

func (b earningBook) update(
	ctx context.Context,
	entity *driverpay.RecurringEarning,
	actor *serviceports.RequestActor,
) (*driverpay.RecurringEarning, error) {
	return b.pay.UpdateEarning(ctx, entity, actor)
}

func provideCreateRecurringDeductionTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &recurringPayTool[driverpay.RecurringDeduction]{
		kind: deductionKind(),
		book: deductionBook{pay: s},
	}
}

func provideUpdateRecurringDeductionTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &recurringPayTool[driverpay.RecurringDeduction]{
		kind:   deductionKind(),
		book:   deductionBook{pay: s},
		update: true,
	}
}

func provideCreateRecurringEarningTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &recurringPayTool[driverpay.RecurringEarning]{
		kind: earningKind(),
		book: earningBook{pay: s},
	}
}

func provideUpdateRecurringEarningTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &recurringPayTool[driverpay.RecurringEarning]{
		kind:   earningKind(),
		book:   earningBook{pay: s},
		update: true,
	}
}
