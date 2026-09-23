package aiproviderservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeProviderRepo struct {
	repositories.AIProviderRepository

	provider *aiprovider.Provider
	marked   []repositories.MarkAIProviderTestedRequest
	markErr  error
}

func (f *fakeProviderRepo) GetByID(
	_ context.Context,
	_ repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
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

func TestTestRecordsProbeOutcomeOnProvider(t *testing.T) {
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

	result, err := svc.Test(t.Context(), repositories.GetAIProviderByIDRequest{
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

func TestTestStillReturnsProbeResultWhenRecordingFails(t *testing.T) {
	repo := &fakeProviderRepo{markErr: errors.New("db down")}
	prober := &fakeProber{
		result: &services.TestAIProviderResult{Success: false, Message: "Refused"},
	}
	svc := newTestService(repo, prober)
	repo.provider = testProvider(t, svc)

	result, err := svc.Test(t.Context(), repositories.GetAIProviderByIDRequest{
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
