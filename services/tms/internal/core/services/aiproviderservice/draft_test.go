package aiproviderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/optional"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sealingTester struct {
	fakeTester
	models *services.ProbeAIProviderModelsRequest
	draft  *services.ProbeAIProviderDraftRequest
}

func (f *sealingTester) ListModels(
	_ context.Context,
	req *services.ProbeAIProviderModelsRequest,
) ([]services.AIProviderModelOption, error) {
	f.models = req
	return nil, nil
}

func (f *sealingTester) TestDraft(
	_ context.Context,
	req *services.ProbeAIProviderDraftRequest,
) (*services.AIProviderDraftTestResult, error) {
	f.draft = req
	return &services.AIProviderDraftTestResult{Success: true}, nil
}

func draftTenant(provider *aiprovider.Provider) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: provider.OrganizationID, BuID: provider.BusinessUnitID}
}

func TestTestDraft_HandsTheWorkerOnlyCiphertext(t *testing.T) {
	t.Parallel()

	prober := &fakeProber{result: &services.TestAIProviderResult{Success: true}}
	svc := newTestService(&fakeProviderRepo{}, prober)
	tester := &sealingTester{}
	svc.tester = tester
	typed := "sk-ant-api03-typed-into-the-editor"

	_, err := svc.TestDraft(t.Context(), &services.TestAIProviderDraftRequest{
		Endpoint: services.AIProviderEndpoint{
			Kind:   aiprovider.KindAnthropicMessages,
			APIKey: &typed,
		},
		Model: "claude-sonnet-4-5",
		Tasks: []aiprovider.Task{aiprovider.TaskAssistantChat},
	})
	require.NoError(t, err)
	require.NotNil(t, tester.draft)
	assert.NotEmpty(t, tester.draft.Endpoint.SealedAPIKey)
	assert.NotContains(t, tester.draft.Endpoint.SealedAPIKey, typed)
	assert.Equal(t, aiprovider.DefaultTimeoutSeconds, tester.draft.TimeoutSeconds)

	result, err := svc.RunTestDraft(t.Context(), tester.draft)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, typed, prober.seen, "the worker decrypts the key it was handed")
}

func TestListModels_UsesTheStoredKeyOnlyForItsOwnEndpoint(t *testing.T) {
	t.Parallel()

	repo := &fakeProviderRepo{}
	svc := newTestService(repo, &fakeProber{})
	tester := &sealingTester{}
	svc.tester = tester
	repo.provider = testProvider(t, svc)
	repo.provider.BaseURL = "https://api.example.com/v1"

	ask := func(baseURL string) string {
		t.Helper()
		_, err := svc.ListModels(t.Context(), &services.ListAIProviderModelsRequest{
			TenantInfo: draftTenant(repo.provider),
			Endpoint: services.AIProviderEndpoint{
				ProviderID: repo.provider.ID,
				Kind:       aiprovider.KindOpenAIChat,
				BaseURL:    baseURL,
			},
		})
		require.NoError(t, err)
		return tester.models.Endpoint.SealedAPIKey
	}

	assert.Equal(t, repo.provider.APIKey, ask("https://api.example.com/v1"))
	assert.Empty(t, ask("https://attacker.example.net/v1"),
		"a stored key is never sent to an address it was not entered for")
}

func TestTestDraft_RefusesADraftWithoutAModel(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	svc.tester = &sealingTester{}
	timeout := 1

	_, err := svc.TestDraft(t.Context(), &services.TestAIProviderDraftRequest{
		Endpoint:       services.AIProviderEndpoint{Kind: aiprovider.KindOllama},
		Tasks:          []aiprovider.Task{aiprovider.TaskAssistantChat},
		TimeoutSeconds: &timeout,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Model")
}

func TestApply_KeepsTheReplacedKeyOnlyForTheSameEndpoint(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	actor := &services.RequestActor{
		PrincipalType: services.PrincipalTypeUser,
		UserID:        pulid.MustNew("usr_"),
	}
	next := "sk-proj-AbCdEfGhIjKlMnOpQrStUvWx9876"

	provider := testProvider(t, svc)
	provider.BaseURL = "https://api.example.com/v1"
	original := provider.APIKey
	req := saveRequest(provider, "https://api.example.com/v1", &next)
	req.KeepPreviousKey = true
	require.NoError(t, svc.apply(provider, req, actor))
	assert.Equal(t, original, provider.PreviousAPIKey)
	require.NotNil(t, provider.RotationExpiresAt)
	assert.Equal(t, "9876", provider.APIKeyLastFour)
	assert.Equal(t, "sk-proj-", provider.APIKeyPrefix)
	assert.Equal(t, actor.UserID, provider.APIKeyAddedByID)
	assert.Equal(t, aiprovider.DefaultTimeoutSeconds, provider.TimeoutSeconds)
	assert.Equal(t, aiprovider.DefaultMaxConcurrent, provider.MaxConcurrent)
	assert.Equal(t, aiprovider.CapActionNext, provider.OnCap)

	moved := testProvider(t, svc)
	moved.BaseURL = "https://api.example.com/v1"
	req = saveRequest(moved, "https://elsewhere.example.net/v1", &next)
	req.KeepPreviousKey = true
	require.NoError(t, svc.apply(moved, req, actor))
	assert.Empty(t, moved.PreviousAPIKey, "the old key is not kept for a new address")
	assert.Nil(t, moved.RotationExpiresAt)
}

func TestPatchEdits_RefusesToClearARequiredSwitch(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})

	_, err := svc.patchEdits(&services.PatchAIProviderRequest{
		Enabled: optional.Some[*bool](nil),
	}, nil)
	require.Error(t, err)

	off := false
	edits, err := svc.patchEdits(&services.PatchAIProviderRequest{
		Enabled: optional.Some(&off),
	}, nil)
	require.NoError(t, err)
	provider := &aiprovider.Provider{Enabled: true, Trusted: true}
	for _, edit := range edits {
		edit(provider)
	}
	assert.False(t, provider.Enabled)
	assert.True(t, provider.Trusted, "a field left out is left alone")
}
