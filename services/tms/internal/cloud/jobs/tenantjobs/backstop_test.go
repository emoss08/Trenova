package tenantjobs

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/interceptors"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workfeature"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type suspendedGate struct{ asked atomic.Int32 }

func (g *suspendedGate) Enabled(ctx context.Context, _ platformcatalog.FeatureKey) (bool, error) {
	if _, ok := dbscope.TenantFrom(ctx); ok {
		g.asked.Add(1)
	}
	return false, nil
}

func (g *suspendedGate) EnabledAnywhere(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, nil
}

type headerFields struct{ header *commonpb.Header }

func (h headerFields) Set(key string, value *commonpb.Payload) { h.header.Fields[key] = value }

type backstopRig struct {
	env    *testsuite.TestWorkflowEnvironment
	mu     sync.Mutex
	order  []string
	scopes []pulid.ID
}

func (r *backstopRig) ran(ctx context.Context, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, name)
	tenant, _ := dbscope.TenantFrom(ctx)
	r.scopes = append(r.scopes, tenant.OrganizationID)
}

func newBackstopRig(
	t *testing.T,
	gate *suspendedGate,
	scheduled platformcatalog.FeatureKey,
) *backstopRig {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	suite.SetContextPropagators([]workflow.ContextPropagator{workfeature.NewPropagator()})
	if scheduled != "" {
		header := &commonpb.Header{Fields: map[string]*commonpb.Payload{}}
		ctx := workfeature.Starting(t.Context(), scheduled)
		require.NoError(t, workfeature.NewPropagator().Inject(ctx, headerFields{header: header}))
		suite.SetHeader(header)
	}
	r := &backstopRig{env: suite.NewTestWorkflowEnvironment()}
	r.env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{
		interceptors.NewTenantScopeInterceptor(),
		interceptors.NewEntitlementInterceptor(gate, nil, zap.NewNop()),
	}})
	r.register()
	return r
}

func (r *backstopRig) register() {
	type payload = *tenantjobscontract.TenantPayload
	named := func(fn any, name string) {
		r.env.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name})
	}
	named(func(ctx context.Context, _ payload) (bool, error) {
		r.ran(ctx, "check")
		return true, nil
	}, "CheckRemovalActivity")
	named(func(ctx context.Context, _ payload) (*tenantlifecycle.RevocationResult, error) {
		r.ran(ctx, "members")
		return &tenantlifecycle.RevocationResult{}, nil
	}, "ReleaseMembersActivity")
	named(func(ctx context.Context, _ payload) (*tenantlifecycle.RevocationResult, error) {
		r.ran(ctx, "sessions")
		return &tenantlifecycle.RevocationResult{}, nil
	}, "RevokeSessionsActivity")
	named(func(ctx context.Context, _ payload) (*cloudtenantports.PurgeRowsResult, error) {
		r.ran(ctx, "rows")
		return &cloudtenantports.PurgeRowsResult{Complete: true}, nil
	}, "PurgeRowsActivity")
	named(func(ctx context.Context, _ payload) (*cloudtenantports.DeleteResult, error) {
		r.ran(ctx, "organization")
		return &cloudtenantports.DeleteResult{Deleted: true}, nil
	}, "DeleteOrganizationActivity")
	named(func(ctx context.Context, _ payload) (*cloudtenantports.DeleteResult, error) {
		r.ran(ctx, "unit")
		return &cloudtenantports.DeleteResult{Deleted: true}, nil
	}, "DeleteBusinessUnitActivity")
	named(func(ctx context.Context, _ payload) (*StoresResult, error) {
		r.ran(ctx, "stores")
		return &StoresResult{}, nil
	}, "PurgeStoresActivity")
	named(func(ctx context.Context, _ payload) ([]cloudtenantports.TableCount, error) {
		r.ran(ctx, "verify")
		return []cloudtenantports.TableCount{}, nil
	}, "VerifyRemovedActivity")
	named(func(ctx context.Context, _ *FinishInput) error {
		r.ran(ctx, "finish")
		return nil
	}, "FinishRemovalActivity")
}

func TestRemovalRunsForASuspendedTenant(t *testing.T) {
	t.Parallel()
	gate := &suspendedGate{}
	r := newBackstopRig(t, gate, "")
	tenant := payload()

	r.env.ExecuteWorkflow(CloudTenantRemovalWorkflow, tenant)

	require.True(t, r.env.IsWorkflowCompleted())
	require.NoError(t, r.env.GetWorkflowError())
	assert.Equal(t, []string{
		"check", "members", "rows", "organization", "rows", "unit", "stores", "verify", "finish",
	}, r.order)
	for i, scope := range r.scopes {
		assert.Equal(
			t,
			tenant.OrganizationID,
			scope,
			"activity %s ran in the tenant's scope",
			r.order[i],
		)
	}
	assert.Zero(t, gate.asked.Load())
}

func TestSessionsAreEndedForASuspendedTenant(t *testing.T) {
	t.Parallel()
	gate := &suspendedGate{}
	r := newBackstopRig(t, gate, "")

	r.env.ExecuteWorkflow(CloudTenantSessionsWorkflow, payload())

	require.True(t, r.env.IsWorkflowCompleted())
	require.NoError(t, r.env.GetWorkflowError())
	assert.Equal(t, []string{"sessions"}, r.order)
}

func TestTheBackstopIsActiveInTheRemovalRig(t *testing.T) {
	t.Parallel()
	gate := &suspendedGate{}
	r := newBackstopRig(t, gate, platformcatalog.FeatureAdministration)

	r.env.ExecuteWorkflow(CloudTenantRemovalWorkflow, payload())

	require.True(t, r.env.IsWorkflowCompleted())
	var refused *temporal.ApplicationError
	require.ErrorAs(t, r.env.GetWorkflowError(), &refused)
	assert.Equal(t, interceptors.NotEntitledErrorType, refused.Type())
	assert.Empty(t, r.order)
	assert.Positive(t, gate.asked.Load())
}
