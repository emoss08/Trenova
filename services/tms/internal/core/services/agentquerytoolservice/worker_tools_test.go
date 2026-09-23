package agentquerytoolservice

import (
	"context"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWorkerRepo records the request a tool built and returns what it is told.
// It embeds the interface so the methods no tool calls stay unimplemented and
// a future addition to WorkerRepository does not break this file.
type fakeWorkerRepo struct {
	repositories.WorkerRepository

	captured *repositories.ListWorkersRequest
	items    []*worker.Worker
	err      error
}

func (f *fakeWorkerRepo) List(
	_ context.Context,
	req *repositories.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	f.captured = req
	if f.err != nil {
		return nil, f.err
	}

	return &pagination.CursorListResult[*worker.Worker]{Items: f.items}, nil
}

func testParams(params map[string]any) serviceports.QueryToolParams {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return serviceports.QueryToolParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &serviceports.RequestActor{
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		Params: params,
	}
}

/*
A model that omits an optional argument is not making a mistake, and a tool that
treats the omission as one turns a question into a dead end. This happened in
production: asked which drivers had a medical card expiring soon, the model
called this tool with no query, was told the parameter "must be a non-empty
string", searched for the literal word "driver" instead, matched nobody, and
told the customer their system had no drivers in it.

The repository never needed the term. ApplyCursorFilters only applies a text
search when Query is non-empty (pkg/querybuilder/cursor_sort.go), so an empty
one already means "everything, newest first" — the tool was rejecting a call the
layer beneath it would have served.
*/
func TestSearchWorker_ListsWithoutAQuery(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{{ID: pulid.MustNew("wrk_")}}}
	tool := newSearchWorkerTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured, "the repository should have been asked")

	assert.Empty(t, repo.captured.Filter.Query, "an absent query filters on nothing")
	assert.NotNil(t, result)
}

func TestSearchWorker_PassesAQueryThrough(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	tool := newSearchWorkerTool(repo)

	_, err := tool.Query(t.Context(), testParams(map[string]any{"query": "  Ortiz  "}))
	require.NoError(t, err)

	assert.Equal(t, "Ortiz", repo.captured.Filter.Query, "a query is trimmed, not dropped")
}

/*
An empty list and "there is no such data" are different answers, and the model
cannot tell them apart from a bare []. It said "there are no driver records
currently in the Trenova system" — a confident claim about the customer's
database, from a search that matched nothing.
*/
func TestSearchWorker_EmptyResultSaysWhatItSearched(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: nil}
	tool := newSearchWorkerTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "Ortiz"}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok, "an empty result must carry its terms, not be a bare slice")
	assert.Zero(t, outcome.Count)
	assert.Contains(t, strings.Join(outcome.SearchedFor, " "), "Ortiz")
	assert.NotEmpty(t, outcome.Note, "an empty result explains itself")
}

func TestSearchWorker_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{}
	tool := newSearchWorkerTool(repo)

	params := testParams(map[string]any{})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, repo.captured, "a mismatched actor never reaches the repository")
}

func TestSearchWorker_IsNamedForOneWorker(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "search_worker", newSearchWorkerTool(&fakeWorkerRepo{}).Name())
}

/*
The tool name is persisted in agent_definitions.tool_names and as a key in
tool_tiers, so a rename is a data change, not a symbol change. The migration
rewrites both columns, but a definition written by an API client that hard-coded
the old name would otherwise lose the tool silently — which is worse than an
error, because the agent simply stops being able to look drivers up.
*/
func TestRegistry_ResolvesTheLegacyWorkerSearchName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []serviceports.AgentQueryTool{newSearchWorkerTool(&fakeWorkerRepo{})},
	})

	tool, ok := reg.Get("search_workers")
	require.True(t, ok, "an agent saved before the rename still has its tool")
	assert.Equal(t, "search_worker", tool.Name())
}

func TestRegistry_DoesNotAdvertiseTheLegacyName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []serviceports.AgentQueryTool{newSearchWorkerTool(&fakeWorkerRepo{})},
	})

	for _, descriptor := range reg.Descriptors() {
		assert.NotEqual(t, "search_workers", descriptor.Name,
			"the alias is for reading old definitions, not for the model to call")
	}
}

/*
Search returns the same curated row the list tools return, not the stored
entity.

It used to return the entity. For a nearly empty worker that was 1,398 bytes
against 231 for the row — six times the context for the same answer, most of
it nulls nobody asked about: businessUnitId, stateId, profilePicUrl,
addressLine2, externalId, gender. Eight drivers overran the 12,000-character
tool-result cap, the result was cut mid-record, and the model announced it
would "see the remaining two workers from the truncated data" and invented
them.

The entity also reported an unrecorded credential as a bare null, which is the
reading that produced a wrong compliance answer.
*/
func TestSearchWorker_ReturnsTheCuratedRowNotTheStoredEntity(t *testing.T) {
	t.Parallel()

	expiry := int64(1791591001)
	repo := &fakeWorkerRepo{items: []*worker.Worker{{
		ID:        pulid.MustNew("wrk_"),
		FirstName: "Mike",
		LastName:  "Johnson",
		Profile:   &worker.WorkerProfile{MedicalCardExpiry: &expiry},
	}}}

	result, err := newSearchWorkerTool(repo).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]workerRow)
	require.True(t, ok, "search must return the curated row, not the entity")
	require.Len(t, rows, 1)

	assert.Equal(t, "Mike Johnson", rows[0].Name)

	encoded, err := sonic.Marshal(rows)
	require.NoError(t, err)
	for _, noise := range []string{"profilePicUrl", "addressLine2", "externalId", "gender"} {
		assert.NotContains(t, string(encoded), noise,
			"the entity's %s costs context and answers nothing", noise)
	}
}

// The absence semantics the rows carry have to reach search too, or the path
// the incident actually used is still the one that reports a missing
// certificate as nothing at all.
func TestSearchWorker_SaysWhenACredentialIsNotOnFile(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{{
		ID: pulid.MustNew("wrk_"), FirstName: "David", LastName: "Park",
		Profile: &worker.WorkerProfile{},
	}}}

	result, err := newSearchWorkerTool(repo).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, _ := result.(searchOutcome)
	encoded, err := sonic.Marshal(outcome.Items)
	require.NoError(t, err)

	assert.Contains(t, string(encoded), "none on file")
	assert.NotContains(t, string(encoded), "null")
}

/*
A NonCompliant driver is not dispatchable, and the row has to say so.

workers.can_be_assigned means "no employment-level block" — no termination, no
leave, no manual hold. The domain's own dispatch decision (worker.Standing)
additionally refuses a worker whose credentials are NonCompliant. The row
reported the column verbatim, so a driver whose medical card had lapsed four
days earlier came back as canBeAssigned: true, alongside
complianceStatus: NonCompliant. Both were in the same object, contradicting
each other, and the reader believed the friendlier one.
*/
func TestWorkerRow_DoesNotCallANonCompliantDriverAssignable(t *testing.T) {
	t.Parallel()

	lapsed := int64(1789516800)
	row := toWorkerRow(&worker.Worker{
		ID:            pulid.MustNew("wrk_"),
		FirstName:     "John",
		LastName:      "Smith",
		CanBeAssigned: true,
		Profile: &worker.WorkerProfile{
			ComplianceStatus:  worker.ComplianceStatusNonCompliant,
			MedicalCardExpiry: &lapsed,
		},
	})

	assert.False(t, row.CanBeAssigned,
		"the stored flag does not know about credentials; the domain does")
	assert.NotEmpty(t, row.AssignmentBlocked, "and it has to say why")
}

// A compliant driver keeps whatever the employment flag said, in both
// directions — this narrows the answer, it never widens it.
func TestWorkerRow_LeavesACompliantDriverAlone(t *testing.T) {
	t.Parallel()

	assignable := toWorkerRow(&worker.Worker{
		ID: pulid.MustNew("wrk_"), CanBeAssigned: true,
		Profile: &worker.WorkerProfile{ComplianceStatus: worker.ComplianceStatusCompliant},
	})
	assert.True(t, assignable.CanBeAssigned)

	held := toWorkerRow(&worker.Worker{
		ID: pulid.MustNew("wrk_"), CanBeAssigned: false,
		Profile: &worker.WorkerProfile{ComplianceStatus: worker.ComplianceStatusCompliant},
	})
	assert.False(t, held.CanBeAssigned)
}

// An existing employment reason is more specific than the compliance one and
// must not be overwritten by it.
func TestWorkerRow_KeepsAMoreSpecificBlockReason(t *testing.T) {
	t.Parallel()

	row := toWorkerRow(&worker.Worker{
		ID: pulid.MustNew("wrk_"), CanBeAssigned: false,
		AssignmentBlocked: "On unpaid leave",
		Profile: &worker.WorkerProfile{
			ComplianceStatus: worker.ComplianceStatusNonCompliant,
		},
	})

	assert.Equal(t, "On unpaid leave", row.AssignmentBlocked)
}

func (f *fakeWorkerRepo) GetByID(
	_ context.Context,
	_ repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.items[0], nil
}

func sensitiveWorker() *worker.Worker {
	termination := int64(1_790_000_000)
	return &worker.Worker{
		ID:                    pulid.MustNew("wrk_"),
		FirstName:             "Sarah",
		LastName:              "Williams",
		Status:                "Active",
		City:                  "Dallas",
		PostalCode:            "75201",
		Email:                 "sarah@example.com",
		PhoneNumber:           "214-555-0100",
		EmergencyContactName:  "Tom Williams",
		EmergencyContactPhone: "214-555-0101",
		Profile: &worker.WorkerProfile{
			DOB:                    500_000_000,
			LicenseNumber:          "TX-9981",
			TWICCardNumber:         "TWIC-1",
			MedicalExaminerNPI:     "1234567890",
			DrugAlcoholStatus:      "Prohibited",
			LastDrugTest:           1_780_000_000,
			HireDate:               1_600_000_000,
			TerminationDate:        &termination,
			LicenseExpiry:          1_800_000_000,
			ComplianceStatus:       worker.ComplianceStatusCompliant,
			IsQualified:            true,
			DisqualificationReason: "",
		},
	}
}

/*
Contract: internal/core/domain/permission/registry.go, ResourceWorker.
get_worker used to hand the model the stored entity whole — date of birth,
licence and TWIC numbers, drug and alcohol status, home address, emergency
contacts — under a plain worker:read grant. Confidential fields never reach
a model, whatever the role; Restricted ones follow the person's ceiling and
are named as withheld when it does not reach them, so their absence reads as
withheld rather than as not on file.
*/
func TestGetWorker_WithholdsConfidentialFieldsFromEveryone(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{sensitiveWorker()}}
	tool := newGetWorkerTool(repo, &fakePermissions{})
	params := testParams(map[string]any{"workerId": repo.items[0].ID.String()})
	params.Actor.PrincipalType = serviceports.PrincipalTypeUser

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	text := string(encoded)

	for _, secret := range []string{"TX-9981", "TWIC-1", "1234567890", "Prohibited", "500000000", "1780000000", "dob", "licenseNumber", "drugAlcohol"} {
		assert.NotContains(t, text, secret, "confidential data must never reach a model")
	}
	// A role that reaches Confidential is capped at Restricted: the address,
	// phone, email and dates are shown, the confidential fields still are not.
	assert.Contains(t, text, "sarah@example.com")
	assert.Contains(t, text, "Tom Williams")
	assert.Contains(t, text, "Dallas")
	assert.NotContains(t, text, "withheldByAccess")
}

func TestGetWorker_WithholdsRestrictedFieldsBelowTheRoleCeilingAndSaysSo(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{sensitiveWorker()}}
	perms := &fakePermissions{readable: map[string]*serviceports.ResourcePermissionDetail{
		"worker": {
			Resource:       "worker",
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityInternal,
		},
	}}
	tool := newGetWorkerTool(repo, perms)
	params := testParams(map[string]any{"workerId": repo.items[0].ID.String()})
	params.Actor.PrincipalType = serviceports.PrincipalTypeUser

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	row, ok := result.(workerDetailRow)
	require.True(t, ok)
	assert.Equal(t, "Sarah Williams", row.Name)
	assert.Empty(t, row.Email)
	assert.Empty(t, row.PhoneNumber)
	assert.Empty(t, row.Emergency)
	assert.Empty(t, row.City, "the city is Restricted too")
	assert.Contains(t, row.Withheld, "email")
	assert.Contains(t, row.Withheld, "hireDate")
	assert.NotContains(t, row.Withheld, "dob", "confidential fields are not even named")
	assert.Equal(t, "Compliant", row.ComplianceStatus, "internal fields still come through")
}

// An agent acting on its own, with no person's role behind it, reads at
// the Internal tier.
func TestGetWorker_AnAgentPrincipalReadsInternalOnly(t *testing.T) {
	t.Parallel()

	repo := &fakeWorkerRepo{items: []*worker.Worker{sensitiveWorker()}}
	tool := newGetWorkerTool(repo, &fakePermissions{})
	params := testParams(map[string]any{"workerId": repo.items[0].ID.String()})
	params.Actor.PrincipalType = serviceports.PrincipalTypeAgent
	params.Actor.UserID = pulid.Nil

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	row := result.(workerDetailRow)
	assert.Empty(t, row.Email)
	assert.Contains(t, row.Withheld, "email")
}

// A credential's number is the licence or card number itself. Who needs a
// new medical card is answered without it.
func TestExpiringCredentialRow_CarriesNoCredentialNumber(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.Marshal(expiringCredentialRow{WorkerID: "wrk_1", CredentialCode: "MED"})
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "number")
}
