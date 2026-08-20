package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

// The contract's window, priority, renewal terms and billing routing are
// negotiated exactly the way its currency is — moving any of them has to read
// as a renegotiation, not vanish because the field lives outside the old
// snapshot.
func TestHeaderTermsChanged_WidenedHeaderFieldsAreTerms(t *testing.T) {
	t.Parallel()

	billTo := pulid.MustNew("cus_")

	cases := []struct {
		name string
		edit func(a *rateagreement.RateAgreement)
		path string
	}{
		{
			name: "agreement window",
			edit: func(a *rateagreement.RateAgreement) { a.EffectiveFrom = 500 },
			path: "agreementEffectiveFrom",
		},
		{
			name: "priority",
			edit: func(a *rateagreement.RateAgreement) { a.Priority = 7 },
			path: "priority",
		},
		{
			name: "auto renew",
			edit: func(a *rateagreement.RateAgreement) { a.AutoRenew = true },
			path: "autoRenew",
		},
		{
			name: "renewal notice",
			edit: func(a *rateagreement.RateAgreement) { a.RenewalNoticeDays = 60 },
			path: "renewalNoticeDays",
		},
		{
			name: "bill-to customer",
			edit: func(a *rateagreement.RateAgreement) { a.BillToCustomerID = &billTo },
			path: "billToCustomerId",
		},
		{
			name: "code",
			edit: func(a *rateagreement.RateAgreement) { a.Code = "ACME-2027" },
			path: "code",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := draftAgreement(rateagreement.StatusDraft)
			edited := *original
			tc.edit(&edited)

			summary, changed := headerTermsChanged(original, &edited)

			require.True(t, changed)
			assert.Contains(t, summary, tc.path)
		})
	}
}

// The accessorial schedule is a negotiated term: a price change, an added or
// dropped accessorial, or a moved window all mint a version, and the summary
// names the exact term keyed by the charge it prices.
func TestHeaderTermsChanged_AccessorialScheduleIsATerm(t *testing.T) {
	t.Parallel()

	chargeID := pulid.MustNew("acc_")
	seed := draftAgreement(rateagreement.StatusDraft)

	// Every reading is the same contract — identity included — so the only
	// differences the diff can see are the ones each case introduces.
	base := func() *rateagreement.RateAgreement {
		agreement := *seed
		agreement.Accessorials = []*rateagreement.RateAgreementAccessorial{{
			AccessorialChargeID: chargeID,
			Method:              "Flat",
			Amount:              decimal.NewFromInt(25),
		}}
		return &agreement
	}

	t.Run("a price change is named", func(t *testing.T) {
		t.Parallel()

		original := base()
		edited := base()
		edited.Accessorials[0].Amount = decimal.NewFromInt(40)

		summary, changed := headerTermsChanged(original, edited)

		require.True(t, changed)
		assert.Contains(t, summary, "accessorialTerms."+chargeID.String()+".amount")
	})

	t.Run("a moved window is named", func(t *testing.T) {
		t.Parallel()

		original := base()
		edited := base()
		from := int64(900)
		edited.Accessorials[0].EffectiveFrom = &from

		summary, changed := headerTermsChanged(original, edited)

		require.True(t, changed)
		assert.Contains(t, summary, "accessorialTerms."+chargeID.String()+".appliesFrom")
	})

	t.Run("an added accessorial is a change", func(t *testing.T) {
		t.Parallel()

		original := base()
		edited := base()
		added := pulid.MustNew("acc_")
		edited.Accessorials = append(edited.Accessorials, &rateagreement.RateAgreementAccessorial{
			AccessorialChargeID: added,
			Method:              "Flat",
			Amount:              decimal.NewFromInt(75),
		})

		summary, changed := headerTermsChanged(original, edited)

		require.True(t, changed)
		assert.Contains(t, summary, "accessorialTerms."+added.String())
	})

	t.Run("a dropped accessorial is a change", func(t *testing.T) {
		t.Parallel()

		original := base()
		edited := base()
		edited.Accessorials = nil

		_, changed := headerTermsChanged(original, edited)

		require.True(t, changed)
	})

	t.Run("a reordered applicability set is not a renegotiation", func(t *testing.T) {
		t.Parallel()

		first, second := pulid.MustNew("st_"), pulid.MustNew("st_")

		original := base()
		original.Accessorials[0].ServiceTypeIDs = []pulid.ID{first, second}
		edited := base()
		edited.Accessorials[0].ServiceTypeIDs = []pulid.ID{second, first}

		_, changed := headerTermsChanged(original, edited)

		assert.False(t, changed)
	})
}

// The fuel binding prices every gallon the contract moves; binding a program,
// waiving it, or overriding its peg is a renegotiation.
func TestHeaderTermsChanged_FuelBindingIsATerm(t *testing.T) {
	t.Parallel()

	original := draftAgreement(rateagreement.StatusDraft)
	edited := draftAgreement(rateagreement.StatusDraft)
	edited.ID = original.ID
	edited.OrganizationID = original.OrganizationID
	edited.BusinessUnitID = original.BusinessUnitID
	edited.CustomerID = original.CustomerID
	edited.FuelBinding = &rateagreement.RateAgreementFuelBinding{
		FuelSurchargeProgramID: pulid.MustNew("fsp_"),
	}

	summary, changed := headerTermsChanged(original, edited)

	require.True(t, changed)
	assert.Contains(t, summary, "fuelTerms")
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
