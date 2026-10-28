package chaptercontract

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ThreadChaptersWorkflowName = "AssistantThreadChaptersWorkflow"
	TurnSignalName             = "chapter-turn"

	IdleWait    = 2 * time.Minute
	TurnsPerRun = 50
)

func WorkflowID(threadID pulid.ID) string {
	return "assistant-chapters:thread:" + threadID.String()
}

type ThreadChaptersInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ThreadID   pulid.ID              `json:"threadId"`
}

type SchedulerParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter `optional:"true"`
	Logger  *zap.Logger
}

type Scheduler struct {
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

func NewScheduler(p SchedulerParams) agentports.ChapterScheduler {
	return &Scheduler{signals: p.Signals, l: p.Logger.Named("job.assistant-chapters")}
}

func (s *Scheduler) Cue(ctx context.Context, req *agentports.ChapterTurnRequest) {
	if s.signals == nil || req == nil || req.ThreadID.IsNil() || req.TurnID.IsNil() {
		return
	}

	tenant := pagination.TenantInfo{OrgID: req.TenantInfo.OrgID, BuID: req.TenantInfo.BuID}
	signal := *req
	signal.TenantInfo = tenant
	if _, err := s.signals.SignalWithStartWorkflow(
		context.WithoutCancel(ctx),
		WorkflowID(req.ThreadID),
		TurnSignalName,
		&signal,
		client.StartWorkflowOptions{
			ID:            WorkflowID(req.ThreadID),
			TaskQueue:     temporaltype.TaskQueueAgentChat.String(),
			StaticSummary: "Mark and name the chapters of a conversation as it moves on",
			Priority: temporal.Priority{
				PriorityKey: agentflowcontract.PriorityOneShot,
				FairnessKey: tenant.OrgID.String(),
			},
		},
		ThreadChaptersWorkflowName,
		&ThreadChaptersInput{TenantInfo: tenant, ThreadID: req.ThreadID},
	); err != nil {
		s.l.Warn("could not cue the chapters of a conversation",
			zap.String("thread", req.ThreadID.String()),
			zap.String("turn", req.TurnID.String()),
			zap.String("phase", string(req.Phase)),
			zap.Error(err),
		)
	}
}
