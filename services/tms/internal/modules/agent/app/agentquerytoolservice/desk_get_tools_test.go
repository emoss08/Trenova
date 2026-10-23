package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/shared/pulid"
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

	intelSpec := carrierIntelEventGetSpec()
	intel, err := newGraphQLGetTool(&intelSpec, &keyedReader{}, nil)
	require.NoError(t, err)

	assert.Equal(
		t,
		permission.ResourceDetention,
		newGetDetentionOccurrenceTool(nil, nil, nil).Policy().Resource,
	)
	assert.Equal(
		t,
		permission.ResourceCarrierIntelligence,
		intel.Policy().Resource,
	)
	assert.Equal(
		t,
		permission.ResourceAgentRun,
		newGetAgentRunTool(getAgentRunParams{}, nil).Policy().Resource,
	)
	assert.Equal(
		t,
		permission.ResourceServiceFailure,
		newGetServiceFailureTool(nil, nil).Policy().Resource,
	)
}

type stubServiceFailures struct {
	servicefailureports.ServiceFailureRepository

	entity *servicefailure.ServiceFailure
}

func (s *stubServiceFailures) GetByID(
	_ context.Context,
	_ *servicefailureports.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	return s.entity, nil
}

func TestGetServiceFailure_NamesThePeopleWithoutReachingThem(t *testing.T) {
	t.Parallel()

	person := func(name, email string) *tenant.User {
		return &tenant.User{
			ID:           pulid.MustNew(tenant.UserIDPrefix),
			Name:         name,
			Username:     email[:5],
			EmailAddress: email,
			Timezone:     "America/Chicago",
		}
	}
	entity := &servicefailure.ServiceFailure{
		ID:         pulid.MustNew("sf_"),
		Notes:      "Consignee closed early",
		CreatedBy:  person("Avery Chen", "avery.chen@example.com"),
		ReviewedBy: person("Morgan Reyes", "morgan.reyes@example.com"),
		ResolvedBy: person("Jordan Blake", "jordan.blake@example.com"),
		VoidedBy:   person("Riley Stone", "riley.stone@example.com"),
	}
	tool := newGetServiceFailureTool(&stubServiceFailures{entity: entity}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"serviceFailureId": entity.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)
	document := encodedDocument(t, result)

	assert.Equal(t, "Consignee closed early", document["notes"])
	for key, user := range map[string]*tenant.User{
		"createdBy":  entity.CreatedBy,
		"reviewedBy": entity.ReviewedBy,
		"resolvedBy": entity.ResolvedBy,
		"voidedBy":   entity.VoidedBy,
	} {
		assert.Equal(t, map[string]any{"id": user.ID.String(), "name": user.Name},
			document[key], key)
	}
	assert.NotContains(t, document, "withheldByAccess",
		"a user reduced to a name is not a withheld field")
}
