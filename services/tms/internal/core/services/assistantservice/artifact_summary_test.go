package assistantservice

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListThreadArtifacts_AsksForSummariesOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	f := newWorkspaceFixture(t)

	_, err := f.svc.ListThreadArtifacts(t.Context(), f.req, serviceports.ListArtifactsOptions{
		Summary: true,
	})
	require.NoError(t, err)
	_, err = f.svc.ListThreadArtifacts(t.Context(), f.req, serviceports.ListArtifactsOptions{})
	require.NoError(t, err)

	require.Len(t, f.repo.paged, 2)
	assert.True(t, f.repo.paged[0].Summary)
	assert.False(t, f.repo.paged[1].Summary)
}
