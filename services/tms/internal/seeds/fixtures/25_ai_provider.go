package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
)

// Names the other AI seeds look providers up by, so an insight can attribute
// its prose to the provider that would have written it.
const (
	SeedAIProviderLocalName  = "Local LM Studio"
	SeedAIProviderServerName = "Workstation vLLM"
	SeedAIProviderHostedName = "Anthropic (add a key to enable)"
)

type AIProviderSeed struct {
}

// AIProviderSeed gives a development organization the three shapes of provider
// the marketplace card and the readiness banner know how to show: a local model
// that answers everything cheap, a self-hosted server that carries the
// assistant, and a hosted frontier model left disabled until someone adds a
// key. Neither enabled provider needs a credential, so the seed stores none and
// the encryption service is never involved.
//
// Depends on:
//   - AdminAccount: the default organization the providers belong to
var AIProvider = Seeds.Add(seed.Seed{
	Name:        "ai_provider",
	Kind:        seed.Fixture,
	Description: "Seeds local, self-hosted and hosted AI providers with task routing",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&AIProviderSeed{}).Run,
})

func (s *AIProviderSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	for _, provider := range s.providers(org.ID, org.BusinessUnitID) {
		if err = aiproviderdb.Providers.Create(ctx, db, provider); err != nil {
			return fmt.Errorf("insert provider %s: %w", provider.Name, err)
		}
	}

	return nil
}

func (s *AIProviderSeed) providers(orgID, buID pulid.ID) []*aiprovider.Provider {
	return []*aiprovider.Provider{
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAIProviderLocalName,
			Description: "A small model on the developer's own machine. Cheap enough to run " +
				"on every chat turn and every insight refresh.",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:1234/v1",
			Model:                "qwen2.5-14b-instruct",
			AllowPrivateNetwork:  true,
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			ReasoningEffort:      aiprovider.ReasoningOff,
			MaxTokens:            8192,
			Tasks: []aiprovider.Task{
				aiprovider.TaskScopeClassification,
				aiprovider.TaskDocumentClassification,
				aiprovider.TaskOperationalInsights,
				aiprovider.TaskDailyBriefing,
				aiprovider.TaskGeneral,
			},
			Priority: 10,
			Enabled:  true,

			ThinkingStyle: aiprovider.ThinkingStyleAuto,

			Platform: aiprovider.PlatformDirect,

			EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNone,

			TimeoutSeconds: 60,

			MaxConcurrent: 8,

			OnCap: aiprovider.CapActionNext,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAIProviderServerName,
			Description: "An OpenAI-compatible server on the office network. Carries the " +
				"assistant and anything that needs a larger model.",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:8000/v1",
			Model:                "meta-llama/Llama-3.3-70B-Instruct",
			AllowPrivateNetwork:  true,
			StructuredOutputMode: aiprovider.StructuredOutputJSONMode,
			ReasoningEffort:      aiprovider.ReasoningOff,
			MaxTokens:            16384,
			Tasks: []aiprovider.Task{
				aiprovider.TaskAssistantChat,
				aiprovider.TaskDocumentExtraction,
				aiprovider.TaskFormulaAssistant,
				aiprovider.TaskOperationalInsights,
				aiprovider.TaskGeneral,
			},
			Priority: 20,
			Enabled:  true,
			Scripts:  true,

			ThinkingStyle: aiprovider.ThinkingStyleAuto,

			Platform: aiprovider.PlatformDirect,

			EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNone,

			TimeoutSeconds: 60,

			MaxConcurrent: 8,

			OnCap: aiprovider.CapActionNext,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAIProviderHostedName,
			Description: "Claude Haiku 5.5, the model the Desk is tuned for, serving every " +
				"task but embeddings and trusted for billing diagnosis. Disabled until an " +
				"API key is added.",
			Kind:                 aiprovider.KindAnthropicMessages,
			Model:                "claude-haiku-5-5",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			ReasoningEffort:      aiprovider.ReasoningOff,
			MaxTokens:            16384,
			Tasks:                hostedTasks(),
			Priority:             5,
			Trusted:              true,
			Enabled:              false,
			Scripts:              true,

			ThinkingStyle: aiprovider.ThinkingStyleAuto,

			Platform: aiprovider.PlatformDirect,

			EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNone,

			TimeoutSeconds: 60,

			MaxConcurrent: 8,

			OnCap: aiprovider.CapActionNext,
		},
	}
}

// hostedTasks is every task the hosted model serves: all but embeddings, which
// the Messages API has no endpoint for. The Desk is tuned for this model, and a
// task left to the local servers alone (the scope classifier, reading a
// document) stops working on a machine that does not run them.
func hostedTasks() []aiprovider.Task {
	tasks := make([]aiprovider.Task, 0, len(aiprovider.AllTasks()))
	for _, task := range aiprovider.AllTasks() {
		if task != aiprovider.TaskEmbedding {
			tasks = append(tasks, task)
		}
	}

	return tasks
}
