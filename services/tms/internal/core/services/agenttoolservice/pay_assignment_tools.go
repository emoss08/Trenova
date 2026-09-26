package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramPayProfileID           = "payProfileId"
	paramAssignmentID           = "assignmentId"
	paramEffectiveFrom          = "effectiveFrom"
	paramEffectiveTo            = "effectiveTo"
	paramSplitPercent           = "splitPercent"
	paramEndDate                = "endDate"
	maxAssignmentNotes          = 1000
	assignmentSourcesTool       = "list_pay_assignments"
	payProfileSourcesTool       = "list_pay_profiles"
	fieldEffectiveTo            = "effectiveTo"
	fieldEffectiveFrom          = "effectiveFrom"
	fieldSplitPercent           = "splitPercent"
	labelPayAssignment          = "Pay assignment"
	payAssignmentResource       = permission.ResourceDriverPayProfile
	fullSplitPercent      int64 = 100
)

type payAssigner interface {
	PlanAssignment(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) (*driverpayservice.AssignmentPlan, error)
	AssignProfileToWorker(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
		actor *serviceports.RequestActor,
	) (*driverpay.WorkerPayAssignment, error)
}

type assignPayProfileTool struct {
	assignments payAssigner
}

var (
	_ serviceports.ToolPreviewer = (*assignPayProfileTool)(nil)
	_ serviceports.ToolValidator = (*assignPayProfileTool)(nil)
)

func provideAssignPayProfileTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &assignPayProfileTool{assignments: s}
}

func (t *assignPayProfileTool) Name() string { return "assign_pay_profile" }

func (t *assignPayProfileTool) Description() string {
	return "Propose putting a driver on a pay profile from a date, which sets how every " +
		"later load is paid. An assignment already in force ends the day this one starts. " +
		"A person always decides; per-component rate overrides are set on the driver's " +
		"pay page."
}

func (t *assignPayProfileTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramWorkerID: workerProperty(),
		paramPayProfileID: idProperty("The pay profile, from " + payProfileSourcesTool +
			". Never guess one."),
		paramEffectiveFrom: dateProperty("The first day pay is computed under it."),
		paramEffectiveTo: dateProperty("The day it stops, when it is temporary; leave it " +
			"out for open-ended."),
		paramSplitPercent: amountProperty("The driver's share of each load, as a percent; " +
			"50 for an even team split. Leave it out for 100."),
		paramDriverPayNotes: stringProperty("Why the driver moves to this profile.",
			maxAssignmentNotes),
	}, paramWorkerID, paramPayProfileID, paramEffectiveFrom)
}

func (t *assignPayProfileTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  payAssignmentResource,
		operation: permission.OpAssign,
		rationale: "Sets how a driver is paid from a date on; only a person decides someone's " +
			"pay.",
	}).policy()
}

func (t *assignPayProfileTool) entity(
	params *serviceports.ToolExecuteParams,
) (*driverpay.WorkerPayAssignment, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	profileID, err := requirePulid(params.Params, paramPayProfileID)
	if err != nil {
		return nil, err
	}
	from, err := requireDay(params.Params, paramEffectiveFrom)
	if err != nil {
		return nil, err
	}
	to, hasTo, err := optionalDay(params.Params, paramEffectiveTo)
	if err != nil {
		return nil, err
	}
	split, hasSplit, err := optionalDecimal(params.Params, paramSplitPercent)
	if err != nil {
		return nil, err
	}
	if !hasSplit {
		split = decimal.NewFromInt(fullSplitPercent)
	}
	notes, err := boundedString(params.Params, paramDriverPayNotes, maxAssignmentNotes, false)
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

func (t *assignPayProfileTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*assignmentPlan, error) {
	entity, err := t.entity(params)
	if err != nil {
		return nil, err
	}
	plan, err := t.assignments.PlanAssignment(ctx, entity)
	if err != nil {
		refusal, failure := refusalOrFailure(err)
		if failure != nil {
			return nil, failure
		}

		return &assignmentPlan{entity: entity, refusal: refusal}, nil
	}

	return &assignmentPlan{entity: entity, plan: plan}, nil
}

func (t *assignPayProfileTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *assignPayProfileTool) Execute(
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
	_, err = t.assignments.AssignProfileToWorker(ctx, entity, params.Actor)

	return err
}

func assignmentRecord(assignment *driverpay.WorkerPayAssignment, label string) toolpreview.Record {
	return toolpreview.Record{
		Resource: payAssignmentResource,
		ID:       assignment.ID,
		Label:    label,
		Version:  pinnedVersion(assignment.Version),
	}
}

func (t *assignPayProfileTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would pay the driver under a new pay profile from %s.",
		dayText(plan.entity.EffectiveFrom),
	)
	if plan.refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.refusal)
	}

	profile := plan.plan.Profile
	summary = fmt.Sprintf(
		"Would pay the driver under %s (%s) from %s at a %s%% share.",
		profile.Name,
		profile.Classification,
		dayText(plan.entity.EffectiveFrom),
		plan.entity.SplitPercent.String(),
	)
	created, err := toolpreview.Create(
		toolpreview.Record{Resource: payAssignmentResource, Label: labelPayAssignment + ": " +
			profile.Name},
		plan.entity,
		toolpreview.Only(paramWorkerID, fieldEffectiveFrom, fieldEffectiveTo,
			fieldSplitPercent, paramDriverPayNotes),
		toolpreview.WithRefs(map[string]permission.Resource{
			paramWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(driverPayDateTypes),
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
			toolpreview.Only(fieldEffectiveTo),
			toolpreview.Types(driverPayDateTypes),
		)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}
	if len(plan.plan.Ended) > 0 {
		summary += fmt.Sprintf(" %s in force ends that day.",
			countOf(len(plan.plan.Ended), "assignment"))
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
		actor *serviceports.RequestActor,
	) (*driverpay.WorkerPayAssignment, error)
}

type endPayAssignmentTool struct {
	assignments assignmentEnder
}

var (
	_ serviceports.ToolPreviewer = (*endPayAssignmentTool)(nil)
	_ serviceports.ToolValidator = (*endPayAssignmentTool)(nil)
)

func provideEndPayAssignmentTool(s *driverpayservice.Service) serviceports.AgentTool {
	return &endPayAssignmentTool{assignments: s}
}

func (t *endPayAssignmentTool) Name() string { return "end_pay_assignment" }

func (t *endPayAssignmentTool) Description() string {
	return "Propose ending a driver's pay profile assignment on a date, such as when they " +
		"leave or change pay. Loads after that day are not paid under it; a driver with no " +
		"assignment in force raises an exception on their next settlement. A person always " +
		"decides."
}

func (t *endPayAssignmentTool) ParamSchema() map[string]any {
	return objectParams(map[string]any{
		paramAssignmentID: idProperty("The assignment, from " + assignmentSourcesTool +
			". Never guess one."),
		paramEndDate: dateProperty("The day it stops applying."),
	}, paramAssignmentID, paramEndDate)
}

func (t *endPayAssignmentTool) Policy() serviceports.ToolPolicy {
	return (&personOnlyPolicy{
		name:      t.Name(),
		resource:  payAssignmentResource,
		operation: permission.OpAssign,
		rationale: "Stops how a driver is paid from a date; only a person decides someone's " +
			"pay.",
	}).policy()
}

type assignmentEnding struct {
	before  *driverpay.WorkerPayAssignment
	after   *driverpay.WorkerPayAssignment
	endDate int64
	refusal error
}

func (t *endPayAssignmentTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*assignmentEnding, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	assignmentID, err := requirePulid(params.Params, paramAssignmentID)
	if err != nil {
		return nil, err
	}
	endDate, err := requireDay(params.Params, paramEndDate)
	if err != nil {
		return nil, err
	}
	before, err := t.assignments.GetAssignment(ctx, tenantFrom(*params), assignmentID)
	if err != nil {
		return nil, err
	}
	after := *before
	refusal := driverpayservice.PlanEndAssignment(&after, endDate)

	return &assignmentEnding{before: before, after: &after, endDate: endDate, refusal: refusal},
		nil
}

func (t *endPayAssignmentTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *endPayAssignmentTool) Execute(
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
	_, err = t.assignments.EndAssignment(
		ctx,
		tenantFrom(params),
		plan.before.ID,
		plan.endDate,
		params.Actor,
	)

	return err
}

func (t *endPayAssignmentTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	label := labelPayAssignment
	if plan.before.PayProfile != nil {
		label += ": " + plan.before.PayProfile.Name
	}
	summary := fmt.Sprintf("Would end %s on %s.", label, dayText(plan.endDate))
	if plan.refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.refusal)
	}
	change, err := toolpreview.Changed(
		assignmentRecord(plan.before, label),
		plan.before,
		plan.after,
		toolpreview.Only(fieldEffectiveTo),
		toolpreview.Types(driverPayDateTypes),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}
