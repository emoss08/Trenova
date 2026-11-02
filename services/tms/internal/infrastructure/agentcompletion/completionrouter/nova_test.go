package completionrouter

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/adapterregistry"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func novaService(t *testing.T, repo *fakeRepo) *Service {
	t.Helper()

	return compose(&completionengine.Engine{
		Logger:     zap.NewNop(),
		AI:         &aiconfig.Config{MaxRetries: 1},
		Repo:       repo,
		Encryption: encryptionservice.NewWithKeyManager(encryptionservice.NewLocalKeyManager("k")),
		Adapters:   adapterregistry.NewRegistry(),
		Clients:    make(map[bool]*http.Client, 2),
	})
}

// opusDeskProvider is an Anthropic provider an organization set up for the
// Desk, on a model of its own choosing with prices entered for that model.
func opusDeskProvider() *aiprovider.Provider {
	input, output := decimal.NewFromInt(4), decimal.NewFromInt(20)

	return &aiprovider.Provider{
		ID:                   pulid.MustNew("aiprv_"),
		Name:                 "Anthropic",
		Kind:                 aiprovider.KindAnthropicMessages,
		Platform:             aiplatform.PlatformDirect,
		BaseURL:              "https://api.anthropic.com",
		Model:                "claude-opus-5-5",
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		ReasoningEffort:      aireasoning.ReasoningHigh,
		AdvisorEnabled:       true,
		AdvisorModel:         "claude-fable-5-1",
		InputCostPerMillion:  &input,
		OutputCostPerMillion: &output,
		Tasks:                []aitask.Task{aitask.TaskAssistantChat},
		Enabled:              true,
		Nova:                 true,
	}
}

func novaRun(task aitask.Task) *completionengine.RunRequest {
	return &completionengine.RunRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Task:       task,
		Nova:       true,
	}
}

func TestCandidatesFor_ANovaCallRunsOnTheNovaProviderWithNovasModelAndPrice(t *testing.T) {
	t.Parallel()

	nova := opusDeskProvider()
	assigned := openAIChatProvider("Workstation", "http://localhost:1234/v1", 1)
	assigned.Tasks = []aitask.Task{aitask.TaskGeneral}
	svc := novaService(t, &fakeRepo{providers: []*aiprovider.Provider{assigned}, nova: nova})

	candidates, err := svc.CandidatesFor(t.Context(), novaRun(aitask.TaskPaperworkReading))
	require.NoError(t, err)

	require.Len(t, candidates, 1, "Nova runs on its provider alone, never on the task's assignees")
	got := candidates[0]
	assert.Equal(t, nova.ID, got.ID)
	assert.Equal(t, "claude-haiku-5-5", got.Model)
	assert.Equal(
		t,
		"https://api.anthropic.com",
		got.BaseURL,
		"the provider's endpoint carries over",
	)
	require.NotNil(t, got.InputCostPerMillion)
	require.NotNil(t, got.OutputCostPerMillion)
	assert.True(t, got.InputCostPerMillion.Equal(decimal.RequireFromString("0.10")),
		"Haiku 5.5's list input price, not the Desk model's: %s", got.InputCostPerMillion)
	assert.True(t, got.OutputCostPerMillion.Equal(decimal.RequireFromString("0.50")),
		"Haiku 5.5's list output price: %s", got.OutputCostPerMillion)
	assert.Equal(t, aireasoning.ReasoningOff, got.ReasoningEffort)
	assert.False(t, got.AdvisorEnabled, "the advisor would bill another model for Nova's calls")
	assert.Equal(t, "claude-opus-5-5", nova.Model, "the stored provider keeps its own model")
}

func TestCandidatesFor_ACallWithoutTheNovaMarkKeepsItsTasksProviders(t *testing.T) {
	t.Parallel()

	assigned := openAIChatProvider("Workstation", "http://localhost:1234/v1", 1)
	svc := novaService(t, &fakeRepo{
		providers: []*aiprovider.Provider{assigned},
		nova:      opusDeskProvider(),
	})
	run := novaRun(aitask.TaskGeneral)
	run.Nova = false

	candidates, err := svc.CandidatesFor(t.Context(), run)
	require.NoError(t, err)

	require.Len(t, candidates, 1)
	assert.Equal(t, assigned.ID, candidates[0].ID)
	assert.Equal(t, "qwen3:32b", candidates[0].Model)
}

func TestCandidatesFor_ANovaCallIsRefusedWithoutANovaProvider(t *testing.T) {
	t.Parallel()

	svc := novaService(t, &fakeRepo{})

	_, err := svc.CandidatesFor(t.Context(), novaRun(aitask.TaskPaperworkReading))

	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured)
	assert.Contains(t, err.Error(), "choose it for Nova")
}

func TestCandidatesFor_ANovaCallIsRefusedWhileTheNovaProviderIsOff(t *testing.T) {
	t.Parallel()

	nova := opusDeskProvider()
	nova.Enabled = false
	svc := novaService(t, &fakeRepo{nova: nova})

	_, err := svc.CandidatesFor(t.Context(), novaRun(aitask.TaskPaperworkReading))

	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured)
	assert.Contains(t, err.Error(), "Anthropic runs Nova but is turned off")
}

func TestNovaReady(t *testing.T) {
	t.Parallel()

	off := opusDeskProvider()
	off.Enabled = false
	disabled := false

	tests := []struct {
		name string
		repo *fakeRepo
		ai   *aiconfig.Config
		want bool
	}{
		{name: "a Nova provider that is on", repo: &fakeRepo{nova: opusDeskProvider()}, want: true},
		{name: "no Nova provider", repo: &fakeRepo{}, want: false},
		{name: "the Nova provider off", repo: &fakeRepo{nova: off}, want: false},
		{
			name: "AI off on the server",
			repo: &fakeRepo{nova: opusDeskProvider()},
			ai:   &aiconfig.Config{Enabled: &disabled},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := novaService(t, tt.repo)
			if tt.ai != nil {
				svc.AI = tt.ai
			}

			ready, err := svc.NovaReady(t.Context(), pagination.TenantInfo{})
			require.NoError(t, err)
			assert.Equal(t, tt.want, ready)
		})
	}
}
