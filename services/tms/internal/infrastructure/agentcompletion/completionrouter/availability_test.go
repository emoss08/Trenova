package completionrouter

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func availabilityTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestCanServe_TrueWhenAProviderServesTheTask(t *testing.T) {
	t.Parallel()

	provider := openAIChatProvider("documents", "http://127.0.0.1:1", 10)
	provider.Tasks = []aitask.Task{aitask.TaskDocumentExtraction}
	svc := newTestService(t, provider)

	ok, err := svc.CanServe(t.Context(), aitask.TaskDocumentExtraction, availabilityTenant())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCanServe_FalseWithoutAProviderForTheTask(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, openAIChatProvider("general", "http://127.0.0.1:1", 10))

	ok, err := svc.CanServe(t.Context(), aitask.TaskDocumentExtraction, availabilityTenant())
	require.NoError(t, err)
	assert.False(t, ok, "a provider that does not serve document work cannot read documents")
}

func TestCanServe_FalseWhenAIIsOff(t *testing.T) {
	t.Parallel()

	provider := openAIChatProvider("documents", "http://127.0.0.1:1", 10)
	provider.Tasks = []aitask.Task{aitask.TaskDocumentExtraction}
	svc := newTestService(t, provider)
	off := false
	svc.ai = &aiconfig.Config{Enabled: &off}

	ok, err := svc.CanServe(t.Context(), aitask.TaskDocumentExtraction, availabilityTenant())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCanServe_ReportsARepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	svc.repo = &fakeRepo{err: errors.New("database unavailable")}

	_, err := svc.CanServe(t.Context(), aitask.TaskDocumentExtraction, availabilityTenant())
	require.Error(t, err, "a failure to look is not the same as nothing found")
}
