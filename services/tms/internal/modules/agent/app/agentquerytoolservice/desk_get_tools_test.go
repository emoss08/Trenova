package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A licence number is Confidential. The credential's operation never selects
// it, so neither a person nor an agent principal is shown it.
func TestGetWorkerCredential_NeverReadsTheNumber(t *testing.T) {
	t.Parallel()

	read := graphQLReads[toolWorkerGetCredential]
	for _, field := range read.Fields {
		if field.Name == "number" {
			assert.False(t, field.Selected, "the credential number is never selected")
		}
	}

	spec := workerCredentialGetSpec()
	tool, err := newGraphQLGetTool(&spec, &keyedReader{}, &fakePermissions{})
	require.NoError(t, err)
	assert.Equal(t, permission.ResourceWorkerCredential, tool.Policy().Resource)
	assert.Equal(t, "worker_get_credential", tool.Name())
	assert.Contains(t, tool.Description(), "worker_list_credentials")
}

func TestDeskGetTools_AreGatedOnTheirRecordsResource(t *testing.T) {
	t.Parallel()

	for _, want := range []struct {
		spec     getSpec
		resource permission.Resource
	}{
		{carrierIntelEventGetSpec(), permission.ResourceCarrierIntelligence},
		{detentionOccurrenceGetSpec(), permission.ResourceDetention},
		{serviceFailureGetSpec(), permission.ResourceServiceFailure},
		{orderGetSpec(), permission.ResourceOrder},
	} {
		built, resource := want.spec, want.resource
		tool, err := newGraphQLGetTool(&built, &keyedReader{}, nil)
		require.NoError(t, err, built.name)
		assert.Equal(t, resource, tool.Policy().Resource, built.name)
	}
	assert.Equal(
		t,
		permission.ResourceAgentRun,
		newGetAgentRunTool(getAgentRunParams{}, nil).Policy().Resource,
	)
}
