package aiproviderservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeProviderRepo struct {
	repositories.AIProviderRepository

	provider *aiprovider.Provider
	getErr   error
	marked   []repositories.MarkAIProviderTestedRequest
	markErr  error
}

func (f *fakeProviderRepo) GetByID(
	_ context.Context,
	_ repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.provider, nil
}

func (f *fakeProviderRepo) MarkTested(
	_ context.Context,
	req repositories.MarkAIProviderTestedRequest,
) error {
	f.marked = append(f.marked, req)
	return f.markErr
}

type fakeProber struct {
	result *services.TestAIProviderResult
	seen   string
}

func (f *fakeProber) Probe(
	_ context.Context,
	_ *aiprovider.Provider,
	apiKey string,
) *services.TestAIProviderResult {
	f.seen = apiKey
	return f.result
}

func newTestService(repo *fakeProviderRepo, prober *fakeProber) *Service {
	return &Service{
		l:    zap.NewNop(),
		repo: repo,
		encryption: encryptionservice.NewWithKeyManager(
			encryptionservice.NewLocalKeyManager("unit-test-encryption-key-with-at-least-32-bytes"),
		),
		prober: prober,
	}
}

func testProvider(t *testing.T, svc *Service) *aiprovider.Provider {
	t.Helper()
	provider := &aiprovider.Provider{
		ID:             pulid.MustNew("aiprv_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Local Qwen",
		Kind:           aiprovider.KindOpenAIChat,
		Model:          "qwen3:32b",
	}
	encrypted, err := svc.encryption.EncryptString("sk-live")
	require.NoError(t, err)
	provider.APIKey = encrypted
	return provider
}

func TestRunTestRecordsProbeOutcomeOnProvider(t *testing.T) {
	repo := &fakeProviderRepo{}
	prober := &fakeProber{result: &services.TestAIProviderResult{
		Success:         true,
		Message:         "Connected",
		ModelIdentifier: "qwen3:32b",
		SchemaHonoured:  true,
		LatencyMS:       412,
	}}
	svc := newTestService(repo, prober)
	repo.provider = testProvider(t, svc)

	result, err := svc.RunTest(t.Context(), repositories.GetAIProviderByIDRequest{
		ID: repo.provider.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: repo.provider.OrganizationID,
			BuID:  repo.provider.BusinessUnitID,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "sk-live", prober.seen, "the decrypted credential reaches the probe")
	assert.True(t, result.Success)

	require.Len(t, repo.marked, 1)
	marked := repo.marked[0]
	assert.Equal(t, repo.provider.ID, marked.ID)
	assert.Equal(t, repo.provider.OrganizationID, marked.TenantInfo.OrgID)
	require.NotNil(t, marked.Outcome)
	assert.True(t, marked.Outcome.Success)
	assert.True(t, marked.Outcome.SchemaHonoured)
	assert.Equal(t, int64(412), marked.Outcome.LatencyMS)
	assert.Equal(t, "qwen3:32b", marked.Outcome.ModelIdentifier)
	assert.Positive(t, marked.Outcome.TestedAt)
}

func TestRunTestStillReturnsProbeResultWhenRecordingFails(t *testing.T) {
	repo := &fakeProviderRepo{markErr: errors.New("db down")}
	prober := &fakeProber{
		result: &services.TestAIProviderResult{Success: false, Message: "Refused"},
	}
	svc := newTestService(repo, prober)
	repo.provider = testProvider(t, svc)

	result, err := svc.RunTest(t.Context(), repositories.GetAIProviderByIDRequest{
		ID: repo.provider.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: repo.provider.OrganizationID,
			BuID:  repo.provider.BusinessUnitID,
		},
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "Refused", result.Message)
	require.Len(t, repo.marked, 1)
	assert.False(t, repo.marked[0].Outcome.Success)
}

type fakeTester struct {
	asked  []repositories.GetAIProviderByIDRequest
	result *services.TestAIProviderResult
}

func (f *fakeTester) Test(
	_ context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*services.TestAIProviderResult, error) {
	f.asked = append(f.asked, req)

	return f.result, nil
}

// A test waits on a worker, which runs the probe. The request only confirms
// the provider exists and hands it over.
func TestTestHandsTheProbeToAWorker(t *testing.T) {
	repo := &fakeProviderRepo{}
	tester := &fakeTester{result: &services.TestAIProviderResult{Success: true}}
	svc := newTestService(repo, &fakeProber{})
	svc.tester = tester
	repo.provider = testProvider(t, svc)
	req := repositories.GetAIProviderByIDRequest{
		ID: repo.provider.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: repo.provider.OrganizationID,
			BuID:  repo.provider.BusinessUnitID,
		},
	}

	result, err := svc.Test(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	require.Len(t, tester.asked, 1)
	assert.Equal(t, req, tester.asked[0])
	assert.Empty(t, repo.marked, "recording is the worker's, in RunTest")
}

// A provider that does not exist is reported as missing, not as a test that
// failed, and nothing is handed to a worker.
func TestTestOfAMissingProviderStartsNothing(t *testing.T) {
	repo := &fakeProviderRepo{getErr: errors.New("not found")}
	tester := &fakeTester{}
	svc := newTestService(repo, &fakeProber{})
	svc.tester = tester

	_, err := svc.Test(t.Context(), repositories.GetAIProviderByIDRequest{
		ID: pulid.MustNew("aiprv_"),
	})
	require.Error(t, err)
	assert.Empty(t, tester.asked)
}

func strPtr(value string) *string { return &value }

func saveRequest(provider *aiprovider.Provider, baseURL string, key *string) *services.SaveAIProviderRequest {
	return &services.SaveAIProviderRequest{
		ID:      provider.ID,
		Name:    provider.Name,
		Kind:    provider.Kind,
		BaseURL: baseURL,
		Model:   provider.Model,
		APIKey:  key,
		Enabled: true,
		TenantInfo: pagination.TenantInfo{
			OrgID: provider.OrganizationID,
			BuID:  provider.BusinessUnitID,
		},
	}
}

/*
A stored key belongs to the endpoint it was entered for.

Someone allowed to edit a provider but not to read its key could point it at
their own server and press Test: the key was kept, decrypted and sent there
in the Authorization header. Moving a provider now needs its key again.
*/
func TestUpdate_RefusesToKeepTheKeyForANewServer(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	provider := testProvider(t, svc)
	provider.BaseURL = "https://api.openai.com/v1"
	svc.repo = &fakeProviderRepo{provider: provider}

	_, err := svc.Update(t.Context(), saveRequest(provider, "https://attacker.example/v1", nil), nil)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Contains(t, fields, "apiKey")
}

func TestKeptKeyForNewEndpoint(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	existing := testProvider(t, svc)
	existing.BaseURL = "https://api.openai.com/v1"

	moved := *existing
	moved.BaseURL = "https://attacker.example/v1"
	assert.True(t, keptKeyForNewEndpoint(existing, &moved, saveRequest(existing, moved.BaseURL, nil)))
	assert.False(t,
		keptKeyForNewEndpoint(existing, &moved, saveRequest(existing, moved.BaseURL, strPtr("sk-new"))),
		"a key entered with the move is the new server's own")

	repathed := *existing
	repathed.BaseURL = "https://api.openai.com/v2"
	assert.False(t, keptKeyForNewEndpoint(existing, &repathed, saveRequest(existing, repathed.BaseURL, nil)),
		"the same server keeps its key")

	rekinded := *existing
	rekinded.Kind = aiprovider.KindOllama
	assert.True(t, keptKeyForNewEndpoint(existing, &rekinded, saveRequest(existing, existing.BaseURL, nil)))

	keyless := *existing
	keyless.APIKey = ""
	assert.False(t, keptKeyForNewEndpoint(&keyless, &moved, saveRequest(existing, moved.BaseURL, nil)),
		"nothing stored, nothing to leak")
}

// A deployment that hosts many organizations turns private addresses off, so
// no organization's administrator can make the server call into its network.
func TestApply_RefusesAPrivateNetworkTheServerDisallows(t *testing.T) {
	t.Parallel()

	off := false
	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	svc.ai = &config.AIConfig{PrivateNetworkProviders: &off}
	provider := testProvider(t, svc)
	req := saveRequest(provider, "http://10.0.0.5:11434", nil)
	req.AllowPrivateNetwork = true

	err := svc.apply(provider, req)

	require.Error(t, err)
	assert.False(t, provider.AllowPrivateNetwork)

	svc.ai = nil
	require.NoError(t, svc.apply(provider, req), "an absent setting keeps self-hosted models working")
	assert.True(t, provider.AllowPrivateNetwork)
}
