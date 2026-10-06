package conversationschedulejobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// scheduleIDPrefix names the Temporal schedule behind each conversation
	// schedule. The static schedule registry leaves these alone: they carry
	// their own owner in their memo.
	scheduleIDPrefix = "conversation-schedule/"

	// scheduleOwner is the memo marker that tells the static registry these
	// schedules are not its to delete.
	scheduleOwner = "conversation-schedules"

	// catchupWindow is how late a slot may still start after the scheduler
	// could not start it on time. A short outage still gets its run; a long
	// one does not post yesterday's answer this afternoon.
	catchupWindow = 5 * time.Minute

	// reconcilePageSize bounds one read of schedules while reconciling.
	reconcilePageSize = 200

	// callTimeout bounds one call to the schedule API made on behalf of a
	// person who is waiting on it.
	callTimeout = 10 * time.Second

	// noteLength bounds the request quoted in the schedule's note.
	noteLength = 120
)

// ScheduleID names the Temporal schedule behind a conversation schedule.
func ScheduleID(scheduleID pulid.ID) string {
	return scheduleIDPrefix + scheduleID.String()
}

type SchedulesParams struct {
	fx.In

	Client    client.Client
	Schedules repositories.ConversationScheduleRepository
	Logger    *zap.Logger
}

// Schedules keeps one Temporal Schedule behind every conversation schedule,
// the way agentjobs.DefinitionSchedules does for scheduled agents.
type Schedules struct {
	client    client.Client
	schedules repositories.ConversationScheduleRepository
	l         *zap.Logger
}

var _ serviceports.ConversationScheduleSyncer = (*Schedules)(nil)

func NewSchedules(p SchedulesParams) *Schedules {
	return &Schedules{
		client:    p.Client,
		schedules: p.Schedules,
		l:         p.Logger.Named("conversation-schedules"),
	}
}

// Sync brings one schedule's Temporal schedule in line with it, as it is
// saved. It never fails the save: the hourly reconcile repairs a schedule a
// failed call left behind.
func (s *Schedules) Sync(ctx context.Context, sched *conversationschedule.Schedule) {
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), callTimeout)
	defer cancel()

	if err := s.sync(callCtx, sched); err != nil {
		s.l.Warn("a conversation schedule could not be updated; the next reconcile will",
			zap.String("schedule", sched.ID.String()),
			zap.Error(err),
		)
	}
}

// Remove deletes a deleted schedule's Temporal schedule.
func (s *Schedules) Remove(ctx context.Context, scheduleID pulid.ID) {
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), callTimeout)
	defer cancel()

	if err := s.remove(callCtx, ScheduleID(scheduleID)); err != nil {
		s.l.Warn("a deleted conversation schedule could not be removed; the next reconcile will",
			zap.String("schedule", scheduleID.String()),
			zap.Error(err),
		)
	}
}

func (s *Schedules) sync(ctx context.Context, sched *conversationschedule.Schedule) error {
	if s.client == nil {
		return errors.New("temporal client is not configured")
	}

	options := optionsFor(sched)
	_, err := s.client.ScheduleClient().Create(ctx, *options)
	if err == nil {
		return nil
	}
	if !scheduleExists(err) {
		return fmt.Errorf("create schedule %s: %w", options.ID, err)
	}

	return s.client.ScheduleClient().GetHandle(ctx, options.ID).Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			desired := input.Description.Schedule
			desired.Spec = &options.Spec
			desired.Action = options.Action
			if desired.Policy == nil {
				desired.Policy = &client.SchedulePolicies{}
			}
			desired.Policy.Overlap = options.Overlap
			desired.Policy.CatchupWindow = options.CatchupWindow
			if desired.State == nil {
				desired.State = &client.ScheduleState{}
			}
			desired.State.Paused = options.Paused
			desired.State.Note = options.Note

			return &client.ScheduleUpdate{Schedule: &desired}, nil
		},
	})
}

func (s *Schedules) remove(ctx context.Context, id string) error {
	if s.client == nil {
		return errors.New("temporal client is not configured")
	}

	err := s.client.ScheduleClient().GetHandle(ctx, id).Delete(ctx)
	var notFound *serviceerror.NotFound
	if err != nil && !errors.As(err, &notFound) {
		return fmt.Errorf("delete schedule %s: %w", id, err)
	}

	return nil
}

// optionsFor is the Temporal schedule a conversation schedule runs on.
//
// Each slot starts a short workflow that asks the service to start the turn;
// the turn itself is its own workflow, so the schedule's workflow ends as soon
// as the turn is handed over. Overlap is skipped: a slot whose previous run
// is still waiting on a busy conversation has nothing to add to it.
func optionsFor(sched *conversationschedule.Schedule) *client.ScheduleOptions {
	timezone := strings.TrimSpace(sched.Timezone)
	if timezone == "" {
		timezone = conversationschedule.DefaultTimezone
	}

	return &client.ScheduleOptions{
		ID: ScheduleID(sched.ID),
		Spec: client.ScheduleSpec{
			CronExpressions: []string{sched.CronExpression},
			TimeZoneName:    timezone,
		},
		Overlap:       enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		CatchupWindow: catchupWindow,
		Paused:        !sched.Enabled,
		Note:          "Asks " + quoted(sched.Prompt) + " · " + sched.Cadence,
		Memo:          map[string]any{schedule.ManagedByMemoKey: scheduleOwner},
		Action: &client.ScheduleWorkflowAction{
			ID:        "conversation-schedule-run/" + sched.ID.String(),
			Workflow:  RunWorkflowName,
			TaskQueue: temporaltype.TaskQueueSystem.String(),
			Args: []any{&RunPayload{
				BasePayload: temporaltype.BasePayload{
					OrganizationID: sched.OrganizationID,
					BusinessUnitID: sched.BusinessUnitID,
					UserID:         sched.UserID,
				},
				ScheduleID: sched.ID,
			}},
		},
	}
}

func quoted(prompt string) string {
	if utf8.RuneCountInString(prompt) > noteLength {
		prompt = string([]rune(prompt)[:noteLength]) + "…"
	}

	return "“" + prompt + "”"
}

func scheduleExists(err error) bool {
	if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return true
	}

	var exists *serviceerror.AlreadyExists

	return errors.As(err, &exists)
}

// Reconcile makes the Temporal schedules match the rows: every schedule has
// one, as it is, and none outlives its row. It repairs a schedule a failed
// save left behind and removes the schedules of conversations deleted with
// their schedules.
func (s *Schedules) Reconcile(ctx context.Context) (*ReconcileResult, error) {
	if s.client == nil {
		return nil, errors.New("temporal client is not configured")
	}

	result := &ReconcileResult{}
	wanted := make(map[string]struct{})

	after := pulid.Nil
	for {
		page, err := s.schedules.ListAcrossTenants(
			ctx,
			repositories.ListConversationSchedulesAcrossTenantsRequest{
				AfterID: after,
				Limit:   reconcilePageSize,
			},
		)
		if err != nil {
			return result, fmt.Errorf("list conversation schedules: %w", err)
		}

		for _, sched := range page {
			wanted[ScheduleID(sched.ID)] = struct{}{}
			if err = s.sync(ctx, sched); err != nil {
				result.Failed++
				s.l.Warn("a conversation schedule could not be reconciled",
					zap.String("schedule", sched.ID.String()),
					zap.Error(err),
				)
				continue
			}
			result.Synced++
		}

		if len(page) < reconcilePageSize {
			break
		}
		after = page[len(page)-1].ID
	}

	iter, err := s.client.ScheduleClient().List(ctx, client.ScheduleListOptions{PageSize: 100})
	if err != nil {
		return result, fmt.Errorf("list schedules: %w", err)
	}
	for iter.HasNext() {
		entry, nextErr := iter.Next()
		if nextErr != nil {
			return result, fmt.Errorf("iterate schedules: %w", nextErr)
		}
		if !strings.HasPrefix(entry.ID, scheduleIDPrefix) {
			continue
		}
		if _, keep := wanted[entry.ID]; keep {
			continue
		}
		if err = s.remove(ctx, entry.ID); err != nil {
			result.Failed++
			s.l.Warn("an orphaned conversation schedule could not be removed",
				zap.String("schedule", entry.ID),
				zap.Error(err),
			)
			continue
		}
		result.Removed++
	}

	return result, nil
}
