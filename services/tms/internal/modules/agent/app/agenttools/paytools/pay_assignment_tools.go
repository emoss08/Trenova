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
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const ParamPayProfileID = "payProfileId"

const ParamAssignmentID = "assignmentId"

const paramSplitPercent = "splitPercent"

const ParamEndDate = "endDate"

const maxAssignmentNotes = 1000

const assignmentSourcesTool = "list_pay_assignments"

const payProfileSourcesTool = "list_pay_profiles"

const fieldSplitPercent = "splitPercent"

const labelPayAssignment = "Pay assignment"

const payAssignmentResource = permission.ResourceDriverPayProfile

const fullSplitPercent int64 = 100

type payAssigner interface {
	PlanAssignment(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) (*driverpayservice.AssignmentPlan, error)
	AssignProfileToWorker(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
		actor *permissionports.RequestActor,
	) (*driverpay.WorkerPayAssignment, error)
}

type AssignPayProfileTool struct {
	Assignments payAssigner `json:"-"`
}

func ProvideAssignPayProfileTool(s *driverpayservice.Service) agentports.AgentTool {
	return &AssignPayProfileTool{Assignments: s}
}

func (t *AssignPayProfileTool) Name() string { return "assign_pay_profile" }

func (t *AssignPayProfileTool) Description() string {
	return "Propose putting a driver on a pay profile from a date, which sets how every " +
		"later load is paid. An assignment already in force ends the day this one starts. " +
		"A person always decides; per-component rate overrides are set on the driver's " +
		"pay page."
}

func (t *AssignPayProfileTool) ParamSchema() map[string]any {
	return common.ObjectParams(map[string]any{
		common.ParamWorkerID: common.WorkerProperty(),
		ParamPayProfileID: agenttoolschema.KindID(
			"The pay profile, from "+payProfileSourcesTool+". Never guess one.",
			permission.KindPayProfile,
		),
		common.ParamEffectiveFrom: agenttoolschema.Date("The first day pay is computed under it."),
		common.ParamEffectiveTo: agenttoolschema.Date(
			"The day it stops, when it is temporary; leave it " +
				"out for open-ended.",
		),
		paramSplitPercent: common.AmountProperty("The driver's share of each load, as a percent; " +
			"50 for an even team split. Leave it out for 100."),
		common.ParamDriverPayNotes: common.StringProperty("Why the driver moves to this profile.",
			maxAssignmentNotes),
	}, common.ParamWorkerID, ParamPayProfileID, common.ParamEffectiveFrom)
}

func (t *AssignPayProfileTool) Policy() agentports.ToolPolicy {
	return (&common.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  payAssignmentResource,
		Operation: permission.OpAssign,
		Rationale: "Sets how a driver is paid from a date on; only a person decides someone's " +
			"pay.",
	}).Policy()
}

func (t *AssignPayProfileTool) entity(
	params *agentports.ToolExecuteParams,
) (*driverpay.WorkerPayAssignment, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	profileID, err := common.RequirePulid(params.Params, ParamPayProfileID)
	if err != nil {
		return nil, err
	}
	from, err := common.RequireDay(params.Params, common.ParamEffectiveFrom)
	if err != nil {
		return nil, err
	}
	to, hasTo, err := common.OptionalDay(params.Params, common.ParamEffectiveTo)
	if err != nil {
		return nil, err
	}
	split, hasSplit, err := common.OptionalDecimal(params.Params, paramSplitPercent)
	if err != nil {
		return nil, err
	}
	if !hasSplit {
		split = decimal.NewFromInt(fullSplitPercent)
	}
	notes, err := common.BoundedString(
		params.Params,
		common.ParamDriverPayNotes,
		maxAssignmentNotes,
		false,
	)
	if err != nil {
		return nil, err
	}

	entity := &driverpay.WorkerPayAssignment{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		WorkerID:       workerID,
		PayProfileID:   profileID,
		EffectiveFrom:  from,
		SplitPercent:   split,
		Notes:          notes,
	}
	if hasTo {
		entity.EffectiveTo = &to
	}

	return entity, nil
}

type assignmentPlan struct {
	entity  *driverpay.WorkerPayAssignment
	plan    *driverpayservice.AssignmentPlan
	refusal error
}

func (t *AssignPayProfileTool) planned(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*assignmentPlan, error) {
	entity, err := t.entity(params)
	if err != nil {
		return nil, err
	}
	plan, err := t.Assignments.PlanAssignment(ctx, entity)
	if err != nil {
		refusal, failure := common.RefusalOrFailure(err)
		if failure != nil {
			return nil, failure
		}

		return &assignmentPlan{entity: entity, refusal: refusal}, nil
	}

	return &assignmentPlan{entity: entity, plan: plan}, nil
}

func (t *AssignPayProfileTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *AssignPayProfileTool) Execute(
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
	_, err = t.Assignments.AssignProfileToWorker(ctx, entity, params.Actor)

	return err
}

func assignmentRecord(assignment *driverpay.WorkerPayAssignment, label string) toolpreview.Record {
	return toolpreview.Record{
		Resource: payAssignmentResource,
		ID:       assignment.ID,
		Label:    label,
		Version:  common.PinnedVersion(assignment.Version),
	}
}

func (t *AssignPayProfileTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would pay the driver under a new pay profile from %s.",
		common.DayText(plan.entity.EffectiveFrom),
	)
	if plan.refusal != nil {
		return common.WouldFail(toolpreview.Build(summary), plan.refusal)
	}

	profile := plan.plan.Profile
	summary = fmt.Sprintf(
		"Would pay the driver under %s (%s) from %s at a %s%% share.",
		profile.Name,
		profile.Classification,
		common.DayText(plan.entity.EffectiveFrom),
		plan.entity.SplitPercent.String(),
	)
	created, err := toolpreview.Create(
		toolpreview.Record{Resource: payAssignmentResource, Label: labelPayAssignment + ": " +
			profile.Name},
		plan.entity,
		toolpreview.Only(common.ParamWorkerID, common.FieldEffectiveFrom, common.FieldEffectiveTo,
			fieldSplitPercent, common.ParamDriverPayNotes),
		toolpreview.WithRefs(map[string]permission.Resource{
			common.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(common.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{created}
	for _, ended := range plan.plan.Ended {
		change, changeErr := toolpreview.Changed(
			assignmentRecord(ended, labelPayAssignment+" in force"),
			endedBefore(ended),
			ended,
			toolpreview.Only(common.FieldEffectiveTo),
			toolpreview.Types(common.DriverPayDateTypes),
		)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}
	if len(plan.plan.Ended) > 0 {
		summary += fmt.Sprintf(" %s in force ends that day.",
			common.CountOf(len(plan.plan.Ended), "assignment"))
	}

	return toolpreview.Build(summary, changes...), nil
}

func endedBefore(ended *driverpay.WorkerPayAssignment) *driverpay.WorkerPayAssignment {
	before := *ended
	before.EffectiveTo = nil

	return &before
}

type assignmentEnder interface {
	GetAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
	) (*driverpay.WorkerPayAssignment, error)
	EndAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
		endDate int64,
		actor *permissionports.RequestActor,
	) (*driverpay.WorkerPayAssignment, error)
}

type EndPayAssignmentTool struct {
	Assignments assignmentEnder `json:"-"`
}

func ProvideEndPayAssignmentTool(s *driverpayservice.Service) agentports.AgentTool {
	return &EndPayAssignmentTool{Assignments: s}
}

func (t *EndPayAssignmentTool) Name() string { return "end_pay_assignment" }

func (t *EndPayAssignmentTool) Description() string {
	return "Propose ending a driver's pay profile assignment on a date, such as when they " +
		"leave or change pay. Loads after that day are not paid under it; a driver with no " +
		"assignment in force raises an exception on their next settlement. A person always " +
		"decides."
}

func (t *EndPayAssignmentTool) ParamSchema() map[string]any {
	return common.ObjectParams(map[string]any{
		ParamAssignmentID: agenttoolschema.KindID(
			"The assignment, from "+assignmentSourcesTool+". Never guess one.",
			permission.KindPayAssignment,
		),
		ParamEndDate: agenttoolschema.Date("The day it stops applying."),
	}, ParamAssignmentID, ParamEndDate)
}

func (t *EndPayAssignmentTool) Policy() agentports.ToolPolicy {
	return (&common.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  payAssignmentResource,
		Operation: permission.OpAssign,
		Rationale: "Stops how a driver is paid from a date; only a person decides someone's " +
			"pay.",
	}).Policy()
}

type assignmentEnding struct {
	before  *driverpay.WorkerPayAssignment
	after   *driverpay.WorkerPayAssignment
	endDate int64
	refusal error
}

func (t *EndPayAssignmentTool) planned(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*assignmentEnding, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}
	assignmentID, err := common.RequirePulid(params.Params, ParamAssignmentID)
	if err != nil {
		return nil, err
	}
	endDate, err := common.RequireDay(params.Params, ParamEndDate)
	if err != nil {
		return nil, err
	}
	before, err := t.Assignments.GetAssignment(ctx, common.TenantFrom(*params), assignmentID)
	if err != nil {
		return nil, err
	}
	after := *before
	refusal := driverpayservice.PlanEndAssignment(&after, endDate)

	return &assignmentEnding{before: before, after: &after, endDate: endDate, refusal: refusal},
		nil
}

func (t *EndPayAssignmentTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *EndPayAssignmentTool) Execute(
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
	_, err = t.Assignments.EndAssignment(
		ctx,
		common.TenantFrom(params),
		plan.before.ID,
		plan.endDate,
		params.Actor,
	)

	return err
}

func (t *EndPayAssignmentTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	label := labelPayAssignment
	if plan.before.PayProfile != nil {
		label += ": " + plan.before.PayProfile.Name
	}
	summary := fmt.Sprintf("Would end %s on %s.", label, common.DayText(plan.endDate))
	if plan.refusal != nil {
		return common.WouldFail(toolpreview.Build(summary), plan.refusal)
	}
	change, err := toolpreview.Changed(
		assignmentRecord(plan.before, label),
		plan.before,
		plan.after,
		toolpreview.Only(common.FieldEffectiveTo),
		toolpreview.Types(common.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}
