package completioncontract

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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

	Workflows kernelports.WorkflowStarter
	Logger    *zap.Logger
}

// Dispatcher hands a one-shot call to a worker and waits for its answer, for
// the request a person is waiting on.
type Dispatcher struct {
	workflows kernelports.WorkflowStarter
	l         *zap.Logger
}

var (
	_ agentports.StructuredCompleter  = (*Dispatcher)(nil)
	_ agentports.AIProviderTester     = (*Dispatcher)(nil)
	_ briefingports.BriefingDayWriter = (*Dispatcher)(nil)
)

func NewDispatcher(p DispatcherParams) *Dispatcher {
	return &Dispatcher{workflows: p.Workflows, l: p.Logger.Named("completion-dispatcher")}
}

// CompleteStructured asks the model one structured question on a worker.
func (d *Dispatcher) CompleteStructured(
	ctx context.Context,
	req *agentports.StructuredCompletionRequest,
) (*agentports.StructuredCompletionResult, error) {
	var result agentports.StructuredCompletionResult
	if err := d.await(ctx, &call{
		id:           "structured-completion/" + pulid.MustNew("scmp_").String(),
		workflow:     StructuredCompletionWorkflowName,
		summary:      "Ask the model: " + string(req.Task),
		feature:      structuredFeature(req),
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
	req agentports.GetAIProviderByIDRequest,
) (*agentports.TestAIProviderResult, error) {
	var result agentports.TestAIProviderResult
	if err := d.await(ctx, &call{
		id:           "ai-provider-test/" + req.ID.String(),
		workflow:     TestAIProviderWorkflowName,
		summary:      "Test an AI provider",
		feature:      jobFeatureProviderTest,
		organization: req.TenantInfo.OrgID,
		payload:      &TestAIProviderPayload{Request: req},
	}, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListModels asks an endpoint for its models on a worker. Each request is
// its own call, abandoned when the editor stops waiting.
func (d *Dispatcher) ListModels(
	ctx context.Context,
	req *agentports.ProbeAIProviderModelsRequest,
) ([]agentports.AIProviderModelOption, error) {
	var result []agentports.AIProviderModelOption
	if err := d.await(ctx, &call{
		id:           "ai-provider-models/" + pulid.MustNew("apml_").String(),
		workflow:     ListAIProviderModelsWorkflowName,
		summary:      "List an AI provider's models",
		feature:      jobFeatureModelList,
		organization: req.TenantInfo.OrgID,
		exclusive:    true,
		payload:      &ListAIProviderModelsPayload{Request: req},
	}, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// TestDraft probes an unsaved endpoint on a worker.
func (d *Dispatcher) TestDraft(
	ctx context.Context,
	req *agentports.ProbeAIProviderDraftRequest,
) (*agentports.AIProviderDraftTestResult, error) {
	var result agentports.AIProviderDraftTestResult
	if err := d.await(ctx, &call{
		id:           "ai-provider-draft-test/" + pulid.MustNew("apdt_").String(),
		workflow:     TestAIProviderDraftWorkflowName,
		summary:      "Test an unsaved AI provider",
		feature:      jobFeatureProviderTest,
		organization: req.TenantInfo.OrgID,
		exclusive:    true,
		payload:      &TestAIProviderDraftPayload{Request: req},
	}, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// WriteForDay writes a day's briefing on a worker. Two requests to rewrite the
// same page at once share the one write.
func (d *Dispatcher) WriteForDay(
	ctx context.Context,
	req briefingports.WriteBriefingRequest,
) (*briefingports.WriteBriefingResult, error) {
	var result briefingports.WriteBriefingResult
	if err := d.await(ctx, &call{
		id:           briefingWriteID(req),
		workflow:     WriteBriefingWorkflowName,
		summary:      "Write the briefing",
		feature:      jobFeatureBriefing,
		organization: req.TenantInfo.OrgID,
		payload:      &WriteBriefingPayload{Request: req},
	}, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func briefingWriteID(req briefingports.WriteBriefingRequest) string {
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
	feature      string
	organization pulid.ID
	// exclusive says the call is this request's alone, so a request that
	// stops waiting cancels it. A shared call is left to whoever else waits.
	exclusive bool
	payload   any
}

func (d *Dispatcher) await(ctx context.Context, c *call, result any) (err error) {
	ctx, span := aitrace.StartJob(ctx, &aitrace.JobSpec{
		Feature:        c.feature,
		WorkflowID:     c.id,
		OrganizationID: c.organization,
	})
	defer func() {
		if err != nil {
			aitrace.MarkFailed(span, jobFailure(ctx, err))
		}
		span.End()
	}()

	run, err := d.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       c.id,
		TaskQueue:                temporaltype.TaskQueueAgentChat.String(),
		WorkflowExecutionTimeout: waitFor(ctx),
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		StaticSummary:            c.summary,
		Priority: temporal.Priority{
			PriorityKey: agentflowcontract.PriorityOneShot,
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

		return modelcallcontract.Err(err)
	}

	return nil
}

func structuredFeature(req *agentports.StructuredCompletionRequest) string {
	if req.Attribution.Feature != "" {
		return string(req.Attribution.Feature)
	}

	return jobFeatureStructured
}

func jobFailure(ctx context.Context, err error) string {
	if ctx.Err() != nil {
		return jobFailureAbandoned
	}
	if kind := agentflowcontract.FailureKind(modelcallcontract.FailureOf(err)); kind != "" {
		return kind
	}

	return jobFailureFailed
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
		return DefaultWait
	}

	return max(time.Until(deadline)-answerMargin, minWait)
}
