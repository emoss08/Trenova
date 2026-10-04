// Package conversationscheduleservice keeps the requests a person scheduled
// in a conversation, and starts the turn each one asks for when its time
// comes.
//
// A schedule is made by sending a message that opens with a cadence ("every
// weekday at 7:30, what's blocking the billing queue?") or with /schedule. The
// message is kept in the conversation as the schedule's card and is not
// answered there and then: the request is asked on its first slot, or at once
// with Run now. Answering it immediately as well would make the card's own
// message a question with a reply under it, and the first scheduled answer a
// repeat of it.
//
// Every run is a turn in the conversation the request was made in, asked as
// the conversation's owner with the access they have when it runs, the same
// way the application reports a decision there (assistantfollowupservice).
package conversationscheduleservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// announceTimeout bounds telling the person's tabs a schedule changed. It is
// a hint riding on a request or an activity that has its own work to finish.
const announceTimeout = 5 * time.Second

// Announcement actions.
const (
	actionCreated = "created"
	actionUpdated = "updated"
	actionDeleted = "deleted"
	actionRan     = "ran"
)

// ErrConversationBusy is a run that found its conversation already writing a
// reply. One turn at a time is the conversation's rule; a scheduled run waits
// for the one in progress rather than being lost.
var ErrConversationBusy = errors.New("the conversation is already working on a reply")

// Skip reasons, as FireConversationScheduleResult.Skipped says them.
const (
	SkipGone       = "deleted"
	SkipPaused     = "paused"
	SkipAlreadyRan = "already ran for this slot"
	SkipNoAccess   = "its owner may no longer ask the agent"
)

// turns is the part of the turn service a run needs.
type turns interface {
	StartTurn(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
	Active(
		ctx context.Context,
		req repositories.ActiveAssistantTurnRequest,
	) (*conversation.AssistantTurn, error)
}

// threads is the part of the conversation repository a schedule needs.
type threads interface {
	GetThread(ctx context.Context, req repositories.GetThreadRequest) (*conversation.Thread, error)
	GetThreadOwned(
		ctx context.Context,
		req repositories.GetThreadOwnedRequest,
	) (*conversation.Thread, error)
}

// clocks reads whose clock a schedule keeps: the person's, else their
// organization's.
type clocks interface {
	userTimezone(ctx context.Context, tenant pagination.TenantInfo, userID pulid.ID) string
	organizationTimezone(ctx context.Context, tenant pagination.TenantInfo) string
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Schedules     repositories.ConversationScheduleRepository
	Conversations repositories.ConversationRepository
	Definitions   repositories.AgentDefinitionRepository
	Users         repositories.UserRepository
	Organizations repositories.OrganizationRepository
	Permissions   serviceports.PermissionEngine
	Turns         *assistantturnservice.Service
	Workflows     serviceports.WorkflowStarter
	Syncer        serviceports.ConversationScheduleSyncer
	Realtime      serviceports.RealtimeService `optional:"true"`
	Plans         serviceports.PlanService     `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	schedules   repositories.ConversationScheduleRepository
	threads     threads
	definitions repositories.AgentDefinitionRepository
	permissions serviceports.PermissionEngine
	turns       turns
	workflows   serviceports.WorkflowStarter
	syncer      serviceports.ConversationScheduleSyncer
	realtime    serviceports.RealtimeService
	clocks      clocks
	plans       serviceports.PlanService
	now         func() int64
}

var _ serviceports.ConversationScheduleRunner = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.conversationschedule"),
		schedules:   p.Schedules,
		threads:     p.Conversations,
		definitions: p.Definitions,
		permissions: p.Permissions,
		turns:       p.Turns,
		workflows:   p.Workflows,
		syncer:      p.Syncer,
		realtime:    p.Realtime,
		clocks:      repositoryClocks{users: p.Users, organizations: p.Organizations},
		plans:       p.Plans,
		now:         timeutils.NowUnix,
	}
}

func (s *Service) requireAutomation(ctx context.Context, tenant pagination.TenantInfo) error {
	return planservice.RequireCapability(
		ctx,
		s.plans,
		tenant,
		platformplan.CapabilityAgentAutomation,
	)
}

// NewRunner is the same service, as what a firing schedule asks to run.
func NewRunner(s *Service) serviceports.ConversationScheduleRunner {
	return s
}

// CreateRequest schedules a message in one of the person's conversations.
type CreateRequest struct {
	ThreadID pulid.ID
	// Text is the message as sent: the cadence and the request.
	Text  string
	Actor serviceports.RequestActor
}

// CreateResult is the schedule and the message that draws its card.
type CreateResult struct {
	Schedule *conversationschedule.Schedule `json:"schedule"`
	Message  *conversation.Message          `json:"message"`
}

// Create reads a message as a scheduled request and keeps it: the schedule,
// its card in the conversation, and the Temporal schedule that fires it.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResult, error) {
	tenant := tenantOf(req.Actor)
	if err := s.requireAutomation(ctx, tenant); err != nil {
		return nil, err
	}

	text := strings.TrimSpace(req.Text)
	parsed, err := conversationschedule.ParseRequest(text)
	if err != nil {
		return nil, err
	}

	thread, err := s.threads.GetThread(ctx, repositories.GetThreadRequest{
		ID:         req.ThreadID,
		UserID:     req.Actor.UserID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	actor := req.Actor
	allowed, err := s.mayUseAgent(ctx, thread, &actor, tenant)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errortypes.NewBusinessError(
			"You can't schedule requests to this agent any more",
		)
	}
	if err = s.withinLimits(ctx, tenant, req.Actor.UserID, thread.ID); err != nil {
		return nil, err
	}

	schedule := &conversationschedule.Schedule{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ThreadID:       thread.ID,
		UserID:         req.Actor.UserID,
		Prompt:         parsed.Prompt,
		Cadence:        parsed.Cadence,
		CronExpression: parsed.CronExpression,
		Timezone:       s.timezoneFor(ctx, tenant, req.Actor.UserID),
		Enabled:        true,
	}
	schedule.NextRunAt = schedule.Next(s.now())

	multiErr := errortypes.NewMultiError()
	schedule.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, message, err := s.schedules.Create(ctx, repositories.CreateConversationScheduleRequest{
		Schedule: schedule,
		Message: conversation.Message{
			Role:    conversation.RoleUser,
			Kind:    conversation.MessageKindSchedule,
			Content: text,
		},
	})
	if err != nil {
		return nil, err
	}

	s.syncer.Sync(ctx, saved)
	s.announce(ctx, saved, actionCreated)

	return &CreateResult{Schedule: saved, Message: message}, nil
}

// withinLimits refuses a schedule past what one conversation or one person
// keeps.
func (s *Service) withinLimits(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID, threadID pulid.ID,
) error {
	inThread, err := s.schedules.Count(ctx, repositories.CountConversationSchedulesRequest{
		UserID:     userID,
		ThreadID:   threadID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}
	if inThread >= conversationschedule.MaxPerThread {
		return errortypes.NewBusinessError(
			"This conversation already has {0} schedules. Delete one to add another.",
			conversationschedule.MaxPerThread,
		)
	}

	total, err := s.schedules.Count(ctx, repositories.CountConversationSchedulesRequest{
		UserID:     userID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}
	if total >= conversationschedule.MaxPerUser {
		return errortypes.NewBusinessError(
			"You already have {0} schedules. Delete one to add another.",
			conversationschedule.MaxPerUser,
		)
	}

	return nil
}

// timezoneFor is the clock a person's schedule keeps: theirs, else their
// organization's, else UTC. "At 7:30" means 7:30 where they are.
func (s *Service) timezoneFor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) string {
	for _, zone := range []string{
		s.clocks.userTimezone(ctx, tenant, userID),
		s.clocks.organizationTimezone(ctx, tenant),
	} {
		zone = strings.TrimSpace(zone)
		if zone == "" {
			continue
		}
		if _, err := time.LoadLocation(zone); err == nil {
			return zone
		}
	}

	return conversationschedule.DefaultTimezone
}

// ListRequest pages through one person's schedules, in one conversation when
// ThreadID is set.
type ListRequest struct {
	ThreadID pulid.ID
	Actor    serviceports.RequestActor
	Limit    int
	Offset   int
}

func (s *Service) List(
	ctx context.Context,
	req ListRequest,
) (*pagination.ListResult[*conversationschedule.Schedule], error) {
	tenant := tenantOf(req.Actor)
	if req.ThreadID.IsNotNil() {
		// Reading a conversation's schedules is reading the conversation.
		if _, err := s.threads.GetThread(ctx, repositories.GetThreadRequest{
			ID:         req.ThreadID,
			UserID:     req.Actor.UserID,
			TenantInfo: tenant,
		}); err != nil {
			return nil, err
		}
	}

	return s.schedules.List(ctx, repositories.ListConversationSchedulesRequest{
		UserID:     req.Actor.UserID,
		ThreadID:   req.ThreadID,
		TenantInfo: tenant,
		Limit:      req.Limit,
		Offset:     req.Offset,
	})
}

// ScheduleRequest names one of the person's schedules.
type ScheduleRequest struct {
	ScheduleID pulid.ID
	Actor      serviceports.RequestActor
}

func (r ScheduleRequest) get() repositories.GetConversationScheduleRequest {
	return repositories.GetConversationScheduleRequest{
		ID:         r.ScheduleID,
		UserID:     r.Actor.UserID,
		TenantInfo: tenantOf(r.Actor),
	}
}

// SetEnabled pauses or resumes a schedule. Resuming works out the next slot
// from now: a slot that passed while it was paused is not run late.
func (s *Service) SetEnabled(
	ctx context.Context,
	req ScheduleRequest,
	enabled bool,
) (*conversationschedule.Schedule, error) {
	schedule, err := s.schedules.Get(ctx, req.get())
	if err != nil {
		return nil, err
	}
	if schedule.Enabled == enabled {
		return schedule, nil
	}
	if enabled {
		if err = s.requireAutomation(ctx, tenantOf(req.Actor)); err != nil {
			return nil, err
		}
	}

	schedule.Enabled = enabled
	if enabled {
		schedule.NextRunAt = schedule.Next(s.now())
	}

	updated, err := s.schedules.UpdateState(ctx, schedule)
	if err != nil {
		return nil, err
	}

	s.syncer.Sync(ctx, updated)
	s.announce(ctx, updated, actionUpdated)

	return updated, nil
}

// Delete removes a schedule. Its card stays in the conversation and says it
// was deleted.
func (s *Service) Delete(ctx context.Context, req ScheduleRequest) error {
	schedule, err := s.schedules.Get(ctx, req.get())
	if err != nil {
		return err
	}
	if err = s.schedules.Delete(ctx, req.get()); err != nil {
		return err
	}

	s.syncer.Remove(ctx, schedule.ID)
	s.announce(ctx, schedule, actionDeleted)

	return nil
}

// RunNow asks a schedule's request at once, paused or not, and returns the
// turn to watch. It is started here rather than by triggering the Temporal
// schedule, so the person gets the turn, or the reason there is none, in the
// answer to their click.
func (s *Service) RunNow(
	ctx context.Context,
	req ScheduleRequest,
) (*conversation.AssistantTurn, error) {
	schedule, err := s.schedules.Get(ctx, req.get())
	if err != nil {
		return nil, err
	}

	turn, skipped, err := s.run(ctx, schedule)
	if errors.Is(err, ErrConversationBusy) {
		return nil, errortypes.NewBusinessError(
			"This conversation is already working on a reply. Wait for it to finish, or stop it first.",
		)
	}
	if err != nil {
		return nil, err
	}
	if skipped != "" {
		return nil, errortypes.NewBusinessError("You can't ask this agent any more")
	}

	return turn, nil
}

// Fire starts the turn a schedule's slot asks for.
//
// A schedule deleted or paused since the slot was set does nothing, and one
// whose Temporal schedule outlived it is reported gone so the caller removes
// it. A retried attempt finds the run it already started and starts no
// second one. A conversation busy with another reply is ErrConversationBusy,
// for the caller to retry once that reply ends.
func (s *Service) Fire(
	ctx context.Context,
	req serviceports.FireConversationScheduleRequest,
) (*serviceports.FireConversationScheduleResult, error) {
	ctx = dbscope.WithValidTenant(ctx, req.TenantInfo.DBTenant())

	schedule, err := s.schedules.Get(ctx, repositories.GetConversationScheduleRequest{
		ID:         req.ScheduleID,
		TenantInfo: req.TenantInfo,
	})
	if errortypes.IsNotFoundError(err) {
		return &serviceports.FireConversationScheduleResult{Skipped: SkipGone, Gone: true}, nil
	}
	if err != nil {
		return nil, err
	}
	if !schedule.Enabled {
		return &serviceports.FireConversationScheduleResult{Skipped: SkipPaused}, nil
	}
	if schedule.LastRunAt != nil && req.FiredAt > 0 && *schedule.LastRunAt >= req.FiredAt {
		return &serviceports.FireConversationScheduleResult{
			Skipped: SkipAlreadyRan,
			TurnID:  schedule.LastTurnID,
		}, nil
	}

	turn, skipped, err := s.run(ctx, schedule)
	if err != nil {
		return nil, err
	}
	if skipped != "" {
		return &serviceports.FireConversationScheduleResult{Skipped: skipped}, nil
	}

	return &serviceports.FireConversationScheduleResult{TurnID: turn.ID}, nil
}

// run starts one turn of a schedule, as its conversation's owner, and records
// it. It returns why it started none when the owner may no longer ask the
// conversation's agent.
func (s *Service) run(
	ctx context.Context,
	schedule *conversationschedule.Schedule,
) (*conversation.AssistantTurn, string, error) {
	tenant := pagination.TenantInfo{
		OrgID:  schedule.OrganizationID,
		BuID:   schedule.BusinessUnitID,
		UserID: schedule.UserID,
	}
	thread, err := s.threads.GetThreadOwned(ctx, repositories.GetThreadOwnedRequest{
		ID:         schedule.ThreadID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, "", err
	}

	// The turn runs as the person whose conversation it is, with the access
	// they have now. One who lost the agent keeps the conversation to read
	// and the schedule to delete, but it asks nothing for them.
	owner := serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    thread.UserID,
		UserID:         thread.UserID,
		OrganizationID: thread.OrganizationID,
		BusinessUnitID: thread.BusinessUnitID,
	}
	allowed, err := s.mayUseAgent(ctx, thread, &owner, tenant)
	if err != nil {
		return nil, "", err
	}
	if !allowed {
		s.l.Info("scheduled run skipped: the owner may no longer use the agent",
			zap.String("schedule", schedule.ID.String()),
			zap.String("thread", thread.ID.String()),
		)
		return nil, SkipNoAccess, nil
	}

	active, err := s.turns.Active(ctx, repositories.ActiveAssistantTurnRequest{
		ThreadID:   thread.ID,
		UserID:     thread.UserID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, "", err
	}
	if active != nil {
		return nil, "", ErrConversationBusy
	}

	turn, err := s.turns.StartTurn(ctx, assistantturnservice.StartRequest{
		ThreadID:   thread.ID,
		UserID:     thread.UserID,
		TenantInfo: tenant,
		Origin:     conversation.AssistantTurnOriginScheduled,
		Input:      schedule.Prompt,
	}, func(turn *conversation.AssistantTurn) (string, error) {
		started, startErr := assistantjobs.StartTurnWorkflow(ctx, s.workflows, turn,
			assistantjobs.TurnStart{
				Actor:   owner,
				Content: schedule.Prompt,
			})
		if startErr != nil {
			return "", startErr
		}

		return started.GetID(), nil
	})
	if err != nil {
		return nil, "", err
	}

	now := s.now()
	next := schedule.Next(now)
	if err = s.schedules.RecordRun(ctx, repositories.RecordConversationScheduleRunRequest{
		ID:         schedule.ID,
		TenantInfo: tenant,
		RunAt:      now,
		NextRunAt:  next,
		TurnID:     turn.ID,
	}); err != nil {
		// The turn is under way and answers whatever this says. A schedule
		// card a run behind is corrected by the next one.
		s.l.Warn("a scheduled run started but could not be recorded",
			zap.String("schedule", schedule.ID.String()),
			zap.String("turn", turn.ID.String()),
			zap.Error(err),
		)
	}
	schedule.LastRunAt = &now
	schedule.NextRunAt = next
	schedule.LastTurnID = turn.ID
	s.announce(ctx, schedule, actionRan)

	return turn, "", nil
}

// mayUseAgent reports whether someone may ask the conversation's agent.
func (s *Service) mayUseAgent(
	ctx context.Context,
	thread *conversation.Thread,
	actor *serviceports.RequestActor,
	tenant pagination.TenantInfo,
) (bool, error) {
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: tenant,
	})
	if errortypes.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return s.permissions.MayUseAgent(ctx, actor, definition)
}

// announce tells the owner's tabs a schedule changed, so a card shows its
// state without being polled. It is best effort: the row is what is true.
func (s *Service) announce(
	ctx context.Context,
	schedule *conversationschedule.Schedule,
	action string,
) {
	if s.realtime == nil {
		return
	}

	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), announceTimeout)
	defer cancel()

	err := s.realtime.PublishResourceInvalidation(
		publishCtx,
		&serviceports.PublishResourceInvalidationRequest{
			OrganizationID: schedule.OrganizationID,
			BusinessUnitID: schedule.BusinessUnitID,
			AudienceUserID: schedule.UserID,
			Resource:       serviceports.ConversationSchedulesResource,
			Action:         action,
			RecordID:       schedule.ID,
			Entity: map[string]any{
				"id":       schedule.ID,
				"threadId": schedule.ThreadID,
				"userId":   schedule.UserID,
			},
		},
	)
	if err != nil {
		s.l.Warn("could not announce a conversation schedule",
			zap.String("schedule", schedule.ID.String()),
			zap.String("action", action),
			zap.Error(err),
		)
	}
}

func tenantOf(actor serviceports.RequestActor) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  actor.OrganizationID,
		BuID:   actor.BusinessUnitID,
		UserID: actor.UserID,
	}
}

// repositoryClocks reads timezones from the user and organization records.
type repositoryClocks struct {
	users         repositories.UserRepository
	organizations repositories.OrganizationRepository
}

func (c repositoryClocks) userTimezone(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) string {
	if c.users == nil {
		return ""
	}
	user, err := c.users.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   tenant,
		LookupUserID: userID,
	})
	if err != nil {
		return ""
	}

	return user.Timezone
}

func (c repositoryClocks) organizationTimezone(
	ctx context.Context,
	tenant pagination.TenantInfo,
) string {
	if c.organizations == nil {
		return ""
	}
	org, err := c.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		return ""
	}

	return org.Timezone
}
