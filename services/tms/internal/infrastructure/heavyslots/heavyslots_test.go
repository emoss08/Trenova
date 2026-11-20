package heavyslots_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/heavyslots"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/redis/slotstore"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryStore struct {
	mu   sync.Mutex
	sets map[string]map[string]struct{}
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sets: map[string]map[string]struct{}{}}
}

func (s *memoryStore) Acquire(_ context.Context, claim slotstore.Claim) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, bound := range claim.Bounds {
		if len(s.sets[bound.Key]) >= bound.Limit {
			return false, nil
		}
	}
	for _, bound := range claim.Bounds {
		if s.sets[bound.Key] == nil {
			s.sets[bound.Key] = map[string]struct{}{}
		}
		s.sets[bound.Key][claim.Token] = struct{}{}
	}
	return true, nil
}

func (s *memoryStore) Refresh(context.Context, slotstore.Claim) (bool, error) {
	return true, nil
}

func (s *memoryStore) Release(_ context.Context, claim slotstore.Claim) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, bound := range claim.Bounds {
		delete(s.sets[bound.Key], claim.Token)
	}
	return nil
}

func tenantCtx(t *testing.T, org pulid.ID) context.Context {
	t.Helper()
	return dbscope.WithValidTenant(t.Context(), dbscope.Tenant{
		OrganizationID: org,
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func limits() *tenantlimitsconfig.Config {
	return &tenantlimitsconfig.Config{HeavyJobs: tenantlimitsconfig.HeavyJobs{
		PDF: tenantlimitsconfig.Slot{
			PerOrganization: 1,
			Total:           2,
			Hold:            time.Minute,
			Wait:            300 * time.Millisecond,
		},
	}}
}

func counter(t *testing.T, registry *prometheus.Registry, name string) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	total := 0.0
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			total += metric.GetCounter().GetValue()
		}
	}
	return total
}

func TestHold_BoundsOneOrganizationAndLetsAnotherThrough(t *testing.T) {
	registry := prometheus.NewRegistry()
	gate := heavyslots.NewGate(
		newMemoryStore(),
		limits(),
		true,
		platformmetrics.NewTenancy(registry, zap.NewNop(), true),
		zap.NewNop(),
	)
	busy := tenantCtx(t, pulid.MustNew("org_"))
	quiet := tenantCtx(t, pulid.MustNew("org_"))

	release, err := gate.Hold(busy, kernelports.HeavyJobPDF)
	require.NoError(t, err)

	_, err = gate.Hold(busy, kernelports.HeavyJobPDF)
	var limited *errortypes.RateLimitError
	require.ErrorAs(
		t,
		err,
		&limited,
		"a second render from one organization waits, then is told the server is busy",
	)
	assert.Positive(t, limited.RetryAfter)
	assert.InDelta(t, 1, counter(t, registry, "trenova_tenancy_slot_waits_total"), 0)
	assert.InDelta(t, 1, counter(t, registry, "trenova_tenancy_limit_refusals_total"), 0)

	other, err := gate.Hold(quiet, kernelports.HeavyJobPDF)
	require.NoError(t, err, "another organization still renders")
	other()

	release()
	again, err := gate.Hold(busy, kernelports.HeavyJobPDF)
	require.NoError(t, err)
	again()
}

func TestHold_AWaitingJobTakesTheSlotWhenItFrees(t *testing.T) {
	gate := heavyslots.NewGate(newMemoryStore(), limits(), true, nil, zap.NewNop())
	ctx := tenantCtx(t, pulid.MustNew("org_"))

	release, err := gate.Hold(ctx, kernelports.HeavyJobPDF)
	require.NoError(t, err)
	time.AfterFunc(100*time.Millisecond, release)

	next, err := gate.Hold(ctx, kernelports.HeavyJobPDF)
	require.NoError(t, err)
	next()
}

func TestHold_TheInstanceBoundAppliesWithoutTheOrganizationBound(t *testing.T) {
	gate := heavyslots.NewGate(newMemoryStore(), limits(), false, nil, zap.NewNop())
	ctx := tenantCtx(t, pulid.MustNew("org_"))

	first, err := gate.Hold(ctx, kernelports.HeavyJobPDF)
	require.NoError(t, err)
	second, err := gate.Hold(ctx, kernelports.HeavyJobPDF)
	require.NoError(t, err, "a dedicated install bounds only the instance")

	_, err = gate.Hold(t.Context(), kernelports.HeavyJobPDF)
	var limited *errortypes.RateLimitError
	require.ErrorAs(t, err, &limited, "the instance holds no more than its total")
	first()
	second()
}

func TestTryHold_SkipsWhenNoSlotIsFree(t *testing.T) {
	gate := heavyslots.NewGate(newMemoryStore(), limits(), true, nil, zap.NewNop())
	ctx := tenantCtx(t, pulid.MustNew("org_"))

	release, ok := gate.TryHold(ctx, kernelports.HeavyJobPDF)
	require.True(t, ok)

	_, ok = gate.TryHold(ctx, kernelports.HeavyJobPDF)
	assert.False(t, ok)

	release()
	release()
	again, ok := gate.TryHold(ctx, kernelports.HeavyJobPDF)
	assert.True(t, ok, "releasing twice frees one slot, once")
	again()
}
