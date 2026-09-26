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
)

const (
	paramAdvanceID       = "advanceId"
	paramAdvanceSource   = "source"
	paramAdvanceRef      = "reference"
	paramIssuedDate      = "issuedDate"
	paramWriteOffReason  = "reason"
	maxWriteOffReason    = 500
	fieldWriteOffReason  = "writeOffReason"
	fieldWrittenOffAt    = "writtenOffAt"
	advanceSourcesTool   = "list_pay_advances"
	advanceSensitiveMark = "amountMinor"
)

type payAdvanceIssuer interface {
	IssueAdvance(
		ctx context.Context,
		entity *driverpay.PayAdvance,
		actor *serviceports.RequestActor,
	) (*driverpay.PayAdvance, error)
}

type issuePayAdvanceTool struct {
	advances payAdvanceIssuer
}

var (
	_ serviceports.ToolPreviewer = (*issuePayAdvanceTool)(nil)
	_ serviceports.ToolValidator = (*issuePayAdvanceTool)(nil)
)

func provideIssuePayAdvanceTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &issuePayAdvanceTool{advances: s}
}

func (t *issuePayAdvanceTool) Name() string { return "issue_pay_advance" }

func (t *issuePayAdvanceTool) Description() string {
	return "Propose recording a cash or money-code advance a driver was given against future " +
		"pay. It is recovered from their next settlements until repaid. A person always " +
		"decides, and only for an advance that was actually handed over; give the code or " +
		"receipt as the reference."
}

func (t *issuePayAdvanceTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramWorkerID:        workerProperty(),
		paramDriverPayAmount: amountProperty("The amount advanced, as a decimal such as 300.00."),
		paramAdvanceSource:   enumProperty("How it was given.", advanceSources),
		paramAdvanceRef: stringProperty("The money code, card transaction or receipt "+
			"number.", maxPayReferenceChars),
		paramIssuedDate: dateProperty("The day it was given; leave it out for today."),
		paramDriverPayNotes: stringProperty("Why it was given, for payroll.",
			maxDriverPayNotes),
	}, paramWorkerID, paramDriverPayAmount, paramAdvanceSource)
}

func (t *issuePayAdvanceTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  permission.ResourcePayAdvance,
		operation: permission.OpCreate,
		rationale: "Records money handed to a driver that their pay then repays; only a " +
			"person records it.",
	}).policy()
}

func (t *issuePayAdvanceTool) entity(
	params *serviceports.ToolExecuteParams,
) (*driverpay.PayAdvance, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	amount, err := requireMoney(params.Params, paramDriverPayAmount)
	if err != nil {
		return nil, err
	}
	source, ok, err := optionalEnum(params.Params, paramAdvanceSource, advanceSources)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("missing required parameter %q", paramAdvanceSource)
	}
	reference, err := boundedString(params.Params, paramAdvanceRef, maxPayReferenceChars, false)
	if err != nil {
		return nil, err
	}
	notes, err := boundedString(params.Params, paramDriverPayNotes, maxDriverPayNotes, false)
	if err != nil {
		return nil, err
	}
	issued, hasIssued, err := optionalDay(params.Params, paramIssuedDate)
	if err != nil {
		return nil, err
	}
	if !hasIssued {
		issued = timeutils.NowUnix()
	}

	return &driverpay.PayAdvance{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		WorkerID:       workerID,
		Source:         source,
		Reference:      reference,
		IssuedDate:     issued,
		AmountMinor:    amount,
		Notes:          notes,
		CurrencyCode:   money.DefaultCurrencyCode,
	}, nil
}

func (t *issuePayAdvanceTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	entity, err := t.entity(&params)
	if err != nil {
		return err
	}

	return driverpayservice.PlanIssueAdvance(entity, params.Actor.UserID)
}

func (t *issuePayAdvanceTool) Execute(
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
	_, err = t.advances.IssueAdvance(ctx, entity, params.Actor)

	return err
}

func (t *issuePayAdvanceTool) Preview(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	entity, err := t.entity(&params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would record an advance of %s given as %s, recovered from the driver's next "+
			"settlements.",
		money.FormatMinor(entity.AmountMinor, entity.CurrencyCode),
		entity.Source,
	)
	if refusal := driverpayservice.PlanIssueAdvance(entity, params.Actor.UserID); refusal != nil {
		return wouldFail(toolpreview.Build(summary), refusal)
	}

	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourcePayAdvance, Label: "New pay advance"},
		entity,
		toolpreview.Only(paramWorkerID, fieldStatus, paramAdvanceSource, paramAdvanceRef,
			paramIssuedDate, paramDriverPayNotes),
		toolpreview.Volatile(paramIssuedDate),
		toolpreview.WithRefs(map[string]permission.Resource{
			paramWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(driverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(entity.CurrencyCode,
		agent.MoneyLine{Label: "Advance", After: minorAmount(entity.AmountMinor)},
	), toolpreview.SensitiveAs(advanceSensitiveMark))

	return toolpreview.Build(summary, change), nil
}

type payAdvanceWriter interface {
	GetAdvance(
		ctx context.Context,
		req repositories.GetPayAdvanceByIDRequest,
	) (*driverpay.PayAdvance, error)
	WriteOffAdvance(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		advanceID pulid.ID,
		reason string,
		actor *serviceports.RequestActor,
	) (*driverpay.PayAdvance, error)
}

type writeOffPayAdvanceTool struct {
	advances payAdvanceWriter
}

var (
	_ serviceports.ToolPreviewer = (*writeOffPayAdvanceTool)(nil)
	_ serviceports.ToolValidator = (*writeOffPayAdvanceTool)(nil)
	_ serviceports.TargetedTool  = (*writeOffPayAdvanceTool)(nil)
)

func provideWriteOffPayAdvanceTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &writeOffPayAdvanceTool{advances: s}
}

func (t *writeOffPayAdvanceTool) Name() string { return "write_off_pay_advance" }

func (t *writeOffPayAdvanceTool) Description() string {
	return "Propose writing off what is still owed on a driver's pay advance, so no more " +
		"is recovered from their pay, with the reason. The organization absorbs the loss, " +
		"so a person always decides; use it when the advance cannot be recovered, such as " +
		"a driver who has left."
}

func (t *writeOffPayAdvanceTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramAdvanceID: idProperty("The advance, from " + advanceSourcesTool +
			". Never guess one."),
		paramWriteOffReason: stringProperty("Why it cannot be recovered.", maxWriteOffReason),
	}, paramAdvanceID, paramWriteOffReason)
}

func (t *writeOffPayAdvanceTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  permission.ResourcePayAdvance,
		operation: permission.OpCancel,
		rationale: "Forgives money a driver owes, which the organization absorbs; only a " +
			"person writes it off.",
	}).policy()
}

func (t *writeOffPayAdvanceTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramAdvanceID, permission.ResourcePayAdvance)
}

type writeOffPlan struct {
	before  *driverpay.PayAdvance
	after   *driverpay.PayAdvance
	reason  string
	refusal error
}

func (t *writeOffPayAdvanceTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*writeOffPlan, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	advanceID, err := requirePulid(params.Params, paramAdvanceID)
	if err != nil {
		return nil, err
	}
	reason, err := boundedString(params.Params, paramWriteOffReason, maxWriteOffReason, true)
	if err != nil {
		return nil, err
	}
	advance, err := t.advances.GetAdvance(ctx, repositories.GetPayAdvanceByIDRequest{
		ID:         advanceID,
		TenantInfo: tenantFrom(*params),
	})
	if err != nil {
		return nil, err
	}
	after := *advance
	refusal := driverpayservice.PlanWriteOffAdvance(
		&after,
		reason,
		params.Actor.UserID,
		timeutils.NowUnix(),
	)

	return &writeOffPlan{before: advance, after: &after, reason: reason, refusal: refusal}, nil
}

func (t *writeOffPayAdvanceTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *writeOffPayAdvanceTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if err := requirePersonsApproval(&params); err != nil {
		return err
	}
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.advances.WriteOffAdvance(
		ctx,
		tenantFrom(params),
		plan.before.ID,
		plan.reason,
		params.Actor,
	)

	return err
}

func (t *writeOffPayAdvanceTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	outstanding := plan.before.OutstandingMinor()
	summary := fmt.Sprintf(
		"Would write off the %s still owed on a %s advance; nothing more is recovered from "+
			"the driver's pay.",
		money.FormatMinor(outstanding, plan.before.CurrencyCode),
		money.FormatMinor(plan.before.AmountMinor, plan.before.CurrencyCode),
	)
	if plan.refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.refusal)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourcePayAdvance,
		ID:       plan.before.ID,
		Label:    advanceLabel(plan.before),
		Version:  pinnedVersion(plan.before.Version),
	}, plan.before, plan.after,
		toolpreview.Only(fieldStatus, fieldWriteOffReason, fieldWrittenOffAt),
		toolpreview.Volatile(fieldWrittenOffAt),
		toolpreview.Types(driverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(plan.before.CurrencyCode,
		agent.MoneyLine{
			Label:  "Still owed",
			Before: minorAmount(outstanding),
			After:  minorAmount(plan.after.OutstandingMinor()),
		},
	), toolpreview.SensitiveAs("writtenOffMinor"))

	return toolpreview.Build(summary, change), nil
}

func advanceLabel(advance *driverpay.PayAdvance) string {
	if advance.Reference != "" {
		return "Advance " + advance.Reference
	}

	return "Advance of " + money.FormatMinor(advance.AmountMinor, advance.CurrencyCode)
}
