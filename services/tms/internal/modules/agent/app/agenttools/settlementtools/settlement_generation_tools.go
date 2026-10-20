package settlementtools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramPayDate = "payDate"

const paramBatchName = "name"

const maxBatchNameChars = 100

const maxBatchNotesChars = 1000

const maxBatchPreviewRecords = 20

var ErrNothingToSettle = errors.New(
	"no settlement was generated: the worker has nothing accrued for the period or it " +
		"is already settled",
)

var errPeriodBackwards = errors.New("periodEnd is before periodStart")

var errHalfAPeriod = errors.New(
	"give both periodStart and periodEnd, or neither for the current period",
)

func periodProperties(what string) map[string]any {
	return map[string]any{
		common.ParamPeriodStart: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The first day " + what + " covers, YYYY-MM-DD. Leave " +
				"both days out for the current pay period.",
		},
		common.ParamPeriodEnd: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The last day " + what + " covers, YYYY-MM-DD, " +
				"included.",
		},
	}
}

type requestedPeriod struct {
	start int64
	end   int64
	given bool
}

func readPeriod(params map[string]any) (requestedPeriod, error) {
	start, hasStart, err := common.OptionalDay(params, common.ParamPeriodStart)
	if err != nil {
		return requestedPeriod{}, err
	}
	lastDay, hasEnd, err := common.OptionalDay(params, common.ParamPeriodEnd)
	if err != nil {
		return requestedPeriod{}, err
	}
	if hasStart != hasEnd {
		return requestedPeriod{}, errHalfAPeriod
	}
	if !hasStart {
		return requestedPeriod{}, nil
	}
	if lastDay < start {
		return requestedPeriod{}, errPeriodBackwards
	}

	return requestedPeriod{start: start, end: lastDay + common.SecondsPerDay, given: true}, nil
}

type workerSettlementGenerator interface {
	CurrentPeriod(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (driversettlementservice.PeriodBounds, error)
	PlanGenerateForWorker(
		ctx context.Context,
		req *driversettlementservice.GenerateForWorkerRequest,
	) (*driversettlementservice.GenerationPlan, error)
	GenerateOffCycle(
		ctx context.Context,
		req *driversettlementservice.GenerateForWorkerRequest,
		actor *permissionports.RequestActor,
	) (*driversettlement.Settlement, error)
}

type GenerateDriverSettlementTool struct {
	settlements workerSettlementGenerator
}

func ProvideGenerateDriverSettlementTool(
	s *driversettlementservice.Service,
) agentports.AgentTool {
	return &GenerateDriverSettlementTool{settlements: s}
}

func (t *GenerateDriverSettlementTool) Name() string { return "settlement_generate_driver" }

func (t *GenerateDriverSettlementTool) Description() string {
	return "Generate one driver's draft settlement for a pay period from the pay they have " +
		"accrued, with their recurring earnings, deductions, escrow and advance recovery. " +
		"Use it for a driver the batch missed or an off-cycle statement; " +
		"settlement_generate_driver_batch settles everyone. A period already settled for " +
		"the driver is refused. The draft is shown to the person before anything is kept, " +
		"so when they name a period (\"last week\", \"the first half of the month\") work " +
		"out its days from today's date and propose it rather than asking which dates."
}

func (t *GenerateDriverSettlementTool) ParamSchema() map[string]any {
	properties := periodProperties("the settlement")
	properties[common.ParamWorkerID] = agenttoolschema.RecordIDText(permission.ResourceWorker,
		"The driver, from worker_search, worker_list or settlement_list_pay_events.")
	properties[paramPayDate] = agenttoolschema.Date("The day it is paid. Leave it out for " +
		"the pay period's own pay date.")

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             []string{common.ParamWorkerID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *GenerateDriverSettlementTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDriverSettlement,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Artifact:      common.DriverSettlementEntity,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Drafts a settlement from pay already accrued inside Trenova; nothing is " +
			"paid until a person approves it, and hands-off approval is only the settlement " +
			"control's clean-settlement rule.",
	}
}

func (t *GenerateDriverSettlementTool) request(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*driversettlementservice.GenerateForWorkerRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	period, err := readPeriod(params.Params)
	if err != nil {
		return nil, err
	}
	payDate, hasPayDate, err := common.OptionalDay(params.Params, paramPayDate)
	if err != nil {
		return nil, err
	}

	tenant := common.TenantFrom(*params)
	req := &driversettlementservice.GenerateForWorkerRequest{
		TenantInfo:  tenant,
		WorkerID:    workerID,
		PeriodStart: period.start,
		PeriodEnd:   period.end,
		PayDate:     payDate,
	}
	if period.given && hasPayDate {
		return req, nil
	}

	current, err := t.settlements.CurrentPeriod(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if !period.given {
		req.PeriodStart = current.PeriodStart
		req.PeriodEnd = current.PeriodEnd
	}
	if !hasPayDate {
		req.PayDate = current.PayDate
		if period.given {
			req.PayDate = req.PeriodEnd + (current.PayDate - current.PeriodEnd)
		}
	}

	return req, nil
}

func (t *GenerateDriverSettlementTool) plan(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*driversettlementservice.GenerateForWorkerRequest, *driversettlementservice.GenerationPlan, error) {
	req, err := t.request(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	plan, err := t.settlements.PlanGenerateForWorker(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return req, plan, nil
}

func generationRefusal(plan *driversettlementservice.GenerationPlan) error {
	switch {
	case plan.Refusal != nil:
		return plan.Refusal
	case plan.AlreadySettled:
		return errors.New("the driver already has a settlement for this period")
	case plan.Settlement == nil:
		return errors.New("the driver has no accrued, unheld pay through the end of this period")
	default:
		return nil
	}
}

func (t *GenerateDriverSettlementTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return generationRefusal(plan)
}

func (t *GenerateDriverSettlementTool) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return nil, err
	}
	req, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}
	created, err := t.settlements.GenerateOffCycle(ctx, req, params.Actor)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, ErrNothingToSettle
	}

	return common.SettlementResult(created), nil
}

func (t *GenerateDriverSettlementTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *GenerateDriverSettlementTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would generate a draft driver settlement for %s through %s, paid %s.",
		common.DayText(req.PeriodStart),
		common.DayText(req.PeriodEnd-common.SecondsPerDay),
		common.DayText(req.PayDate),
	)
	if refusal := generationRefusal(plan); refusal != nil {
		return common.WouldFail(toolpreview.Build(summary), refusal)
	}

	change, err := common.NewSettlementChange(plan.Settlement, "New driver settlement")
	if err != nil {
		return nil, err
	}
	if plan.Settlement.HasExceptions {
		summary += fmt.Sprintf(" It raises %s for review.",
			common.CountOf(len(plan.Settlement.Exceptions), "exception"))
	}
	if plan.AutoApprove {
		summary += " The settlement control approves a clean settlement as soon as it is " +
			"generated."
	}

	return toolpreview.Build(summary, change), nil
}

type batchText struct {
	name      string
	recipe    []string
	resource  permission.Resource
	rationale string
}

func batchSchema() map[string]any {
	properties := periodProperties("the batch")
	properties[paramBatchName] = common.StringProperty("A name for the batch; leave it out for "+
		"\"Pay period ending\" and the period's last day.", maxBatchNameChars)
	properties[common.ParamNote] = common.StringProperty("Notes kept on the batch for payroll.",
		maxBatchNotesChars)

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
}

func batchPolicy(text batchText) agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          text.name,
		Kind:          agent.ToolKindAction,
		Resource:      text.resource,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     text.rationale,
	}
}

type batchInput struct {
	period requestedPeriod
	name   string
	notes  string
}

func readBatchInput(params map[string]any) (batchInput, error) {
	period, err := readPeriod(params)
	if err != nil {
		return batchInput{}, err
	}
	name, err := common.BoundedString(params, paramBatchName, maxBatchNameChars, false)
	if err != nil {
		return batchInput{}, err
	}
	notes, err := common.BoundedString(params, common.ParamNote, maxBatchNotesChars, false)
	if err != nil {
		return batchInput{}, err
	}

	return batchInput{period: period, name: name, notes: notes}, nil
}

type driverBatchGenerator interface {
	PlanBatch(
		ctx context.Context,
		req *driversettlementservice.GenerateBatchRequest,
	) (*driversettlementservice.BatchPlan, error)
	GenerateBatch(
		ctx context.Context,
		req *driversettlementservice.GenerateBatchRequest,
		actor *permissionports.RequestActor,
	) (*driversettlement.SettlementBatch, error)
}

type GenerateDriverBatchTool struct {
	settlements driverBatchGenerator
}

func ProvideGenerateDriverSettlementBatchTool(
	s *driversettlementservice.Service,
) agentports.AgentTool {
	return &GenerateDriverBatchTool{settlements: s}
}

func driverBatchText() batchText {
	return batchText{
		name: "settlement_generate_driver_batch",
		recipe: []string{
			"settlement_list_pay_events",
			"settlement_generate_driver_batch",
			toolListDriverSettlements,
		},
		resource: permission.ResourceDriverSettlement,
		rationale: "Drafts the period's settlements from pay already accrued inside Trenova; " +
			"nothing is paid until a person approves them, and hands-off approval is only " +
			"the settlement control's clean-settlement rule.",
	}
}

func (t *GenerateDriverBatchTool) Name() string { return driverBatchText().name }

func (t *GenerateDriverBatchTool) Recipe() []string { return driverBatchText().recipe }

func (t *GenerateDriverBatchTool) Description() string {
	return "Generate the pay period's driver settlement batch: a draft settlement for every " +
		"driver with accrued, unheld pay who has none for the period yet. An open batch for " +
		"the period is added to; a completed one is refused. Review the drafts with " +
		"settlement_list_driver afterwards."
}

func (t *GenerateDriverBatchTool) ParamSchema() map[string]any {
	return batchSchema()
}

func (t *GenerateDriverBatchTool) Policy() agentports.ToolPolicy {
	return batchPolicy(driverBatchText())
}

func (t *GenerateDriverBatchTool) request(
	params *agentports.ToolExecuteParams,
) (*driversettlementservice.GenerateBatchRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	input, err := readBatchInput(params.Params)
	if err != nil {
		return nil, err
	}

	return &driversettlementservice.GenerateBatchRequest{
		TenantInfo:  common.TenantFrom(*params),
		PeriodStart: input.period.start,
		PeriodEnd:   input.period.end,
		Name:        input.name,
		Notes:       input.notes,
	}, nil
}

func (t *GenerateDriverBatchTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	plan, err := t.settlements.PlanBatch(ctx, req)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *GenerateDriverBatchTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.settlements.GenerateBatch(ctx, req, params.Actor)

	return err
}

type plannedWorkerSettlement struct {
	WorkerID      pulid.ID `json:"workerId"`
	PeriodStart   int64    `json:"periodStart"`
	PeriodEnd     int64    `json:"periodEnd"`
	PayDate       int64    `json:"payDate"`
	PayEventCount int      `json:"payEventCount"`
	HeldCount     int      `json:"heldCount"`
}

func (t *GenerateDriverBatchTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	plan, err := t.settlements.PlanBatch(ctx, req)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would generate the driver settlement batch for %s through %s, paid %s.",
		common.DayText(plan.Bounds.PeriodStart),
		common.DayText(plan.Bounds.PeriodEnd-common.SecondsPerDay),
		common.DayText(plan.Bounds.PayDate),
	)
	if plan.Refusal != nil {
		return common.WouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	changes := make([]*agent.RecordChange, 0, len(plan.Workers))
	for _, worker := range plan.Workers {
		if worker == nil || worker.HasSettlement || worker.EventCount == 0 {
			continue
		}
		change, changeErr := plannedWorkerChange(worker, plan.Bounds)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	summary += fmt.Sprintf(
		" %s would get a draft settlement.",
		common.CountOf(len(changes), "driver"),
	)
	if plan.ExistingBatch != nil {
		summary += " They join the open batch " + plan.ExistingBatch.Name + "."
	}
	if plan.AutoApprove {
		summary += " The settlement control approves each clean settlement as soon as it is " +
			"generated."
	}
	summary += " Recurring deductions, escrow and advance recovery are computed as each " +
		"settlement is generated."

	preview := toolpreview.Build(summary, changes...)
	preview.Partial = true

	return preview, nil
}

func plannedWorkerChange(
	worker *driversettlementports.UnsettledWorkerSummary,
	bounds driversettlementservice.PeriodBounds,
) (*agent.RecordChange, error) {
	label := worker.WorkerName
	if label == "" {
		label = "Driver settlement"
	}
	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceDriverSettlement, Label: label},
		&plannedWorkerSettlement{
			WorkerID:      worker.WorkerID,
			PeriodStart:   bounds.PeriodStart,
			PeriodEnd:     bounds.PeriodEnd,
			PayDate:       bounds.PayDate,
			PayEventCount: worker.EventCount,
			HeldCount:     worker.HeldCount,
		},
		toolpreview.WithRefs(map[string]permission.Resource{
			common.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(common.SettlementDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		money.DefaultCurrencyCode,
		agent.MoneyLine{
			Label: "Accrued earnings",
			After: common.MinorAmount(worker.GrossAmountMinor),
		},
	), toolpreview.SensitiveAs("grossAmountMinor"))

	return change, nil
}

type carrierBatchGenerator interface {
	PlanBatch(
		ctx context.Context,
		req *carriersettlementservice.BatchPlanRequest,
	) (*carriersettlementservice.BatchPlan, error)
	GenerateBatch(
		ctx context.Context,
		req *carriersettlementservice.GenerateBatchRequest,
		actor *permissionports.RequestActor,
	) (*carriersettlement.CarrierSettlementBatch, error)
}

type GenerateCarrierBatchTool struct {
	settlements carrierBatchGenerator
}

func ProvideGenerateCarrierSettlementBatchTool(
	s *carriersettlementservice.Service,
) agentports.AgentTool {
	return &GenerateCarrierBatchTool{settlements: s}
}

func carrierBatchText() batchText {
	return batchText{
		name:     "settlement_generate_carrier_batch",
		resource: permission.ResourceCarrierSettlement,
		rationale: "Drafts the period's carrier settlements from cost already accrued inside " +
			"Trenova; nothing is paid until a person approves and posts them.",
	}
}

func (t *GenerateCarrierBatchTool) Name() string { return carrierBatchText().name }

func (t *GenerateCarrierBatchTool) Description() string {
	return "Generate the pay period's carrier settlement batch, a draft settlement for each " +
		"carrier with pending cost. A carrier that already has one for the period is skipped. " +
		"Review the drafts with settlement_list_carrier afterwards."
}

func (t *GenerateCarrierBatchTool) ParamSchema() map[string]any {
	return batchSchema()
}

func (t *GenerateCarrierBatchTool) Policy() agentports.ToolPolicy {
	return batchPolicy(carrierBatchText())
}

func (t *GenerateCarrierBatchTool) request(
	params *agentports.ToolExecuteParams,
) (*carriersettlementservice.GenerateBatchRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	input, err := readBatchInput(params.Params)
	if err != nil {
		return nil, err
	}

	return &carriersettlementservice.GenerateBatchRequest{
		TenantInfo:  common.TenantFrom(*params),
		PeriodStart: input.period.start,
		PeriodEnd:   input.period.end,
		Name:        input.name,
		Notes:       input.notes,
	}, nil
}

func (t *GenerateCarrierBatchTool) planned(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*carriersettlementservice.BatchPlan, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.settlements.PlanBatch(ctx, &carriersettlementservice.BatchPlanRequest{
		Batch: req,
		Limit: maxBatchPreviewRecords,
	})
}

func (t *GenerateCarrierBatchTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *GenerateCarrierBatchTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.settlements.GenerateBatch(ctx, req, params.Actor)

	return err
}

func (t *GenerateCarrierBatchTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would generate the carrier settlement batch for %s through %s, paid %s.",
		common.DayText(plan.Bounds.PeriodStart),
		common.DayText(plan.Bounds.PeriodEnd-common.SecondsPerDay),
		common.DayText(plan.Bounds.PayDate),
	)
	if plan.Refusal != nil {
		return common.WouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	changes := make([]*agent.RecordChange, 0, len(plan.Settlements))
	for _, settlement := range plan.Settlements {
		change, changeErr := toolpreview.Create(
			toolpreview.Record{
				Resource: permission.ResourceCarrierSettlement,
				Label:    "New carrier settlement",
			},
			settlement,
			toolpreview.Only(
				common.ParamCarrierID,
				common.FieldStatus,
				common.FieldPeriodStart,
				common.FieldPeriodEnd,
				common.FieldPayDate,
				"shipmentCount",
			),
			toolpreview.WithRefs(map[string]permission.Resource{
				common.ParamCarrierID: permission.ResourceCarrier,
			}),
			toolpreview.Types(common.SettlementDateTypes),
		)
		if changeErr != nil {
			return nil, changeErr
		}
		toolpreview.AttachMoney(
			change,
			common.SettlementMoney(&common.SettlementFacts{}, carrierSettlementFacts(settlement)),
			toolpreview.SensitiveAs(carrierSettlementLedger().Sensitive...),
		)
		changes = append(changes, change)
	}

	remaining := plan.CarrierCount - plan.SettledCount - len(plan.Settlements)
	summary += fmt.Sprintf(
		" Pending cost for the period covers %s; each without a settlement for it would get "+
			"a draft.",
		common.CountOf(plan.CarrierCount, "carrier"),
	)
	if plan.ExistingBatch != nil {
		summary += " They join the batch " + plan.ExistingBatch.Name + "."
	}
	preview := toolpreview.Build(summary, changes...)
	if remaining > 0 {
		preview.Partial = true
		preview.OmittedRecords += remaining
	}

	return preview, nil
}
