package selfserviceservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/selfserviceservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	repositories.SelfServiceRepository

	policy     *worker.WorkerPolicy
	policies   []*worker.WorkerPolicy
	acks       []*worker.WorkerPolicyAcknowledgement
	ackCount   int
	saved      *worker.WorkerPolicy
	pending    []*worker.WorkerProfileChangeRequest
	request    *worker.WorkerProfileChangeRequest
	created    *worker.WorkerProfileChangeRequest
	updated    *worker.WorkerProfileChangeRequest
	compliance []repositories.PolicyComplianceRow
}

func (r *fakeRepo) GetPolicyByID(
	_ context.Context,
	_ *repositories.GetWorkerPolicyByIDRequest,
) (*worker.WorkerPolicy, error) {
	return r.policy, nil
}

func (r *fakeRepo) ListPolicies(
	_ context.Context,
	_ *repositories.ListWorkerPoliciesRequest,
) ([]*worker.WorkerPolicy, error) {
	return r.policies, nil
}

func (r *fakeRepo) UpdatePolicy(
	_ context.Context,
	entity *worker.WorkerPolicy,
) (*worker.WorkerPolicy, error) {
	r.saved = entity
	return entity, nil
}

func (r *fakeRepo) CountAcknowledgements(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ string,
) (int, error) {
	return r.ackCount, nil
}

func (r *fakeRepo) ListAcknowledgements(
	_ context.Context,
	req *repositories.ListPolicyAcknowledgementsRequest,
) ([]*worker.WorkerPolicyAcknowledgement, error) {
	out := make([]*worker.WorkerPolicyAcknowledgement, 0, len(r.acks))
	for _, ack := range r.acks {
		if !req.PolicyID.IsNil() && ack.PolicyID != req.PolicyID {
			continue
		}
		if req.VersionLabel != "" && ack.VersionLabel != req.VersionLabel {
			continue
		}
		out = append(out, ack)
	}
	return out, nil
}

func (r *fakeRepo) CreateAcknowledgement(
	_ context.Context,
	entity *worker.WorkerPolicyAcknowledgement,
) (*worker.WorkerPolicyAcknowledgement, error) {
	entity.ID = pulid.MustNew("wpak_")
	r.acks = append(r.acks, entity)
	return entity, nil
}

func (r *fakeRepo) PolicyCompliance(
	_ context.Context,
	_ *repositories.PolicyComplianceRequest,
) ([]repositories.PolicyComplianceRow, error) {
	return r.compliance, nil
}

func (r *fakeRepo) ListChangeRequests(
	_ context.Context,
	_ *repositories.ListProfileChangeRequestsRequest,
) ([]*worker.WorkerProfileChangeRequest, error) {
	return r.pending, nil
}

func (r *fakeRepo) GetChangeRequestByID(
	_ context.Context,
	_ *repositories.GetProfileChangeRequestByIDRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	return r.request, nil
}

func (r *fakeRepo) CreateChangeRequest(
	_ context.Context,
	entity *worker.WorkerProfileChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	entity.ID = pulid.MustNew("wpcr_")
	r.created = entity
	return entity, nil
}

func (r *fakeRepo) UpdateChangeRequest(
	_ context.Context,
	entity *worker.WorkerProfileChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	r.updated = entity
	return entity, nil
}

type fakeWorkers struct {
	wrk     *worker.Worker
	written *worker.Worker
}

func (f *fakeWorkers) GetByID(
	_ context.Context,
	_ repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	return f.wrk, nil
}

func (f *fakeWorkers) Update(
	_ context.Context,
	entity *worker.Worker,
	_ *services.RequestActor,
) (*worker.Worker, error) {
	f.written = entity
	return entity, nil
}

type fakeChecksums struct{}

func (fakeChecksums) Checksum(_ context.Context, _ pagination.TenantInfo, _ pulid.ID) (string, error) {
	return "abc123", nil
}

func newService(repo *fakeRepo, workers *fakeWorkers) *selfserviceservice.Service {
	return selfserviceservice.NewWithDeps(selfserviceservice.Deps{
		Repo:          repo,
		WorkerRepo:    workers,
		WorkerUpdater: workers,
		Checksums:     fakeChecksums{},
	})
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func employee() *worker.Worker {
	return &worker.Worker{
		ID:        "wrk_1",
		FirstName: "Ada",
		LastName:  "Byron",
		Type:      worker.WorkerTypeEmployee,
		City:      "Springfield",
	}
}

func handbook() *worker.WorkerPolicy {
	return &worker.WorkerPolicy{
		ID:                pulid.MustNew("wpol_"),
		Status:            domaintypes.StatusActive,
		Code:              "HANDBOOK",
		Title:             "Driver handbook",
		Body:              "Be safe.",
		VersionLabel:      "1",
		RequiresSignature: true,
		AppliesTo:         worker.PolicyAudienceEmployees,
		EffectiveFrom:     1_700_000_000,
	}
}

// Every acknowledgement records the version it was given for. Rewriting the
// text beneath it would turn each into a signature on words nobody saw.
func TestUpdatePolicy_RefusesNewWordsUnderASignedVersion(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{policy: handbook(), ackCount: 12}
	revised := handbook()
	revised.ID = repo.policy.ID
	revised.Body = "Be safer."

	_, err := newService(repo, &fakeWorkers{}).UpdatePolicy(
		t.Context(),
		&selfserviceservice.PolicyRequest{Entity: revised, TenantInfo: tenant()},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "12 person(s) have signed version 1")
	assert.Nil(t, repo.saved)
}

func TestUpdatePolicy_NewWordsUnderANewVersionAreFine(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{policy: handbook(), ackCount: 12}
	revised := handbook()
	revised.ID = repo.policy.ID
	revised.Body = "Be safer."
	revised.VersionLabel = "2"

	updated, err := newService(repo, &fakeWorkers{}).UpdatePolicy(
		t.Context(),
		&selfserviceservice.PolicyRequest{Entity: revised, TenantInfo: tenant()},
	)

	require.NoError(t, err)
	assert.Equal(t, "2", updated.VersionLabel)
}

// The title is not words anybody signed, so it can move without a new version.
func TestUpdatePolicy_ATitleChangeNeedsNoNewVersion(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{policy: handbook(), ackCount: 12}
	revised := handbook()
	revised.ID = repo.policy.ID
	revised.Title = "Driver handbook (2026)"

	_, err := newService(repo, &fakeWorkers{}).UpdatePolicy(
		t.Context(),
		&selfserviceservice.PolicyRequest{Entity: revised, TenantInfo: tenant()},
	)

	require.NoError(t, err)
	require.NotNil(t, repo.saved)
}

// Being asked to sign a handbook that does not apply to you is worse than not
// seeing it.
func TestPoliciesForWorker_HidesPoliciesForOtherAudiences(t *testing.T) {
	t.Parallel()

	forContractors := handbook()
	forContractors.ID = pulid.MustNew("wpol_")
	forContractors.AppliesTo = worker.PolicyAudienceContractors
	repo := &fakeRepo{policies: []*worker.WorkerPolicy{handbook(), forContractors}}

	standings, err := newService(repo, &fakeWorkers{}).PoliciesForWorker(
		t.Context(),
		tenant(),
		employee(),
	)

	require.NoError(t, err)
	require.Len(t, standings, 1)
	assert.Equal(t, worker.PolicyAudienceEmployees, standings[0].Policy.AppliesTo)
	assert.True(t, standings[0].Outstanding())
}

func TestPoliciesForWorker_ANewVersionIsOutstandingAgain(t *testing.T) {
	t.Parallel()

	policy := handbook()
	policy.VersionLabel = "2"
	repo := &fakeRepo{
		policies: []*worker.WorkerPolicy{policy},
		acks: []*worker.WorkerPolicyAcknowledgement{{
			PolicyID:     policy.ID,
			WorkerID:     "wrk_1",
			VersionLabel: "1",
		}},
	}

	standings, err := newService(repo, &fakeWorkers{}).PoliciesForWorker(
		t.Context(),
		tenant(),
		employee(),
	)

	require.NoError(t, err)
	require.Len(t, standings, 1)
	assert.True(t, standings[0].Outstanding(), "the signature was on version 1")
}

func TestAcknowledge_RequiresTheWorkersOwnName(t *testing.T) {
	t.Parallel()

	t.Run("a blank signature is refused", func(t *testing.T) {
		t.Parallel()
		repo := &fakeRepo{policy: handbook()}

		_, err := newService(repo, &fakeWorkers{}).Acknowledge(
			t.Context(),
			&selfserviceservice.AcknowledgeRequest{
				PolicyID:   repo.policy.ID,
				Worker:     employee(),
				TenantInfo: tenant(),
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Type your full name")
	})

	// A signature is a claim that a specific person agreed, and "ok" is not a
	// person.
	t.Run("somebody else's name is refused", func(t *testing.T) {
		t.Parallel()
		repo := &fakeRepo{policy: handbook()}

		_, err := newService(repo, &fakeWorkers{}).Acknowledge(
			t.Context(),
			&selfserviceservice.AcknowledgeRequest{
				PolicyID:      repo.policy.ID,
				Worker:        employee(),
				SignatureName: "ok",
				TenantInfo:    tenant(),
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Sign as Ada Byron")
	})

	t.Run("case and spacing do not make a different person", func(t *testing.T) {
		t.Parallel()
		repo := &fakeRepo{policy: handbook()}

		ack, err := newService(repo, &fakeWorkers{}).Acknowledge(
			t.Context(),
			&selfserviceservice.AcknowledgeRequest{
				PolicyID:      repo.policy.ID,
				Worker:        employee(),
				SignatureName: "  ada   BYRON ",
				IP:            "10.0.0.1",
				UserAgent:     "Dash/1.0",
				TenantInfo:    tenant(),
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "ada   BYRON", ack.SignatureName)
		assert.Equal(t, "1", ack.VersionLabel)
		assert.Equal(t, "10.0.0.1", ack.SignatureIP)
		assert.Positive(t, ack.AcknowledgedAt)
	})
}

// Signing the same words twice adds nothing, and a second row would be the
// one an audit asks about.
func TestAcknowledge_SigningTwiceReturnsTheFirstSignature(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{policy: handbook()}
	service := newService(repo, &fakeWorkers{})
	req := &selfserviceservice.AcknowledgeRequest{
		PolicyID:      repo.policy.ID,
		Worker:        employee(),
		SignatureName: "Ada Byron",
		TenantInfo:    tenant(),
	}

	first, err := service.Acknowledge(t.Context(), req)
	require.NoError(t, err)
	second, err := service.Acknowledge(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Len(t, repo.acks, 1)
}

func TestAcknowledge_RefusesAPolicyThatDoesNotApply(t *testing.T) {
	t.Parallel()

	policy := handbook()
	policy.AppliesTo = worker.PolicyAudienceContractors
	repo := &fakeRepo{policy: policy}

	_, err := newService(repo, &fakeWorkers{}).Acknowledge(
		t.Context(),
		&selfserviceservice.AcknowledgeRequest{
			PolicyID:      policy.ID,
			Worker:        employee(),
			SignatureName: "Ada Byron",
			TenantInfo:    tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not apply to you")
}

// The document checksum ties the signature to the bytes, not just the row.
func TestAcknowledge_CopiesTheDocumentChecksum(t *testing.T) {
	t.Parallel()

	policy := handbook()
	policy.Body = ""
	policy.DocumentID = "doc_1"
	policy.RequiresSignature = false
	repo := &fakeRepo{policy: policy}

	ack, err := newService(repo, &fakeWorkers{}).Acknowledge(
		t.Context(),
		&selfserviceservice.AcknowledgeRequest{
			PolicyID:   policy.ID,
			Worker:     employee(),
			TenantInfo: tenant(),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "abc123", ack.DocumentChecksum)
	assert.False(t, ack.Signed(), "a read receipt, not a signature")
}

func TestPolicyCompliance_CountsBothSides(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{
		policy: handbook(),
		compliance: []repositories.PolicyComplianceRow{
			{WorkerID: "wrk_1", AcknowledgedAt: 1},
			{WorkerID: "wrk_2"},
			{WorkerID: "wrk_3"},
		},
	}

	view, err := newService(repo, &fakeWorkers{}).PolicyCompliance(
		t.Context(),
		tenant(),
		repo.policy.ID,
	)

	require.NoError(t, err)
	assert.Equal(t, 1, view.Signed)
	assert.Equal(t, 2, view.Outstanding)
}

func TestSubmitChange_RefusesARequestThatChangesNothing(t *testing.T) {
	t.Parallel()

	wrk := employee()
	repo := &fakeRepo{}

	_, err := newService(repo, &fakeWorkers{wrk: wrk}).SubmitChange(
		t.Context(),
		&selfserviceservice.SubmitChangeRequest{
			Worker:     wrk,
			Wanted:     worker.ContactSnapshotOf(wrk),
			TenantInfo: tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nothing would change")
	assert.Nil(t, repo.created)
}

// A manager deciding two versions of the same address will approve the wrong
// one.
func TestSubmitChange_RefusesASecondRequestWhileOneWaits(t *testing.T) {
	t.Parallel()

	wrk := employee()
	repo := &fakeRepo{pending: []*worker.WorkerProfileChangeRequest{{ID: "wpcr_1"}}}
	wanted := worker.ContactSnapshotOf(wrk)
	wanted.City = "Shelbyville"

	_, err := newService(repo, &fakeWorkers{wrk: wrk}).SubmitChange(
		t.Context(),
		&selfserviceservice.SubmitChangeRequest{Worker: wrk, Wanted: wanted, TenantInfo: tenant()},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already have a change waiting")
}

func TestSubmitChange_StoresOnlyWhatMoves(t *testing.T) {
	t.Parallel()

	wrk := employee()
	repo := &fakeRepo{}
	wanted := worker.ContactSnapshotOf(wrk)
	wanted.City = "Shelbyville"
	wanted.PhoneNumber = "555-0100"

	created, err := newService(repo, &fakeWorkers{wrk: wrk}).SubmitChange(
		t.Context(),
		&selfserviceservice.SubmitChangeRequest{
			Worker:     wrk,
			Wanted:     wanted,
			Note:       "Moved house",
			TenantInfo: tenant(),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, worker.ProfileChangePending, created.Status)
	require.Len(t, created.Changes, 2)
	assert.Equal(t, worker.ProfileFieldPhoneNumber, created.Changes[0].Field)
	assert.Equal(t, worker.ProfileFieldCity, created.Changes[1].Field)
	assert.Equal(t, "Springfield", created.Changes[1].From)
}

// An approval applies exactly the stored changes — not whatever the driver's
// record looks like now, and not whatever else was on the form.
func TestDecideChange_ApprovalWritesTheStoredChangesOntoTheWorker(t *testing.T) {
	t.Parallel()

	wrk := employee()
	workers := &fakeWorkers{wrk: wrk}
	repo := &fakeRepo{request: &worker.WorkerProfileChangeRequest{
		ID:          pulid.MustNew("wpcr_"),
		WorkerID:    "wrk_1",
		Status:      worker.ProfileChangePending,
		SubmittedAt: 1_800_000_000,
		Changes: []worker.FieldChange{
			{Field: worker.ProfileFieldCity, From: "Springfield", To: "Shelbyville"},
		},
	}}

	decided, err := newService(repo, workers).DecideChange(
		t.Context(),
		&selfserviceservice.DecideChangeRequest{
			ID:         repo.request.ID,
			Approve:    true,
			TenantInfo: tenant(),
			UserID:     pulid.MustNew("usr_"),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, worker.ProfileChangeApproved, decided.Status)
	require.NotNil(t, workers.written)
	assert.Equal(t, "Shelbyville", workers.written.City)
	require.NotNil(t, decided.DecidedAt)
}

func TestDecideChange_RejectionNeedsAReasonAndTouchesNothing(t *testing.T) {
	t.Parallel()

	workers := &fakeWorkers{wrk: employee()}
	repo := &fakeRepo{request: &worker.WorkerProfileChangeRequest{
		ID:          pulid.MustNew("wpcr_"),
		WorkerID:    "wrk_1",
		Status:      worker.ProfileChangePending,
		SubmittedAt: 1_800_000_000,
		Changes: []worker.FieldChange{
			{Field: worker.ProfileFieldCity, From: "Springfield", To: "Shelbyville"},
		},
	}}

	_, err := newService(repo, workers).DecideChange(
		t.Context(),
		&selfserviceservice.DecideChangeRequest{
			ID:         repo.request.ID,
			Approve:    false,
			TenantInfo: tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs a reason")
	assert.Nil(t, workers.written)
	assert.Nil(t, repo.updated)
}

func TestWithdrawChange_OnlyTheRequesterCan(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{request: &worker.WorkerProfileChangeRequest{
		ID:       pulid.MustNew("wpcr_"),
		WorkerID: "wrk_1",
		Status:   worker.ProfileChangePending,
	}}

	_, err := newService(repo, &fakeWorkers{}).WithdrawChange(
		t.Context(),
		&selfserviceservice.WithdrawChangeRequest{
			ID:            repo.request.ID,
			ActorWorkerID: "wrk_2",
			TenantInfo:    tenant(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your request")
}
