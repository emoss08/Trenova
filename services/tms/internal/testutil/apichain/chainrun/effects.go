package chainrun

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/shared/pulid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

var ErrWorkflowRecorded = errors.New(
	"the chain harness records workflow starts and never runs them",
)

type Workflow struct {
	Type      string `json:"type"`
	TaskQueue string `json:"queue,omitempty"`
	ID        string `json:"id,omitempty"`
	Args      []any  `json:"args,omitempty"`
}

type Signal struct {
	WorkflowID string `json:"workflowId"`
	Name       string `json:"name"`
	Arg        any    `json:"arg,omitempty"`
}

type Realtime struct {
	Event    string `json:"event"`
	Scope    string `json:"scope,omitempty"`
	Audience string `json:"audience,omitempty"`
	Payload  any    `json:"payload,omitempty"`
}

type Audit struct {
	Resource   string `json:"resource"`
	Operation  string `json:"operation"`
	ResourceID string `json:"resourceId,omitempty"`
	Critical   bool   `json:"critical,omitempty"`
}

type Captured struct {
	Workflows []Workflow `json:"workflows,omitempty"`
	Signals   []Signal   `json:"signals,omitempty"`
	Cancels   []string   `json:"cancels,omitempty"`
	Realtime  []Realtime `json:"realtime,omitempty"`
	Audit     []Audit    `json:"audit,omitempty"`
}

func (c Captured) Empty() bool {
	return len(c.Workflows) == 0 && len(c.Signals) == 0 && len(c.Cancels) == 0 &&
		len(c.Realtime) == 0 && len(c.Audit) == 0
}

type Effects struct {
	mu      sync.Mutex
	pending Captured
}

func (e *Effects) Drain() Captured {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := e.pending
	e.pending = Captured{}

	return out
}

func (e *Effects) add(fn func(*Captured)) {
	e.mu.Lock()
	defer e.mu.Unlock()

	fn(&e.pending)
}

type workflowRecorder struct {
	effects *Effects
}

func (r workflowRecorder) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	id := options.ID
	if id == "" {
		id = pulid.MustNew("wf_").String()
	}
	recorded := Workflow{
		Type:      workflowName(workflow),
		TaskQueue: options.TaskQueue,
		ID:        id,
		Args:      make([]any, 0, len(args)),
	}
	for _, arg := range args {
		recorded.Args = append(recorded.Args, plain(arg))
	}
	r.effects.add(func(c *Captured) { c.Workflows = append(c.Workflows, recorded) })

	return recordedRun{id: id}, nil
}

func (r workflowRecorder) CancelWorkflow(_ context.Context, workflowID, _ string) error {
	r.effects.add(func(c *Captured) { c.Cancels = append(c.Cancels, workflowID) })

	return nil
}

func (r workflowRecorder) SignalWorkflow(
	_ context.Context,
	workflowID, _, signalName string,
	arg any,
) error {
	recorded := Signal{WorkflowID: workflowID, Name: signalName, Arg: plain(arg)}
	r.effects.add(func(c *Captured) { c.Signals = append(c.Signals, recorded) })

	return nil
}

func (r workflowRecorder) SignalWithStartWorkflow(
	ctx context.Context,
	workflowID, signalName string,
	signalArg any,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	options.ID = workflowID
	run, err := r.StartWorkflow(ctx, options, workflow, args...)
	if err != nil {
		return nil, err
	}
	if err = r.SignalWorkflow(ctx, workflowID, "", signalName, signalArg); err != nil {
		return nil, err
	}

	return run, nil
}

func (r workflowRecorder) WorkflowResult(context.Context, string, any) error {
	return ErrWorkflowRecorded
}

var (
	_ kernelports.WorkflowStarter       = workflowRecorder{}
	_ kernelports.WorkflowSignalStarter = workflowRecorder{}
	_ kernelports.WorkflowResults       = workflowRecorder{}
	_ converter.StorageDriver           = refusingPayloads{}
)

type refusingPayloads struct{}

func (refusingPayloads) Name() string { return "chain-harness" }

func (refusingPayloads) Type() string { return "chain-harness" }

func (refusingPayloads) Store(
	converter.StorageDriverStoreContext,
	[]*commonpb.Payload,
) ([]converter.StorageDriverClaim, error) {
	return nil, ErrWorkflowRecorded
}

func (refusingPayloads) Retrieve(
	converter.StorageDriverRetrieveContext,
	[]converter.StorageDriverClaim,
) ([]*commonpb.Payload, error) {
	return nil, ErrWorkflowRecorded
}

type recordedRun struct {
	id string
}

func (r recordedRun) GetID() string    { return r.id }
func (r recordedRun) GetRunID() string { return r.id }

func (r recordedRun) GetFirstExecutionRunID() string { return r.id }

func (r recordedRun) Get(context.Context, any) error { return ErrWorkflowRecorded }

func (r recordedRun) GetWithOptions(context.Context, any, client.WorkflowRunGetOptions) error {
	return ErrWorkflowRecorded
}

type realtimeRecorder struct {
	inner   realtimeports.RealtimePublisher
	effects *Effects
}

func (r realtimeRecorder) Enqueue(env *realtimeports.RealtimeEnvelope) bool {
	r.record(env)

	return r.inner.Enqueue(env)
}

func (r realtimeRecorder) Publish(ctx context.Context, env *realtimeports.RealtimeEnvelope) error {
	r.record(env)

	return r.inner.Publish(ctx, env)
}

func (r realtimeRecorder) record(env *realtimeports.RealtimeEnvelope) {
	if env == nil || env.PresenceAction != "" || env.Ephemeral {
		return
	}
	recorded := Realtime{Event: env.Event, Scope: env.Scope}
	if !env.AudienceUserID.IsNil() {
		recorded.Audience = env.AudienceUserID.String()
	}
	if len(env.Payload) > 0 {
		var payload any
		if err := sonic.Unmarshal(env.Payload, &payload); err == nil {
			recorded.Payload = payload
		} else {
			recorded.Payload = string(env.Payload)
		}
	}
	r.effects.add(func(c *Captured) { c.Realtime = append(c.Realtime, recorded) })
}

type auditRecorder struct {
	auditports.AuditService
	effects *Effects
}

func (r auditRecorder) LogAction(
	params *auditports.LogActionParams,
	opts ...auditports.LogOption,
) error {
	r.record(params)

	return r.AuditService.LogAction(params, opts...)
}

func (r auditRecorder) LogActions(entries []auditports.BulkLogEntry) error {
	for _, entry := range entries {
		r.record(entry.Params)
	}

	return r.AuditService.LogActions(entries)
}

func (r auditRecorder) record(params *auditports.LogActionParams) {
	if params == nil {
		return
	}
	recorded := Audit{
		Resource:   params.Resource.String(),
		Operation:  string(params.Operation),
		ResourceID: params.ResourceID,
		Critical:   params.Critical,
	}
	r.effects.add(func(c *Captured) { c.Audit = append(c.Audit, recorded) })
}

func workflowName(workflow any) string {
	if name, ok := workflow.(string); ok {
		return name
	}
	value := reflect.ValueOf(workflow)
	if value.Kind() != reflect.Func {
		return reflect.TypeOf(workflow).String()
	}
	full := runtime.FuncForPC(value.Pointer()).Name()
	full = strings.TrimSuffix(full, "-fm")
	parts := strings.Split(full, ".")

	return parts[len(parts)-1]
}

func plain(value any) any {
	raw, err := sonic.Marshal(value)
	if err != nil {
		return reflect.TypeOf(value).String()
	}
	var out any
	if err = sonic.Unmarshal(raw, &out); err != nil {
		return string(raw)
	}

	return out
}
