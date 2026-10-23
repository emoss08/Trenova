package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubWorkerCredentials struct {
	workerports.WorkerCredentialRepository

	credential *worker.WorkerCredential
	last       *workerports.GetWorkerCredentialByIDRequest
}

func (s *stubWorkerCredentials) GetByID(
	_ context.Context,
	req *workerports.GetWorkerCredentialByIDRequest,
) (*worker.WorkerCredential, error) {
	s.last = req

	return s.credential, nil
}

func sampleCredential() *worker.WorkerCredential {
	expires := int64(1_800_000_000)
	return &worker.WorkerCredential{
		ID:               pulid.MustNew("wcred_"),
		WorkerID:         pulid.MustNew("wrk_"),
		CredentialTypeID: pulid.MustNew("wct_"),
		Status:           worker.CredentialStatusActive,
		Number:           "D1234567",
		IssuingAuthority: "TX DPS",
		ExpiresAt:        &expires,
		CredentialType: &worker.WorkerCredentialType{
			Name:     "CDL",
			Category: worker.CredentialCategoryLicense,
		},
		Worker: &worker.Worker{FirstName: "Maria", LastName: "Ortiz"},
	}
}

// A licence number is Confidential. Neither a person nor an agent principal is
// shown it; both are told it was withheld rather than shown a credential with
// no number on file.
func TestGetWorkerCredential_WithholdsTheNumberFromEveryone(t *testing.T) {
	t.Parallel()

	repo := &stubWorkerCredentials{credential: sampleCredential()}
	tool := newGetWorkerCredentialTool(repo, &fakePermissions{})

	params := testParams(map[string]any{"credentialId": repo.credential.ID.String()})
	params.Actor.PrincipalType = permissionports.PrincipalTypeUser
	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	detail := result.(workerCredentialDetail)
	assert.Empty(t, detail.Number)
	assert.Equal(t, []string{"number"}, detail.Withheld)
	assert.Equal(t, "Maria Ortiz", detail.WorkerName)
	assert.Equal(t, "CDL", detail.CredentialType)
	assert.True(t, repo.last.IncludeType)
	assert.True(t, repo.last.IncludeWorker)

	agentParams := testParams(map[string]any{"credentialId": repo.credential.ID.String()})
	agentParams.Actor.PrincipalType = permissionports.PrincipalTypeAgent
	result, err = tool.Query(t.Context(), agentParams)
	require.NoError(t, err)
	detail = result.(workerCredentialDetail)
	assert.Empty(t, detail.Number)
	assert.Equal(t, []string{"number"}, detail.Withheld)
	assert.Equal(t, "Maria Ortiz", detail.WorkerName, "the holder's name is Internal and stays")
}

func TestGetWorkerCredential_IsGatedOnTheCredentialResource(t *testing.T) {
	t.Parallel()

	tool := newGetWorkerCredentialTool(&stubWorkerCredentials{}, &fakePermissions{})
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
