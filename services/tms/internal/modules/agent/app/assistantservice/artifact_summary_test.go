package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListThreadArtifacts_AsksForSummariesOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	f := newWorkspaceFixture(t)

	_, err := f.svc.ListThreadArtifacts(t.Context(), f.req, assistantports.ListArtifactsOptions{
		Summary: true,
	})
	require.NoError(t, err)
	_, err = f.svc.ListThreadArtifacts(t.Context(), f.req, assistantports.ListArtifactsOptions{})
	require.NoError(t, err)

	require.Len(t, f.repo.paged, 2)
	assert.True(t, f.repo.paged[0].Summary)
	assert.False(t, f.repo.paged[1].Summary)
}
