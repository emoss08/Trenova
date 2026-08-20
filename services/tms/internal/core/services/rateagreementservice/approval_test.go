package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestApprove_ActivatesAgreementAndStampsApprover(t *testing.T) {
	t.Parallel()

	agreement := draftAgreement(rateagreement.StatusInReview)
	svc, repo := serviceFor(t, agreement)

	userID := pulid.MustNew("usr_")

	updated, err := svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  agreement.OrganizationID,
			BuID:   agreement.BusinessUnitID,
			UserID: userID,
		},
		EntityID: agreement.ID,
		Comment:  "Signed contract on file",
	})

	require.NoError(t, err)
	assert.Equal(t, rateagreement.StatusActive, updated.Status)
	require.NotNil(t, updated.ApprovedByID)
	assert.Equal(t, userID, *updated.ApprovedByID)
	require.NotNil(t, updated.ApprovedAt)
	assert.Equal(t, "Signed contract on file", updated.ReviewComment)

	repo.AssertExpectations(t)
}

// An illegal move must leave the agreement exactly as it was, which means it
// must be refused before anything is written.
func TestApprove_RefusesFromTheWrongStatus(t *testing.T) {
	t.Parallel()

	agreement := draftAgreement(rateagreement.StatusDraft)

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateAgreementByIDRequest")).
		Return(agreement, nil).
		Once()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: mocks.NewMockAuditService(t),
	})

	_, err := svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: pagination.TenantInfo{OrgID: agreement.OrganizationID},
		EntityID:   agreement.ID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot transition rate agreement status")
	assert.Equal(t, rateagreement.StatusDraft, agreement.Status,
		"a refused transition must not have written anything")
}

// Sending a contract back without saying why wastes the next person's time, and
// by then the reviewer has moved on.
func TestReject_RequiresAReason(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRateAgreementRepository(t)

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: mocks.NewMockAuditService(t),
	})

	_, err := svc.Reject(t.Context(), &ApprovalActionRequest{
		EntityID: pulid.MustNew("ragr_"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "comment is required")
}

func TestReject_ClearsTheSubmission(t *testing.T) {
	t.Parallel()

	agreement := draftAgreement(rateagreement.StatusInReview)
	submitter := pulid.MustNew("usr_")
	submittedAt := int64(500)
	agreement.SubmittedByID = &submitter
	agreement.SubmittedAt = &submittedAt

	svc, _ := serviceFor(t, agreement)

	updated, err := svc.Reject(t.Context(), &ApprovalActionRequest{
		TenantInfo: pagination.TenantInfo{OrgID: agreement.OrganizationID},
		EntityID:   agreement.ID,
		Comment:    "The fuel terms do not match the signed contract",
	})

	require.NoError(t, err)
	assert.Equal(t, rateagreement.StatusDraft, updated.Status)
	assert.Nil(t, updated.SubmittedByID, "a rejected agreement is no longer submitted")
	assert.Nil(t, updated.SubmittedAt)
	assert.Equal(t, "The fuel terms do not match the signed contract", updated.ReviewComment)
}

// Suspension is the lever for a customer on credit hold, so resuming must not
// require another approval round.
func TestSuspendThenResume_RoundTripsWithoutReapproval(t *testing.T) {
	t.Parallel()

	agreement := draftAgreement(rateagreement.StatusActive)
	svc, _ := serviceFor(t, agreement)

	tenantInfo := pagination.TenantInfo{OrgID: agreement.OrganizationID}

	suspended, err := svc.Suspend(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenantInfo,
		EntityID:   agreement.ID,
		Comment:    "Customer on credit hold",
	})
	require.NoError(t, err)
	assert.Equal(t, rateagreement.StatusSuspended, suspended.Status)

	resumed, err := svc.Resume(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenantInfo,
		EntityID:   agreement.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, rateagreement.StatusActive, resumed.Status)
}

// A new agreement starts in Draft whatever the payload says, so creating one
// already active cannot route around the review the organization asked for.
func TestCreate_AlwaysStartsInDraft(t *testing.T) {
	t.Parallel()

	agreement := draftAgreement(rateagreement.StatusActive)
	agreement.ID = pulid.Nil

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreement")).
		RunAndReturn(func(
			_ context.Context,
			entity *rateagreement.RateAgreement,
		) (*rateagreement.RateAgreement, error) {
			return entity, nil
		}).
		Once()
	repo.EXPECT().
		CreateVersion(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreementVersion")).
		RunAndReturn(func(
			_ context.Context,
			version *rateagreement.RateAgreementVersion,
		) (*rateagreement.RateAgreementVersion, error) {
			return version, nil
		}).
		Once()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: audit,
	})

	created, err := svc.Create(t.Context(), agreement, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.Equal(t, rateagreement.StatusDraft, created.Status)
}

func draftAgreement(status rateagreement.Status) *rateagreement.RateAgreement {
	customerID := pulid.MustNew("cus_")

	return &rateagreement.RateAgreement{
		ID:             pulid.MustNew("ragr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		PartyType:      rateagreement.PartyTypeCustomer,
		CustomerID:     &customerID,
		Code:           "ACME-2026",
		Name:           "Acme Freight Agreement",
		AgreementType:  rateagreement.AgreementTypeContract,
		Status:         status,
		Currency:       "USD",
		EffectiveFrom:  100,
	}
}

// serviceFor wires a service whose repository reads and writes the one
// agreement under test, so a transition can be observed on the object itself.
func serviceFor(
	t *testing.T,
	agreement *rateagreement.RateAgreement,
) (*Service, *mocks.MockRateAgreementRepository) {
	t.Helper()

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateAgreementByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.GetRateAgreementByIDRequest,
		) (*rateagreement.RateAgreement, error) {
			require.Equal(t, agreement.ID, req.RateAgreementID)
			return agreement, nil
		}).
		Maybe()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreement")).
		RunAndReturn(func(
			_ context.Context,
			entity *rateagreement.RateAgreement,
		) (*rateagreement.RateAgreement, error) {
			return entity, nil
		}).
		Maybe()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	return New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: audit,
	}), repo
}
