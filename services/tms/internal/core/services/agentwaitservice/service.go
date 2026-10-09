// Package agentwaitservice parks an agent's work until something happens in
// the world, and picks it up again when it does.
//
// A wait never holds a conversation or a run open. The turn or run that sets
// one ends as usual; a workflow per wait holds the timer and listens for the
// signal, and when the wait ends the work comes back as a new turn of the
// same conversation, or a new run of the same agent on the same record, told
// what it waited for, what came of it and what it said it would do then.
package agentwaitservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentwaitjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WaitsResource is the realtime resource a conversation's waits move under.
const WaitsResource = "agent_waits"

const (
	// minLead is how far ahead a wait on a time must come due; anything
	// sooner is now, and the agent should act rather than wait.
	minLead int64 = 60
	// rearmSlack is how close to now a timed wait may come due again before
	// it counts as due.
	rearmSlack int64 = 30
	// expiryAfterTime keeps a wait on a time open a little past that time.
	expiryAfterTime int64 = 5 * 60
	timeLayout            = "Mon Jan 2 15:04 MST"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Waits         repositories.AgentWaitRepository
	Runs          repositories.AgentRunRepository
	Moves         repositories.ShipmentMoveRepository
	Assignments   repositories.AssignmentRepository
	Occurrences   repositories.DetentionOccurrenceRepository
	Telematics    repositories.TelematicsRepository
	Carriers      repositories.CarrierRepository
	Customers     repositories.CustomerRepository
	Subjects      repositories.AgentSubjectRepository
	Conversations repositories.ConversationRepository
	Turns         *assistantturnservice.Service
	RunService    serviceports.AgentRunService
	Workflows     serviceports.WorkflowStarter
	Realtime      serviceports.RealtimeService `optional:"true"`
}

type turnStarter interface {
	Active(
		ctx context.Context,
		req repositories.ActiveAssistantTurnRequest,
	) (*conversation.AssistantTurn, error)
	StartTurn(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
}

type runStarter interface {
	StartForDefinition(
		ctx context.Context,
		req *serviceports.StartAgentRunForDefinitionRequest,
		actor *serviceports.RequestActor,
	) (*agent.AgentRun, error)
}

type Service struct {
	l             *zap.Logger
	waits         repositories.AgentWaitRepository
	runs          repositories.AgentRunRepository
	facts         *facts
	conversations repositories.ConversationRepository
	turns         turnStarter
	runService    runStarter
	workflows     serviceports.WorkflowStarter
	realtime      serviceports.RealtimeService
	now           func() int64
}

var (
	_ serviceports.AgentWaitService  = (*Service)(nil)
	_ serviceports.AgentWaitNotifier = (*Service)(nil)
	_ agentwaitjobs.Worker           = (*Service)(nil)
)

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	s := &Service{
		l:     p.Logger.Named("service.agentwait"),
		waits: p.Waits,
		runs:  p.Runs,
		facts: &facts{
			moves:       p.Moves,
			assignments: p.Assignments,
			occurrences: p.Occurrences,
			telematics:  p.Telematics,
			carriers:    p.Carriers,
			customers:   p.Customers,
			subjects:    p.Subjects,
		},
		conversations: p.Conversations,
		runService:    p.RunService,
		workflows:     p.Workflows,
		realtime:      p.Realtime,
		now:           timeutils.NowUnix,
	}
	if p.Turns != nil {
		s.turns = p.Turns
	}

	return s
}

func NewWaitService(s *Service) serviceports.AgentWaitService { return s }

func NewNotifier(s *Service) serviceports.AgentWaitNotifier { return s }

func NewWorker(s *Service) agentwaitjobs.Worker { return s }

// Register parks the work. It reads the records the wait names first: a wait
// on something that has already happened, or on a record that is not there,
// is refused with what is true now, so the agent acts instead of waiting.
func (s *Service) Register(
	ctx context.Context,
	req *serviceports.RegisterWaitRequest,
) (*agentwait.Wait, error) {
	if req.Delegated {
		return nil, errortypes.NewBusinessError(
			"A task another agent handed you cannot be parked on a wait. Finish what you " +
				"can and tell that agent what to wait for.",
		)
	}
	if req.Actor == nil {
		return nil, errortypes.NewBusinessError("A wait is set by an agent's work")
	}

	wait, err := s.draft(ctx, req)
	if err != nil {
		return nil, err
	}
	wait.Normalize()
	multiErr := errortypes.NewMultiError()
	wait.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	now := s.now()
	due, err := s.facts.assess(ctx, wait, now)
	if err != nil {
		return nil, err
	}
	wait.DueAt = due
	wait.ExpiresAt = expiry(wait, now, req.LifetimeSeconds)

	created, err := s.waits.Insert(ctx, wait)
	if errors.Is(err, repositories.ErrTooManyWaits) {
		return nil, errortypes.NewBusinessError(fmt.Sprintf(
			"There are already %d open waits here. Cancel one you no longer need first.",
			agentwait.MaxOpenPerOwner,
		))
	}
	if err != nil {
		return nil, err
	}

	if err = s.start(ctx, created); err != nil {
		s.close(ctx, created, agentwait.StatusFailed, "The wait could not be started.")
		return nil, fmt.Errorf("start the wait: %w", err)
	}
	s.announce(ctx, created)

	return created, nil
}

// draft is the wait as asked for, owned by the conversation the work is
// answering or, for a background run, by the run's agent and record.
func (s *Service) draft(
	ctx context.Context,
	req *serviceports.RegisterWaitRequest,
) (*agentwait.Wait, error) {
	condition := req.Condition
	wait := &agentwait.Wait{
		OrganizationID:    req.Actor.OrganizationID,
		BusinessUnitID:    req.Actor.BusinessUnitID,
		Kind:              req.Kind,
		Condition:         &condition,
		Description:       req.Description,
		Then:              req.Then,
		AgentDefinitionID: req.DefinitionID,
		Status:            agentwait.StatusWaiting,
	}

	switch {
	case req.ThreadID.IsNotNil():
		wait.ThreadID = req.ThreadID
		wait.UserID = req.Actor.UserID
	case req.RunID.IsNotNil():
		tenant := pagination.TenantInfo{OrgID: wait.OrganizationID, BuID: wait.BusinessUnitID}
		run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
			ID:         req.RunID,
			TenantInfo: &tenant,
		})
		if err != nil {
			return nil, fmt.Errorf("read the run setting the wait: %w", err)
		}
		wait.RunID = run.ID
		wait.SubjectType = run.SubjectType
		wait.SubjectID = run.SubjectID
		wait.Taint = run.Taint
	default:
		return nil, errortypes.NewBusinessError(
			"A wait can only be set from a conversation or an agent run",
		)
	}

	return wait, nil
}

func expiry(wait *agentwait.Wait, now, lifetime int64) int64 {
	if lifetime <= 0 {
		lifetime = agentwait.DefaultLifetimeSecs
	}
	lifetime = min(max(lifetime, agentwait.MinLifetimeSeconds), agentwait.MaxLifetimeSeconds)
	expires := now + lifetime
	if wait.DueAt != nil {
		expires = max(expires, *wait.DueAt+expiryAfterTime)
	}

	return expires
}

func (s *Service) start(ctx context.Context, wait *agentwait.Wait) error {
	workflowID := agentwaitjobs.WorkflowIDFor(wait.ID)
	_, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.TaskQueueSystem.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		StaticSummary:         "Agent wait: " + wait.Description,
	}, agentwaitjobs.WorkflowName, &agentwaitjobs.Payload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: wait.OrganizationID,
			BusinessUnitID: wait.BusinessUnitID,
			UserID:         wait.UserID,
			Timestamp:      s.now(),
		},
		WaitID: wait.ID,
	})
	if err != nil {
		return err
	}
	wait.WorkflowID = workflowID

	return s.waits.SetDue(ctx, &repositories.SetAgentWaitDueRequest{
		ID:         wait.ID,
		TenantInfo: tenantOf(wait),
		DueAt:      wait.DueAt,
		WorkflowID: workflowID,
	})
}

// Cancel ends a wait without picking the work up.
func (s *Service) Cancel(
	ctx context.Context,
	req *serviceports.CancelWaitRequest,
) (*agentwait.Wait, error) {
	wait, err := s.waits.Get(ctx, &repositories.GetAgentWaitRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		ThreadID:   req.ThreadID,
		UserID:     req.UserID,
	})
	if err != nil {
		return nil, err
	}
	if req.ThreadID.IsNil() && req.UserID.IsNil() &&
		(req.DefinitionID.IsNil() || wait.AgentDefinitionID != req.DefinitionID) {
		return nil, errortypes.NewNotFoundError("{0} not found within your organization", "Wait")
	}
	if !wait.Status.Open() {
		return wait, nil
	}

	outcome := "Cancelled."
	if req.By != "" {
		outcome = "Cancelled by " + req.By + "."
	}
	closed, changed, err := s.waits.Resolve(ctx, &repositories.ResolveAgentWaitRequest{
		ID:         wait.ID,
		TenantInfo: tenantOf(wait),
		Status:     agentwait.StatusCancelled,
		Outcome:    outcome,
		ResolvedAt: s.now(),
	})
	if err != nil {
		return nil, err
	}
	if changed && closed.WorkflowID != "" {
		if cErr := s.workflows.CancelWorkflow(ctx, closed.WorkflowID, ""); cErr != nil {
			var gone *serviceerror.NotFound
			if !errors.As(cErr, &gone) {
				s.l.Warn("a cancelled wait's workflow could not be stopped; it ends on its own",
					zap.String("wait", closed.ID.String()), zap.Error(cErr))
			}
		}
	}
	s.announce(ctx, closed)

	return closed, nil
}

// List is a conversation's waits, newest first, as its owner sees them.
func (s *Service) List(
	ctx context.Context,
	req *repositories.ListThreadWaitsRequest,
) ([]*agentwait.Wait, error) {
	if _, err := s.conversations.GetThread(ctx, repositories.GetThreadRequest{
		ID:         req.ThreadID,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	return s.waits.ListByThread(ctx, req)
}

// NotifyEvent ends the waits watching for what the event says happened.
func (s *Service) NotifyEvent(ctx context.Context, event *serviceports.AgentEvent) {
	kinds, watched := eventWatch(event)
	if len(kinds) == 0 || len(watched) == 0 {
		return
	}
	ctx = dbscope.WithValidTenant(context.WithoutCancel(ctx), event.TenantInfo.DBTenant())

	waits, err := s.waits.ListOpenWatching(ctx, &repositories.ListOpenWaitsWatchingRequest{
		TenantInfo: event.TenantInfo,
		Kinds:      kinds,
		WatchIDs:   watched,
	})
	if err != nil {
		s.l.Warn("could not read the waits an event may end",
			zap.String("kind", string(event.Kind)), zap.Error(err))
		return
	}

	detail := event.Detail
	if detail == "" {
		detail = fmt.Sprintf("Seen at %s.", stamp(s.now()))
	}
	for _, wait := range waits {
		if eventMeets(wait, event) {
			s.meet(ctx, wait, detail)
		}
	}
}

func eventWatch(event *serviceports.AgentEvent) ([]agentwait.Kind, []pulid.ID) {
	switch event.Kind { //nolint:exhaustive // only these events end a wait
	case agent.EventShipmentMoveArrived:
		return []agentwait.Kind{agentwait.KindStopArrival}, []pulid.ID{event.SubjectID}
	case agent.EventShipmentMoveDeparted:
		return []agentwait.Kind{agentwait.KindStopDeparture}, []pulid.ID{event.SubjectID}
	case agent.EventInboundMessageClassified:
		return []agentwait.Kind{agentwait.KindReply}, event.Related
	default:
		return nil, nil
	}
}

// eventMeets reports whether the event is the one the wait is for. A wait on
// one stop of a move ends only at that stop.
func eventMeets(wait *agentwait.Wait, event *serviceports.AgentEvent) bool {
	switch wait.Kind { //nolint:exhaustive // the kinds an event ends
	case agentwait.KindStopArrival, agentwait.KindStopDeparture:
		return wait.Condition.StopID.IsNil() ||
			slices.Contains(event.Related, wait.Condition.StopID)
	case agentwait.KindReply:
		return slices.Contains(event.Related, wait.WatchID)
	default:
		return false
	}
}

// NotifyHOS ends the waits on drivers whose drive time has fallen below what
// the wait named. The clocks are polled every minute, so a wait hears within
// a minute of the clock crossing.
func (s *Service) NotifyHOS(
	ctx context.Context,
	tenant pagination.TenantInfo,
	states []*telematics.WorkerHOSState,
) {
	if len(states) == 0 {
		return
	}
	byWorker := make(map[pulid.ID]*telematics.WorkerHOSState, len(states))
	workers := make([]pulid.ID, 0, len(states))
	for _, state := range states {
		if state == nil || state.WorkerID.IsNil() {
			continue
		}
		byWorker[state.WorkerID] = state
		workers = append(workers, state.WorkerID)
	}
	ctx = dbscope.WithValidTenant(context.WithoutCancel(ctx), tenant.DBTenant())

	waits, err := s.waits.ListOpenWatching(ctx, &repositories.ListOpenWaitsWatchingRequest{
		TenantInfo: tenant,
		Kinds:      []agentwait.Kind{agentwait.KindHOSDriveBelow},
		WatchIDs:   workers,
	})
	if err != nil {
		s.l.Warn("could not read the waits on drivers' hours", zap.Error(err))
		return
	}
	for _, wait := range waits {
		state := byWorker[wait.WatchID]
		if state == nil {
			continue
		}
		minutes := wait.Condition.DriveMinutesBelow
		if driveBelow(state, minutes) {
			s.meet(ctx, wait, fmt.Sprintf("Drive time left: %s, as of %s.",
				durationText(state.DriveRemainingMs), stamp(state.RecordedAt)))
			continue
		}
		s.reschedule(ctx, wait, crossingAt(state, minutes))
	}
}

// reschedule moves when a timed wait comes due and tells its workflow, when
// the time moved. A driver who stops driving has no crossing to time; the
// next clock read that finds them driving sets one again.
func (s *Service) reschedule(ctx context.Context, wait *agentwait.Wait, due *int64) {
	if sameDue(wait.DueAt, due) {
		return
	}
	if err := s.waits.SetDue(ctx, &repositories.SetAgentWaitDueRequest{
		ID:         wait.ID,
		TenantInfo: tenantOf(wait),
		DueAt:      due,
	}); err != nil {
		s.l.Warn("could not move when a wait comes due", zap.String("wait", wait.ID.String()),
			zap.Error(err))
		return
	}
	wait.DueAt = due
	if err := s.workflows.SignalWorkflow(ctx, wait.WorkflowID, "",
		agentwaitjobs.RescheduleSignal, nil); err != nil {
		s.l.Debug("could not tell a wait its due time moved; it reads it when it next wakes",
			zap.String("wait", wait.ID.String()), zap.Error(err))
	}
}

func sameDue(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}

// crossingAt is when a driving driver's drive clock crosses the minutes
// given, counted from the clock's own reading. Nil for one not driving, whose
// clock is not running down.
func crossingAt(state *telematics.WorkerHOSState, minutes int) *int64 {
	if state.DutyStatus != telematics.DutyStatusDriving {
		return nil
	}
	leftMs := state.DriveRemainingMs - int64(minutes)*int64(time.Minute/time.Millisecond)
	at := state.RecordedAt + leftMs/int64(time.Second/time.Millisecond)

	return &at
}

// projectedDriveMs is the drive time left now, from a clock read at an
// earlier moment: the same, unless the driver was driving, when it ran down
// by the time since.
func projectedDriveMs(state *telematics.WorkerHOSState, now int64) int64 {
	if state.DutyStatus != telematics.DutyStatusDriving || now <= state.RecordedAt {
		return state.DriveRemainingMs
	}

	return state.DriveRemainingMs - (now-state.RecordedAt)*int64(time.Second/time.Millisecond)
}

func driveBelow(state *telematics.WorkerHOSState, minutes int) bool {
	return state.DriveRemainingMs < int64(minutes)*int64(time.Minute/time.Millisecond)
}

// meet tells a wait's workflow it is over. A wait whose workflow is gone is
// closed and picked up here, so a lost workflow never strands the work. Gone
// most often means the workflow already ended it, which a second notice of
// the same arrival finds; the wait is read again first so nothing is picked
// up twice.
func (s *Service) meet(ctx context.Context, wait *agentwait.Wait, detail string) {
	err := s.workflows.SignalWorkflow(ctx, wait.WorkflowID, "", agentwaitjobs.MetSignal,
		agentwaitjobs.Met{Detail: detail, At: s.now()})
	if err == nil {
		return
	}

	var gone *serviceerror.NotFound
	if !errors.As(err, &gone) {
		s.l.Warn("could not tell a wait it was met; it looks again when it comes due",
			zap.String("wait", wait.ID.String()), zap.Error(err))
		return
	}
	current, gErr := s.waits.Get(ctx, &repositories.GetAgentWaitRequest{
		ID:         wait.ID,
		TenantInfo: tenantOf(wait),
	})
	if gErr != nil || !current.Status.Open() {
		return
	}
	s.finishLost(ctx, current, agentwait.StatusMet, detail)
}

// finishLost closes and picks up a wait whose workflow is gone.
func (s *Service) finishLost(
	ctx context.Context,
	wait *agentwait.Wait,
	status agentwait.Status,
	detail string,
) {
	if fErr := s.Finish(ctx, &agentwaitjobs.FinishInput{
		Payload: &agentwaitjobs.Payload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: wait.OrganizationID,
				BusinessUnitID: wait.BusinessUnitID,
			},
			WaitID: wait.ID,
		},
		Status:  status,
		Outcome: detail,
	}); fErr != nil {
		s.l.Warn("a wait whose workflow is gone could not be picked up",
			zap.String("wait", wait.ID.String()), zap.Error(fErr))
	}
}

// overdueGrace is how long past its expiry a wait is left to its own
// workflow before the reconcile takes it as lost.
const overdueGrace int64 = 15 * 60

// ReconcileOverdue closes and picks up the open waits past their expiry that
// their own workflow never closed: one terminated by hand, or lost with a
// namespace. Each is picked up as a wait that ran out.
func (s *Service) ReconcileOverdue(ctx context.Context) (int, error) {
	overdue, err := s.waits.ListOverdueAcrossTenants(ctx, &repositories.ListOverdueWaitsRequest{
		ExpiredBefore: s.now() - overdueGrace,
	})
	if err != nil {
		return 0, err
	}

	for _, wait := range overdue {
		waitCtx := dbscope.WithValidTenant(ctx, tenantOf(wait).DBTenant())
		s.l.Info("closing a wait its workflow never closed", zap.String("wait", wait.ID.String()))
		s.finishLost(waitCtx, wait, agentwait.StatusTimedOut,
			"The wait ran out without being closed; it was found and closed later.")
	}

	return len(overdue), nil
}

// Schedule is when the workflow next needs to look at the wait.
func (s *Service) Schedule(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*agentwaitjobs.Schedule, error) {
	wait, err := s.waits.Get(ctx, &repositories.GetAgentWaitRequest{ID: id, TenantInfo: tenant})
	if err != nil {
		return nil, err
	}

	return &agentwaitjobs.Schedule{
		Open:      wait.Status.Open(),
		DueAt:     wait.DueAt,
		ExpiresAt: wait.ExpiresAt,
	}, nil
}

// Check reads a timed wait again when it comes due. An appointment or a free
// time that moved puts it off; one that is gone ends it, so the agent hears.
func (s *Service) Check(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	now int64,
) (*agentwaitjobs.Check, error) {
	wait, err := s.waits.Get(ctx, &repositories.GetAgentWaitRequest{ID: id, TenantInfo: tenant})
	if err != nil {
		return nil, err
	}
	if !wait.Status.Open() {
		return &agentwaitjobs.Check{Closed: true}, nil
	}

	check, err := s.facts.recheck(ctx, wait, now)
	if err != nil {
		return nil, err
	}
	if !check.Met && check.DueAt != nil {
		if err = s.waits.SetDue(ctx, &repositories.SetAgentWaitDueRequest{
			ID:         wait.ID,
			TenantInfo: tenant,
			DueAt:      check.DueAt,
		}); err != nil {
			return nil, err
		}
	}

	return check, nil
}

// Finish closes a wait as met or run out and picks the work up. A retried
// attempt finds the wait closed and picks up only what it has not.
func (s *Service) Finish(ctx context.Context, in *agentwaitjobs.FinishInput) error {
	tenant := pagination.TenantInfo{
		OrgID: in.Payload.OrganizationID,
		BuID:  in.Payload.BusinessUnitID,
	}
	ctx = dbscope.WithValidTenant(ctx, tenant.DBTenant())

	wait, changed, err := s.waits.Resolve(ctx, &repositories.ResolveAgentWaitRequest{
		ID:         in.Payload.WaitID,
		TenantInfo: tenant,
		Status:     in.Status,
		Outcome:    clip(in.Outcome, agentwait.MaxOutcomeRunes),
		ResolvedAt: s.now(),
	})
	if err != nil {
		return err
	}
	if changed {
		s.announce(ctx, wait)
	}
	if wait.Status != agentwait.StatusMet && wait.Status != agentwait.StatusTimedOut {
		return nil
	}
	if wait.ResumedTurnID.IsNotNil() || wait.ResumedRunID.IsNotNil() {
		return nil
	}

	if wait.Conversational() {
		return s.resumeConversation(ctx, wait)
	}

	return s.resumeRun(ctx, wait)
}

func (s *Service) resumeConversation(ctx context.Context, wait *agentwait.Wait) error {
	tenant := pagination.TenantInfo{
		OrgID:  wait.OrganizationID,
		BuID:   wait.BusinessUnitID,
		UserID: wait.UserID,
	}
	thread, err := s.conversations.GetThreadOwned(ctx, repositories.GetThreadOwnedRequest{
		ID:         wait.ThreadID,
		TenantInfo: tenant,
	})
	if errortypes.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}

	active, err := s.turns.Active(ctx, repositories.ActiveAssistantTurnRequest{
		ThreadID:   thread.ID,
		UserID:     thread.UserID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}
	if active != nil {
		return agentwaitjobs.ErrConversationBusy
	}

	owner := serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    thread.UserID,
		UserID:         thread.UserID,
		OrganizationID: thread.OrganizationID,
		BusinessUnitID: thread.BusinessUnitID,
	}
	turn, err := s.turns.StartTurn(ctx, assistantturnservice.StartRequest{
		ThreadID:   thread.ID,
		UserID:     thread.UserID,
		TenantInfo: tenant,
		Origin:     conversation.AssistantTurnOriginWaitResolved,
	}, func(turn *conversation.AssistantTurn) (string, error) {
		run, startErr := assistantjobs.StartTurnWorkflow(ctx, s.workflows, turn,
			assistantjobs.TurnStart{
				Actor:   owner,
				Request: assistantjobs.AssistantTurnRequest{ResumeWaitID: wait.ID},
			})
		if startErr != nil {
			return "", startErr
		}

		return run.GetID(), nil
	})
	if errortypes.IsBusinessError(err) {
		return agentwaitjobs.ErrConversationBusy
	}
	if err != nil {
		return err
	}
	s.wakeCase(ctx, thread, tenant)

	return s.waits.MarkResumed(ctx, &repositories.MarkAgentWaitResumedRequest{
		ID:         wait.ID,
		TenantInfo: tenant,
		TurnID:     turn.ID,
	})
}

// wakeCase brings a snoozed case back when its wait picks it up: the person
// put it away until later, and what they were waiting for is later.
func (s *Service) wakeCase(
	ctx context.Context,
	thread *conversation.Thread,
	tenant pagination.TenantInfo,
) {
	if !thread.Snooze().Set() {
		return
	}
	if _, err := s.conversations.WakeThread(ctx, &repositories.WakeThreadRequest{
		ThreadID:   thread.ID,
		UserID:     thread.UserID,
		TenantInfo: tenant,
	}); err != nil {
		s.l.Warn("could not wake the case a wait picked up",
			zap.String("thread", thread.ID.String()), zap.Error(err))
	}
}

func (s *Service) resumeRun(ctx context.Context, wait *agentwait.Wait) error {
	tenant := tenantOf(wait)
	run, err := s.runService.StartForDefinition(
		ctx,
		&serviceports.StartAgentRunForDefinitionRequest{
			DefinitionID: wait.AgentDefinitionID,
			SubjectType:  wait.SubjectType,
			SubjectID:    wait.SubjectID,
			Trigger:      agent.RunTriggerWait,
			WaitID:       wait.ID,
			TenantInfo:   tenant,
		},
		nil,
	)
	if errors.Is(err, serviceports.ErrAgentRunAlreadyOpen) {
		return agentwaitjobs.ErrConversationBusy
	}
	if errortypes.IsBusinessError(err) {
		s.l.Info("a wait ended but its agent could not pick the work up",
			zap.String("wait", wait.ID.String()), zap.Error(err))
		return nil
	}
	if err != nil {
		return err
	}

	return s.waits.MarkResumed(ctx, &repositories.MarkAgentWaitResumedRequest{
		ID:         wait.ID,
		TenantInfo: tenant,
		RunID:      run.ID,
	})
}

// close ends a wait without picking anything up, for one that never started.
func (s *Service) close(
	ctx context.Context,
	wait *agentwait.Wait,
	status agentwait.Status,
	outcome string,
) {
	if _, _, err := s.waits.Resolve(
		context.WithoutCancel(ctx),
		&repositories.ResolveAgentWaitRequest{
			ID:         wait.ID,
			TenantInfo: tenantOf(wait),
			Status:     status,
			Outcome:    outcome,
			ResolvedAt: s.now(),
		},
	); err != nil {
		s.l.Error("a wait that could not start could not be closed",
			zap.String("wait", wait.ID.String()), zap.Error(err))
	}
}

func (s *Service) announce(ctx context.Context, wait *agentwait.Wait) {
	if s.realtime == nil || wait == nil || !wait.Conversational() {
		return
	}
	if err := s.realtime.PublishResourceInvalidation(
		context.WithoutCancel(ctx),
		&serviceports.PublishResourceInvalidationRequest{
			OrganizationID: wait.OrganizationID,
			BusinessUnitID: wait.BusinessUnitID,
			AudienceUserID: wait.UserID,
			Resource:       WaitsResource,
			Action:         "updated",
			RecordID:       wait.ThreadID,
			Entity:         map[string]string{"threadId": wait.ThreadID.String()},
		},
	); err != nil {
		s.l.Debug("could not announce a change to a conversation's waits",
			zap.String("wait", wait.ID.String()), zap.Error(err))
	}
}

func tenantOf(wait *agentwait.Wait) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: wait.OrganizationID, BuID: wait.BusinessUnitID}
}

func clip(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	return string(runes[:limit])
}

func durationText(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	minutes := ms / int64(time.Minute/time.Millisecond)

	return fmt.Sprintf("%dh %02dm", minutes/60, minutes%60)
}

func stopOf(move *shipment.ShipmentMove, stopID pulid.ID) *shipment.Stop {
	for _, stop := range move.Stops {
		if stop != nil && stop.ID == stopID {
			return stop
		}
	}

	return nil
}
