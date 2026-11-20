package completionengine

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/testutil/metricstest"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func builtIn(org pulid.ID) *aiprovider.Provider {
	return aiprovider.BuiltInEmbedding(
		org,
		pulid.MustNew("bu_"),
		aiprovider.BuiltInEmbeddingService{URL: "http://embed.internal", Model: "nomic"},
	)
}

func TestBreakerKey_RestsTheBuiltInProviderForOneOrganizationAlone(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_790_000_000, 0)
	health := NewProviderHealth(func() time.Time { return now })
	flooding := builtIn(pulid.MustNew("org_"))
	quiet := builtIn(pulid.MustNew("org_"))
	down := &modeladapter.TransportError{StatusCode: http.StatusBadGateway, Retryable: true}

	for range BreakerThreshold {
		health.Observe(t.Context(), BreakerKey(flooding), down)
	}

	_, resting := health.Resting(BreakerKey(flooding))
	assert.True(t, resting)
	_, resting = health.Resting(BreakerKey(quiet))
	assert.False(t, resting, "another organization still reaches the built-in provider")

	owned := &aiprovider.Provider{
		ID:             pulid.MustNew("aiprv_"),
		OrganizationID: pulid.MustNew("org_"),
	}
	assert.Equal(
		t,
		owned.ID,
		BreakerKey(owned),
		"a provider an organization owns keeps one breaker",
	)
}

type sharingSlots struct {
	mu   sync.Mutex
	seen []llmports.AcquireProviderSlotRequest
}

func (s *sharingSlots) Acquire(
	_ context.Context,
	req llmports.AcquireProviderSlotRequest,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, req)
	return true, nil
}

func (*sharingSlots) Refresh(context.Context, llmports.ProviderSlotRequest) (bool, error) {
	return true, nil
}

func (*sharingSlots) Release(context.Context, llmports.ProviderSlotRequest) error {
	return nil
}

func TestClaimSlot_BoundsAnOrganizationsShareOfTheBuiltInProviderOnASharedServer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		managed  *managedconfig.Config
		provider func(org pulid.ID) *aiprovider.Provider
		share    int
	}{
		{
			name:     "built-in on a shared server",
			managed:  &managedconfig.Config{Enabled: true, Tenancy: managedconfig.TenancyShared},
			provider: builtIn,
			share:    2,
		},
		{
			name:     "built-in on a dedicated server",
			managed:  &managedconfig.Config{Enabled: true, Tenancy: managedconfig.TenancyDedicated},
			provider: builtIn,
		},
		{
			name:    "an organization's own provider on a shared server",
			managed: &managedconfig.Config{Enabled: true, Tenancy: managedconfig.TenancyShared},
			provider: func(org pulid.ID) *aiprovider.Provider {
				return &aiprovider.Provider{ID: pulid.MustNew("aiprv_"), OrganizationID: org}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			slots := &sharingSlots{}
			engine := &Engine{
				Logger: zap.NewNop(),
				Limits: NewProviderLimits(&Params{
					Slots:        slots,
					Managed:      tc.managed,
					TenantLimits: &tenantlimitsconfig.Config{},
					Metrics:      metricstest.Disabled(),
				}),
			}
			org := pulid.MustNew("org_")

			release, err := engine.ClaimSlot(t.Context(), tc.provider(org), time.Minute)
			require.NoError(t, err)
			release()

			require.Len(t, slots.seen, 1)
			assert.Equal(t, tc.share, slots.seen[0].OrganizationLimit)
			if tc.share > 0 {
				assert.Equal(t, org, slots.seen[0].OrganizationID)
			} else {
				assert.True(t, slots.seen[0].OrganizationID.IsNil())
			}
		})
	}
}
