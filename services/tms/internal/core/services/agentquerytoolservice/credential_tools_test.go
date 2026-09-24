package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCredentialRepo struct {
	repositories.WorkerCredentialRepository

	captured *repositories.ListExpiringWorkerCredentialsRequest
	items    []*worker.WorkerCredential
}

func (f *fakeCredentialRepo) ListExpiring(
	_ context.Context,
	req *repositories.ListExpiringWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	f.captured = req

	return f.items, nil
}

func expiringCredential(name string, expiresAt int64) *worker.WorkerCredential {
	return &worker.WorkerCredential{
		ID:        pulid.MustNew("wcred_"),
		WorkerID:  pulid.MustNew("wrk_"),
		ExpiresAt: &expiresAt,
		CredentialType: &worker.WorkerCredentialType{
			Code: "MED_CARD",
			Name: "Medical Card",
		},
		Worker: &worker.Worker{FirstName: name, LastName: "Ortiz"},
	}
}

/*
The question the assistant could not answer: "which drivers have a medical card
expiring in the next 30 days?" It is a filtered list over a date, which neither
get_worker nor search_worker can express, so the model had nowhere to go.

WorkerCredentialRepository.ListExpiring already answers it — the sweep that
mails compliance digests uses the same call — so this tool is a schema over an
existing query rather than new SQL.
*/
func TestListExpiringCredentials_WindowsOnTheRequestedDays(t *testing.T) {
	t.Parallel()

	repo := &fakeCredentialRepo{items: []*worker.WorkerCredential{expiringCredential("Maria", 1)}}
	tool := newListExpiringCredentialsTool(repo)

	_, err := tool.Query(t.Context(), testParams(map[string]any{"withinDays": float64(30)}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured)

	assert.Equal(t, 30, repo.captured.HorizonDays)
	assert.Zero(t, repo.captured.GraceDays, "already-expired rows are opt-in")
}

func TestListExpiringCredentials_IncludesExpiredOnRequest(t *testing.T) {
	t.Parallel()

	repo := &fakeCredentialRepo{}
	tool := newListExpiringCredentialsTool(repo)

	_, err := tool.Query(t.Context(), testParams(map[string]any{"includeExpired": true}))
	require.NoError(t, err)

	assert.Positive(
		t,
		repo.captured.GraceDays,
		"a lapsed card is what someone is usually asking about",
	)
}

/*
The type filter has to reach the query, not be applied to its result. Asking
for 25 expiring credentials and then keeping the medical cards among them
reports two when there are forty — the same shape of confidently wrong answer
this whole tool exists to prevent.
*/
func TestListExpiringCredentials_FiltersTypeInTheQuery(t *testing.T) {
	t.Parallel()

	repo := &fakeCredentialRepo{}
	tool := newListExpiringCredentialsTool(repo)

	_, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"credentialTypeCode": "med_card"}),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"MED_CARD"}, repo.captured.CredentialTypeCodes,
		"the code is normalized and pushed into the query")
}

// The raw entity carries the worker's whole profile, the document and the
// verifying user. A model asked who needs a new medical card needs a name and a
// date; the rest is context window spent on nothing.
func TestListExpiringCredentials_ReturnsReadableRows(t *testing.T) {
	t.Parallel()

	repo := &fakeCredentialRepo{items: []*worker.WorkerCredential{expiringCredential("Maria", 99)}}
	tool := newListExpiringCredentialsTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	require.Equal(t, 1, outcome.Count)

	rows, ok := outcome.Items.([]expiringCredentialRow)
	require.True(t, ok, "rows are shaped for reading, not the raw entity")
	assert.Equal(t, "Maria Ortiz", rows[0].WorkerName)
	assert.Equal(t, "Medical Card", rows[0].CredentialType)
	assert.NotEmpty(t, rows[0].WorkerID)
	assert.Equal(t, repo.items[0].ID.String(), rows[0].ID,
		"get_worker_credential takes the credential's id, so the row has to carry it")
}

func TestListExpiringCredentials_EmptyResultSaysWhatItLookedFor(t *testing.T) {
	t.Parallel()

	repo := &fakeCredentialRepo{items: nil}
	tool := newListExpiringCredentialsTool(repo)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"withinDays": float64(30), "credentialTypeCode": "MED_CARD"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note)
}

/*
ListExpiring crosses every tenant when handed a zero TenantInfo — that is what
the nightly sweep needs, and it is a loaded gun pointed at a tool whose caller
is a language model. The tenant comes from the actor, never from the model, and
the guard runs before anything else.
*/
func TestListExpiringCredentials_NeverQueriesWithoutATenant(t *testing.T) {
	t.Parallel()

	repo := &fakeCredentialRepo{}
	tool := newListExpiringCredentialsTool(repo)

	params := testParams(map[string]any{})
	_, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, params.OrganizationID, repo.captured.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, repo.captured.TenantInfo.BuID)
	assert.False(t, repo.captured.TenantInfo.OrgID.IsNil(), "a nil org would read every tenant")
}

func TestListExpiringCredentials_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	repo := &fakeCredentialRepo{}
	tool := newListExpiringCredentialsTool(repo)

	params := testParams(map[string]any{})
	params.Actor.BusinessUnitID = pulid.MustNew("bu_")

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, repo.captured)
}

// A credential is its own resource, with its own sensitivities, and the
// application gates the credential pages on it. The tool that lists them is
// gated the same way, so a person who may read workers but not their
// credentials is not shown expiry dates in the chat that the page refuses.
func TestListExpiringCredentials_IsGatedOnTheCredentialNotTheWorker(t *testing.T) {
	t.Parallel()

	tool := newListExpiringCredentialsTool(&fakeCredentialRepo{})
	assert.Equal(t, permission.ResourceWorkerCredential, tool.Policy().Resource)
	assert.True(t, permission.IsAgentAllowed(tool.Policy().Resource, permission.OpRead))
}
