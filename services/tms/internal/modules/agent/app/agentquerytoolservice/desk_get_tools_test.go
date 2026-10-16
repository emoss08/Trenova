package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCarrierIntelEvents struct {
	carrierintelports.CarrierIntelEventRepository

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
	tool := newGetCarrierIntelEventTool(repo, &fakePermissions{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{"eventId": id.String()}))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{id}, repo.asked)
	assert.Equal(t, "Insurance lapsed", encodedDocument(t, result)["summary"])

	repo.events = nil
	_, err = tool.Query(t.Context(), testParams(map[string]any{"eventId": id.String()}))
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

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
	assert.Equal(t, "get_worker_credential", tool.Name())
	assert.Contains(t, tool.Description(), "list_expiring_credentials")
}

func TestDeskGetTools_AreGatedOnTheirRecordsResource(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		permission.ResourceDetention,
		newGetDetentionOccurrenceTool(nil, nil, nil).Policy().Resource,
	)
	assert.Equal(
		t,
		permission.ResourceCarrierIntelligence,
		newGetCarrierIntelEventTool(nil, nil).Policy().Resource,
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
