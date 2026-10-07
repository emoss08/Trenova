package aiproviderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chainRepo struct {
	repositories.AIProviderRepository
	providers map[pulid.ID]*aiprovider.Provider
	order     []pulid.ID
	updated   []*aiprovider.Provider
}

func newChainRepo(providers ...*aiprovider.Provider) *chainRepo {
	repo := &chainRepo{providers: map[pulid.ID]*aiprovider.Provider{}}
	for _, provider := range providers {
		repo.providers[provider.ID] = provider
		repo.order = append(repo.order, provider.ID)
	}
	return repo
}

func (r *chainRepo) ListOrdered(context.Context, pagination.TenantInfo) ([]*aiprovider.Provider, error) {
	out := make([]*aiprovider.Provider, 0, len(r.order))
	for _, id := range r.order {
		copied := *r.providers[id]
		copied.APIKey = ""
		out = append(out, &copied)
	}
	return out, nil
}

func (r *chainRepo) GetByID(
	_ context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	provider, ok := r.providers[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("AIProvider not found")
	}
	copied := *provider
	return &copied, nil
}

func (r *chainRepo) Update(_ context.Context, entity *aiprovider.Provider) (*aiprovider.Provider, error) {
	saved := *entity
	saved.Version++
	r.providers[saved.ID] = &saved
	r.updated = append(r.updated, &saved)
	return &saved, nil
}

type auditLog struct {
	services.AuditService
	comments int
}

func (a *auditLog) LogAction(*services.LogActionParams, ...services.LogOption) error {
	a.comments++
	return nil
}

func chainProvider(id string, priority int, tasks ...aiprovider.Task) *aiprovider.Provider {
	return &aiprovider.Provider{
		ID:                   pulid.ID(id),
		OrganizationID:       pulid.ID("org_1"),
		BusinessUnitID:       pulid.ID("bu_1"),
		Name:                 id,
		Kind:                 aiprovider.KindOpenAIChat,
		BaseURL:              "https://api.example.com/v1",
		Model:                "model",
		APIKey:               "encrypted-" + id,
		StructuredOutputMode: aiprovider.StructuredOutputJSONMode,
		ReasoningEffort:      aiprovider.ReasoningOff,
		ThinkingStyle:        aiprovider.ThinkingStyleAuto,
		EmbeddingInputStyle:  aiprovider.EmbeddingInputStyleNone,
		MaxTokens:            8192,
		Priority:             priority,
		Tasks:                tasks,
		Enabled:              true,
		Version:              1,
	}
}

func chainService(repo *chainRepo) (*Service, *auditLog, *settingVersions) {
	audit := &auditLog{}
	versions := &settingVersions{}
	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	svc.repo = repo
	svc.db = dbtest.NopConnection{}
	svc.audit = audit
	svc.versions = versions
	return svc, audit, versions
}

var chainTenant = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}

func TestReorderWritesOnlyTheProvidersWhosePlaceChanged(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(
		chainProvider("aip_a", 10, aiprovider.TaskGeneral),
		chainProvider("aip_b", 20, aiprovider.TaskGeneral),
		chainProvider("aip_c", 30, aiprovider.TaskGeneral),
	)
	svc, audit, versions := chainService(repo)

	ordered, err := svc.Reorder(t.Context(), &services.ReorderAIProvidersRequest{
		TenantInfo:  chainTenant,
		ProviderIDs: []pulid.ID{"aip_a", "aip_c", "aip_b"},
	}, &services.RequestActor{UserID: pulid.ID("usr_1")})
	require.NoError(t, err)

	assert.Equal(t, 10, repo.providers["aip_a"].Priority)
	assert.Equal(t, 20, repo.providers["aip_c"].Priority)
	assert.Equal(t, 30, repo.providers["aip_b"].Priority)
	assert.Len(t, repo.updated, 2, "the first provider kept its place")
	assert.Len(t, versions.created, 2)
	assert.Equal(t, 2, audit.comments)
	assert.Equal(t, "encrypted-aip_c", repo.providers["aip_c"].APIKey, "the stored key survives the move")
	for _, provider := range ordered {
		assert.Empty(t, provider.APIKey)
	}
}

func TestReorderRefusesAnOrderThatDoesNotNameEveryProviderOnce(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(
		chainProvider("aip_a", 10, aiprovider.TaskGeneral),
		chainProvider("aip_b", 20, aiprovider.TaskGeneral),
	)
	svc, _, _ := chainService(repo)

	for name, ids := range map[string][]pulid.ID{
		"missing":   {"aip_a"},
		"duplicate": {"aip_a", "aip_a"},
		"unknown":   {"aip_a", "aip_x"},
	} {
		_, err := svc.Reorder(t.Context(), &services.ReorderAIProvidersRequest{
			TenantInfo: chainTenant, ProviderIDs: ids,
		}, nil)
		require.Error(t, err, name)
	}
	assert.Empty(t, repo.updated)
}

func TestAssignTaskAddsTheTaskOnce(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(chainProvider("aip_a", 10, aiprovider.TaskAssistantChat))
	svc, audit, _ := chainService(repo)

	saved, err := svc.AssignTask(t.Context(), &services.AssignAIProviderTaskRequest{
		TenantInfo: chainTenant, ProviderID: "aip_a", Task: aiprovider.TaskOperationalInsights,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t,
		[]aiprovider.Task{aiprovider.TaskAssistantChat, aiprovider.TaskOperationalInsights},
		saved.Tasks)
	assert.Empty(t, saved.APIKey)
	assert.Equal(t, 1, audit.comments)

	again, err := svc.AssignTask(t.Context(), &services.AssignAIProviderTaskRequest{
		TenantInfo: chainTenant, ProviderID: "aip_a", Task: aiprovider.TaskOperationalInsights,
	}, nil)
	require.NoError(t, err)
	assert.Len(t, again.Tasks, 2)
}

func TestAssignTaskRefusesAnUnknownTask(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(chainProvider("aip_a", 10, aiprovider.TaskGeneral))
	svc, _, _ := chainService(repo)

	_, err := svc.AssignTask(t.Context(), &services.AssignAIProviderTaskRequest{
		TenantInfo: chainTenant, ProviderID: "aip_a", Task: aiprovider.Task("Juggling"),
	}, nil)
	require.Error(t, err)
	assert.Empty(t, repo.updated)
}
