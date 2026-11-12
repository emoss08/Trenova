package agentrunclose

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// stoppedReason is what a run stopped while it waited on decisions records.
const stoppedReason = "The run was stopped while its proposals waited for a decision."

// settledBatch bounds the runs one sweep closes, so the statement and the
// announcements after it stay small; the hourly sweep takes the rest.
const settledBatch = 500

type Params struct {
	fx.In

	Runs        agentruntimeports.AgentRunRepository
	Proposals   agentproposalports.AgentProposalRepository
	Activity    agentactivityports.AgentActivityPublisher
	Reflections agentreflectionports.AgentReflectionScheduler
	Logger      *zap.Logger
}

type Closer struct {
	runs        agentruntimeports.AgentRunRepository
	proposals   agentproposalports.AgentProposalRepository
	activity    agentactivityports.AgentActivityPublisher
	reflections agentreflectionports.AgentReflectionScheduler
	l           *zap.Logger
}

var _ agentruntimeports.RunCloser = (*Closer)(nil)

func New(p Params) *Closer {
	return &Closer{
		runs:        p.Runs,
		proposals:   p.Proposals,
		activity:    p.Activity,
		reflections: p.Reflections,
		l:           p.Logger.Named("service.agentrunclose"),
	}
}

func (c *Closer) CloseIfDecided(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
) error {
	if runID.IsNil() {
		return nil
	}

	run, err := c.runs.CloseDecided(ctx, &agentruntimeports.CloseDecidedRunRequest{
		TenantInfo: tenantInfo,
		RunID:      runID,
		Now:        timeutils.NowUnix(),
	})
	if err != nil || run == nil {
		return err
	}

	c.announce(ctx, run)
	c.reflections.AfterRun(ctx, &agentreflectionports.ReflectOnRunRequest{
		TenantInfo: tenantInfo,
		RunID:      run.ID,
	})

	return nil
}

// CloseSettled completes the runs whose proposals all expired. An expired
// run is not reflected on: nobody decided anything it could learn from.
func (c *Closer) CloseSettled(ctx context.Context, now int64) (int, error) {
	runs, err := c.runs.CloseSettled(ctx, &agentruntimeports.CloseSettledRunsRequest{
		Now:   now,
		Limit: settledBatch,
	})
	if err != nil {
		return 0, err
	}

	for _, run := range runs {
		tenant := pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
		c.announce(dbscope.WithTenant(ctx, tenant.DBTenant()), run)
	}
	if len(runs) > 0 {
		c.l.Info("closed agent runs whose proposals all expired", zap.Int("closed", len(runs)))
	}

	return len(runs), nil
}

func (c *Closer) StopWaiting(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
) (bool, error) {
	run, err := c.runs.FailWaiting(ctx, &agentruntimeports.FailWaitingRunRequest{
		TenantInfo: tenantInfo,
		RunID:      runID,
		Reason:     stoppedReason,
		Now:        timeutils.NowUnix(),
	})
	if err != nil || run == nil {
		return false, err
	}

	if _, err = c.proposals.ExpirePendingByRun(ctx, agentproposalports.ExpireAgentProposalsByRunRequest{
		RunID:      runID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return true, err
	}
	c.announce(ctx, run)

	return true, nil
}

func (c *Closer) announce(ctx context.Context, run *agent.AgentRun) {
	c.activity.RunChanged(
		ctx,
		run,
		permissionports.SystemAuditActor(),
		agentactivityports.ActivityUpdated,
	)
}
