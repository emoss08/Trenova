package paytools

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
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const ParamPayProfileID = "payProfileId"

const ParamAssignmentID = "assignmentId"

const paramSplitPercent = "splitPercent"

const ParamEndDate = "endDate"

const maxAssignmentNotes = 1000

const assignmentSourcesTool = "settlement_list_pay_assignments"

const payProfileSourcesTool = "settlement_list_pay_profiles"

const fieldSplitPercent = "splitPercent"

const labelPayAssignment = "Pay assignment"

const payAssignmentResource = permission.ResourceDriverPayProfile

const fullSplitPercent int64 = 100

type payAssigner interface {
	PlanAssignment(
		ctx context.Context,
		entity *payassignment.WorkerPayAssignment,
	) (*driverpayservice.AssignmentPlan, error)
	AssignProfileToWorker(
		ctx context.Context,
		entity *payassignment.WorkerPayAssignment,
		actor *permissionports.RequestActor,
	) (*payassignment.WorkerPayAssignment, error)
}

type AssignPayProfileTool struct {
	Assignments payAssigner `json:"-"`
}

func ProvideAssignPayProfileTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &AssignPayProfileTool{Assignments: s}
}

func (t *AssignPayProfileTool) Name() string { return "settlement_assign_pay_profile" }

func (t *AssignPayProfileTool) Description() string {
	return "Propose putting a driver on a pay profile from a date, which sets how every " +
		"later load is paid. An assignment already in force ends the day this one starts. " +
		"A person always decides; per-component rate overrides are set on the driver's " +
		"pay page."
}

func (t *AssignPayProfileTool) ParamSchema() map[string]any {
	return toolparams.ObjectParams(map[string]any{
		toolparams.ParamWorkerID: toolparams.WorkerProperty(),
		ParamPayProfileID: agenttoolschema.KindID(
			"The pay profile, from "+payProfileSourcesTool+". Never guess one.",
			recordkind.PayProfile,
		),
		toolparams.ParamEffectiveFrom: agenttoolschema.Date(
			"The first day pay is computed under it.",
		),
		toolparams.ParamEffectiveTo: agenttoolschema.Date(
			"The day it stops, when it is temporary; leave it " +
				"out for open-ended.",
		),
		paramSplitPercent: toolparams.AmountProperty(
			"The driver's share of each load, as a percent; " +
				"50 for an even team split. Leave it out for 100.",
		),
		toolparams.ParamNote: toolparams.StringProperty("Why the driver moves to this profile.",
			maxAssignmentNotes),
	}, toolparams.ParamWorkerID, ParamPayProfileID, toolparams.ParamEffectiveFrom)
}

func (t *AssignPayProfileTool) Policy() agenttoolports.ToolPolicy {
	return (&settlementpreview.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  payAssignmentResource,
		Operation: permission.OpAssign,
		Rationale: "Sets how a driver is paid from a date on; only a person decides someone's " +
			"pay.",
	}).Policy()
}

func (t *AssignPayProfileTool) entity(
	params *agenttoolports.ToolExecuteParams,
) (*payassignment.WorkerPayAssignment, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	profileID, err := toolparams.RequirePulid(params.Params, ParamPayProfileID)
	if err != nil {
		return nil, err
	}
	from, err := toolparams.RequireDay(params.Params, toolparams.ParamEffectiveFrom)
	if err != nil {
		return nil, err
	}
	to, hasTo, err := toolparams.OptionalDay(params.Params, toolparams.ParamEffectiveTo)
	if err != nil {
		return nil, err
	}
	split, hasSplit, err := toolparams.OptionalDecimal(params.Params, paramSplitPercent)
	if err != nil {
		return nil, err
	}
	if !hasSplit {
		split = decimal.NewFromInt(fullSplitPercent)
	}
	notes, err := toolparams.BoundedString(
		params.Params,
		toolparams.ParamNote,
		maxAssignmentNotes,
		false,
	)
	if err != nil {
		return nil, err
	}

	entity := &payassignment.WorkerPayAssignment{
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
	entity  *payassignment.WorkerPayAssignment
	plan    *driverpayservice.AssignmentPlan
	refusal error
}

func (t *AssignPayProfileTool) planned(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*assignmentPlan, error) {
	entity, err := t.entity(params)
	if err != nil {
		return nil, err
	}
	plan, err := t.Assignments.PlanAssignment(ctx, entity)
	if err != nil {
		refusal, failure := changepreview.RefusalOrFailure(err)
		if failure != nil {
			return nil, failure
		}

		return &assignmentPlan{entity: entity, refusal: refusal}, nil
	}

	return &assignmentPlan{entity: entity, plan: plan}, nil
}

func (t *AssignPayProfileTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *AssignPayProfileTool) Execute(
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
	_, err = t.Assignments.AssignProfileToWorker(ctx, entity, params.Actor)

	return err
}

func assignmentRecord(
	assignment *payassignment.WorkerPayAssignment,
	label string,
) toolpreview.Record {
	return toolpreview.Record{
		Resource: payAssignmentResource,
		ID:       assignment.ID,
		Label:    label,
		Version:  changepreview.PinnedVersion(assignment.Version),
	}
}

func (t *AssignPayProfileTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would pay the driver under a new pay profile from %s.",
		changepreview.DayText(plan.entity.EffectiveFrom),
	)
	if plan.refusal != nil {
		return changepreview.WouldFail(toolpreview.Build(summary), plan.refusal)
	}

	profile := plan.plan.Profile
	summary = fmt.Sprintf(
		"Would pay the driver under %s (%s) from %s at a %s%% share.",
		profile.Name,
		profile.Classification,
		changepreview.DayText(plan.entity.EffectiveFrom),
		plan.entity.SplitPercent.String(),
	)
	created, err := toolpreview.Create(
		toolpreview.Record{Resource: payAssignmentResource, Label: labelPayAssignment + ": " +
			profile.Name},
		plan.entity,
		toolpreview.Only(
			toolparams.ParamWorkerID,
			changepreview.FieldEffectiveFrom,
			changepreview.FieldEffectiveTo,
			fieldSplitPercent,
			changepreview.FieldNotes,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			toolparams.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(settlementpreview.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}
	changes := []*agentpreview.RecordChange{created}
	for _, ended := range plan.plan.Ended {
		change, changeErr := toolpreview.Changed(
			assignmentRecord(ended, labelPayAssignment+" in force"),
			endedBefore(ended),
			ended,
			toolpreview.Only(changepreview.FieldEffectiveTo),
			toolpreview.Types(settlementpreview.DriverPayDateTypes),
		)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}
	if len(plan.plan.Ended) > 0 {
		summary += fmt.Sprintf(" %s in force ends that day.",
			changepreview.CountOf(len(plan.plan.Ended), "assignment"))
	}

	return toolpreview.Build(summary, changes...), nil
}

func endedBefore(ended *payassignment.WorkerPayAssignment) *payassignment.WorkerPayAssignment {
	before := *ended
	before.EffectiveTo = nil

	return &before
}

type assignmentEnder interface {
	GetAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
	) (*payassignment.WorkerPayAssignment, error)
	EndAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
		endDate int64,
		actor *permissionports.RequestActor,
	) (*payassignment.WorkerPayAssignment, error)
}

type EndPayAssignmentTool struct {
	Assignments assignmentEnder `json:"-"`
}

func ProvideEndPayAssignmentTool(s *driverpayservice.Service) agenttoolports.AgentTool {
	return &EndPayAssignmentTool{Assignments: s}
}

func (t *EndPayAssignmentTool) Name() string { return "settlement_end_pay_assignment" }

func (t *EndPayAssignmentTool) Description() string {
	return "Propose ending a driver's pay profile assignment on a date, such as when they " +
		"leave or change pay. Loads after that day are not paid under it; a driver with no " +
		"assignment in force raises an exception on their next settlement. A person always " +
		"decides."
}

func (t *EndPayAssignmentTool) ParamSchema() map[string]any {
	return toolparams.ObjectParams(map[string]any{
		ParamAssignmentID: agenttoolschema.KindID(
			"The assignment, from "+assignmentSourcesTool+". Never guess one.",
			recordkind.PayAssignment,
		),
		ParamEndDate: agenttoolschema.Date("The day it stops applying."),
	}, ParamAssignmentID, ParamEndDate)
}

func (t *EndPayAssignmentTool) Policy() agenttoolports.ToolPolicy {
	return (&settlementpreview.PersonOnlyPolicy{
		Name:      t.Name(),
		Resource:  payAssignmentResource,
		Operation: permission.OpAssign,
		Rationale: "Stops how a driver is paid from a date; only a person decides someone's " +
			"pay.",
	}).Policy()
}

type assignmentEnding struct {
	before  *payassignment.WorkerPayAssignment
	after   *payassignment.WorkerPayAssignment
	endDate int64
	refusal error
}

func (t *EndPayAssignmentTool) planned(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*assignmentEnding, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	assignmentID, err := toolparams.RequirePulid(params.Params, ParamAssignmentID)
	if err != nil {
		return nil, err
	}
	endDate, err := toolparams.RequireDay(params.Params, ParamEndDate)
	if err != nil {
		return nil, err
	}
	before, err := t.Assignments.GetAssignment(ctx, toolparams.TenantFrom(*params), assignmentID)
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
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *EndPayAssignmentTool) Execute(
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
	_, err = t.Assignments.EndAssignment(
		ctx,
		toolparams.TenantFrom(params),
		plan.before.ID,
		plan.endDate,
		params.Actor,
	)

	return err
}

func (t *EndPayAssignmentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	label := labelPayAssignment
	if plan.before.PayProfile != nil {
		label += ": " + plan.before.PayProfile.Name
	}
	summary := fmt.Sprintf("Would end %s on %s.", label, changepreview.DayText(plan.endDate))
	if plan.refusal != nil {
		return changepreview.WouldFail(toolpreview.Build(summary), plan.refusal)
	}
	change, err := toolpreview.Changed(
		assignmentRecord(plan.before, label),
		plan.before,
		plan.after,
		toolpreview.Only(changepreview.FieldEffectiveTo),
		toolpreview.Types(settlementpreview.DriverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}
