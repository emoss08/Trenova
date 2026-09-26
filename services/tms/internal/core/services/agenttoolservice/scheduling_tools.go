package agenttoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
	"github.com/emoss08/trenova/internal/core/services/teamscope"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramShiftTemplateID      = "shiftTemplateId"
	paramCycleOffsetWeeks     = "cycleOffsetWeeks"
	paramShiftAssignmentID    = "shiftAssignmentId"
	paramDayOfWeek            = "dayOfWeek"
	paramPreference           = "preference"
	paramSwapID               = "swapId"
	paramRequestingWorkerID   = "requestingWorkerId"
	paramCounterpartyWorkerID = "counterpartyWorkerId"
	paramShiftDate            = "shiftDate"
	paramCounterpartyDate     = "counterpartyShiftDate"
	kindShiftAssignment       = "shift assignment"
	kindShiftSwap             = "shift swap"
	maxScheduleNote           = 255
	maxCycleOffsetWeeks       = 7
)

var (
	availabilityPreferences = []worker.AvailabilityPreference{
		worker.AvailabilityPreferred,
		worker.AvailabilityAvailable,
		worker.AvailabilityUnavailable,
	}
	scheduleUserRefs = map[string]permission.Resource{
		paramWorkerID:             permission.ResourceWorker,
		paramShiftTemplateID:      permission.ResourceShiftTemplate,
		"assignedById":            permission.ResourceUser,
		paramRequestingWorkerID:   permission.ResourceWorker,
		paramCounterpartyWorkerID: permission.ResourceWorker,
		"decidedById":             permission.ResourceUser,
	}
	scheduleDays = map[string]assistantartifact.DisplayType{
		fieldEffectiveFrom:    assistantartifact.DisplayDate,
		fieldEffectiveTo:      assistantartifact.DisplayDate,
		paramShiftDate:        assistantartifact.DisplayDate,
		paramCounterpartyDate: assistantartifact.DisplayDate,
		"decidedAt":           assistantartifact.DisplayDateTime,
		"respondedAt":         assistantartifact.DisplayDateTime,
	}
)

type scheduleKeeper interface {
	GetAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerShiftAssignment, error)
	AssignShift(
		ctx context.Context,
		req *schedulingservice.AssignShiftRequest,
	) (*worker.WorkerShiftAssignment, error)
	PreviewAssignShift(
		ctx context.Context,
		req *schedulingservice.AssignShiftRequest,
	) (*schedulingservice.ShiftAssignmentPlan, error)
	EndAssignment(
		ctx context.Context,
		req *schedulingservice.EndAssignmentRequest,
	) (*worker.WorkerShiftAssignment, error)
	PreviewEndAssignment(
		ctx context.Context,
		req *schedulingservice.EndAssignmentRequest,
	) (*schedulingservice.AssignmentChange, error)
	SetPreference(
		ctx context.Context,
		req *schedulingservice.SetPreferenceRequest,
	) (*worker.WorkerAvailabilityPreference, error)
	PreviewSetPreference(
		ctx context.Context,
		req *schedulingservice.SetPreferenceRequest,
	) (*schedulingservice.PreferenceChange, error)
	ProposeSwap(
		ctx context.Context,
		req *schedulingservice.ProposeSwapRequest,
	) (*worker.ShiftSwapRequest, error)
	PreviewProposeSwap(req *schedulingservice.ProposeSwapRequest) (*worker.ShiftSwapRequest, error)
	TransitionSwap(
		ctx context.Context,
		req *schedulingservice.TransitionSwapRequest,
	) (*worker.ShiftSwapRequest, error)
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

func newTeamScope(permissions teamscope.Permissions, teams teamscope.Teams) teamScope {
	return teamScope{guard: teamscope.Guard{Permissions: permissions, Teams: teams}}
}

func (s teamScope) require(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
	operation permission.Operation,
	workerID pulid.ID,
) error {
	return s.guard.Require(ctx, &teamscope.Request{
		Actor:      params.Actor,
		TenantInfo: tenantFrom(*params),
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
	name, err := requireEnum(params, key, weekdayNames())
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

func optionalScheduleDay(params map[string]any, key string) (*int64, error) {
	raw, given := params[key]
	if !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent day is no day and no error
	}
	if text, isText := raw.(string); !isText || strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("parameter %q must be YYYY-MM-DD", key)
	}
	day, err := requireUTCDay(params, key)
	if err != nil {
		return nil, err
	}
	seconds := day.Unix()

	return &seconds, nil
}

func requireScheduleDay(params map[string]any, key string) (int64, error) {
	day, err := requireUTCDay(params, key)
	if err != nil {
		return 0, err
	}

	return day.Unix(), nil
}

func shiftAssignmentRecord(assignment *worker.WorkerShiftAssignment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceWorkerSchedule,
		ID:       assignment.ID,
		Label:    "Shift assignment",
		Version:  pinnedVersion(assignment.Version),
	}
}

func swapRecord(swap *worker.ShiftSwapRequest) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceShiftSwap,
		ID:       swap.ID,
		Label:    "Shift swap for " + time.Unix(swap.ShiftDate, 0).UTC().Format(dayLayout),
		Version:  pinnedVersion(swap.Version),
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
	target func(map[string]any) (serviceports.ToolTarget, bool),
) *receivableSpec {
	return &receivableSpec{
		name:        name,
		description: description,
		artifact:    workerRecordEntity,
		resource:    permission.ResourceWorkerSchedule,
		operation:   operation,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale:   rationale,
		properties:  properties,
		required:    required,
		target:      target,
	}
}

func targetScheduledWorker(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramWorkerID, permission.ResourceWorker)
}

func newAssignWorkerShiftTool(schedules scheduleKeeper, scope teamScope) serviceports.AgentTool {
	return newReportingReceivableTool(scheduleSpec(
		"assign_worker_shift",
		"Put a worker on a shift pattern from a day onward. The pattern they are on "+
			"now ends the day before; a worker who already has a pattern starting on or "+
			"after that day is refused. Take the shift from list_shift_templates and the "+
			"day from what the person asked for.",
		"Changes a worker's standing schedule inside Trenova; nothing is sent, and "+
			"end_worker_shift_assignment or a new assignment changes it back.",
		permission.OpAssign,
		map[string]any{
			paramWorkerID: workerProperty(),
			paramShiftTemplateID: idProperty("The shift pattern, from list_shift_templates. " +
				"Never guess one."),
			fieldEffectiveFrom: dayProperty("The first day the worker works it."),
			paramCycleOffsetWeeks: integerProperty("Which week of a rotating pattern the "+
				"worker starts on, 0 for the first.", 0, maxCycleOffsetWeeks),
			fieldNotes: stringProperty("Anything the planner should know.",
				maxOperationNoteChars),
		},
		[]string{paramWorkerID, paramShiftTemplateID, fieldEffectiveFrom},
		targetScheduledWorker,
	), receivablePlan[*schedulingservice.AssignShiftRequest, *schedulingservice.ShiftAssignmentPlan]{
		request: assignShiftRequest,
		plan: func(
			ctx context.Context,
			req *schedulingservice.AssignShiftRequest,
			params *serviceports.ToolExecuteParams,
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
		refused: func(*schedulingservice.AssignShiftRequest) string {
			return "Would put the worker on a shift pattern."
		},
		render: renderAssignShift,
		run: func(
			ctx context.Context,
			req *schedulingservice.AssignShiftRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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

			return &agent.ToolExecutionResult{
				Action: "assigned",
				Kind:   kindShiftAssignment,
				IDs: map[string]string{
					paramShiftAssignmentID: created.ID.String(),
					paramWorkerID:          created.WorkerID.String(),
				},
				Record: recordOf(workerRecordEntity, created.WorkerID),
			}, nil
		},
	})
}

func renderAssignShift(
	_ *schedulingservice.AssignShiftRequest,
	plan *schedulingservice.ShiftAssignmentPlan,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.Ended)+1)
	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceWorkerSchedule,
			Label:    "New shift assignment",
		},
		plan.Created,
		scheduleOptions(paramWorkerID, paramShiftTemplateID, fieldEffectiveFrom,
			paramCycleOffsetWeeks, fieldNotes)...,
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
			scheduleOptions(fieldEffectiveTo)...)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf("Would put the worker on %s from %s.", plan.Template.Name,
		time.Unix(plan.Created.EffectiveFrom, 0).UTC().Format(dayLayout))
	if len(plan.Ended) > 0 {
		summary += " The pattern they are on now ends the day before."
	}

	return toolpreview.Build(summary, changes...), nil
}

type assignmentEnd struct {
	req *schedulingservice.EndAssignmentRequest
}

func newEndWorkerShiftAssignmentTool(
	schedules scheduleKeeper,
	scope teamScope,
) serviceports.AgentTool {
	return newReportingReceivableTool(scheduleSpec(
		"end_worker_shift_assignment",
		"End a worker's shift pattern after a day, leaving them on no pattern from the "+
			"day after. Use assign_worker_shift instead to move them onto another pattern.",
		"Ends a worker's standing schedule inside Trenova; nothing is sent, and "+
			"assign_worker_shift puts them back on it.",
		permission.OpAssign,
		map[string]any{
			paramShiftAssignmentID: idProperty("The assignment, from get_worker_schedule. " +
				"Never guess one."),
			fieldEffectiveTo: dayProperty("The last day the worker works the pattern."),
		},
		[]string{paramShiftAssignmentID, fieldEffectiveTo},
		func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramShiftAssignmentID, permission.ResourceWorkerSchedule)
		},
	), receivablePlan[*assignmentEnd, *schedulingservice.AssignmentChange]{
		request: func(params *serviceports.ToolExecuteParams) (*assignmentEnd, error) {
			id, err := requirePulid(params.Params, paramShiftAssignmentID)
			if err != nil {
				return nil, err
			}
			to, err := requireScheduleDay(params.Params, fieldEffectiveTo)
			if err != nil {
				return nil, err
			}

			return &assignmentEnd{req: &schedulingservice.EndAssignmentRequest{
				ID:          id,
				EffectiveTo: to,
				TenantInfo:  tenantFrom(*params),
				UserID:      params.Actor.UserID,
			}}, nil
		},
		plan: func(
			ctx context.Context,
			end *assignmentEnd,
			params *serviceports.ToolExecuteParams,
		) (*schedulingservice.AssignmentChange, error) {
			if err := requireAssignmentScope(ctx, schedules, scope, end, params); err != nil {
				return nil, err
			}

			return schedules.PreviewEndAssignment(ctx, end.req)
		},
		refused: func(*assignmentEnd) string {
			return "Would end the worker's shift pattern."
		},
		render: func(
			_ *assignmentEnd,
			plan *schedulingservice.AssignmentChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(shiftAssignmentRecord(plan.Before), plan.Before,
				plan.After, scheduleOptions(fieldEffectiveTo)...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would end the worker's shift pattern after %s.",
				time.Unix(*plan.After.EffectiveTo, 0).UTC().Format(dayLayout),
			), change), nil
		},
		run: func(
			ctx context.Context,
			end *assignmentEnd,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if err := requireAssignmentScope(ctx, schedules, scope, end, params); err != nil {
				return nil, err
			}
			ended, err := schedules.EndAssignment(ctx, end.req)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "ended",
				Kind:   kindShiftAssignment,
				IDs:    map[string]string{paramShiftAssignmentID: ended.ID.String()},
				Record: recordOf(workerRecordEntity, ended.WorkerID),
			}, nil
		},
	})
}

// requireAssignmentScope reads the worker off the assignment being ended, the
// way the resolver does, so the team check is about whose schedule changes.
func requireAssignmentScope(
	ctx context.Context,
	schedules scheduleKeeper,
	scope teamScope,
	end *assignmentEnd,
	params *serviceports.ToolExecuteParams,
) error {
	assignment, err := schedules.GetAssignment(ctx, end.req.TenantInfo, end.req.ID)
	if err != nil {
		return err
	}

	return scope.require(ctx, params, permission.OpAssign, assignment.WorkerID)
}

func newSetWorkerAvailabilityPreferenceTool(
	schedules scheduleKeeper,
	scope teamScope,
) serviceports.AgentTool {
	return newReportingReceivableTool(scheduleSpec(
		"set_worker_availability_preference",
		"Record whether a worker prefers, is available for or cannot work one weekday, "+
			"as they told the office. The rota shows it beside their pattern; it does not "+
			"change the pattern.",
		"Records a worker's own stated preference inside Trenova; nothing is sent, and "+
			"it is set again the same way.",
		permission.OpUpdate,
		map[string]any{
			paramWorkerID: workerProperty(),
			paramDayOfWeek: enumProperty("The weekday it is about.",
				weekdayNames()),
			paramPreference: enumProperty("What the worker said about that day.",
				availabilityPreferences),
			paramNote: stringProperty("Why, in the worker's words where you have them.",
				maxScheduleNote),
		},
		[]string{paramWorkerID, paramDayOfWeek, paramPreference},
		targetScheduledWorker,
	), receivablePlan[*schedulingservice.SetPreferenceRequest, *schedulingservice.PreferenceChange]{
		request: preferenceRequest,
		plan: func(
			ctx context.Context,
			req *schedulingservice.SetPreferenceRequest,
			params *serviceports.ToolExecuteParams,
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
		refused: func(*schedulingservice.SetPreferenceRequest) string {
			return "Would record the worker's availability."
		},
		render: renderPreference,
		run: func(
			ctx context.Context,
			req *schedulingservice.SetPreferenceRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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

			return &agent.ToolExecutionResult{
				Action: "recorded " + string(saved.Preference),
				Kind:   "availability preference",
				Name:   time.Weekday(saved.DayOfWeek).String(),
				IDs:    map[string]string{paramWorkerID: saved.WorkerID.String()},
				Record: recordOf(workerRecordEntity, saved.WorkerID),
			}, nil
		},
	})
}

type preferenceView struct {
	Preference worker.AvailabilityPreference `json:"preference"`
	Note       string                        `json:"note"`
}

func renderPreference(
	req *schedulingservice.SetPreferenceRequest,
	plan *schedulingservice.PreferenceChange,
) (*agent.ToolPreview, error) {
	day := time.Weekday(req.Entity.DayOfWeek).String()
	rec := toolpreview.Record{
		Resource: permission.ResourceWorker,
		ID:       req.Entity.WorkerID,
		Label:    day + " availability",
	}
	after := &preferenceView{Preference: plan.After.Preference, Note: plan.After.Note}

	var (
		change *agent.RecordChange
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

func newProposeShiftSwapTool(schedules scheduleKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "propose_shift_swap",
		artifact: workerRecordEntity,
		description: "Raise a shift swap for a worker who asked the office: they give up a " +
			"day, handed to another worker or traded for one of theirs. It waits for the " +
			"other worker to accept and the office to approve.",
		resource:    permission.ResourceShiftSwap,
		operation:   permission.OpCreate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Opens a request inside Trenova that changes nobody's schedule until it " +
			"is approved; withdraw_shift_swap takes it back.",
		properties: map[string]any{
			paramRequestingWorkerID: idProperty("The worker giving the day up, from " +
				"search_worker. Never guess one."),
			paramCounterpartyWorkerID: idProperty("The worker asked to take it, from " +
				"search_worker. Leave out to offer it to anyone."),
			paramShiftDate: dayProperty("The day given up."),
			paramCounterpartyDate: dayProperty("The day offered back, when it is a trade " +
				"rather than a hand-off."),
			fieldReason: stringProperty("Why, in the worker's words where you have them.",
				maxScheduleNote),
		},
		required: []string{paramRequestingWorkerID, paramShiftDate},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramRequestingWorkerID, permission.ResourceWorker)
		},
	}, receivablePlan[*schedulingservice.ProposeSwapRequest, *worker.ShiftSwapRequest]{
		request: proposeSwapRequest,
		plan: func(
			_ context.Context,
			req *schedulingservice.ProposeSwapRequest,
			_ *serviceports.ToolExecuteParams,
		) (*worker.ShiftSwapRequest, error) {
			entity := *req.Entity

			return schedules.PreviewProposeSwap(&schedulingservice.ProposeSwapRequest{
				Entity:     &entity,
				TenantInfo: req.TenantInfo,
				UserID:     req.UserID,
			})
		},
		refused: func(*schedulingservice.ProposeSwapRequest) string {
			return "Would raise a shift swap."
		},
		render: func(
			_ *schedulingservice.ProposeSwapRequest,
			swap *worker.ShiftSwapRequest,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				toolpreview.Record{Resource: permission.ResourceShiftSwap, Label: "New shift swap"},
				swap,
				scheduleOptions(paramRequestingWorkerID, paramCounterpartyWorkerID,
					paramShiftDate, paramCounterpartyDate, fieldReason, fieldStatus)...,
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would raise a swap of %s for the worker; nobody's schedule changes until "+
					"the office approves it.",
				time.Unix(swap.ShiftDate, 0).UTC().Format(dayLayout),
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *schedulingservice.ProposeSwapRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := schedules.ProposeSwap(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "proposed",
				Kind:   kindShiftSwap,
				IDs:    map[string]string{paramSwapID: created.ID.String()},
				Record: recordOf(workerRecordEntity, created.RequestingWorkerID),
			}, nil
		},
	})
}

func proposeSwapRequest(
	params *serviceports.ToolExecuteParams,
) (*schedulingservice.ProposeSwapRequest, error) {
	requesting, err := requirePulid(params.Params, paramRequestingWorkerID)
	if err != nil {
		return nil, err
	}
	counterparty, _, err := optionalPulid(params.Params, paramCounterpartyWorkerID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramCounterpartyWorkerID,
			err)
	}
	shiftDate, err := requireScheduleDay(params.Params, paramShiftDate)
	if err != nil {
		return nil, err
	}
	counterDate, err := optionalScheduleDay(params.Params, paramCounterpartyDate)
	if err != nil {
		return nil, err
	}
	reason, err := boundedText(params.Params, fieldReason, maxScheduleNote)
	if err != nil {
		return nil, err
	}

	return &schedulingservice.ProposeSwapRequest{
		Entity: &worker.ShiftSwapRequest{
			RequestingWorkerID:    requesting,
			CounterpartyWorkerID:  counterparty,
			ShiftDate:             shiftDate,
			CounterpartyShiftDate: counterDate,
			Reason:                reason,
		},
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}

type swapDecisionSpec struct {
	name        string
	description string
	rationale   string
	operation   permission.Operation
	status      worker.ShiftSwapStatus
	verb        string
	egress      agent.EgressClass
	maxTier     agent.AutonomyTier
	personOnly  bool
}

func newSwapDecisionTool(schedules scheduleKeeper, spec *swapDecisionSpec) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:        spec.name,
		description: spec.description,
		artifact:    workerRecordEntity,
		resource:    permission.ResourceShiftSwap,
		operation:   spec.operation,
		egress:      spec.egress,
		defaultTier: agent.TierPropose,
		maxTier:     spec.maxTier,
		personOnly:  spec.personOnly,
		rationale:   spec.rationale,
		properties: map[string]any{
			paramSwapID: idProperty("The swap, from get_worker_schedule. Never guess one."),
			paramNote: stringProperty("What the workers are told about it.",
				maxScheduleNote),
		},
		required: []string{paramSwapID},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramSwapID, permission.ResourceShiftSwap)
		},
	}, receivablePlan[*schedulingservice.TransitionSwapRequest, *schedulingservice.SwapChange]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*schedulingservice.TransitionSwapRequest, error) {
			id, err := requirePulid(params.Params, paramSwapID)
			if err != nil {
				return nil, err
			}
			note, err := boundedText(params.Params, paramNote, maxScheduleNote)
			if err != nil {
				return nil, err
			}

			return &schedulingservice.TransitionSwapRequest{
				ID:         id,
				Status:     spec.status,
				Note:       note,
				TenantInfo: tenantFrom(*params),
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *schedulingservice.TransitionSwapRequest,
			_ *serviceports.ToolExecuteParams,
		) (*schedulingservice.SwapChange, error) {
			return schedules.PreviewTransitionSwap(ctx, req)
		},
		refused: func(*schedulingservice.TransitionSwapRequest) string {
			return "Would " + spec.verb + " the shift swap."
		},
		render: func(
			_ *schedulingservice.TransitionSwapRequest,
			plan *schedulingservice.SwapChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(swapRecord(plan.Before), plan.Before, plan.After,
				append(scheduleOptions(fieldStatus, "responseNote", "decidedAt", "decidedById",
					"respondedAt"), toolpreview.Volatile("decidedAt", "respondedAt"))...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would %s the swap of %s.", spec.verb,
				time.Unix(plan.Before.ShiftDate, 0).UTC().Format(dayLayout)), change), nil
		},
		run: func(
			ctx context.Context,
			req *schedulingservice.TransitionSwapRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := schedules.TransitionSwap(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: strings.ToLower(string(updated.Status)),
				Kind:   kindShiftSwap,
				IDs:    map[string]string{paramSwapID: updated.ID.String()},
				Record: recordOf(workerRecordEntity, updated.RequestingWorkerID),
			}, nil
		},
	})
}

func newApproveShiftSwapTool(schedules scheduleKeeper) serviceports.AgentTool {
	return newSwapDecisionTool(schedules, &swapDecisionSpec{
		name: "approve_shift_swap",
		description: "Approve a shift swap the other worker has accepted, so the day " +
			"changes hands on the rota. Only a person approves one.",
		rationale: "Changes who works a day; approving is the office's decision, so " +
			"only a person makes it.",
		operation:  permission.OpApprove,
		status:     worker.SwapApproved,
		verb:       "approve",
		egress:     agent.EgressDriverVisible,
		maxTier:    agent.TierPropose,
		personOnly: true,
	})
}

func newRejectShiftSwapTool(schedules scheduleKeeper) serviceports.AgentTool {
	return newSwapDecisionTool(schedules, &swapDecisionSpec{
		name: "reject_shift_swap",
		description: "Reject a shift swap, with the reason the workers read, when the " +
			"office cannot let the day change hands.",
		rationale: "Refuses a request the workers read in their portal; a person approves " +
			"each refusal.",
		operation: permission.OpReject,
		status:    worker.SwapRejected,
		verb:      "reject",
		egress:    agent.EgressDriverVisible,
		maxTier:   agent.TierActWithApproval,
	})
}

func newWithdrawShiftSwapTool(schedules scheduleKeeper) serviceports.AgentTool {
	return newSwapDecisionTool(schedules, &swapDecisionSpec{
		name: "withdraw_shift_swap",
		description: "Withdraw a shift swap on the proposing worker's behalf, when they " +
			"told the office they no longer want it.",
		rationale: "Takes back a request the workers read in their portal; a person " +
			"approves each withdrawal.",
		operation: permission.OpCancel,
		status:    worker.SwapWithdrawn,
		verb:      "withdraw",
		egress:    agent.EgressDriverVisible,
		maxTier:   agent.TierActWithApproval,
	})
}

func assignShiftRequest(
	params *serviceports.ToolExecuteParams,
) (*schedulingservice.AssignShiftRequest, error) {
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	templateID, err := requirePulid(params.Params, paramShiftTemplateID)
	if err != nil {
		return nil, err
	}
	from, err := requireScheduleDay(params.Params, fieldEffectiveFrom)
	if err != nil {
		return nil, err
	}
	offset := 0
	if _, given := params.Params[paramCycleOffsetWeeks]; given {
		if offset, err = requireIntInRange(params.Params, paramCycleOffsetWeeks, 0,
			maxCycleOffsetWeeks); err != nil {
			return nil, err
		}
	}
	notes, err := boundedText(params.Params, fieldNotes, maxOperationNoteChars)
	if err != nil {
		return nil, err
	}

	return &schedulingservice.AssignShiftRequest{
		Entity: &worker.WorkerShiftAssignment{
			WorkerID:         workerID,
			ShiftTemplateID:  templateID,
			EffectiveFrom:    from,
			CycleOffsetWeeks: int16(offset),
			Notes:            notes,
		},
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}

func preferenceRequest(
	params *serviceports.ToolExecuteParams,
) (*schedulingservice.SetPreferenceRequest, error) {
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	day, err := requireWeekday(params.Params, paramDayOfWeek)
	if err != nil {
		return nil, err
	}
	preference, err := requireEnum(params.Params, paramPreference,
		availabilityPreferences)
	if err != nil {
		return nil, err
	}
	note, err := boundedText(params.Params, paramNote, maxScheduleNote)
	if err != nil {
		return nil, err
	}

	return &schedulingservice.SetPreferenceRequest{
		Entity: &worker.WorkerAvailabilityPreference{
			WorkerID:   workerID,
			DayOfWeek:  day,
			Preference: preference,
			Note:       note,
		},
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}
