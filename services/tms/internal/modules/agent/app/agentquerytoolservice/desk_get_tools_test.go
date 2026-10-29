package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/agentrunquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/operationsquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/partyquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workerquerytools"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A licence number is Confidential. The credential's operation never selects
// it, so neither a person nor an agent principal is shown it.
func TestGetWorkerCredential_NeverReadsTheNumber(t *testing.T) {
	t.Parallel()

	read := graphqlread.Reads[workerquerytools.ToolWorkerGetCredential]
	for _, field := range read.Fields {
		if field.Name == "number" {
			assert.False(t, field.Selected, "the credential number is never selected")
		}
	}

	spec := workerquerytools.CredentialGetSpec()
	tool, err := gettool.NewGraphQL(
		&spec,
		&querytooltest.KeyedReader{},
		&querytooltest.FakePermissions{},
	)
	require.NoError(t, err)
	assert.Equal(t, permission.ResourceWorkerCredential, tool.Policy().Resource)
	assert.Equal(t, "worker_get_credential", tool.Name())
	assert.Contains(t, tool.Description(), "worker_list_credentials")
}

func TestDeskGetTools_AreGatedOnTheirRecordsResource(t *testing.T) {
	t.Parallel()

	for _, want := range []struct {
		spec     readspec.Get
		resource permission.Resource
	}{
		{partyquerytools.CarrierIntelEventGetSpec(), permission.ResourceCarrierIntelligence},
		{operationsquerytools.DetentionOccurrenceGetSpec(), permission.ResourceDetention},
		{operationsquerytools.ServiceFailureGetSpec(), permission.ResourceServiceFailure},
		{operationsquerytools.OrderGetSpec(), permission.ResourceOrder},
	} {
		built, resource := want.spec, want.resource
		tool, err := gettool.NewGraphQL(&built, &querytooltest.KeyedReader{}, nil)
		require.NoError(t, err, built.Name)
		assert.Equal(t, resource, tool.Policy().Resource, built.Name)
	}
	assert.Equal(
		t,
		permission.ResourceAgentRun,
		agentrunquerytools.NewGetAgentRunTool(agentrunquerytools.GetAgentRunParams{}, nil).
			Policy().
			Resource,
	)
}
