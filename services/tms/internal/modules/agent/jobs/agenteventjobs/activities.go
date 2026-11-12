package agenteventjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agenteventjobs/agenteventcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// maxStartsPerPage bounds one page however large an agent's cap.
	maxStartsPerPage = 20
	// startRetryAfter is how soon a queued event comes back: one deferred
	// for room, or one whose start failed for a reason that may pass.
	startRetryAfter = 15 * time.Second
)

type ActivitiesParams struct {
	fx.In

	Queue       workqueueports.Queue
	Definitions agentdefinitionports.AgentDefinitionRepository
	Runs        agentruntimeports.AgentRunService
	RunRepo     agentruntimeports.AgentRunRepository
	Logger      *zap.Logger
}

type Activities struct {
	queue       workqueueports.Queue
	definitions agentdefinitionports.AgentDefinitionRepository
	runs        agentruntimeports.AgentRunService
	runRepo     agentruntimeports.AgentRunRepository
	l           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		queue:       p.Queue,
		definitions: p.Definitions,
		runs:        p.Runs,
		runRepo:     p.RunRepo,
		l:           p.Logger.Named("job.agent-event-runs"),
	}
}

// StartEventRunsActivity starts as many of an agent's queued event runs as it
// has room for: its cap less the runs still working. With no room it claims
// nothing and says the drain is blocked, so the drain comes back for them.
func (a *Activities) StartEventRunsActivity(
	ctx context.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainPage, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	definitionID, ok := agenteventcontract.DefinitionOf(payload.Kind)
	if !ok {
		return &temporaljobs.DrainPage{}, nil
	}

	room, err := a.room(ctx, tenant, definitionID)
	if err != nil {
		return nil, err
	}
	if room == 0 {
		return &temporaljobs.DrainPage{Blocked: true}, nil
	}

	return temporaljobs.DrainDue(ctx, &temporaljobs.DrainDueParams{
		Queue:      a.queue,
		Kind:       payload.Kind,
		Tenant:     tenant,
		PageSize:   room,
		RetryAfter: startRetryAfter,
		Handle: func(ctx context.Context, subjects []pulid.ID, kinds map[pulid.ID]string) map[pulid.ID]error {
			failed := make(map[pulid.ID]error)
			for _, subjectID := range subjects {
				if sErr := a.start(
					ctx,
					tenant,
					definitionID,
					subjectID,
					kinds[subjectID],
				); sErr != nil {
					failed[subjectID] = sErr
				}
			}

			return failed
		},
	})
}

// room is how many more runs the agent may start: its cap less those still
// working. A run waiting on a person's decision has finished its work and
// takes no room. A deleted agent's queue is let through so start spends its
// events, as an off agent's are spent: neither starts a run.
func (a *Activities) room(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
) (int, error) {
	definition, err := a.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         definitionID,
			TenantInfo: tenant,
		},
	)
	if errortypes.IsNotFoundError(err) {
		return maxStartsPerPage, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read the agent: %w", err)
	}

	working, err := a.runRepo.CountOpen(ctx, &agentruntimeports.CountOpenAgentRunsRequest{
		TenantInfo:   tenant,
		DefinitionID: definitionID,
		WorkingOnly:  true,
	})
	if err != nil {
		return 0, fmt.Errorf("count the agent's working runs: %w", err)
	}

	return min(max(definition.MaxConcurrentRuns-working, 0), maxStartsPerPage), nil
}

// start starts one queued run. A subject whose run is still open, an agent
// that is off or past its budget, and a plan that leaves automation out are
// settled: the event is spent as it would have been when it arrived. Anything
// else is tried again.
func (a *Activities) start(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID, subjectID pulid.ID,
	rawKind string,
) error {
	kind := agenttrigger.EventKind(rawKind)
	_, err := a.runs.StartForDefinition(ctx, &agentruntimeports.StartAgentRunForDefinitionRequest{
		DefinitionID: definitionID,
		SubjectType:  kind.SubjectType(),
		SubjectID:    subjectID,
		Trigger:      agent.RunTriggerEvent,
		EventKind:    kind,
		TenantInfo:   tenant,
	}, nil)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, agentruntimeports.ErrAgentRunAlreadyOpen):
		return nil
	case errortypes.IsNotFoundError(err), errortypes.IsBusinessError(err),
		errortypes.IsPlanRestrictionError(err):
		a.l.Info("an agent's queued event did not start a run",
			zap.String("definitionId", definitionID.String()),
			zap.String("subjectId", subjectID.String()),
			zap.Error(err))

		return nil
	default:
		return err
	}
}
