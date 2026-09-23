package agentguard_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeShared is the store every replica would share, in memory.
type fakeShared struct {
	mu     sync.Mutex
	rows   map[string]*repositories.CachedScopeVerdict
	ttls   map[string]time.Duration
	getErr error
	setErr error
	gets   int
	sets   int
}

func newFakeShared() *fakeShared {
	return &fakeShared{
		rows: map[string]*repositories.CachedScopeVerdict{},
		ttls: map[string]time.Duration{},
	}
}

func (f *fakeShared) Get(_ context.Context, key string) (*repositories.CachedScopeVerdict, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.rows[key], nil
}

func (f *fakeShared) Set(
	_ context.Context,
	key string,
	v *repositories.CachedScopeVerdict,
	ttl time.Duration,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sets++
	if f.setErr != nil {
		return f.setErr
	}
	f.rows[key] = v
	f.ttls[key] = ttl

	return nil
}

func guardSharing(
	t *testing.T,
	stub *stubCompletion,
	shared repositories.ScopeVerdictCacheRepository,
) *agentguard.Service {
	t.Helper()

	return agentguard.New(agentguard.Params{
		Logger:     zap.NewNop(),
		Completion: stub,
		Verdicts:   shared,
	})
}

const hazmat = "Which drivers hold a hazmat endorsement?"

// Behind a load balancer the second ask lands on a different pod. Each pod
// used to classify it again. A verdict one replica produced is now read by
// the next, so the model is asked once per question, not once per replica.
func TestEvaluate_AReplicaReusesAnotherReplicasVerdict(t *testing.T) {
	t.Parallel()

	shared := newFakeShared()
	tenantInfo := tenant()

	first := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	replicaA := guardSharing(t, first, shared)
	decisionA := replicaA.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: hazmat},
	)
	require.True(t, decisionA.Allowed)
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, shared.sets, "the verdict is shared")

	second := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	replicaB := guardSharing(t, second, shared)
	decisionB := replicaB.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: hazmat},
	)

	require.True(t, decisionB.Allowed)
	assert.Zero(t, second.calls, "the other replica never asks the model")
	assert.Equal(t, decisionA.Category, decisionB.Category)

	// And having read it once, the replica keeps it locally: replica A's own
	// miss and replica B's hit are the only two reads, however often B is asked.
	replicaB.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: hazmat},
	)
	assert.Equal(t, 2, shared.gets, "a local hit does not go back to the store")
}

func TestEvaluate_SharesTheVerdictForADayByDefault(t *testing.T) {
	t.Parallel()

	shared := newFakeShared()
	guard := guardSharing(
		t,
		&stubCompletion{category: string(agentguard.CategoryTransportationOperations)},
		shared,
	)

	guard.Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenant(), Input: hazmat})

	for _, ttl := range shared.ttls {
		assert.Equal(t, 24*time.Hour, ttl)
	}
	assert.Len(t, shared.ttls, 1)
}

// A store that is down is a miss, never a failure: the classifier answers and
// the person is none the wiser. A cache that can take the guard down with it
// is worse than no cache.
func TestEvaluate_ClassifiesWhenTheSharedStoreIsDown(t *testing.T) {
	t.Parallel()

	shared := newFakeShared()
	shared.getErr = errors.New("redis: connection refused")
	shared.setErr = errors.New("redis: connection refused")
	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	guard := guardSharing(t, stub, shared)
	tenantInfo := tenant()

	decision := guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: hazmat},
	)
	require.True(t, decision.Allowed)
	assert.Equal(t, 1, stub.calls)

	// The local tier still works while the store is away.
	guard.Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: hazmat})
	assert.Equal(t, 1, stub.calls, "remembered locally despite the store")
}

// A classifier failure is a degraded state and is never shared: freezing it
// would turn one replica's blip into every replica's outage.
func TestEvaluate_NeverSharesAFailure(t *testing.T) {
	t.Parallel()

	shared := newFakeShared()
	stub := &stubCompletion{err: errors.New("provider down")}
	guard := guardSharing(t, stub, shared)

	guard.Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenant(), Input: hazmat})

	assert.Zero(t, shared.sets)
}
