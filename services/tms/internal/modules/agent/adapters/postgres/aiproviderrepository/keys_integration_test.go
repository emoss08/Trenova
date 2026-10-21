//go:build integration

package aiproviderrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRebindSealedKeyReplacesOnlyTheKeyItWasHanded(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	providers := New(Params{DB: conn, Logger: zap.NewNop()})
	keys := NewKeyRepository(KeyParams{DB: conn})
	rotationExpiresAt := time.Now().Add(time.Hour).Unix()
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	created, err := providers.Create(ctx, &aiprovider.Provider{
		OrganizationID:       data.Organization.ID,
		BusinessUnitID:       data.BusinessUnit.ID,
		Name:                 "Primary",
		Kind:                 aiprovider.KindOpenAIChat,
		BaseURL:              "https://api.example.com/v1",
		Model:                "gpt-test",
		Tasks:                []aiprovider.Task{aiprovider.TaskGeneral},
		StructuredOutputMode: aiprovider.KindOpenAIChat.DefaultStructuredOutputMode(),
		ReasoningEffort:      aiprovider.ReasoningOff,
		ThinkingStyle:        aiprovider.ThinkingStyleAuto,
		EmbeddingInputStyle:  aiprovider.EmbeddingInputStyleNone,
		TimeoutSeconds:       aiprovider.DefaultTimeoutSeconds,
		MaxTokens:            aiprovider.ClampMaxTokens(8192),
		Enabled:              true,
		APIKey:               "legacy-current",
		PreviousAPIKey:       "legacy-previous",
		RotationExpiresAt:    &rotationExpiresAt,
	})
	require.NoError(t, err)

	require.NoError(t, keys.RebindSealedKey(ctx, &agentports.RebindAIProviderKeyRequest{
		ID:          created.ID,
		TenantInfo:  tenant,
		Expected:    "someone-else-replaced-it",
		Replacement: "bound-current",
	}))
	require.NoError(t, keys.RebindSealedKey(ctx, &agentports.RebindAIProviderKeyRequest{
		ID:          created.ID,
		TenantInfo:  tenant,
		Previous:    true,
		Expected:    "legacy-previous",
		Replacement: "bound-previous",
	}))

	loaded, err := providers.GetByID(ctx, agentports.GetAIProviderByIDRequest{
		ID:         created.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, "legacy-current", loaded.APIKey)
	assert.Equal(t, "bound-previous", loaded.PreviousAPIKey)
}
