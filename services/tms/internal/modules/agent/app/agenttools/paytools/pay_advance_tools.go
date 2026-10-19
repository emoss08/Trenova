package paytools

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
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const ParamAdvanceID = "advanceId"

const ParamAdvanceSource = "source"

const ParamAdvanceRef = "reference"

const ParamIssuedDate = "issuedDate"

const ParamWriteOffReason = "reason"

const maxWriteOffReason = 500

const fieldWriteOffReason = "writeOffReason"

const fieldWrittenOffAt = "writtenOffAt"

const advanceSourcesTool = "settlement_list_advances"

const advanceSensitiveMark = "amountMinor"

type payAdvanceIssuer interface {
	IssueAdvance(
		ctx context.Context,
		entity *driverpay.PayAdvance,
		actor *permissionports.RequestActor,
	) (*driverpay.PayAdvance, error)
}

type IssuePayAdvanceTool struct {
	Advances payAdvanceIssuer `json:"-"`
}

func ProvideIssuePayAdvanceTool(s *driverpayservice.Service) agentports.AgentTool {
	return &IssuePayAdvanceTool{Advances: s}
}

func (t *IssuePayAdvanceTool) Name() string { return "settlement_issue_advance" }

func (t *IssuePayAdvanceTool) Description() string {
	return "Propose recording a cash or money-code advance a driver was given against future " +
		"pay. It is recovered from their next settlements until repaid. A person always " +
		"decides, and only for an advance that was actually handed over; give the code or " +
		"receipt as the reference."
}

func (t *IssuePayAdvanceTool) ParamSchema() map[string]any {
	return common.ObjectParams(map[string]any{
		common.ParamWorkerID: common.WorkerProperty(),
		common.ParamDriverPayAmount: common.AmountProperty(
			"The amount advanced, as a decimal such as 300.00.",
		),
		ParamAdvanceSource: agenttoolschema.Enum(
			"How it was given.",
			common.AdvanceSources,
		),
		ParamAdvanceRef: common.StringProperty("The money code, card transaction or receipt "+
			"number.", common.MaxPayReferenceChars),
		ParamIssuedDate: agenttoolschema.Date("The day it was given; leave it out for today."),
		common.ParamNote: common.StringProperty("Why it was given, for payroll.",
			common.MaxDriverPayNotes),
	}, common.ParamWorkerID, common.ParamDriverPayAmount, ParamAdvanceSource)
}

func (t *IssuePayAdvanceTool) Policy() agentports.ToolPolicy {
	return (&common.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  permission.ResourcePayAdvance,
		Operation: permission.OpCreate,
		Rationale: "Records money handed to a driver that their pay then repays; only a " +
			"person records it.",
	}).Policy()
}

func (t *IssuePayAdvanceTool) entity(
	params *agentports.ToolExecuteParams,
) (*driverpay.PayAdvance, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	amount, err := common.RequireMoney(params.Params, common.ParamDriverPayAmount)
	if err != nil {
		return nil, err
	}
	source, ok, err := common.OptionalEnum(
		params.Params,
		ParamAdvanceSource,
		common.AdvanceSources.Values,
	)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("missing required parameter %q", ParamAdvanceSource)
	}
	reference, err := common.BoundedString(
		params.Params,
		ParamAdvanceRef,
		common.MaxPayReferenceChars,
		false,
	)
	if err != nil {
		return nil, err
	}
	notes, err := common.BoundedString(
		params.Params,
		common.ParamNote,
		common.MaxDriverPayNotes,
		false,
	)
	if err != nil {
		return nil, err
	}
	issued, hasIssued, err := common.OptionalDay(params.Params, ParamIssuedDate)
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

func (t *IssuePayAdvanceTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	entity, err := t.entity(&params)
	if err != nil {
		return err
	}

	return driverpayservice.PlanIssueAdvance(entity, params.Actor.UserID)
}

func (t *IssuePayAdvanceTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	if err := common.RequirePersonsApproval(&params); err != nil {
		return err
	}
	entity, err := t.entity(&params)
	if err != nil {
		return err
	}
	_, err = t.Advances.IssueAdvance(ctx, entity, params.Actor)

	return err
}

func (t *IssuePayAdvanceTool) Preview(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		return common.WouldFail(toolpreview.Build(summary), refusal)
	}

	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourcePayAdvance, Label: "New pay advance"},
		entity,
		toolpreview.Only(
			common.ParamWorkerID,
			common.FieldStatus,
			ParamAdvanceSource,
			ParamAdvanceRef,
			ParamIssuedDate,
			common.FieldNotes,
		),
		toolpreview.Volatile(ParamIssuedDate),
		toolpreview.WithRefs(map[string]permission.Resource{
			common.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(common.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(entity.CurrencyCode,
		agent.MoneyLine{Label: "Advance", After: common.MinorAmount(entity.AmountMinor)},
	), toolpreview.SensitiveAs(advanceSensitiveMark))

	return toolpreview.Build(summary, change), nil
}

type payAdvanceWriter interface {
	GetAdvance(
		ctx context.Context,
		req driverpayports.GetPayAdvanceByIDRequest,
	) (*driverpay.PayAdvance, error)
	WriteOffAdvance(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		advanceID pulid.ID,
		reason string,
		actor *permissionports.RequestActor,
	) (*driverpay.PayAdvance, error)
}

type WriteOffPayAdvanceTool struct {
	Advances payAdvanceWriter `json:"-"`
}

func ProvideWriteOffPayAdvanceTool(s *driverpayservice.Service) agentports.AgentTool {
	return &WriteOffPayAdvanceTool{Advances: s}
}

func (t *WriteOffPayAdvanceTool) Name() string { return "settlement_forgive_advance" }

func (t *WriteOffPayAdvanceTool) SearchTerms() []string { return []string{"write off"} }

func (t *WriteOffPayAdvanceTool) Description() string {
	return "Propose writing off what is still owed on a driver's pay advance, so no more " +
		"is recovered from their pay, with the reason. The organization absorbs the loss, " +
		"so a person always decides; use it when the advance cannot be recovered, such as " +
		"a driver who has left."
}

func (t *WriteOffPayAdvanceTool) ParamSchema() map[string]any {
	return common.ObjectParams(map[string]any{
		ParamAdvanceID: agenttoolschema.RecordIDText(permission.ResourcePayAdvance,
			"The advance, from "+advanceSourcesTool+". Never guess one."),
		ParamWriteOffReason: common.StringProperty(
			"Why it cannot be recovered.",
			maxWriteOffReason,
		),
	}, ParamAdvanceID, ParamWriteOffReason)
}

func (t *WriteOffPayAdvanceTool) Policy() agentports.ToolPolicy {
	return (&common.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  permission.ResourcePayAdvance,
		Operation: permission.OpCancel,
		Rationale: "Forgives money a driver owes, which the organization absorbs; only a " +
			"person writes it off.",
	}).Policy()
}

func (t *WriteOffPayAdvanceTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, ParamAdvanceID, permission.ResourcePayAdvance)
}

type writeOffPlan struct {
	before  *driverpay.PayAdvance
	after   *driverpay.PayAdvance
	reason  string
	refusal error
}

func (t *WriteOffPayAdvanceTool) plan(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*writeOffPlan, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	advanceID, err := common.RequirePulid(params.Params, ParamAdvanceID)
	if err != nil {
		return nil, err
	}
	reason, err := common.BoundedString(params.Params, ParamWriteOffReason, maxWriteOffReason, true)
	if err != nil {
		return nil, err
	}
	advance, err := t.Advances.GetAdvance(ctx, driverpayports.GetPayAdvanceByIDRequest{
		ID:         advanceID,
		TenantInfo: common.TenantFrom(*params),
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

func (t *WriteOffPayAdvanceTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *WriteOffPayAdvanceTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	if err := common.RequirePersonsApproval(&params); err != nil {
		return err
	}
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.Advances.WriteOffAdvance(
		ctx,
		common.TenantFrom(params),
		plan.before.ID,
		plan.reason,
		params.Actor,
	)

	return err
}

func (t *WriteOffPayAdvanceTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		return common.WouldFail(toolpreview.Build(summary), plan.refusal)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourcePayAdvance,
		ID:       plan.before.ID,
		Label:    advanceLabel(plan.before),
		Version:  common.PinnedVersion(plan.before.Version),
	}, plan.before, plan.after,
		toolpreview.Only(common.FieldStatus, fieldWriteOffReason, fieldWrittenOffAt),
		toolpreview.Volatile(fieldWrittenOffAt),
		toolpreview.Types(common.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(plan.before.CurrencyCode,
		agent.MoneyLine{
			Label:  "Still owed",
			Before: common.MinorAmount(outstanding),
			After:  common.MinorAmount(plan.after.OutstandingMinor()),
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
