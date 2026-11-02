package agentfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/shared/pulid"
)

// Names the other AI seeds look providers up by, so an insight can attribute
// its prose to the provider that would have written it.
const (
	SeedAIProviderLocalName  = "Local LM Studio"
	SeedAIProviderServerName = "Workstation vLLM"
	SeedAIProviderHostedName = "Anthropic (add a key to enable)"
)

// AIProviderSeed gives a development organization the three shapes of provider
// the marketplace card and the readiness banner know how to show: a local model
// that answers everything cheap, a self-hosted server that carries the
// assistant, and a hosted frontier model left disabled until someone adds a
// key. Neither enabled provider needs a credential, so the seed stores none and
// the encryption service is never involved.
//
// Depends on:
//   - AdminAccount: the default organization the providers belong to
type AIProviderSeed struct {
}

func (s *AIProviderSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
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
				"the Desk's scope guard on every chat turn.",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:1234/v1",
			Model:                "qwen2.5-14b-instruct",
			AllowPrivateNetwork:  true,
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			ReasoningEffort:      aireasoning.ReasoningOff,
			MaxTokens:            8192,
			Tasks: []aitask.Task{
				aitask.TaskScopeClassification,
				aitask.TaskGeneral,
			},
			Priority: 10,
			Enabled:  true,

			ThinkingStyle: aireasoning.ThinkingStyleAuto,

			Platform: aiplatform.PlatformDirect,

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
				"Desk when the hosted model is off.",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:8000/v1",
			Model:                "meta-llama/Llama-3.3-70B-Instruct",
			AllowPrivateNetwork:  true,
			StructuredOutputMode: aiprovider.StructuredOutputJSONMode,
			ReasoningEffort:      aireasoning.ReasoningOff,
			MaxTokens:            16384,
			Tasks: []aitask.Task{
				aitask.TaskAssistantChat,
				aitask.TaskGeneral,
			},
			Priority: 20,
			Enabled:  true,
			Scripts:  true,

			ThinkingStyle: aireasoning.ThinkingStyleAuto,

			Platform: aiplatform.PlatformDirect,

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
				"Desk task and Nova. Disabled until an API key is added.",
			Kind:                 aiprovider.KindAnthropicMessages,
			Model:                "claude-haiku-5-5",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			ReasoningEffort:      aireasoning.ReasoningOff,
			MaxTokens:            16384,
			Tasks:                hostedTasks(),
			Priority:             5,
			Trusted:              true,
			Enabled:              false,
			Nova:                 true,
			Scripts:              true,

			ThinkingStyle: aireasoning.ThinkingStyleAuto,

			Platform: aiplatform.PlatformDirect,

			EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNone,

			TimeoutSeconds: 60,

			MaxConcurrent: 8,

			OnCap: aiprovider.CapActionNext,
		},
	}
}

// hostedTasks is every task the hosted model is assigned: the Desk's, but not
// embeddings, which the Messages API has no endpoint for. It also runs Nova,
// which is not assigned. A Desk task left to the local servers alone (the
// scope classifier) stops working on a machine that does not run them.
func hostedTasks() []aitask.Task {
	routed := aitask.RoutedTasks()
	tasks := make([]aitask.Task, 0, len(routed))
	for _, task := range routed {
		if task != aitask.TaskEmbedding {
			tasks = append(tasks, task)
		}
	}

	return tasks
}
