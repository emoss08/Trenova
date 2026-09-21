package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

// Names the other AI seeds look providers up by, so an insight can attribute
// its prose to the provider that would have written it.
const (
	SeedAIProviderLocalName  = "Local Ollama"
	SeedAIProviderServerName = "Workstation vLLM"
	SeedAIProviderHostedName = "Anthropic (add a key to enable)"
)

type AIProviderSeed struct {
	seedhelpers.BaseSeed
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
func NewAIProviderSeed() *AIProviderSeed {
	seed := &AIProviderSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AIProvider",
		"1.0.0",
		"Seeds local, self-hosted and hosted AI providers with task routing",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount)
	return seed
}

func (s *AIProviderSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			cols := buncolgen.ProviderColumns
			count, err := tx.NewSelect().
				Model((*aiprovider.Provider)(nil)).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing providers: %w", err)
			}
			if count > 0 {
				return nil
			}

			for _, provider := range s.providers(org.ID, org.BusinessUnitID) {
				if _, err = tx.NewInsert().Model(provider).Exec(ctx); err != nil {
					return fmt.Errorf("insert provider %s: %w", provider.Name, err)
				}
				if err = sc.TrackCreated(ctx, "ai_providers", provider.ID, s.Name()); err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func (s *AIProviderSeed) providers(orgID, buID pulid.ID) []*aiprovider.Provider {
	return []*aiprovider.Provider{
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAIProviderLocalName,
			Description: "A small model on the developer's own machine. Cheap enough to run " +
				"on every chat turn and every insight refresh.",
			Kind:                 aiprovider.KindOllama,
			Model:                "qwen2.5:14b",
			AllowPrivateNetwork:  true,
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			ReasoningEffort:      aiprovider.ReasoningOff,
			MaxTokens:            8192,
			Tasks: []aiprovider.Task{
				aiprovider.TaskScopeClassification,
				aiprovider.TaskDocumentClassification,
				aiprovider.TaskOperationalInsights,
				aiprovider.TaskGeneral,
			},
			Priority: 10,
			Enabled:  true,
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
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAIProviderHostedName,
			Description: "A hosted frontier model, marked trusted so it may serve billing " +
				"diagnosis. Disabled until an API key is added.",
			Kind:                 aiprovider.KindAnthropicMessages,
			Model:                "claude-sonnet-5",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			ReasoningEffort:      aiprovider.ReasoningOff,
			MaxTokens:            16384,
			Tasks: []aiprovider.Task{
				aiprovider.TaskBillingDiagnosis,
				aiprovider.TaskAssistantChat,
			},
			Priority: 5,
			Trusted:  true,
			Enabled:  false,
		},
	}
}

func (s *AIProviderSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *AIProviderSeed) CanRollback() bool {
	return true
}
