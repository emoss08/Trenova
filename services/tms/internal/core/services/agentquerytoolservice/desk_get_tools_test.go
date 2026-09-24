package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCarrierIntelEvents struct {
	repositories.CarrierIntelEventRepository

	events []*carrierintel.CarrierIntelEvent
	asked  []pulid.ID
}

func (s *stubCarrierIntelEvents) GetByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*carrierintel.CarrierIntelEvent, error) {
	s.asked = ids

	return s.events, nil
}

// A batch reader answers a single get; an empty batch is a missing record,
// not an empty success the model would report as "nothing on file".
func TestGetCarrierIntelEvent_ReadsOneThroughTheBatchReader(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("cievt_")
	repo := &stubCarrierIntelEvents{
		events: []*carrierintel.CarrierIntelEvent{{ID: id, Summary: "Insurance lapsed"}},
	}
	tool := newGetCarrierIntelEventTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"eventId": id.String()}))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{id}, repo.asked)
	assert.Equal(t, "Insurance lapsed", result.(*carrierintel.CarrierIntelEvent).Summary)

	repo.events = nil
	_, err = tool.Query(t.Context(), testParams(map[string]any{"eventId": id.String()}))
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

type stubWorkerCredentials struct {
	repositories.WorkerCredentialRepository

	credential *worker.WorkerCredential
	last       *repositories.GetWorkerCredentialByIDRequest
}

func (s *stubWorkerCredentials) GetByID(
	_ context.Context,
	req *repositories.GetWorkerCredentialByIDRequest,
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

// A licence number is Restricted. A person whose role reaches it sees it; an
// agent principal, which reads at Internal, is told it was withheld rather
// than shown a credential with no number on file.
func TestGetWorkerCredential_ShowsTheNumberOnlyUnderAReachingCeiling(t *testing.T) {
	t.Parallel()

	repo := &stubWorkerCredentials{credential: sampleCredential()}
	tool := newGetWorkerCredentialTool(repo, &fakePermissions{})

	params := testParams(map[string]any{"credentialId": repo.credential.ID.String()})
	params.Actor.PrincipalType = serviceports.PrincipalTypeUser
	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	detail := result.(workerCredentialDetail)
	assert.Equal(t, "D1234567", detail.Number)
	assert.Empty(t, detail.Withheld)
	assert.Equal(t, "Maria Ortiz", detail.WorkerName)
	assert.Equal(t, "CDL", detail.CredentialType)
	assert.True(t, repo.last.IncludeType)
	assert.True(t, repo.last.IncludeWorker)

	agentParams := testParams(map[string]any{"credentialId": repo.credential.ID.String()})
	agentParams.Actor.PrincipalType = serviceports.PrincipalTypeAgent
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
	assert.Equal(t, "get_worker_credential", tool.Name())
	assert.Contains(t, tool.Description(), "list_expiring_credentials")
}

func TestDeskGetTools_AreGatedOnTheirRecordsResource(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		permission.ResourceDetentionPolicy,
		newGetDetentionOccurrenceTool(nil).Policy().Resource,
	)
	assert.Equal(
		t,
		permission.ResourceCarrierIntelligence,
		newGetCarrierIntelEventTool(nil).Policy().Resource,
	)
	assert.Equal(t, permission.ResourceAgentRun, newGetAgentRunTool(getAgentRunParams{}, nil).Policy().Resource)
	assert.Equal(
		t,
		permission.ResourceServiceFailure,
		newGetServiceFailureTool(nil).Policy().Resource,
	)
}
