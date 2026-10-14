// Package agentprofilecontract is how the API asks a worker to write an agent
// profile again: the request workflow's name, id and signal, and the scheduler
// that starts or joins it.
package agentprofilecontract

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RequestWorkflowName = "AgentProfileRequestWorkflow"
	RequestedSignalName = "profile-requested"

	// GuidanceQuietPeriod is how long a request waits for more guidance before
	// writing, so several lines given together cost one write; LongestWait
	// bounds it however steadily guidance keeps coming.
	GuidanceQuietPeriod = 2 * time.Minute
	LongestWait         = 10 * time.Minute
	RoundsPerRun        = 20
)

var (
	errNoWorkflowClient = errors.New("no workflow client is available to write profiles on request")
	errRequestUnnamed   = errors.New("a profile request names a profile")
)

// RequestWorkflowID is one per profile, so asking twice while a write waits
// joins the one already waiting.
func RequestWorkflowID(ref agentprofileports.SubjectRef) string {
	id := "agent-profile:" + ref.OrganizationID.String() + ":" + ref.BusinessUnitID.String()
	if ref.Subject == agentprofile.SubjectUser {
		return id + ":user:" + ref.PersonID.String()
	}

	return id + ":organization"
}

// RequestInput starts a request workflow; the signal carries a
// GenerateRequest.
type RequestInput struct {
	Request agentprofileports.GenerateRequest `json:"request"`
	Rounds  int                               `json:"rounds"`
}

// Priority keeps profile work behind people's turns and fair across
// organizations.
func Priority(tenantInfo pagination.TenantInfo) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflowcontract.PriorityBackground,
		FairnessKey: tenantInfo.OrgID.String(),
	}
}

type SchedulerParams struct {
	fx.In

	Logger  *zap.Logger
	Signals kernelports.WorkflowSignalStarter `optional:"true"`
}

type Scheduler struct {
	l       *zap.Logger
	signals kernelports.WorkflowSignalStarter
}

func NewScheduler(p SchedulerParams) agentprofileports.Scheduler {
	return &Scheduler{
		l:       p.Logger.Named("job.agent-profile-scheduler"),
		signals: p.Signals,
	}
}

// Request starts the profile's request workflow, or joins the one already
// waiting, so asking twice while a write waits costs one write.
func (s *Scheduler) Request(ctx context.Context, req *agentprofileports.GenerateRequest) error {
	if s.signals == nil {
		return errNoWorkflowClient
	}
	if req == nil || !req.Ref.Valid() {
		return errRequestUnnamed
	}

	request := *req
	id := RequestWorkflowID(request.Ref)
	if _, err := s.signals.SignalWithStartWorkflow(
		ctx,
		id,
		RequestedSignalName,
		request,
		client.StartWorkflowOptions{
			ID:            id,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			StaticSummary: "Write an agent profile again",
			Priority:      Priority(request.Ref.Tenant()),
		},
		RequestWorkflowName,
		&RequestInput{Request: request},
	); err != nil {
		s.l.Warn("could not ask for an agent profile to be written",
			zap.String("organization", request.Ref.OrganizationID.String()),
			zap.String("subject", string(request.Ref.Subject)),
			zap.Error(err),
		)

		return err
	}

	return nil
}
