package aiproviderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingWaker struct {
	woken []pagination.TenantInfo
}

func (w *recordingWaker) Wake(_ context.Context, tenant pagination.TenantInfo) {
	w.woken = append(w.woken, tenant)
}

func (r *chainRepo) Delete(
	_ context.Context,
	req aiproviderports.DeleteAIProviderRequest,
) error {
	delete(r.providers, req.ID)
	return nil
}

func routeService(repo *chainRepo) (*Service, *recordingWaker) {
	svc, _, _ := chainService(repo)
	waker := &recordingWaker{}
	svc.indexer = waker
	return svc, waker
}

var routeActor = &permissionports.RequestActor{UserID: pulid.ID("usr_1")}

func TestRoutesEmbedding(t *testing.T) {
	t.Parallel()

	chat := chainProvider("aip_chat", 10, aitask.TaskAssistantChat)
	embed := chainProvider("aip_embed", 20, aitask.TaskEmbedding)

	assert.True(t, RoutesEmbedding(chat, embed))
	assert.True(t, RoutesEmbedding(nil, embed))
	assert.False(t, RoutesEmbedding(chat))
	assert.False(t, RoutesEmbedding(nil))
	assert.False(t, RoutesEmbedding())
}

func TestDeletingTheEmbeddingRouteWakesTheIndexer(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(chainProvider("aip_embed", 10, aitask.TaskEmbedding))
	svc, waker := routeService(repo)

	err := svc.Delete(t.Context(), aiproviderports.DeleteAIProviderRequest{
		ID:         pulid.ID("aip_embed"),
		TenantInfo: chainTenant,
	}, routeActor)
	require.NoError(t, err)

	assert.Equal(t, []pagination.TenantInfo{chainTenant}, waker.woken,
		"with its own route gone, the organization falls back to the built-in service and reindexes")
}

func TestDeletingAnotherProviderLeavesTheIndexerAlone(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(chainProvider("aip_chat", 10, aitask.TaskAssistantChat))
	svc, waker := routeService(repo)

	err := svc.Delete(t.Context(), aiproviderports.DeleteAIProviderRequest{
		ID:         pulid.ID("aip_chat"),
		TenantInfo: chainTenant,
	}, routeActor)
	require.NoError(t, err)

	assert.Empty(t, waker.woken)
}

func TestAssigningEmbeddingWakesTheIndexer(t *testing.T) {
	t.Parallel()

	dims := 768
	provider := chainProvider("aip_embed", 10)
	provider.EmbeddingDimensions = &dims
	repo := newChainRepo(provider)
	svc, waker := routeService(repo)

	_, err := svc.AssignTask(t.Context(), &aiproviderports.AssignAIProviderTaskRequest{
		TenantInfo: chainTenant,
		ProviderID: pulid.ID("aip_embed"),
		Task:       aitask.TaskEmbedding,
	}, routeActor)
	require.NoError(t, err)

	assert.Equal(t, []pagination.TenantInfo{chainTenant}, waker.woken)
}

func TestAssigningAnotherTaskLeavesTheIndexerAlone(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(chainProvider("aip_chat", 10, aitask.TaskGeneral))
	svc, waker := routeService(repo)

	_, err := svc.AssignTask(t.Context(), &aiproviderports.AssignAIProviderTaskRequest{
		TenantInfo: chainTenant,
		ProviderID: pulid.ID("aip_chat"),
		Task:       aitask.TaskAssistantChat,
	}, routeActor)
	require.NoError(t, err)

	assert.Empty(t, waker.woken)
}

func TestReorderingEmbeddingProvidersWakesTheIndexer(t *testing.T) {
	t.Parallel()

	dims := 768
	first := chainProvider("aip_a", 10, aitask.TaskEmbedding)
	first.EmbeddingDimensions = &dims
	second := chainProvider("aip_b", 20, aitask.TaskEmbedding)
	second.EmbeddingDimensions = &dims
	repo := newChainRepo(first, second)
	svc, waker := routeService(repo)

	_, err := svc.Reorder(t.Context(), &aiproviderports.ReorderAIProvidersRequest{
		TenantInfo:  chainTenant,
		ProviderIDs: []pulid.ID{"aip_b", "aip_a"},
	}, routeActor)
	require.NoError(t, err)

	assert.Equal(t, []pagination.TenantInfo{chainTenant}, waker.woken,
		"the first embedding provider is the model the index uses, so its order is routing")
}

func TestAReorderThatMovesNothingLeavesTheIndexerAlone(t *testing.T) {
	t.Parallel()

	dims := 768
	provider := chainProvider("aip_a", 10, aitask.TaskEmbedding)
	provider.EmbeddingDimensions = &dims
	repo := newChainRepo(provider, chainProvider("aip_b", 20, aitask.TaskGeneral))
	svc, waker := routeService(repo)

	_, err := svc.Reorder(t.Context(), &aiproviderports.ReorderAIProvidersRequest{
		TenantInfo:  chainTenant,
		ProviderIDs: []pulid.ID{"aip_a", "aip_b"},
	}, routeActor)
	require.NoError(t, err)

	assert.Empty(t, waker.woken)
}

func TestRoutingEmbeddingToAnotherProviderWakesTheIndexer(t *testing.T) {
	t.Parallel()

	dims := 768
	target := chainProvider("aip_b", 5)
	target.EmbeddingDimensions = &dims
	current := chainProvider("aip_a", 10, aitask.TaskEmbedding)
	current.EmbeddingDimensions = &dims
	repo := newChainRepo(target, current)
	svc, waker := routeService(repo)

	_, err := svc.RouteTask(t.Context(), &aiproviderports.RouteAIProviderTaskRequest{
		TenantInfo: chainTenant,
		Task:       aitask.TaskEmbedding,
		ProviderID: pulid.ID("aip_b"),
	}, routeActor)
	require.NoError(t, err)

	assert.Equal(t, []pagination.TenantInfo{chainTenant}, waker.woken)
}

func TestRoutingAnotherTaskLeavesTheIndexerAlone(t *testing.T) {
	t.Parallel()

	repo := newChainRepo(
		chainProvider("aip_a", 10, aitask.TaskGeneral, aitask.TaskAssistantChat),
		chainProvider("aip_b", 20, aitask.TaskGeneral),
	)
	svc, waker := routeService(repo)

	_, err := svc.RouteTask(t.Context(), &aiproviderports.RouteAIProviderTaskRequest{
		TenantInfo: chainTenant,
		Task:       aitask.TaskAssistantChat,
		ProviderID: pulid.ID("aip_b"),
	}, routeActor)
	require.NoError(t, err)

	assert.Empty(t, waker.woken)
}
