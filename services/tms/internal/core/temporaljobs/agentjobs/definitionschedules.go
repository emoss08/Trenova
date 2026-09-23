package agentjobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
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
	// definitionScheduleIDPrefix names the schedule behind each scheduled or
	// continuous agent. The static schedule registry leaves these alone: they
	// carry their own owner in their memo.
	definitionScheduleIDPrefix = "agent-definition/"

	// definitionScheduleOwner is the memo marker that tells the static
	// registry these schedules are not its to delete.
	definitionScheduleOwner = "agent-definitions"

	// scheduleCatchupWindow is how late a slot may still start after the
	// scheduler could not start it on time, a Temporal outage most often. A
	// short outage still gets its run; a long one does not come back as a
	// burst of stale runs.
	scheduleCatchupWindow = 5 * time.Minute

	// reconcilePageSize bounds one read of definitions while reconciling.
	reconcilePageSize = 200

	// scheduleCallTimeout bounds one call to the schedule API made on behalf
	// of a person saving an agent, who is waiting on it.
	scheduleCallTimeout = 10 * time.Second
)

// DefinitionScheduleID names the schedule behind a definition.
func DefinitionScheduleID(definitionID pulid.ID) string {
	return definitionScheduleIDPrefix + definitionID.String()
}

type DefinitionSchedulesParams struct {
	fx.In

	Client      client.Client
	Definitions repositories.AgentDefinitionRepository
	Logger      *zap.Logger
}

// DefinitionSchedules keeps one Temporal Schedule behind every scheduled or
// continuous agent, so Temporal decides when each one runs.
//
// It replaces a sweep that woke every minute and polled every tenant for the
// agents whose next slot had passed. A schedule fires on its slot, in the
// agent's own timezone, stops at the agent's end date, and is paused while the
// agent is disabled, and the Temporal UI shows each one with its next run.
type DefinitionSchedules struct {
	client      client.Client
	definitions repositories.AgentDefinitionRepository
	l           *zap.Logger
}

var _ serviceports.AgentDefinitionScheduler = (*DefinitionSchedules)(nil)

func NewDefinitionSchedules(p DefinitionSchedulesParams) *DefinitionSchedules {
	return &DefinitionSchedules{
		client:      p.Client,
		definitions: p.Definitions,
		l:           p.Logger.Named("agent-definition-schedules"),
	}
}

// Scheduled reports whether an agent runs on a schedule of its own.
func Scheduled(definition *agentdefinition.Definition) bool {
	return definition.TriggerMode == agentdefinition.TriggerScheduled ||
		definition.TriggerMode == agentdefinition.TriggerContinuous
}

// Sync brings one agent's schedule in line with the agent, as it is saved. It
// never fails the save: the reconcile that runs every quarter hour repairs a
// schedule a failed call left behind.
func (s *DefinitionSchedules) Sync(ctx context.Context, definition *agentdefinition.Definition) {
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), scheduleCallTimeout)
	defer cancel()

	if err := s.sync(callCtx, definition); err != nil {
		s.l.Warn("an agent's schedule could not be updated; the next reconcile will",
			zap.String("definition", definition.ID.String()),
			zap.Error(err),
		)
	}
}

// Remove deletes a deleted agent's schedule.
func (s *DefinitionSchedules) Remove(ctx context.Context, definitionID pulid.ID) {
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), scheduleCallTimeout)
	defer cancel()

	if err := s.remove(callCtx, DefinitionScheduleID(definitionID)); err != nil {
		s.l.Warn("a deleted agent's schedule could not be removed; the next reconcile will",
			zap.String("definition", definitionID.String()),
			zap.Error(err),
		)
	}
}

func (s *DefinitionSchedules) sync(
	ctx context.Context,
	definition *agentdefinition.Definition,
) error {
	id := DefinitionScheduleID(definition.ID)
	if !Scheduled(definition) {
		return s.remove(ctx, id)
	}

	options, err := scheduleOptionsFor(definition)
	if err != nil {
		return err
	}

	_, err = s.client.ScheduleClient().Create(ctx, *options)
	if err == nil {
		return nil
	}
	if !scheduleExists(err) {
		return fmt.Errorf("create schedule %s: %w", id, err)
	}

	return s.client.ScheduleClient().GetHandle(ctx, id).Update(ctx, client.ScheduleUpdateOptions{
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

func (s *DefinitionSchedules) remove(ctx context.Context, id string) error {
	err := s.client.ScheduleClient().GetHandle(ctx, id).Delete(ctx)
	var notFound *serviceerror.NotFound
	if err != nil && !errors.As(err, &notFound) {
		return fmt.Errorf("delete schedule %s: %w", id, err)
	}

	return nil
}

// scheduleOptionsFor is the schedule an agent runs on.
//
// Each firing starts a short workflow that checks the agent may run now — it is
// still enabled, within its budget, under its concurrency limit — and starts
// the run. Overlap is skipped because that check is where concurrency is
// decided; the schedule only says when to ask.
func scheduleOptionsFor(definition *agentdefinition.Definition) (*client.ScheduleOptions, error) {
	spec := client.ScheduleSpec{}
	switch definition.TriggerMode {
	case agentdefinition.TriggerScheduled:
		timezone := strings.TrimSpace(definition.CronTimezone)
		if timezone == "" {
			timezone = agentdefinition.DefaultCronTimezone
		}
		spec.CronExpressions = []string{definition.CronExpression}
		spec.TimeZoneName = timezone
	case agentdefinition.TriggerContinuous:
		if definition.IntervalSeconds <= 0 {
			return nil, fmt.Errorf("agent %s has no interval to run on", definition.ID)
		}
		spec.Intervals = []client.ScheduleIntervalSpec{{
			Every: time.Duration(definition.IntervalSeconds) * time.Second,
		}}
	default:
		return nil, fmt.Errorf("agent %s does not run on a schedule", definition.ID)
	}
	if definition.EndsAt != nil {
		spec.EndAt = time.Unix(*definition.EndsAt, 0)
	}

	return &client.ScheduleOptions{
		ID:            DefinitionScheduleID(definition.ID),
		Spec:          spec,
		Overlap:       enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		CatchupWindow: scheduleCatchupWindow,
		Paused:        !definition.Enabled,
		Note:          "Runs " + definition.Name,
		Memo:          map[string]any{schedule.ManagedByMemoKey: definitionScheduleOwner},
		Action: &client.ScheduleWorkflowAction{
			ID:        "agent-scheduled/" + definition.ID.String(),
			Workflow:  AgentScheduledRunWorkflowName,
			TaskQueue: temporaltype.TaskQueueAgentBackground.String(),
			Args: []any{&ScheduledRunPayload{
				DefinitionID:   definition.ID,
				OrganizationID: definition.OrganizationID,
				BusinessUnitID: definition.BusinessUnitID,
			}},
			Priority: temporal.Priority{
				PriorityKey: agentflow.PriorityBackground,
				FairnessKey: definition.OrganizationID.String(),
			},
		},
	}, nil
}

func scheduleExists(err error) bool {
	if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return true
	}

	var exists *serviceerror.AlreadyExists

	return errors.As(err, &exists)
}

// ReconcileResult is what one reconcile changed.
type ReconcileResult struct {
	Synced  int `json:"synced"`
	Removed int `json:"removed"`
	Failed  int `json:"failed"`
}

// Reconcile makes the schedules match the agents: every scheduled or
// continuous agent has one, as it is, and no schedule outlives its agent. It
// is what backfills the schedules of agents saved before this existed, and
// what repairs a schedule a failed save left behind.
func (s *DefinitionSchedules) Reconcile(ctx context.Context) (*ReconcileResult, error) {
	result := &ReconcileResult{}
	wanted := make(map[string]struct{})

	after := pulid.Nil
	for {
		page, err := s.definitions.ListScheduledAcrossTenants(
			ctx,
			repositories.ListScheduledAcrossTenantsRequest{
				AfterID: after,
				Limit:   reconcilePageSize,
			},
		)
		if err != nil {
			return result, fmt.Errorf("list scheduled agents: %w", err)
		}

		for _, definition := range page {
			wanted[DefinitionScheduleID(definition.ID)] = struct{}{}
			if err = s.sync(ctx, definition); err != nil {
				result.Failed++
				s.l.Warn("an agent's schedule could not be reconciled",
					zap.String("definition", definition.ID.String()),
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
		if !strings.HasPrefix(entry.ID, definitionScheduleIDPrefix) {
			continue
		}
		if _, keep := wanted[entry.ID]; keep {
			continue
		}
		if err = s.remove(ctx, entry.ID); err != nil {
			result.Failed++
			s.l.Warn("an orphaned agent schedule could not be removed",
				zap.String("schedule", entry.ID),
				zap.Error(err),
			)
			continue
		}
		result.Removed++
	}

	return result, nil
}
