package completionjobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DispatcherParams struct {
	fx.In

	Workflows serviceports.WorkflowStarter
	Logger    *zap.Logger
}

// Dispatcher hands a one-shot call to a worker and waits for its answer, for
// the request a person is waiting on.
type Dispatcher struct {
	workflows serviceports.WorkflowStarter
	l         *zap.Logger
}

var (
	_ serviceports.StructuredCompleter = (*Dispatcher)(nil)
	_ serviceports.AIProviderTester    = (*Dispatcher)(nil)
	_ serviceports.BriefingDayWriter   = (*Dispatcher)(nil)
)

func NewDispatcher(p DispatcherParams) *Dispatcher {
	return &Dispatcher{workflows: p.Workflows, l: p.Logger.Named("completion-dispatcher")}
}

// CompleteStructured asks the model one structured question on a worker.
func (d *Dispatcher) CompleteStructured(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	var result serviceports.StructuredCompletionResult
	if err := d.await(ctx, call{
		id:           "structured-completion/" + pulid.MustNew("scmp_").String(),
		workflow:     StructuredCompletionWorkflowName,
		summary:      "Ask the model: " + string(req.Task),
		organization: req.TenantInfo.OrgID,
		exclusive:    true,
		payload:      &StructuredCompletionPayload{Request: req},
	}, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Test probes a provider on a worker. Two tests of one provider at once share
// the one probe.
func (d *Dispatcher) Test(
	ctx context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*serviceports.TestAIProviderResult, error) {
	var result serviceports.TestAIProviderResult
	if err := d.await(ctx, call{
		id:           "ai-provider-test/" + req.ID.String(),
		workflow:     TestAIProviderWorkflowName,
		summary:      "Test an AI provider",
		organization: req.TenantInfo.OrgID,
		payload:      &TestAIProviderPayload{Request: req},
	}, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// WriteForDay writes a day's briefing on a worker. Two requests to rewrite the
// same page at once share the one write.
func (d *Dispatcher) WriteForDay(
	ctx context.Context,
	req serviceports.WriteBriefingRequest,
) (*serviceports.WriteBriefingResult, error) {
	var result serviceports.WriteBriefingResult
	if err := d.await(ctx, call{
		id:           briefingWriteID(req),
		workflow:     WriteBriefingWorkflowName,
		summary:      "Write the briefing",
		organization: req.TenantInfo.OrgID,
		payload:      &WriteBriefingPayload{Request: req},
	}, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func briefingWriteID(req serviceports.WriteBriefingRequest) string {
	roles := make([]string, 0, len(req.Roles))
	for _, role := range req.Roles {
		roles = append(roles, string(role))
	}
	if len(roles) == 0 {
		roles = append(roles, "all")
	}
	day := req.BriefingDate
	if day == "" {
		day = "today"
	}

	return fmt.Sprintf(
		"briefing-write/%s/%s/%s",
		req.TenantInfo.OrgID,
		strings.Join(roles, ","),
		day,
	)
}

// call is one workflow a request waits on.
type call struct {
	id           string
	workflow     string
	summary      string
	organization pulid.ID
	// exclusive says the call is this request's alone, so a request that
	// stops waiting cancels it. A shared call is left to whoever else waits.
	exclusive bool
	payload   any
}

func (d *Dispatcher) await(ctx context.Context, c call, result any) error {
	run, err := d.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       c.id,
		TaskQueue:                temporaltype.TaskQueueAgentChat.String(),
		WorkflowExecutionTimeout: waitFor(ctx),
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		StaticSummary:            c.summary,
		Priority: temporal.Priority{
			PriorityKey: agentflow.PriorityOneShot,
			FairnessKey: c.organization.String(),
		},
	}, c.workflow, c.payload)
	if err != nil {
		return fmt.Errorf("hand the call to a worker: %w", err)
	}

	if err = run.Get(ctx, result); err != nil {
		if ctx.Err() != nil && c.exclusive {
			d.abandon(ctx, run)
		}

		return modelcall.Err(err)
	}

	return nil
}

// abandon cancels a call its caller stopped waiting for, so a person who
// navigated away does not keep paying for the answer.
func (d *Dispatcher) abandon(ctx context.Context, run client.WorkflowRun) {
	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), abandonTimeout)
	defer cancel()

	if err := d.workflows.CancelWorkflow(cancelCtx, run.GetID(), run.GetRunID()); err != nil {
		d.l.Warn("a call nobody is waiting for could not be cancelled",
			zap.String("workflow", run.GetID()),
			zap.Error(err),
		)
	}
}

// waitFor is how long the caller can wait: up to its own deadline, less the
// margin it needs to answer.
func waitFor(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return defaultWait
	}

	return max(time.Until(deadline)-answerMargin, minWait)
}
