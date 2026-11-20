package entitlementtest

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/interceptors"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workfeature"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type Shape int

const (
	FansOut Shape = iota + 1
	InOneActivity
	NoTenant
)

func (s Shape) String() string {
	switch s {
	case FansOut:
		return "fans out per tenant"
	case InOneActivity:
		return "works for tenants inside one activity"
	case NoTenant:
		return "belongs to no tenant"
	default:
		return "undeclared"
	}
}

type Package struct {
	Schedules  []*schedule.Schedule
	Activities []any
	Workflows  []registry.WorkflowDefinition
	Shapes     map[string]Shape
}

func RunSchedules(t *testing.T, pkg Package) {
	t.Helper()

	require.NotEmpty(t, pkg.Schedules, "a package with no schedules has nothing to check")
	seen := make(map[string]struct{}, len(pkg.Schedules))
	for _, sched := range pkg.Schedules {
		shape, declared := pkg.Shapes[sched.ID]
		require.True(t, declared, "schedule %q declares no shape", sched.ID)
		seen[sched.ID] = struct{}{}
		t.Run(sched.ID, func(t *testing.T) {
			runSchedule(t, pkg, sched, shape)
		})
	}
	for id := range pkg.Shapes {
		_, ok := seen[id]
		assert.True(t, ok, "shape declared for %q, which the package does not schedule", id)
	}
}

func runSchedule(t *testing.T, pkg Package, sched *schedule.Schedule, shape Shape) {
	world := NewWorld(sched.Feature)
	observed := &observations{world: world}
	refused := &refusals{}

	var suite testsuite.WorkflowTestSuite
	suite.SetContextPropagators([]workflow.ContextPropagator{workfeature.NewPropagator()})
	suite.SetHeader(featureHeader(t, sched))
	suite.SetLogger(nopLogger{})

	env := suite.NewTestWorkflowEnvironment()
	env.SetTestTimeout(20 * time.Second)
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{
			&recordingInterceptor{observed: observed},
			interceptors.NewTenantScopeInterceptor(),
			interceptors.NewEntitlementInterceptor(world.Gate(), refused, zap.NewNop()),
		},
	})

	env.RegisterWorkflowWithOptions(sched.Workflow, workflow.RegisterOptions{
		Name:                          workflowName(sched, pkg.Workflows),
		DisableAlreadyRegisteredCheck: true,
	})
	for _, def := range pkg.Workflows {
		env.RegisterWorkflowWithOptions(def.Fn, workflow.RegisterOptions{
			Name:                          def.Name,
			DisableAlreadyRegisteredCheck: true,
		})
	}
	env.RegisterDynamicWorkflow(
		func(workflow.Context, converter.EncodedValues) error { return nil },
		workflow.DynamicRegisterOptions{},
	)

	fakes := &fakeActivities{world: world, answered: map[string]bool{}}
	for _, acts := range pkg.Activities {
		fakes.register(env, acts)
	}
	env.RegisterActivity(temporaljobs.NewEntitledActivities(world.Filter()))
	env.RegisterDynamicActivity(
		func(context.Context, converter.EncodedValues) error { return nil },
		activity.DynamicRegisterOptions{},
	)

	func() {
		defer func() {
			if r := recover(); r != nil {
				observed.note(fmt.Sprintf("the workflow panicked: %v", r))
			}
		}()
		env.ExecuteWorkflow(sched.Workflow, sched.Args...)
	}()

	for _, served := range observed.outsiders() {
		assert.Failf(t, "work scheduled for a tenant outside the feature",
			"%s scheduled %s for a tenant whose plan lacks %s or that is suspended",
			sched.ID, served, sched.Feature)
	}
	assert.Empty(t, refused.snapshot(),
		"%s: the activity backstop refused work the fan-out should have skipped", sched.ID)

	switch shape {
	case FansOut:
		assert.NotEmpty(
			t,
			observed.entitledWork(),
			"%s is declared to fan out per tenant but scheduled no work for the entitled tenant (%s)",
			sched.ID,
			observed.summary(),
		)
	case InOneActivity, NoTenant:
		assert.Empty(t, observed.tenantWork(),
			"%s is declared to %s but scheduled per-tenant work", sched.ID, shape)
	default:
		assert.Failf(t, "undeclared shape", "%s", sched.ID)
	}
}

func workflowName(sched *schedule.Schedule, defs []registry.WorkflowDefinition) string {
	target := reflect.ValueOf(sched.Workflow).Pointer()
	for _, def := range defs {
		if reflect.ValueOf(def.Fn).Pointer() == target {
			return def.Name
		}
	}
	name := sched.GetWorkflowName()
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return name
}

func featureHeader(t *testing.T, sched *schedule.Schedule) *commonpb.Header {
	t.Helper()
	header := &commonpb.Header{Fields: map[string]*commonpb.Payload{}}
	ctx := workfeature.Starting(context.Background(), sched.Feature)
	require.NoError(t, workfeature.NewPropagator().Inject(ctx, headerWriter{header: header}))
	return header
}

type headerWriter struct{ header *commonpb.Header }

func (w headerWriter) Set(key string, value *commonpb.Payload) { w.header.Fields[key] = value }

type observations struct {
	world *World
	mu    sync.Mutex
	work  []scheduled
	notes []string
}

type scheduled struct {
	kind   string
	name   string
	tenant pulid.ID
}

func (s scheduled) String() string {
	return s.kind + " " + s.name
}

func (o *observations) record(kind, name string, args []any) {
	tenant, ok := dbscope.TenantOf(args...)
	if !ok || tenant.OrganizationID.IsNil() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.work = append(o.work, scheduled{kind: kind, name: name, tenant: tenant.OrganizationID})
}

func (o *observations) note(text string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.notes = append(o.notes, text)
}

func (o *observations) outsiders() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []string
	for _, w := range o.work {
		if o.world.Outsider(w.tenant) {
			out = append(out, w.String())
		}
	}
	sort.Strings(out)
	return out
}

func (o *observations) entitledWork() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var out []string
	for _, w := range o.work {
		if w.tenant == o.world.Entitled.OrganizationID {
			out = append(out, w.String())
		}
	}
	return out
}

func (o *observations) tenantWork() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]string, 0, len(o.work))
	for _, w := range o.work {
		out = append(out, w.String())
	}
	return out
}

func (o *observations) summary() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return fmt.Sprintf("%d per-tenant starts; notes: %v", len(o.work), o.notes)
}

type recordingInterceptor struct {
	interceptor.InterceptorBase
	observed *observations
}

func (r *recordingInterceptor) InterceptWorkflow(
	_ workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &recordingInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		observed:                       r.observed,
	}
}

type recordingInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	observed *observations
}

func (r *recordingInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return r.Next.Init(&recordingOutbound{
		WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{
			Next: outbound,
		},
		observed: r.observed,
	})
}

type recordingOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	observed *observations
}

func (r *recordingOutbound) ExecuteActivity(
	ctx workflow.Context,
	activityType string,
	args ...any,
) workflow.Future {
	if activityType != temporaljobs.KeepEntitledActivityName {
		r.observed.record("activity", activityType, args)
	}
	return r.Next.ExecuteActivity(ctx, activityType, args...)
}

func (r *recordingOutbound) ExecuteLocalActivity(
	ctx workflow.Context,
	activityType string,
	args ...any,
) workflow.Future {
	r.observed.record("local activity", activityType, args)
	return r.Next.ExecuteLocalActivity(ctx, activityType, args...)
}

func (r *recordingOutbound) ExecuteChildWorkflow(
	ctx workflow.Context,
	childWorkflowType string,
	args ...any,
) workflow.ChildWorkflowFuture {
	r.observed.record("child workflow", childWorkflowType, args)
	return r.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}
