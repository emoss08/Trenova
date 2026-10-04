package reflectionjobs

import (
	"context"
	"errors"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SchedulerParams struct {
	fx.In

	Logger    *zap.Logger
	Signals   serviceports.WorkflowSignalStarter `optional:"true"`
	Workflows serviceports.WorkflowStarter       `optional:"true"`
}

type Scheduler struct {
	l         *zap.Logger
	signals   serviceports.WorkflowSignalStarter
	workflows serviceports.WorkflowStarter
}

func NewScheduler(p SchedulerParams) serviceports.AgentReflectionScheduler {
	return &Scheduler{
		l:         p.Logger.Named("job.agent-reflection"),
		signals:   p.Signals,
		workflows: p.Workflows,
	}
}

func (s *Scheduler) AfterTurn(ctx context.Context, req *serviceports.ReflectOnThreadRequest) {
	if s.signals == nil || req == nil || req.ThreadID.IsNil() {
		return
	}

	tenant := tenantOnly(req.TenantInfo)
	if _, err := s.signals.SignalWithStartWorkflow(
		ctx,
		ThreadWorkflowID(req.ThreadID),
		TurnFinishedSignalName,
		TurnFinished{TurnID: req.TurnID},
		client.StartWorkflowOptions{
			ID:            ThreadWorkflowID(req.ThreadID),
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			StaticSummary: "Look back over a conversation once it goes quiet",
			Priority:      priorityFor(tenant),
		},
		ThreadReflectionWorkflowName,
		&ThreadReflectionInput{
			TenantInfo: tenant,
			ThreadID:   req.ThreadID,
			UserID:     req.UserID,
		},
	); err != nil {
		s.l.Warn("could not cue a look back over a conversation",
			zap.String("thread", req.ThreadID.String()),
			zap.Error(err),
		)
	}
}

func (s *Scheduler) AfterRun(ctx context.Context, req *serviceports.ReflectOnRunRequest) {
	if s.workflows == nil || req == nil || req.RunID.IsNil() {
		return
	}

	tenant := tenantOnly(req.TenantInfo)
	_, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    RunWorkflowID(req.RunID),
		TaskQueue:             temporaltype.TaskQueueAgentBackground.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		StaticSummary:         "Look back over a settled agent run",
		Priority:              priorityFor(tenant),
	}, RunReflectionWorkflowName, &RunReflectionInput{
		TenantInfo: tenant,
		RunID:      req.RunID,
	})
	if err == nil {
		return
	}

	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &started) {
		return
	}
	s.l.Warn("could not cue a look back over a run",
		zap.String("run", req.RunID.String()),
		zap.Error(err),
	)
}

func tenantOnly(tenant pagination.TenantInfo) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID}
}

func priorityFor(tenant pagination.TenantInfo) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflow.PriorityBackground,
		FairnessKey: tenant.OrgID.String(),
	}
}
