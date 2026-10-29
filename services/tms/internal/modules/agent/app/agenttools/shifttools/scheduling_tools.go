package shifttools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/teamscope"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramShiftTemplateID = "shiftTemplateId"

const paramCycleOffsetWeeks = "cycleOffsetWeeks"

const paramShiftAssignmentID = "shiftAssignmentId"

const paramDayOfWeek = "dayOfWeek"

const paramPreference = "preference"

const paramSwapID = "swapId"

const paramRequestingWorkerID = "requestingWorkerId"

const paramCounterpartyWorkerID = "counterpartyWorkerId"

const paramShiftDate = "shiftDate"

const paramCounterpartyDate = "counterpartyShiftDate"

const kindShiftAssignment = "shift assignment"

const kindShiftSwap = "shift swap"

const maxScheduleNote = 255

const maxCycleOffsetWeeks = 7

var availabilityPreferences = agenttoolschema.Source(
	"worker.availabilityPreference",
	workerschedule.AvailabilityPreferenceValues(),
)

var weekdays = agenttoolschema.Source("time.weekday", weekdayNames())

var scheduleUserRefs = map[string]permission.Resource{
	toolparams.ParamWorkerID:  permission.ResourceWorker,
	paramShiftTemplateID:      permission.ResourceShiftTemplate,
	"assignedById":            permission.ResourceUser,
	paramRequestingWorkerID:   permission.ResourceWorker,
	paramCounterpartyWorkerID: permission.ResourceWorker,
	"decidedById":             permission.ResourceUser,
}

var scheduleDays = map[string]assistantartifact.DisplayType{
	changepreview.FieldEffectiveFrom: assistantartifact.DisplayDate,
	changepreview.FieldEffectiveTo:   assistantartifact.DisplayDate,
	paramShiftDate:                   assistantartifact.DisplayDate,
	paramCounterpartyDate:            assistantartifact.DisplayDate,
	"decidedAt":                      assistantartifact.DisplayDateTime,
	"respondedAt":                    assistantartifact.DisplayDateTime,
}

type scheduleKeeper interface {
	GetAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerschedule.WorkerShiftAssignment, error)
	AssignShift(
		ctx context.Context,
		req *schedulingservice.AssignShiftRequest,
	) (*workerschedule.WorkerShiftAssignment, error)
	PreviewAssignShift(
		ctx context.Context,
		req *schedulingservice.AssignShiftRequest,
	) (*schedulingservice.ShiftAssignmentPlan, error)
	EndAssignment(
		ctx context.Context,
		req *schedulingservice.EndAssignmentRequest,
	) (*workerschedule.WorkerShiftAssignment, error)
	PreviewEndAssignment(
		ctx context.Context,
		req *schedulingservice.EndAssignmentRequest,
	) (*schedulingservice.AssignmentChange, error)
	SetPreference(
		ctx context.Context,
		req *schedulingservice.SetPreferenceRequest,
	) (*workerschedule.WorkerAvailabilityPreference, error)
	PreviewSetPreference(
		ctx context.Context,
		req *schedulingservice.SetPreferenceRequest,
	) (*schedulingservice.PreferenceChange, error)
	ProposeSwap(
		ctx context.Context,
		req *schedulingservice.ProposeSwapRequest,
	) (*workerschedule.ShiftSwapRequest, error)
	PreviewProposeSwap(
		req *schedulingservice.ProposeSwapRequest,
	) (*workerschedule.ShiftSwapRequest, error)
	TransitionSwap(
		ctx context.Context,
		req *schedulingservice.TransitionSwapRequest,
	) (*workerschedule.ShiftSwapRequest, error)
	PreviewTransitionSwap(
		ctx context.Context,
		req *schedulingservice.TransitionSwapRequest,
	) (*schedulingservice.SwapChange, error)
}

// teamScope is the check the scheduling resolvers make before a manager changes
// a worker's schedule. The worker is always the one the record names, never one
// taken from alongside it.
type teamScope struct {
	guard teamscope.Guard
}

func NewTeamScope(permissions teamscope.Permissions, teams teamscope.Teams) teamScope {
	return teamScope{guard: teamscope.Guard{Permissions: permissions, Teams: teams}}
}

func (s teamScope) require(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	operation permission.Operation,
	workerID pulid.ID,
) error {
	return s.guard.Require(ctx, &teamscope.Request{
		Actor:      params.Actor,
		TenantInfo: toolparams.TenantFrom(*params),
		Resource:   permission.ResourceWorkerSchedule,
		Operation:  operation,
		WorkerID:   workerID,
	})
}

func weekdayNames() []string {
	names := make([]string, 0, 7)
	for day := time.Sunday; day <= time.Saturday; day++ {
		names = append(names, day.String())
	}

	return names
}

func requireWeekday(params map[string]any, key string) (int16, error) {
	name, err := toolparams.RequireEnum(params, key, weekdays.Values)
	if err != nil {
		return 0, err
	}
	for day := time.Sunday; day <= time.Saturday; day++ {
		if day.String() == name {
			return int16(day), nil
		}
	}

	return 0, fmt.Errorf("parameter %q must be a weekday", key)
}

func shiftAssignmentRecord(assignment *workerschedule.WorkerShiftAssignment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceWorkerSchedule,
		ID:       assignment.ID,
		Label:    "Shift assignment",
		Version:  changepreview.PinnedVersion(assignment.Version),
	}
}

func swapRecord(swap *workerschedule.ShiftSwapRequest) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceShiftSwap,
		ID:       swap.ID,
		Label: "Shift swap for " + time.Unix(swap.ShiftDate, 0).
			UTC().
			Format(toolparams.DayLayout),
		Version: changepreview.PinnedVersion(swap.Version),
	}
}

func scheduleOptions(fields ...string) []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(fields...),
		toolpreview.WithRefs(scheduleUserRefs),
		toolpreview.Types(scheduleDays),
	}
}

func scheduleSpec(
	name, description, rationale string,
	operation permission.Operation,
	properties map[string]any,
	required []string,
	target func(map[string]any) (agenttoolports.ToolTarget, bool),
) *toolspec.ReceivableSpec {
	return &toolspec.ReceivableSpec{
		Name:        name,
		Description: description,
		Artifact:    changepreview.WorkerRecordEntity,
		Resource:    permission.ResourceWorkerSchedule,
		Operation:   operation,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale:   rationale,
		Properties:  properties,
		Required:    required,
		Target:      target,
	}
}

func targetScheduledWorker(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamWorkerID, permission.ResourceWorker)
}

func NewAssignWorkerShiftTool(schedules scheduleKeeper, scope teamScope) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(scheduleSpec(
		"worker_assign_shift",
		"Put a worker on a shift pattern from a day onward. The pattern they are on "+
			"now ends the day before; a worker who already has a pattern starting on or "+
			"after that day is refused. Take the shift from worker_list_shift_templates and the "+
			"day from what the person asked for.",
		"Changes a worker's standing schedule inside Trenova; nothing is sent, and "+
			"worker_end_shift_assignment or a new assignment changes it back.",
		permission.OpAssign,
		map[string]any{
			toolparams.ParamWorkerID: toolparams.WorkerProperty(),
			paramShiftTemplateID: agenttoolschema.RecordIDText(
				permission.ResourceShiftTemplate,
				"The shift pattern, from worker_list_shift_templates. "+
					"Never guess one.",
			),
			changepreview.FieldEffectiveFrom: agenttoolschema.Date(
				"The first day the worker works it.",
			),
			paramCycleOffsetWeeks: toolparams.IntegerProperty(
				"Which week of a rotating pattern the "+
					"worker starts on, 0 for the first.",
				0,
				maxCycleOffsetWeeks,
			),
			toolparams.ParamNote: toolparams.StringProperty("Anything the planner should know.",
				toolparams.MaxOperationNoteChars),
		},
		[]string{toolparams.ParamWorkerID, paramShiftTemplateID, changepreview.FieldEffectiveFrom},
		targetScheduledWorker,
	), toolspec.ReceivablePlan[*schedulingservice.AssignShiftRequest, *schedulingservice.ShiftAssignmentPlan]{
		Request: assignShiftRequest,
		Plan: func(
			ctx context.Context,
			req *schedulingservice.AssignShiftRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*schedulingservice.ShiftAssignmentPlan, error) {
			if err := scope.require(
				ctx,
				params,
				permission.OpAssign,
				req.Entity.WorkerID,
			); err != nil {
				return nil, err
			}
			entity := *req.Entity

			return schedules.PreviewAssignShift(ctx, &schedulingservice.AssignShiftRequest{
				Entity:     &entity,
				TenantInfo: req.TenantInfo,
				UserID:     req.UserID,
			})
		},
		Refused: func(*schedulingservice.AssignShiftRequest) string {
			return "Would put the worker on a shift pattern."
		},
		Render: renderAssignShift,
		Run: func(
			ctx context.Context,
			req *schedulingservice.AssignShiftRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if err := scope.require(
				ctx,
				params,
				permission.OpAssign,
				req.Entity.WorkerID,
			); err != nil {
				return nil, err
			}
			created, err := schedules.AssignShift(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "assigned",
				Kind:   kindShiftAssignment,
				IDs: map[string]string{
					paramShiftAssignmentID:   created.ID.String(),
					toolparams.ParamWorkerID: created.WorkerID.String(),
				},
				Record: changepreview.RecordOf(changepreview.WorkerRecordEntity, created.WorkerID),
			}, nil
		},
	})
}

func renderAssignShift(
	_ *schedulingservice.AssignShiftRequest,
	plan *schedulingservice.ShiftAssignmentPlan,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(plan.Ended)+1)
	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceWorkerSchedule,
			Label:    "New shift assignment",
		},
		plan.Created,
		scheduleOptions(
			toolparams.ParamWorkerID,
			paramShiftTemplateID,
			changepreview.FieldEffectiveFrom,
			paramCycleOffsetWeeks,
			changepreview.FieldNotes,
		)...,
	)
	if err != nil {
		return nil, err
	}
	changes = append(changes, created)
	for _, open := range plan.Ended {
		ended := *open
		endAt := plan.EndedAt
		ended.EffectiveTo = &endAt
		change, changeErr := toolpreview.Changed(shiftAssignmentRecord(open), open, &ended,
			scheduleOptions(changepreview.FieldEffectiveTo)...)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf("Would put the worker on %s from %s.", plan.Template.Name,
		time.Unix(plan.Created.EffectiveFrom, 0).UTC().Format(toolparams.DayLayout))
	if len(plan.Ended) > 0 {
		summary += " The pattern they are on now ends the day before."
	}

	return toolpreview.Build(summary, changes...), nil
}

type assignmentEnd struct {
	req *schedulingservice.EndAssignmentRequest
}

func NewEndWorkerShiftAssignmentTool(
	schedules scheduleKeeper,
	scope teamScope,
) agenttoolports.AgentTool {
	spec := scheduleSpec(
		"worker_end_shift_assignment",
		"End a worker's shift pattern after a day, leaving them on no pattern from the "+
			"day after. Use worker_assign_shift instead to move them onto another pattern.",
		"Ends a worker's standing schedule inside Trenova; nothing is sent, and "+
			"worker_assign_shift puts them back on it.",
		permission.OpAssign,
		map[string]any{
			paramShiftAssignmentID: agenttoolschema.KindID(
				"The assignment, from worker_get_schedule. "+
					"Never guess one.",
				permission.KindShiftAssignment,
			),
			changepreview.FieldEffectiveTo: agenttoolschema.Date(
				"The last day the worker works the pattern.",
			),
		},
		[]string{paramShiftAssignmentID, changepreview.FieldEffectiveTo},
		func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(
				params,
				paramShiftAssignmentID,
				permission.ResourceWorkerSchedule,
			)
		},
	)
	spec.SearchTerms = []string{"shift pattern", "end shift", "remove from shift pattern"}

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*assignmentEnd, *schedulingservice.AssignmentChange,
	]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*assignmentEnd, error) {
			id, err := toolparams.RequirePulid(params.Params, paramShiftAssignmentID)
			if err != nil {
				return nil, err
			}
			to, err := toolparams.RequireScheduleDay(params.Params, changepreview.FieldEffectiveTo)
			if err != nil {
				return nil, err
			}

			return &assignmentEnd{req: &schedulingservice.EndAssignmentRequest{
				ID:          id,
				EffectiveTo: to,
				TenantInfo:  toolparams.TenantFrom(*params),
				UserID:      params.Actor.UserID,
			}}, nil
		},
		Plan: func(
			ctx context.Context,
			end *assignmentEnd,
			params *agenttoolports.ToolExecuteParams,
		) (*schedulingservice.AssignmentChange, error) {
			if err := requireAssignmentScope(ctx, schedules, scope, end, params); err != nil {
				return nil, err
			}

			return schedules.PreviewEndAssignment(ctx, end.req)
		},
		Refused: func(*assignmentEnd) string {
			return "Would end the worker's shift pattern."
		},
		Render: renderShiftAssignmentEnd,
		Run: func(
			ctx context.Context,
			end *assignmentEnd,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if err := requireAssignmentScope(ctx, schedules, scope, end, params); err != nil {
				return nil, err
			}
			ended, err := schedules.EndAssignment(ctx, end.req)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "ended",
				Kind:   kindShiftAssignment,
				IDs:    map[string]string{paramShiftAssignmentID: ended.ID.String()},
				Record: changepreview.RecordOf(changepreview.WorkerRecordEntity, ended.WorkerID),
			}, nil
		},
	})
}

func renderShiftAssignmentEnd(
	_ *assignmentEnd,
	plan *schedulingservice.AssignmentChange,
) (*agentpreview.ToolPreview, error) {
	change, err := toolpreview.Changed(shiftAssignmentRecord(plan.Before), plan.Before,
		plan.After, scheduleOptions(changepreview.FieldEffectiveTo)...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would end the worker's shift pattern after %s.",
		time.Unix(*plan.After.EffectiveTo, 0).UTC().Format(toolparams.DayLayout),
	), change), nil
}

// requireAssignmentScope reads the worker off the assignment being ended, the
// way the resolver does, so the team check is about whose schedule changes.
func requireAssignmentScope(
	ctx context.Context,
	schedules scheduleKeeper,
	scope teamScope,
	end *assignmentEnd,
	params *agenttoolports.ToolExecuteParams,
) error {
	assignment, err := schedules.GetAssignment(ctx, end.req.TenantInfo, end.req.ID)
	if err != nil {
		return err
	}

	return scope.require(ctx, params, permission.OpAssign, assignment.WorkerID)
}

func NewSetWorkerAvailabilityPreferenceTool(
	schedules scheduleKeeper,
	scope teamScope,
) agenttoolports.AgentTool {
	spec := scheduleSpec(
		"worker_set_availability_preference",
		"Record whether a worker prefers, is available for or cannot work one weekday, "+
			"as they told the office. The rota shows it beside their pattern; it does not "+
			"change the pattern.",
		"Records a worker's own stated preference inside Trenova; nothing is sent, and "+
			"it is set again the same way.",
		permission.OpUpdate,
		map[string]any{
			toolparams.ParamWorkerID: toolparams.WorkerProperty(),
			paramDayOfWeek:           agenttoolschema.Enum("The weekday it is about.", weekdays),
			paramPreference: agenttoolschema.Enum("What the worker said about that day.",
				availabilityPreferences),
			toolparams.ParamNote: toolparams.StringProperty(
				"Why, in the worker's words where you have them.",
				maxScheduleNote,
			),
		},
		[]string{toolparams.ParamWorkerID, paramDayOfWeek, paramPreference},
		targetScheduledWorker,
	)
	spec.SearchTerms = []string{"availability", "unavailable", "prefers", "weekday preference"}

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*schedulingservice.SetPreferenceRequest, *schedulingservice.PreferenceChange,
	]{
		Request: preferenceRequest,
		Plan: func(
			ctx context.Context,
			req *schedulingservice.SetPreferenceRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*schedulingservice.PreferenceChange, error) {
			if err := scope.require(
				ctx,
				params,
				permission.OpUpdate,
				req.Entity.WorkerID,
			); err != nil {
				return nil, err
			}
			entity := *req.Entity

			return schedules.PreviewSetPreference(ctx, &schedulingservice.SetPreferenceRequest{
				Entity:     &entity,
				TenantInfo: req.TenantInfo,
				UserID:     req.UserID,
			})
		},
		Refused: func(*schedulingservice.SetPreferenceRequest) string {
			return "Would record the worker's availability."
		},
		Render: renderPreference,
		Run: func(
			ctx context.Context,
			req *schedulingservice.SetPreferenceRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if err := scope.require(
				ctx,
				params,
				permission.OpUpdate,
				req.Entity.WorkerID,
			); err != nil {
				return nil, err
			}
			saved, err := schedules.SetPreference(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "recorded " + string(saved.Preference),
				Kind:   "availability preference",
				Name:   time.Weekday(saved.DayOfWeek).String(),
				IDs:    map[string]string{toolparams.ParamWorkerID: saved.WorkerID.String()},
				Record: changepreview.RecordOf(changepreview.WorkerRecordEntity, saved.WorkerID),
			}, nil
		},
	})
}

type preferenceView struct {
	Preference workerschedule.AvailabilityPreference `json:"preference"`
	Note       string                                `json:"note"`
}

func renderPreference(
	req *schedulingservice.SetPreferenceRequest,
	plan *schedulingservice.PreferenceChange,
) (*agentpreview.ToolPreview, error) {
	day := time.Weekday(req.Entity.DayOfWeek).String()
	rec := toolpreview.Record{
		Resource: permission.ResourceWorker,
		ID:       req.Entity.WorkerID,
		Label:    day + " availability",
	}
	after := &preferenceView{Preference: plan.After.Preference, Note: plan.After.Note}

	var (
		change *agentpreview.RecordChange
		err    error
	)
	if plan.Before == nil {
		change, err = toolpreview.Create(rec, after)
	} else {
		change, err = toolpreview.Changed(rec,
			&preferenceView{Preference: plan.Before.Preference, Note: plan.Before.Note}, after)
	}
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf("Would record the worker as %s on %s.",
		strings.ToLower(string(plan.After.Preference)), day), change), nil
}

func NewProposeShiftSwapTool(schedules scheduleKeeper) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "worker_request_shift_swap",
		SearchTerms: []string{"trade shifts", "shift trade", "swap shifts"},
		Artifact:    changepreview.WorkerRecordEntity,
		Description: "Raise a shift swap for a worker who asked the office: they give up a " +
			"day, handed to another worker or traded for one of theirs. It waits for the " +
			"other worker to accept and the office to approve.",
		Resource:    permission.ResourceShiftSwap,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Opens a request inside Trenova that changes nobody's schedule until it " +
			"is approved; worker_withdraw_shift_swap takes it back.",
		Properties: map[string]any{
			paramRequestingWorkerID: agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The worker giving the day up, from "+
					"worker_list. Never guess one.",
			),
			paramCounterpartyWorkerID: agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The worker asked to take it, from "+
					"worker_list. Leave out to offer it to anyone.",
			),
			paramShiftDate: agenttoolschema.Date("The day given up."),
			paramCounterpartyDate: agenttoolschema.Date(
				"The day offered back, when it is a trade " +
					"rather than a hand-off.",
			),
			changepreview.FieldReason: toolparams.StringProperty(
				"Why, in the worker's words where you have them.",
				maxScheduleNote,
			),
		},
		Required: []string{paramRequestingWorkerID, paramShiftDate},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(params, paramRequestingWorkerID, permission.ResourceWorker)
		},
	}, toolspec.ReceivablePlan[*schedulingservice.ProposeSwapRequest, *workerschedule.ShiftSwapRequest]{
		Request: proposeSwapRequest,
		Plan: func(
			_ context.Context,
			req *schedulingservice.ProposeSwapRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerschedule.ShiftSwapRequest, error) {
			entity := *req.Entity

			return schedules.PreviewProposeSwap(&schedulingservice.ProposeSwapRequest{
				Entity:     &entity,
				TenantInfo: req.TenantInfo,
				UserID:     req.UserID,
			})
		},
		Refused: func(*schedulingservice.ProposeSwapRequest) string {
			return "Would raise a shift swap."
		},
		Render: renderShiftSwapProposal,
		Run: func(
			ctx context.Context,
			req *schedulingservice.ProposeSwapRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := schedules.ProposeSwap(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "proposed",
				Kind:   kindShiftSwap,
				IDs:    map[string]string{paramSwapID: created.ID.String()},
				Record: changepreview.RecordOf(
					changepreview.WorkerRecordEntity,
					created.RequestingWorkerID,
				),
			}, nil
		},
	})
}

func renderShiftSwapProposal(
	_ *schedulingservice.ProposeSwapRequest,
	swap *workerschedule.ShiftSwapRequest,
) (*agentpreview.ToolPreview, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceShiftSwap, Label: "New shift swap"},
		swap,
		scheduleOptions(
			paramRequestingWorkerID,
			paramCounterpartyWorkerID,
			paramShiftDate,
			paramCounterpartyDate,
			changepreview.FieldReason,
			changepreview.FieldStatus,
		)...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would raise a swap of %s for the worker; nobody's schedule changes until "+
			"the office approves it.",
		time.Unix(swap.ShiftDate, 0).UTC().Format(toolparams.DayLayout),
	), change), nil
}

func proposeSwapRequest(
	params *agenttoolports.ToolExecuteParams,
) (*schedulingservice.ProposeSwapRequest, error) {
	requesting, err := toolparams.RequirePulid(params.Params, paramRequestingWorkerID)
	if err != nil {
		return nil, err
	}
	counterparty, _, err := toolparams.OptionalPulid(params.Params, paramCounterpartyWorkerID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramCounterpartyWorkerID,
			err)
	}
	shiftDate, err := toolparams.RequireScheduleDay(params.Params, paramShiftDate)
	if err != nil {
		return nil, err
	}
	counterDate, err := toolparams.OptionalScheduleDay(params.Params, paramCounterpartyDate)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.BoundedText(params.Params, changepreview.FieldReason, maxScheduleNote)
	if err != nil {
		return nil, err
	}

	return &schedulingservice.ProposeSwapRequest{
		Entity: &workerschedule.ShiftSwapRequest{
			RequestingWorkerID:    requesting,
			CounterpartyWorkerID:  counterparty,
			ShiftDate:             shiftDate,
			CounterpartyShiftDate: counterDate,
			Reason:                reason,
		},
		TenantInfo: toolparams.TenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}

type swapDecisionSpec struct {
	name        string
	description string
	rationale   string
	operation   permission.Operation
	status      workerschedule.ShiftSwapStatus
	verb        string
	egress      agent.EgressClass
	maxTier     agent.AutonomyTier
	personOnly  bool
	searchTerms []string
}

func newSwapDecisionTool(
	schedules scheduleKeeper,
	spec *swapDecisionSpec,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        spec.name,
		Description: spec.description,
		SearchTerms: spec.searchTerms,
		Artifact:    changepreview.WorkerRecordEntity,
		Resource:    permission.ResourceShiftSwap,
		Operation:   spec.operation,
		Egress:      spec.egress,
		DefaultTier: agent.TierPropose,
		MaxTier:     spec.maxTier,
		PersonOnly:  spec.personOnly,
		Rationale:   spec.rationale,
		Properties: map[string]any{
			paramSwapID: agenttoolschema.RecordIDText(
				permission.ResourceShiftSwap,
				"The swap, from worker_get_schedule. Never guess one.",
			),
			toolparams.ParamNote: toolparams.StringProperty("What the workers are told about it.",
				maxScheduleNote),
		},
		Required: []string{paramSwapID},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(params, paramSwapID, permission.ResourceShiftSwap)
		},
	}, toolspec.ReceivablePlan[*schedulingservice.TransitionSwapRequest, *schedulingservice.SwapChange]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*schedulingservice.TransitionSwapRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramSwapID)
			if err != nil {
				return nil, err
			}
			note, err := toolparams.BoundedText(
				params.Params,
				toolparams.ParamNote,
				maxScheduleNote,
			)
			if err != nil {
				return nil, err
			}

			return &schedulingservice.TransitionSwapRequest{
				ID:         id,
				Status:     spec.status,
				Note:       note,
				TenantInfo: toolparams.TenantFrom(*params),
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *schedulingservice.TransitionSwapRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*schedulingservice.SwapChange, error) {
			return schedules.PreviewTransitionSwap(ctx, req)
		},
		Refused: func(*schedulingservice.TransitionSwapRequest) string {
			return "Would " + spec.verb + " the shift swap."
		},
		Render: func(
			_ *schedulingservice.TransitionSwapRequest,
			plan *schedulingservice.SwapChange,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(
				swapRecord(plan.Before),
				plan.Before,
				plan.After,
				append(
					scheduleOptions(
						changepreview.FieldStatus,
						"responseNote",
						"decidedAt",
						"decidedById",
						"respondedAt",
					),
					toolpreview.Volatile("decidedAt", "respondedAt"),
				)...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would %s the swap of %s.",
				spec.verb,
				time.Unix(plan.Before.ShiftDate, 0).
					UTC().
					Format(toolparams.DayLayout),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			req *schedulingservice.TransitionSwapRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := schedules.TransitionSwap(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: strings.ToLower(string(updated.Status)),
				Kind:   kindShiftSwap,
				IDs:    map[string]string{paramSwapID: updated.ID.String()},
				Record: changepreview.RecordOf(
					changepreview.WorkerRecordEntity,
					updated.RequestingWorkerID,
				),
			}, nil
		},
	})
}

func NewApproveShiftSwapTool(schedules scheduleKeeper) agenttoolports.AgentTool {
	return newSwapDecisionTool(schedules, &swapDecisionSpec{
		name: "worker_approve_shift_swap",
		description: "Approve a shift swap the other worker has accepted, so the day " +
			"changes hands on the rota. Only a person approves one.",
		rationale: "Changes who works a day; approving is the office's decision, so " +
			"only a person makes it.",
		operation:  permission.OpApprove,
		status:     workerschedule.SwapApproved,
		verb:       "approve",
		egress:     agent.EgressDriverVisible,
		maxTier:    agent.TierPropose,
		personOnly: true,
	})
}

func NewRejectShiftSwapTool(schedules scheduleKeeper) agenttoolports.AgentTool {
	return newSwapDecisionTool(schedules, &swapDecisionSpec{
		name:        "worker_reject_shift_swap",
		searchTerms: []string{"deny swap", "decline swap"},
		description: "Reject a shift swap, with the reason the workers read, when the " +
			"office cannot let the day change hands.",
		rationale: "Refuses a request the workers read in their portal; a person approves " +
			"each refusal.",
		operation: permission.OpReject,
		status:    workerschedule.SwapRejected,
		verb:      "reject",
		egress:    agent.EgressDriverVisible,
		maxTier:   agent.TierActWithApproval,
	})
}

func NewWithdrawShiftSwapTool(schedules scheduleKeeper) agenttoolports.AgentTool {
	return newSwapDecisionTool(schedules, &swapDecisionSpec{
		name: "worker_withdraw_shift_swap",
		description: "Withdraw a shift swap on the proposing worker's behalf, when they " +
			"told the office they no longer want it.",
		rationale: "Takes back a request the workers read in their portal; a person " +
			"approves each withdrawal.",
		operation: permission.OpCancel,
		status:    workerschedule.SwapWithdrawn,
		verb:      "withdraw",
		egress:    agent.EgressDriverVisible,
		maxTier:   agent.TierActWithApproval,
	})
}

func assignShiftRequest(
	params *agenttoolports.ToolExecuteParams,
) (*schedulingservice.AssignShiftRequest, error) {
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	templateID, err := toolparams.RequirePulid(params.Params, paramShiftTemplateID)
	if err != nil {
		return nil, err
	}
	from, err := toolparams.RequireScheduleDay(params.Params, changepreview.FieldEffectiveFrom)
	if err != nil {
		return nil, err
	}
	offset := 0
	if _, given := params.Params[paramCycleOffsetWeeks]; given {
		if offset, err = toolparams.RequireIntInRange(params.Params, paramCycleOffsetWeeks, 0,
			maxCycleOffsetWeeks); err != nil {
			return nil, err
		}
	}
	notes, err := toolparams.BoundedText(
		params.Params,
		toolparams.ParamNote,
		toolparams.MaxOperationNoteChars,
	)
	if err != nil {
		return nil, err
	}

	return &schedulingservice.AssignShiftRequest{
		Entity: &workerschedule.WorkerShiftAssignment{
			WorkerID:         workerID,
			ShiftTemplateID:  templateID,
			EffectiveFrom:    from,
			CycleOffsetWeeks: int16(offset),
			Notes:            notes,
		},
		TenantInfo: toolparams.TenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}

func preferenceRequest(
	params *agenttoolports.ToolExecuteParams,
) (*schedulingservice.SetPreferenceRequest, error) {
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	day, err := requireWeekday(params.Params, paramDayOfWeek)
	if err != nil {
		return nil, err
	}
	preference, err := toolparams.RequireEnum(params.Params, paramPreference,
		availabilityPreferences.Values)
	if err != nil {
		return nil, err
	}
	note, err := toolparams.BoundedText(params.Params, toolparams.ParamNote, maxScheduleNote)
	if err != nil {
		return nil, err
	}

	return &schedulingservice.SetPreferenceRequest{
		Entity: &workerschedule.WorkerAvailabilityPreference{
			WorkerID:   workerID,
			DayOfWeek:  day,
			Preference: preference,
			Note:       note,
		},
		TenantInfo: toolparams.TenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}
