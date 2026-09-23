package completionjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type fakeRun struct {
	client.WorkflowRun
	id  string
	get func(ctx context.Context, result any) error
}

func (r fakeRun) GetID() string    { return r.id }
func (r fakeRun) GetRunID() string { return "run-1" }

func (r fakeRun) Get(ctx context.Context, result any) error { return r.get(ctx, result) }

type fakeStarter struct {
	serviceports.WorkflowStarter
	options   []client.StartWorkflowOptions
	workflows []any
	payloads  []any
	cancelled []string
	get       func(ctx context.Context, result any) error
}

func (f *fakeStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	f.options = append(f.options, options)
	f.workflows = append(f.workflows, workflow)
	f.payloads = append(f.payloads, args[0])

	return fakeRun{id: options.ID, get: f.get}, nil
}

func (f *fakeStarter) CancelWorkflow(_ context.Context, workflowID, _ string) error {
	f.cancelled = append(f.cancelled, workflowID)

	return nil
}

func dispatcher(starter *fakeStarter) *Dispatcher {
	return &Dispatcher{workflows: starter, l: zap.NewNop()}
}

func TestDispatcher_RunsTheCallOnTheChatQueueWithinTheRequestsDeadline(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{get: func(_ context.Context, result any) error {
		*result.(*serviceports.StructuredCompletionResult) = serviceports.StructuredCompletionResult{
			Text: "{}",
		}

		return nil
	}}
	req := request()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	result, err := dispatcher(starter).CompleteStructured(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "{}", result.Text)

	require.Len(t, starter.options, 1)
	options := starter.options[0]
	assert.Equal(t, temporaltype.TaskQueueAgentChat.String(), options.TaskQueue)
	assert.Equal(t, StructuredCompletionWorkflowName, starter.workflows[0])
	assert.Same(t, req, starter.payloads[0].(*StructuredCompletionPayload).Request)
	assert.Equal(t, agentflow.PriorityOneShot, options.Priority.PriorityKey)
	assert.Equal(t, req.TenantInfo.OrgID.String(), options.Priority.FairnessKey)
	assert.LessOrEqual(t, options.WorkflowExecutionTimeout, 30*time.Second-answerMargin,
		"the call ends before the request that waits on it")
	assert.Greater(t, options.WorkflowExecutionTimeout, 20*time.Second)
}

func TestDispatcher_GivesEveryCallItsOwnExecution(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{get: func(context.Context, any) error { return nil }}
	d := dispatcher(starter)

	_, err := d.CompleteStructured(t.Context(), request())
	require.NoError(t, err)
	_, err = d.CompleteStructured(t.Context(), request())
	require.NoError(t, err)

	require.Len(t, starter.options, 2)
	assert.NotEqual(t, starter.options[0].ID, starter.options[1].ID)
	assert.Equal(t, defaultWait, starter.options[0].WorkflowExecutionTimeout,
		"a caller with no deadline of its own waits the API's request timeout")
}

// Two administrators testing one provider at once share the probe, and so do
// two people rewriting the same page.
func TestDispatcher_SharesACallThatIsTheSameQuestion(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{get: func(context.Context, any) error { return nil }}
	d := dispatcher(starter)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	providerID := pulid.MustNew("aiprv_")

	_, err := d.Test(
		t.Context(),
		repositories.GetAIProviderByIDRequest{ID: providerID, TenantInfo: tenant},
	)
	require.NoError(t, err)
	_, err = d.WriteForDay(t.Context(), serviceports.WriteBriefingRequest{
		TenantInfo:   tenant,
		Roles:        []briefing.RoleKey{briefing.RoleBilling},
		BriefingDate: "2026-09-23",
	})
	require.NoError(t, err)

	require.Len(t, starter.options, 2)
	assert.Equal(t, "ai-provider-test/"+providerID.String(), starter.options[0].ID)
	assert.Equal(
		t,
		"briefing-write/"+tenant.OrgID.String()+"/Billing/2026-09-23",
		starter.options[1].ID,
	)
	for _, options := range starter.options {
		assert.Equal(
			t,
			enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			options.WorkflowIDConflictPolicy,
		)
		assert.Equal(
			t,
			enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			options.WorkflowIDReusePolicy,
		)
	}
}

// A person who stops waiting stops paying for the answer, unless someone else
// is waiting on the same call.
func TestDispatcher_CancelsACallNobodyWaitsFor(t *testing.T) {
	t.Parallel()

	stopWaiting := func(ctx context.Context, cancel context.CancelFunc) *fakeStarter {
		return &fakeStarter{get: func(context.Context, any) error {
			cancel()

			return ctx.Err()
		}}
	}

	exclusiveCtx, cancelExclusive := context.WithCancel(t.Context())
	defer cancelExclusive()
	exclusive := stopWaiting(exclusiveCtx, cancelExclusive)

	_, err := dispatcher(exclusive).CompleteStructured(exclusiveCtx, request())
	require.Error(t, err)
	require.Len(t, exclusive.cancelled, 1)
	assert.Equal(t, exclusive.options[0].ID, exclusive.cancelled[0])

	sharedCtx, cancelShared := context.WithCancel(t.Context())
	defer cancelShared()
	shared := stopWaiting(sharedCtx, cancelShared)

	_, err = dispatcher(shared).Test(sharedCtx, repositories.GetAIProviderByIDRequest{
		ID: pulid.MustNew("aiprv_"),
	})
	require.Error(t, err)
	assert.Empty(t, shared.cancelled, "a shared call is left to the others waiting on it")
}

// What the person is told is what they would have been told had the call run
// in their request.
func TestDispatcher_ReturnsTheCallsOwnError(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{get: func(context.Context, any) error {
		return modelcall.Classify(errors.Join(serviceports.ErrNoProviderConfigured))
	}}

	_, err := dispatcher(starter).CompleteStructured(t.Context(), request())
	assert.ErrorIs(t, err, serviceports.ErrNoProviderConfigured)

	starter.get = func(context.Context, any) error {
		return modelcall.Classify(errortypes.NewBusinessError("AI is turned off"))
	}
	_, err = dispatcher(starter).CompleteStructured(t.Context(), request())
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
}
