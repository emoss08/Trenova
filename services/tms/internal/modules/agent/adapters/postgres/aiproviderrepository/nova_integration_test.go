//go:build integration

package aiproviderrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func anthropicProvider(orgID, buID pulid.ID, name string) *aiprovider.Provider {
	return &aiprovider.Provider{
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		Name:                 name,
		Kind:                 aiprovider.KindAnthropicMessages,
		Platform:             aiplatform.PlatformDirect,
		Model:                "claude-opus-5-5",
		Tasks:                []aitask.Task{aitask.TaskAssistantChat},
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		ReasoningEffort:      aireasoning.ReasoningOff,
		ThinkingStyle:        aireasoning.ThinkingStyleAuto,
		EmbeddingInputStyle:  aiprovider.EmbeddingInputStyleNone,
		TimeoutSeconds:       aiprovider.DefaultTimeoutSeconds,
		MaxTokens:            aiprovider.ClampMaxTokens(8192),
		Enabled:              true,
		APIKey:               "sealed-" + name,
		Nova:                 true,
	}
}

// An organization runs Nova on one provider: choosing another, on create or on
// save, takes Nova from the one that had it in the same write, and the Nova
// read returns the provider with its credential.
func TestChoosingAProviderForNovaMovesNovaToIt(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	conn := postgres.WrapDB(db)
	providers := New(Params{DB: conn, Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	novaOf := func() pulid.ID {
		t.Helper()
		nova, err := providers.GetNova(ctx, tenant)
		require.NoError(t, err)
		require.NotNil(t, nova)
		return nova.ID
	}
	isNova := func(id pulid.ID) bool {
		t.Helper()
		loaded, err := providers.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
			ID:         id,
			TenantInfo: tenant,
		})
		require.NoError(t, err)
		return loaded.Nova
	}

	first, err := providers.Create(
		ctx,
		anthropicProvider(data.Organization.ID, data.BusinessUnit.ID, "First"),
	)
	require.NoError(t, err)
	nova, err := providers.GetNova(ctx, tenant)
	require.NoError(t, err)
	require.NotNil(t, nova)
	assert.Equal(t, first.ID, nova.ID)
	assert.Equal(t, "sealed-First", nova.APIKey, "Nova calls with the provider's credential")

	second, err := providers.Create(
		ctx,
		anthropicProvider(data.Organization.ID, data.BusinessUnit.ID, "Second"),
	)
	require.NoError(t, err)
	assert.Equal(t, second.ID, novaOf())
	assert.False(t, isNova(first.ID), "creating a Nova provider takes Nova from the first")

	reloaded, err := providers.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
		ID:         first.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	reloaded.Nova = true
	_, err = providers.Update(ctx, reloaded)
	require.NoError(t, err)
	assert.Equal(t, first.ID, novaOf())
	assert.False(t, isNova(second.ID), "saving a provider as Nova's takes Nova from the other")
}
