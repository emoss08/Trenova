package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramWorkerID          = "workerId"
	paramPayDate           = "payDate"
	paramBatchName         = "name"
	paramBatchNotes        = "notes"
	maxBatchNameChars      = 100
	maxBatchNotesChars     = 1000
	secondsPerDay          = int64(24 * time.Hour / time.Second)
	driverSettlementEntity = "driver_settlement"
	maxBatchPreviewRecords = 20
	fieldPeriodStart       = "periodStart"
	fieldPeriodEnd         = "periodEnd"
	fieldPayDate           = "payDate"
)

var (
	ErrNothingToSettle = errors.New(
		"no settlement was generated: the worker has nothing accrued for the period or it " +
			"is already settled",
	)
	errPeriodBackwards = errors.New("periodEnd is before periodStart")
	errHalfAPeriod     = errors.New(
		"give both periodStart and periodEnd, or neither for the current period",
	)
)

var settlementDateTypes = map[string]assistantartifact.DisplayType{
	fieldPeriodStart: assistantartifact.DisplayDate,
	fieldPeriodEnd:   assistantartifact.DisplayDate,
	fieldPayDate:     assistantartifact.DisplayDate,
}

func periodProperties(what string) map[string]any {
	return map[string]any{
		paramPeriodStart: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The first day " + what + " covers, YYYY-MM-DD. Leave " +
				"both days out for the current pay period.",
		},
		paramPeriodEnd: map[string]any{
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
	start, hasStart, err := optionalDay(params, paramPeriodStart)
	if err != nil {
		return requestedPeriod{}, err
	}
	lastDay, hasEnd, err := optionalDay(params, paramPeriodEnd)
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

	return requestedPeriod{start: start, end: lastDay + secondsPerDay, given: true}, nil
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
		actor *serviceports.RequestActor,
	) (*driversettlement.Settlement, error)
}

type generateDriverSettlementTool struct {
	settlements workerSettlementGenerator
}

var (
	_ serviceports.ToolPreviewer      = (*generateDriverSettlementTool)(nil)
	_ serviceports.ToolValidator      = (*generateDriverSettlementTool)(nil)
	_ serviceports.ToolResultReporter = (*generateDriverSettlementTool)(nil)
)

func provideGenerateDriverSettlementTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return &generateDriverSettlementTool{settlements: s}
}

func (t *generateDriverSettlementTool) Name() string { return "generate_driver_settlement" }

func (t *generateDriverSettlementTool) Description() string {
	return "Generate one driver's draft settlement for a pay period from the pay they have " +
		"accrued, with their recurring earnings, deductions, escrow and advance recovery. " +
		"Use it for a driver the batch missed or an off-cycle statement; " +
		"generate_driver_settlement_batch settles everyone. A period already settled for " +
		"the driver is refused."
}

func (t *generateDriverSettlementTool) ParamSchema() map[string]any {
	properties := periodProperties("the settlement")
	properties[paramWorkerID] = map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The driver, from search_worker, list_workers or " +
			"list_driver_pay_events.",
	}
	properties[paramPayDate] = map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The day it is paid, YYYY-MM-DD. Leave it out for the " +
			"pay period's own pay date.",
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             []string{paramWorkerID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *generateDriverSettlementTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDriverSettlement,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Artifact:      driverSettlementEntity,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Drafts a settlement from pay already accrued inside Trenova; nothing is " +
			"paid until a person approves it, and hands-off approval is only the settlement " +
			"control's clean-settlement rule.",
	}
}

func (t *generateDriverSettlementTool) request(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.GenerateForWorkerRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	period, err := readPeriod(params.Params)
	if err != nil {
		return nil, err
	}
	payDate, hasPayDate, err := optionalDay(params.Params, paramPayDate)
	if err != nil {
		return nil, err
	}

	tenant := tenantFrom(*params)
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

func (t *generateDriverSettlementTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
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

func (t *generateDriverSettlementTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return generationRefusal(plan)
}

func (t *generateDriverSettlementTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
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

	return settlementResult(created), nil
}

func settlementResult(created *driversettlement.Settlement) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: resultCreated,
		Kind:   nounDriverSettlement,
		Name:   created.SettlementNumber,
		IDs:    map[string]string{paramSettlementID: created.ID.String()},
		Record: &agent.RecordRef{EntityType: driverSettlementEntity, ID: created.ID.String()},
	}
}

func (t *generateDriverSettlementTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func newSettlementChange(
	settlement *driversettlement.Settlement,
	label string,
) (*agent.RecordChange, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceDriverSettlement, Label: label},
		settlement,
		toolpreview.Only(
			paramWorkerID, fieldStatus, fieldPeriodStart, fieldPeriodEnd, fieldPayDate,
			"payProfileName", "classification", "shipmentCount", fieldHasExceptions,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			paramWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(settlementDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		settlementMoney(&settlementFacts{}, driverSettlementFacts(settlement)),
		toolpreview.SensitiveAs(driverSettlementLedger().sensitive...),
	)

	return change, nil
}

func (t *generateDriverSettlementTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would generate a draft driver settlement for %s through %s, paid %s.",
		dayText(req.PeriodStart),
		dayText(req.PeriodEnd-secondsPerDay),
		dayText(req.PayDate),
	)
	if refusal := generationRefusal(plan); refusal != nil {
		return wouldFail(toolpreview.Build(summary), refusal)
	}

	change, err := newSettlementChange(plan.Settlement, "New driver settlement")
	if err != nil {
		return nil, err
	}
	if plan.Settlement.HasExceptions {
		summary += fmt.Sprintf(" It raises %s for review.",
			countOf(len(plan.Settlement.Exceptions), "exception"))
	}
	if plan.AutoApprove {
		summary += " The settlement control approves a clean settlement as soon as it is " +
			"generated."
	}

	return toolpreview.Build(summary, change), nil
}

func dayText(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format("Jan 2, 2006")
}

type batchText struct {
	name      string
	resource  permission.Resource
	rationale string
}

func batchSchema() map[string]any {
	properties := periodProperties("the batch")
	properties[paramBatchName] = stringProperty("A name for the batch; leave it out for "+
		"\"Pay period ending\" and the period's last day.", maxBatchNameChars)
	properties[paramBatchNotes] = stringProperty("Notes kept on the batch for payroll.",
		maxBatchNotesChars)

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
}

func batchPolicy(text batchText) serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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
	name, err := boundedString(params, paramBatchName, maxBatchNameChars, false)
	if err != nil {
		return batchInput{}, err
	}
	notes, err := boundedString(params, paramBatchNotes, maxBatchNotesChars, false)
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
		actor *serviceports.RequestActor,
	) (*driversettlement.SettlementBatch, error)
}

type generateDriverBatchTool struct {
	settlements driverBatchGenerator
}

var (
	_ serviceports.ToolPreviewer = (*generateDriverBatchTool)(nil)
	_ serviceports.ToolValidator = (*generateDriverBatchTool)(nil)
)

func provideGenerateDriverSettlementBatchTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return &generateDriverBatchTool{settlements: s}
}

func driverBatchText() batchText {
	return batchText{
		name:     "generate_driver_settlement_batch",
		resource: permission.ResourceDriverSettlement,
		rationale: "Drafts the period's settlements from pay already accrued inside Trenova; " +
			"nothing is paid until a person approves them, and hands-off approval is only " +
			"the settlement control's clean-settlement rule.",
	}
}

func (t *generateDriverBatchTool) Name() string { return driverBatchText().name }

func (t *generateDriverBatchTool) Description() string {
	return "Generate the pay period's driver settlement batch: a draft settlement for every " +
		"driver with accrued, unheld pay who has none for the period yet. An open batch for " +
		"the period is added to; a completed one is refused. Review the drafts with " +
		"list_driver_settlements afterwards."
}

func (t *generateDriverBatchTool) ParamSchema() map[string]any {
	return batchSchema()
}

func (t *generateDriverBatchTool) Policy() serviceports.ToolPolicy {
	return batchPolicy(driverBatchText())
}

func (t *generateDriverBatchTool) request(
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.GenerateBatchRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	input, err := readBatchInput(params.Params)
	if err != nil {
		return nil, err
	}

	return &driversettlementservice.GenerateBatchRequest{
		TenantInfo:  tenantFrom(*params),
		PeriodStart: input.period.start,
		PeriodEnd:   input.period.end,
		Name:        input.name,
		Notes:       input.notes,
	}, nil
}

func (t *generateDriverBatchTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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

func (t *generateDriverBatchTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
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

func (t *generateDriverBatchTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		dayText(plan.Bounds.PeriodStart),
		dayText(plan.Bounds.PeriodEnd-secondsPerDay),
		dayText(plan.Bounds.PayDate),
	)
	if plan.Refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.Refusal)
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

	summary += fmt.Sprintf(" %s would get a draft settlement.", countOf(len(changes), "driver"))
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
	worker *repositories.UnsettledWorkerSummary,
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
			paramWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(settlementDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(money.DefaultCurrencyCode,
		agent.MoneyLine{Label: "Accrued earnings", After: minorAmount(worker.GrossAmountMinor)},
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
		actor *serviceports.RequestActor,
	) (*carriersettlement.CarrierSettlementBatch, error)
}

type generateCarrierBatchTool struct {
	settlements carrierBatchGenerator
}

var (
	_ serviceports.ToolPreviewer = (*generateCarrierBatchTool)(nil)
	_ serviceports.ToolValidator = (*generateCarrierBatchTool)(nil)
)

func provideGenerateCarrierSettlementBatchTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return &generateCarrierBatchTool{settlements: s}
}

func carrierBatchText() batchText {
	return batchText{
		name:     "generate_carrier_settlement_batch",
		resource: permission.ResourceCarrierSettlement,
		rationale: "Drafts the period's carrier settlements from cost already accrued inside " +
			"Trenova; nothing is paid until a person approves and posts them.",
	}
}

func (t *generateCarrierBatchTool) Name() string { return carrierBatchText().name }

func (t *generateCarrierBatchTool) Description() string {
	return "Generate the pay period's carrier settlement batch, a draft settlement for each " +
		"carrier with pending cost. A carrier that already has one for the period is skipped. " +
		"Review the drafts with list_carrier_settlements afterwards."
}

func (t *generateCarrierBatchTool) ParamSchema() map[string]any {
	return batchSchema()
}

func (t *generateCarrierBatchTool) Policy() serviceports.ToolPolicy {
	return batchPolicy(carrierBatchText())
}

func (t *generateCarrierBatchTool) request(
	params *serviceports.ToolExecuteParams,
) (*carriersettlementservice.GenerateBatchRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	input, err := readBatchInput(params.Params)
	if err != nil {
		return nil, err
	}

	return &carriersettlementservice.GenerateBatchRequest{
		TenantInfo:  tenantFrom(*params),
		PeriodStart: input.period.start,
		PeriodEnd:   input.period.end,
		Name:        input.name,
		Notes:       input.notes,
	}, nil
}

func (t *generateCarrierBatchTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
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

func (t *generateCarrierBatchTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *generateCarrierBatchTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.settlements.GenerateBatch(ctx, req, params.Actor)

	return err
}

func (t *generateCarrierBatchTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would generate the carrier settlement batch for %s through %s, paid %s.",
		dayText(plan.Bounds.PeriodStart),
		dayText(plan.Bounds.PeriodEnd-secondsPerDay),
		dayText(plan.Bounds.PayDate),
	)
	if plan.Refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.Refusal)
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
				paramCarrierID, fieldStatus, fieldPeriodStart, fieldPeriodEnd, fieldPayDate,
				"shipmentCount",
			),
			toolpreview.WithRefs(map[string]permission.Resource{
				paramCarrierID: permission.ResourceCarrier,
			}),
			toolpreview.Types(settlementDateTypes),
		)
		if changeErr != nil {
			return nil, changeErr
		}
		toolpreview.AttachMoney(
			change,
			settlementMoney(&settlementFacts{}, carrierSettlementFacts(settlement)),
			toolpreview.SensitiveAs(carrierSettlementLedger().sensitive...),
		)
		changes = append(changes, change)
	}

	remaining := plan.CarrierCount - plan.SettledCount - len(plan.Settlements)
	summary += fmt.Sprintf(
		" Pending cost for the period covers %s; each without a settlement for it would get "+
			"a draft.",
		countOf(plan.CarrierCount, "carrier"),
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
