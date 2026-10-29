package aiproviderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routingRepo struct {
	fakeProviderRepo
	enabled []*aiprovider.Provider
}

func (r *routingRepo) ListEnabled(
	context.Context,
	pagination.TenantInfo,
) ([]*aiprovider.Provider, error) {
	return r.enabled, nil
}

func TestRoutePreviewEditsTheStoredProvider(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	stored := testProvider(t, svc)
	stored.Enabled = true
	stored.Tasks = []aitask.Task{aitask.TaskAssistantChat}
	dims := 1536
	stored.EmbeddingDimensions = &dims
	svc.repo = &routingRepo{
		fakeProviderRepo: fakeProviderRepo{provider: stored},
		enabled:          []*aiprovider.Provider{stored},
	}

	routes, err := svc.RoutePreview(t.Context(), &aiproviderports.AIProviderRoutePreviewRequest{
		Draft: aiproviderports.AIProviderRoutingDraft{
			ID:      stored.ID,
			Name:    stored.Name,
			Kind:    stored.Kind,
			Tasks:   []aitask.Task{aitask.TaskDailyBriefing},
			Enabled: true,
		},
	})

	require.NoError(t, err)
	for _, route := range routes {
		switch route.Task {
		case aitask.TaskAssistantChat:
			require.NotNil(t, route.Before)
			assert.Nil(t, route.After, "the edit takes chat away from the only provider")
		case aitask.TaskDailyBriefing:
			assert.Nil(t, route.Before)
			require.NotNil(t, route.After)
			assert.True(t, route.After.Draft)
			assert.Equal(t, stored.ID, route.After.ProviderID)
		}
	}
	assert.NotEmpty(t, stored.APIKey, "the stored provider is not changed")
	assert.Equal(t, []aitask.Task{aitask.TaskAssistantChat}, stored.Tasks)
}
