package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCredentialRepo struct {
	workerports.WorkerCredentialRepository

	captured *workerports.ListExpiringWorkerCredentialsRequest
	items    []*worker.WorkerCredential
}

func (f *fakeCredentialRepo) ListExpiring(
	_ context.Context,
	req *workerports.ListExpiringWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	f.captured = req

	return f.items, nil
}

type fakeCredentialReader struct {
	workerID        pulid.ID
	includeArchived bool
	listed          bool
	items           []*worker.WorkerCredential
	types           []*worker.WorkerCredentialType
}

func (f *fakeCredentialReader) ListForWorker(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	includeArchived bool,
) ([]*worker.WorkerCredential, error) {
	f.listed = true
	f.workerID = workerID
	f.includeArchived = includeArchived

	return f.items, nil
}

func (f *fakeCredentialReader) ListTypes(
	_ context.Context,
	_ *workerports.ListCredentialTypesRequest,
) (*pagination.CursorListResult[*worker.WorkerCredentialType], error) {
	return &pagination.CursorListResult[*worker.WorkerCredentialType]{Items: f.types}, nil
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

func heldCredential(code string, required bool) *worker.WorkerCredential {
	expires := int64(1_800_000_000)
	return &worker.WorkerCredential{
		ID:               pulid.MustNew("wcred_"),
		CredentialTypeID: pulid.MustNew("wctyp_"),
		Status:           worker.CredentialStatusActive,
		Number:           "TX-9981",
		ExpiresAt:        &expires,
		CredentialType: &worker.WorkerCredentialType{
			Code:       code,
			Name:       code,
			IsRequired: required,
		},
	}
}

type credentialFixture struct {
	repo   *fakeCredentialRepo
	reader *fakeCredentialReader
	tool   *listWorkerCredentialsTool
}

func newCredentialFixture(
	expiring []*worker.WorkerCredential,
	held []*worker.WorkerCredential,
) credentialFixture {
	repo := &fakeCredentialRepo{items: expiring}
	reader := &fakeCredentialReader{
		items: held,
		types: []*worker.WorkerCredentialType{{
			ID: pulid.MustNew("wctyp_"), Code: "MED_CARD", Name: "Medical Card", IsRequired: true,
		}},
	}
	tool := newListWorkerCredentialsTool(reader, repo, &fakePermissions{allowed: true})

	return credentialFixture{
		repo:   repo,
		reader: reader,
		tool:   tool.(*listWorkerCredentialsTool),
	}
}

/*
The question the assistant could not answer: "which drivers have a medical card
expiring in the next 30 days?" It is a filtered list over a date, which neither
worker_get nor the worker name search can express, so the model had nowhere to
go.

WorkerCredentialRepository.ListExpiring already answers it — the sweep that
mails compliance digests uses the same call — so the organization-wide answer
is a schema over an existing query rather than new SQL.
*/
func TestListWorkerCredentials_WindowsOnTheRequestedDays(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture([]*worker.WorkerCredential{expiringCredential("Maria", 1)}, nil)

	_, err := fx.tool.Query(t.Context(), testParams(map[string]any{"withinDays": float64(30)}))
	require.NoError(t, err)
	require.NotNil(t, fx.repo.captured)

	assert.Equal(t, 30, fx.repo.captured.HorizonDays)
	assert.Zero(t, fx.repo.captured.GraceDays, "already-expired rows are opt-in")
	assert.False(t, fx.reader.listed, "an expiry question is not one worker's full list")
}

func TestListWorkerCredentials_WithoutAWorkerListsWhatFallsDue(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	_, err := fx.tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, fx.repo.captured)

	assert.Equal(t, defaultExpiryHorizonDays, fx.repo.captured.HorizonDays)
	assert.True(t, fx.repo.captured.SoonestFirst, "the soonest expiry comes first")
	assert.True(t, fx.repo.captured.WorkerID.IsNil())
	assert.False(t, fx.reader.listed)
}

func TestListWorkerCredentials_ClampsTheHorizon(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	_, err := fx.tool.Query(t.Context(), testParams(map[string]any{"withinDays": float64(900)}))
	require.NoError(t, err)

	assert.Equal(t, maxExpiryHorizonDays, fx.repo.captured.HorizonDays,
		"a horizon past the bound is read as the bound, not as the default")
}

func TestListWorkerCredentials_IncludesExpiredOnRequest(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	_, err := fx.tool.Query(t.Context(), testParams(map[string]any{"includeExpired": true}))
	require.NoError(t, err)

	assert.Equal(t, expiredGraceDays, fx.repo.captured.GraceDays,
		"a lapsed card is what someone is usually asking about")
}

func TestListWorkerCredentials_RequiredOnlyReachesTheQuery(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	_, err := fx.tool.Query(t.Context(), testParams(map[string]any{"requiredOnly": true}))
	require.NoError(t, err)

	assert.True(t, fx.repo.captured.RequiredOnly)
}

/*
The type filter has to reach the query, not be applied to its result. Asking
for 25 expiring credentials and then keeping the medical cards among them
reports two when there are forty — the same shape of confidently wrong answer
this whole tool exists to prevent.
*/
func TestListWorkerCredentials_FiltersTypeInTheQuery(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	_, err := fx.tool.Query(
		t.Context(),
		testParams(map[string]any{"credentialTypeCode": "med_card"}),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"MED_CARD"}, fx.repo.captured.CredentialTypeCodes,
		"the code is normalized and pushed into the query")
}

func TestListWorkerCredentials_PagesWhatFallsDue(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture([]*worker.WorkerCredential{
		expiringCredential("Ana", 10),
		expiringCredential("Ben", 20),
		expiringCredential("Cal", 30),
	}, nil)

	result, err := fx.tool.Query(t.Context(), testParams(map[string]any{
		"limit":  float64(2),
		"offset": float64(4),
	}))
	require.NoError(t, err)

	assert.Equal(t, 3, fx.repo.captured.Limit, "one row past the page says whether more exist")
	assert.Equal(t, 4, fx.repo.captured.Offset)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Equal(t, 2, outcome.Count)
	assert.True(t, outcome.HasMore)
	require.NotNil(t, outcome.NextOffset)
	assert.Equal(t, 6, *outcome.NextOffset)
}

// The raw entity carries the worker's whole profile, the document and the
// verifying user. A model asked who needs a new medical card needs a name and a
// date; the rest is context window spent on nothing.
func TestListWorkerCredentials_ReturnsReadableFallingDueRows(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture([]*worker.WorkerCredential{expiringCredential("Maria", 99)}, nil)

	result, err := fx.tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	require.Equal(t, 1, outcome.Count)

	rows, ok := outcome.Items.([]expiringCredentialRow)
	require.True(t, ok, "rows are shaped for reading, not the raw entity")
	assert.Equal(t, "Maria Ortiz", rows[0].WorkerName)
	assert.Equal(t, "Medical Card", rows[0].CredentialType)
	assert.NotEmpty(t, rows[0].WorkerID)
	assert.Equal(t, fx.repo.items[0].ID.String(), rows[0].ID,
		"worker_get_credential takes the credential's id, so the row has to carry it")
}

func TestListWorkerCredentials_EmptyResultSaysWhatItLookedFor(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	result, err := fx.tool.Query(
		t.Context(),
		testParams(map[string]any{"withinDays": float64(30), "credentialTypeCode": "MED_CARD"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note)
}

func TestListWorkerCredentials_OneWorkerReturnsEveryCredentialAndTheTrackedTypes(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	fx := newCredentialFixture(nil, []*worker.WorkerCredential{
		heldCredential("MED_CARD", true),
		heldCredential("TWIC", false),
	})

	result, err := fx.tool.Query(t.Context(), testParams(map[string]any{
		"workerId":        workerID.String(),
		"includeArchived": true,
	}))
	require.NoError(t, err)
	assert.Nil(t, fx.repo.captured, "one worker's full list is not the expiry walk")
	assert.Equal(t, workerID, fx.reader.workerID)
	assert.True(t, fx.reader.includeArchived)

	outcome, ok := result.(*workerCredentialsOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]credentialRow)
	require.True(t, ok)
	require.Len(t, rows, 2)
	assert.Equal(t, "MED_CARD", rows[0].CredentialCode)
	assert.NotEmpty(t, rows[0].CredentialTypeID)
	require.Len(t, outcome.Types, 1)
	assert.True(t, outcome.Types[0].IsRequired)
	assert.False(t, outcome.HasMore)
}

func TestListWorkerCredentials_OneWorkerNarrowsByTypeAndRequired(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	held := []*worker.WorkerCredential{
		heldCredential("MED_CARD", true),
		heldCredential("TWIC", false),
		heldCredential("CDL", true),
	}

	fx := newCredentialFixture(nil, held)
	result, err := fx.tool.Query(t.Context(), testParams(map[string]any{
		"workerId":           workerID,
		"credentialTypeCode": "twic",
	}))
	require.NoError(t, err)
	rows := result.(*workerCredentialsOutcome).Items.([]credentialRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "TWIC", rows[0].CredentialCode)

	fx = newCredentialFixture(nil, held)
	result, err = fx.tool.Query(t.Context(), testParams(map[string]any{
		"workerId":     workerID,
		"requiredOnly": true,
	}))
	require.NoError(t, err)
	rows = result.(*workerCredentialsOutcome).Items.([]credentialRow)
	require.Len(t, rows, 2)
	for _, row := range rows {
		assert.NotEqual(t, "TWIC", row.CredentialCode)
	}
}

func TestListWorkerCredentials_OneWorkerPages(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, []*worker.WorkerCredential{
		heldCredential("MED_CARD", true),
		heldCredential("TWIC", false),
		heldCredential("CDL", true),
	})

	result, err := fx.tool.Query(t.Context(), testParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
		"limit":    float64(2),
	}))
	require.NoError(t, err)

	outcome := result.(*workerCredentialsOutcome)
	assert.Equal(t, 2, outcome.Count)
	assert.True(t, outcome.HasMore)
	require.NotNil(t, outcome.NextOffset)
	assert.Equal(t, 2, *outcome.NextOffset)
}

func TestListWorkerCredentials_OneWorkerWithNothingOnFileSaysSo(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	result, err := fx.tool.Query(t.Context(), testParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
	}))
	require.NoError(t, err)

	outcome := result.(*workerCredentialsOutcome)
	assert.Zero(t, outcome.Count)
	assert.Contains(t, outcome.Note, "worse than one expiring soon")

	encoded, err := sonic.Marshal(outcome)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"items":[]`)
	assert.Contains(t, string(encoded), `"credentialTypes"`)
}

func TestListWorkerCredentials_OneWorkerFallingDueNarrowsTheExpiryWalk(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	fx := newCredentialFixture([]*worker.WorkerCredential{expiringCredential("Maria", 5)}, nil)

	result, err := fx.tool.Query(t.Context(), testParams(map[string]any{
		"workerId":       workerID.String(),
		"withinDays":     float64(60),
		"includeExpired": true,
	}))
	require.NoError(t, err)
	require.NotNil(t, fx.repo.captured)
	assert.False(t, fx.reader.listed)

	assert.Equal(t, workerID, fx.repo.captured.WorkerID)
	assert.Equal(t, 60, fx.repo.captured.HorizonDays)
	assert.Equal(t, expiredGraceDays, fx.repo.captured.GraceDays)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Equal(t, 1, outcome.Count)
	assert.Contains(t, outcome.SearchedFor, "worker "+workerID.String())
}

func TestListWorkerCredentials_RefusesAWorkerThatIsNotAnID(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	_, err := fx.tool.Query(t.Context(), testParams(map[string]any{"workerId": "Maria Ortiz"}))
	require.Error(t, err)
	assert.Nil(t, fx.repo.captured)
	assert.False(t, fx.reader.listed)
}

func TestListWorkerCredentials_WithholdsTheNumberBelowTheCeiling(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, []*worker.WorkerCredential{heldCredential("CDL", true)})

	result, err := fx.tool.Query(t.Context(), agentParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
	}, permission.SensitivityInternal))
	require.NoError(t, err)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "TX-9981")
}

/*
ListExpiring crosses every tenant when handed a zero TenantInfo — that is what
the nightly sweep needs, and it is a loaded gun pointed at a tool whose caller
is a language model. The tenant comes from the actor, never from the model, and
the guard runs before anything else.
*/
func TestListWorkerCredentials_NeverQueriesWithoutATenant(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	params := testParams(map[string]any{})
	_, err := fx.tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, params.OrganizationID, fx.repo.captured.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, fx.repo.captured.TenantInfo.BuID)
	assert.False(t, fx.repo.captured.TenantInfo.OrgID.IsNil(), "a nil org would read every tenant")
}

func TestListWorkerCredentials_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)

	params := testParams(map[string]any{})
	params.Actor.BusinessUnitID = pulid.MustNew("bu_")

	_, err := fx.tool.Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, fx.repo.captured)
}

// A credential is its own resource, with its own sensitivities, and the
// application gates the credential pages on it. The tool that lists them is
// gated the same way, so a person who may read workers but not their
// credentials is not shown expiry dates in the chat that the page refuses.
func TestListWorkerCredentials_IsGatedOnTheCredentialNotTheWorker(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)
	assert.Equal(t, toolWorkerListCredentials, fx.tool.Name())
	assert.Equal(t, permission.ResourceWorkerCredential, fx.tool.Policy().Resource)
	assert.True(t, permission.IsAgentAllowed(fx.tool.Policy().Resource, permission.OpRead))
}

func TestListWorkerCredentials_SchemaIsTheUnionOfBothReads(t *testing.T) {
	t.Parallel()

	fx := newCredentialFixture(nil, nil)
	schema := fx.tool.ParamSchema()
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	for _, name := range []string{
		"workerId", "withinDays", "includeExpired", "credentialTypeCode", "requiredOnly",
		"includeArchived", "limit", "offset",
	} {
		assert.Contains(t, properties, name)
	}
	assert.NotContains(t, schema, "required", "workerId narrows; it is not required")
	assert.Contains(t, fx.tool.Description(), "worse than expiring soon")
}
