package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A dispute asks what the terms said when the rate was quoted, and the answer
// has to exist from the very first save.
func TestCreate_WritesTheInitialHeaderVersion(t *testing.T) {
	t.Parallel()

	agreement := draftAgreement(rateagreement.StatusDraft)
	agreement.ID = pulid.Nil

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreement")).
		RunAndReturn(func(
			_ context.Context,
			entity *rateagreement.RateAgreement,
		) (*rateagreement.RateAgreement, error) {
			entity.ID = pulid.MustNew("ragr_")
			return entity, nil
		}).
		Once()
	repo.EXPECT().
		CreateVersion(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreementVersion")).
		RunAndReturn(func(
			_ context.Context,
			version *rateagreement.RateAgreementVersion,
		) (*rateagreement.RateAgreementVersion, error) {
			assert.EqualValues(t, 1, version.VersionNumber)
			assert.Equal(t, agreement.Name, version.Name)
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
	assert.EqualValues(t, 1, created.CurrentVersionNumber)
}

// Changing a negotiated term closes the old version and opens the next, so
// "what did the contract say in March" stays answerable after a renegotiation.
func TestUpdate_ChangedHeaderTermsWriteTheNextVersion(t *testing.T) {
	t.Parallel()

	original := draftAgreement(rateagreement.StatusDraft)
	original.CurrentVersionNumber = 3

	edited := *original
	edited.Name = "Acme Freight Agreement (renegotiated)"

	repo, audit := updateHarness(t, original)
	repo.EXPECT().
		CreateVersion(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreementVersion")).
		RunAndReturn(func(
			_ context.Context,
			version *rateagreement.RateAgreementVersion,
		) (*rateagreement.RateAgreementVersion, error) {
			assert.EqualValues(t, 4, version.VersionNumber)
			assert.NotEmpty(t, version.ChangeSummary)
			return version, nil
		}).
		Once()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: audit,
	})

	updated, err := svc.Update(t.Context(), &edited, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.EqualValues(t, 4, updated.CurrentVersionNumber)
}

// A save that restates the same terms — a lane edit, a description touch-up
// that never happened — must not mint a version, or the history becomes noise
// nobody can read a renegotiation out of.
func TestUpdate_UnchangedHeaderTermsWriteNoVersion(t *testing.T) {
	t.Parallel()

	original := draftAgreement(rateagreement.StatusDraft)
	original.CurrentVersionNumber = 3

	restated := *original
	// The client cannot bump the version by sending a bigger number.
	restated.CurrentVersionNumber = 99

	repo, audit := updateHarness(t, original)

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewTestValidator(),
		AuditService: audit,
	})

	updated, err := svc.Update(t.Context(), &restated, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.EqualValues(t, 3, updated.CurrentVersionNumber)
}

// updateHarness wires the reads and writes an update makes, leaving version
// expectations to each test — a strict mock makes "no version written" an
// assertion rather than an accident.
func updateHarness(
	t *testing.T,
	original *rateagreement.RateAgreement,
) (*mocks.MockRateAgreementRepository, *mocks.MockAuditService) {
	t.Helper()

	var saved *rateagreement.RateAgreement

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateAgreementByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.GetRateAgreementByIDRequest,
		) (*rateagreement.RateAgreement, error) {
			require.Equal(t, original.ID, req.RateAgreementID)
			if saved != nil {
				return saved, nil
			}
			return original, nil
		}).
		Times(2)
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*rateagreement.RateAgreement")).
		RunAndReturn(func(
			_ context.Context,
			entity *rateagreement.RateAgreement,
		) (*rateagreement.RateAgreement, error) {
			saved = entity
			return entity, nil
		}).
		Once()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()

	return repo, audit
}
