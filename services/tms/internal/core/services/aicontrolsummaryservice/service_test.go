package aicontrolsummaryservice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type factsRepo struct {
	counts   aicontrolsummary.AgentCounts
	failures []repositories.ProviderFailureCount
}

func (f *factsRepo) AgentCounts(
	context.Context,
	*repositories.AgentCountsRequest,
) (aicontrolsummary.AgentCounts, error) {
	return f.counts, nil
}

func (f *factsRepo) ProviderFailures(
	context.Context,
	*repositories.ProviderFailuresRequest,
) ([]repositories.ProviderFailureCount, error) {
	return f.failures, nil
}

type providerRepo struct {
	repositories.AIProviderRepository
	enabled []*aiprovider.Provider
}

func (p *providerRepo) ListEnabled(context.Context, pagination.TenantInfo) ([]*aiprovider.Provider, error) {
	return p.enabled, nil
}

type controlRepo struct {
	repositories.AgentControlRepository
	paused bool
}

func (c *controlRepo) GetOrCreate(context.Context, pagination.TenantInfo) (*tenant.AgentControl, error) {
	return &tenant.AgentControl{ShadowMode: c.paused}, nil
}

type memoryCache struct {
	mu      sync.Mutex
	stored  map[string]aicontrolsummary.Summary
	ttls    map[string]time.Duration
	claims  int
	allowed bool
}

func newMemoryCache() *memoryCache {
	return &memoryCache{
		stored:  map[string]aicontrolsummary.Summary{},
		ttls:    map[string]time.Duration{},
		allowed: true,
	}
}

func (c *memoryCache) Get(
	_ context.Context,
	key *repositories.AIControlSummaryKey,
	dest *aicontrolsummary.Summary,
) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.stored[key.FactsHash]
	if ok {
		*dest = value
	}
	return ok, nil
}

func (c *memoryCache) Set(
	_ context.Context,
	key *repositories.AIControlSummaryKey,
	value *aicontrolsummary.Summary,
	ttl time.Duration,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stored[key.FactsHash] = *value
	c.ttls[key.FactsHash] = ttl
	return nil
}

func (c *memoryCache) ClaimNarration(context.Context, *repositories.AIControlSummaryKey, int) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.claims++
	return c.allowed, nil
}

type completion struct {
	services.CompletionService
	reply string
	calls int
}

func (c *completion) CompleteStructured(
	context.Context,
	*services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	c.calls++
	return &services.StructuredCompletionResult{Text: c.reply, ModelIdentifier: "test"}, nil
}

func newService(cache *memoryCache, model *completion, enabled ...*aiprovider.Provider) *Service {
	svc := &Service{
		l:         zap.NewNop(),
		facts:     &factsRepo{counts: aicontrolsummary.AgentCounts{Total: 3, On: 2, Waiting: 4}},
		providers: &providerRepo{enabled: enabled},
		controls:  &controlRepo{},
		cache:     cache,
		now:       func() time.Time { return time.Unix(1_800_000_000, 0) },
		detach:    func(run func()) { run() },
	}
	if model != nil {
		svc.completion = model
	}
	return svc
}

func chatProvider() *aiprovider.Provider {
	dimensions := 1536
	return &aiprovider.Provider{
		EmbeddingDimensions: &dimensions,
		ID:                  pulid.MustNew("aiprv_"),
		Name:                "Primary",
		Enabled:             true,
		Tasks:               aiprovider.AllTasks(),
		Trusted:             true,
		Kind:                aiprovider.KindOpenAIChat,
	}
}

const reworded = `{"segments":[
 {"text":"Two agents are on and ","strong":false,"target":"","providerId":""},
 {"text":"4 proposals","strong":true,"target":"watchtower","providerId":""},
 {"text":" are waiting on someone in Watchtower.","strong":false,"target":"","providerId":""}]}`

func request() *services.AIControlSummaryRequest {
	return &services.AIControlSummaryRequest{Tab: aicontrolsummary.TabOverview}
}

func TestSummaryRewordsOnceAndThenReadsWhatWasKept(t *testing.T) {
	t.Parallel()

	cache := newMemoryCache()
	model := &completion{reply: reworded}
	svc := newService(cache, model, chatProvider())

	first, err := svc.Summary(t.Context(), request())
	require.NoError(t, err)
	assert.True(t, first.Pending, "the first read gets the plain sentence while a model rewords it")
	assert.False(t, first.Narrated)

	second, err := svc.Summary(t.Context(), request())
	require.NoError(t, err)
	assert.True(t, second.Narrated)
	assert.False(t, second.Pending)
	assert.Equal(t, "Two agents are on and 4 proposals are waiting on someone in Watchtower.",
		aicontrolsummary.Text(second.Segments))
	assert.Equal(t, aicontrolsummary.ToneWarn, second.Segments[1].Tone, "the link keeps its tone")

	_, err = svc.Summary(t.Context(), request())
	require.NoError(t, err)
	assert.Equal(t, 1, model.calls, "a read with the same facts never asks the model again")
	assert.Equal(t, narratedTTL, cache.ttls[first.FactsHash])
}

func TestSummaryKeepsThePlainSentenceWhenTheModelInventsAFigure(t *testing.T) {
	t.Parallel()

	cache := newMemoryCache()
	model := &completion{reply: `{"segments":[
 {"text":"9 proposals","strong":true,"target":"watchtower","providerId":""},
 {"text":" wait.","strong":false,"target":"","providerId":""}]}`}
	svc := newService(cache, model, chatProvider())

	first, err := svc.Summary(t.Context(), request())
	require.NoError(t, err)
	second, err := svc.Summary(t.Context(), request())
	require.NoError(t, err)

	assert.False(t, second.Narrated)
	assert.Equal(t, aicontrolsummary.Text(first.Segments), aicontrolsummary.Text(second.Segments))
	assert.Equal(t, plainTTL, cache.ttls[first.FactsHash])
}

func TestSummaryWithNoProviderNeverAsksAModel(t *testing.T) {
	t.Parallel()

	cache := newMemoryCache()
	model := &completion{reply: reworded}
	svc := newService(cache, model)

	summary, err := svc.Summary(t.Context(), request())

	require.NoError(t, err)
	assert.False(t, summary.Pending)
	assert.Zero(t, model.calls)
	assert.Zero(t, cache.claims)
}

func TestSummaryPastTheDaysAllowanceStaysPlain(t *testing.T) {
	t.Parallel()

	cache := newMemoryCache()
	cache.allowed = false
	model := &completion{reply: reworded}
	svc := newService(cache, model, chatProvider())

	summary, err := svc.Summary(t.Context(), request())

	require.NoError(t, err)
	assert.False(t, summary.Pending)
	assert.Zero(t, model.calls)
}

func TestSummaryRefusesAnUnknownTab(t *testing.T) {
	t.Parallel()

	_, err := newService(newMemoryCache(), nil).Summary(t.Context(), &services.AIControlSummaryRequest{Tab: "Nope"})

	require.Error(t, err)
}

func TestFactsNameOnlyEnabledProvidersAsFailing(t *testing.T) {
	t.Parallel()

	on := chatProvider()
	svc := newService(newMemoryCache(), nil, on)
	svc.facts = &factsRepo{failures: []repositories.ProviderFailureCount{
		{ProviderID: on.ID, FailedCalls: 24, LastFailureAt: 10},
		{ProviderID: pulid.MustNew("aiprv_"), FailedCalls: 3},
	}}

	facts, err := svc.Facts(t.Context(), pagination.TenantInfo{})

	require.NoError(t, err)
	require.Len(t, facts.Failing, 1)
	assert.Equal(t, "Primary", facts.Failing[0].Name)
	assert.Zero(t, facts.Uncovered)
}

func TestAcceptRefusesALinkThePlainSentenceDoesNotHave(t *testing.T) {
	t.Parallel()

	plain := []aicontrolsummary.Segment{{Text: "4 proposals", Target: aicontrolsummary.TargetWatchtower}}
	facts := &aicontrolsummary.Facts{Agents: aicontrolsummary.AgentCounts{Waiting: 4}}

	_, ok := Accept([]aicontrolsummary.Segment{
		{Text: "4 proposals", Target: aicontrolsummary.TargetWatchtower},
		{Text: " and routing", Target: aicontrolsummary.TargetRouting},
	}, facts, plain)

	assert.False(t, ok)
}
