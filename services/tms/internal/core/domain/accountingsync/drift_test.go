package accountingsync_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func minor(v int64) *int64 { return &v }

func driftObservation(
	objectType accountingsync.SyncObjectType,
	kind accountingsync.DriftKind,
) *accountingsync.DriftObservation {
	return &accountingsync.DriftObservation{
		TenantInfo:    pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ConnectionID:  pulid.MustNew("acctc_"),
		ObjectType:    objectType,
		ObjectID:      pulid.MustNew("inv_"),
		ObjectNumber:  "INV-1001",
		Kind:          kind,
		CurrencyCode:  "usd",
		TrenovaMinor:  minor(150_000),
		ProviderMinor: minor(145_000),
		TrenovaState:  "Posted",
		ProviderState: "Open",
		At:            1_790_000_000,
	}
}

func TestDriftFinding_RecordsBothValuesAndTheDifference(t *testing.T) {
	t.Parallel()

	finding := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch),
	)

	assert.Equal(t, accountingsync.DriftStatusOpen, finding.Status)
	assert.Equal(t, "USD", finding.CurrencyCode)
	require.NotNil(t, finding.DifferenceMinor)
	assert.Equal(t, int64(-5_000), *finding.DifferenceMinor, "the provider holds 50.00 less")
	assert.Equal(t, int64(1_790_000_000), finding.DetectedAt)
	assert.Equal(t, finding.DetectedAt, finding.LastSeenAt)
}

func TestDriftFinding_NoDifferenceWithoutBothValues(t *testing.T) {
	t.Parallel()

	obs := driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftDeletedInProvider)
	obs.ProviderMinor = nil
	finding := accountingsync.NewAccountingDriftFinding(obs)

	assert.Nil(t, finding.DifferenceMinor)
	assert.False(t, finding.WithinTolerance(1_000_000), "a deleted document is never within tolerance")
}

func TestDriftFinding_OffersTheFixesTrenovaCanMake(t *testing.T) {
	t.Parallel()

	both := []accountingsync.DriftDirection{
		accountingsync.DriftPushTrenovaValue,
		accountingsync.DriftAdjustTrenova,
	}
	pushOnly := []accountingsync.DriftDirection{accountingsync.DriftPushTrenovaValue}

	cases := []struct {
		objectType accountingsync.SyncObjectType
		kind       accountingsync.DriftKind
		want       []accountingsync.DriftDirection
	}{
		{accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch, both},
		{accountingsync.SyncObjectDebitMemo, accountingsync.DriftVoidedInProvider, both},
		{accountingsync.SyncObjectInvoice, accountingsync.DriftDeletedInProvider, both},
		{accountingsync.SyncObjectInvoice, accountingsync.DriftStatusMismatch, pushOnly},
		{accountingsync.SyncObjectCustomerPayment, accountingsync.DriftVoidedInProvider, both},
		{accountingsync.SyncObjectCustomerPayment, accountingsync.DriftAmountMismatch, pushOnly},
		{accountingsync.SyncObjectCreditMemo, accountingsync.DriftAmountMismatch, pushOnly},
		{accountingsync.SyncObjectCarrierBill, accountingsync.DriftAmountMismatch, pushOnly},
		{accountingsync.SyncObjectDriverBillPay, accountingsync.DriftVoidedInProvider, pushOnly},
		{
			accountingsync.SyncObjectCustomer,
			accountingsync.DriftCustomerBalanceMismatch,
			[]accountingsync.DriftDirection{},
		},
	}
	for _, tc := range cases {
		finding := accountingsync.NewAccountingDriftFinding(driftObservation(tc.objectType, tc.kind))
		assert.Equal(t, tc.want, finding.Directions(), "%s %s", tc.objectType, tc.kind)
	}
}

func TestDriftFinding_CanFixRefusesWhatIsNotOffered(t *testing.T) {
	t.Parallel()

	finding := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectCarrierBill, accountingsync.DriftAmountMismatch),
	)
	require.NoError(t, finding.CanFix(accountingsync.DriftPushTrenovaValue))
	require.ErrorIs(t, finding.CanFix(accountingsync.DriftAdjustTrenova), accountingsync.ErrDriftFixUnavailable)

	require.True(t, finding.Clear(1_790_000_100))
	require.ErrorIs(t, finding.CanFix(accountingsync.DriftPushTrenovaValue), accountingsync.ErrDriftClosed)
}

func TestDriftFinding_ToleranceComparesTheSizeOfTheDifference(t *testing.T) {
	t.Parallel()

	finding := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch),
	)
	assert.True(t, finding.WithinTolerance(5_000))
	assert.False(t, finding.WithinTolerance(4_999))
	assert.False(t, finding.WithinTolerance(0))

	status := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftStatusMismatch),
	)
	assert.False(t, status.WithinTolerance(1_000_000), "only money findings have a tolerance")
}

func TestDriftFinding_ClearingAfterAPushCreditsThePush(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	record := pulid.MustNew("acctsr_")
	pushed := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch),
	)
	pushed.MarkPushed(record, actor)
	assert.True(t, pushed.Pushed())
	assert.True(t, pushed.IsOpen(), "a push resolves only once a compare matches")

	require.True(t, pushed.Clear(1_790_000_200))
	assert.Equal(t, accountingsync.DriftStatusResolved, pushed.Status)
	assert.Equal(t, accountingsync.DriftPushedTrenovaValue, pushed.Resolution)
	assert.Equal(t, record, pushed.FixObjectID)
	assert.Equal(t, actor, pushed.ResolvedByID)
	require.NotNil(t, pushed.ResolvedAt)
	assert.Equal(t, int64(1_790_000_200), *pushed.ResolvedAt)

	untouched := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch),
	)
	require.True(t, untouched.Clear(1_790_000_200))
	assert.Equal(t, accountingsync.DriftNoLongerDiffers, untouched.Resolution)
	assert.False(t, untouched.Clear(1_790_000_300), "a closed finding clears once")
}

func TestDriftFinding_AdjustingResolvesWithWhatWasPosted(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	memo := pulid.MustNew("inv_")
	finding := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch),
	)
	finding.MarkAdjusted(accountingsync.DriftFixCreditMemo, memo, actor, 1_790_000_300)

	assert.Equal(t, accountingsync.DriftStatusResolved, finding.Status)
	assert.Equal(t, accountingsync.DriftAdjustedTrenova, finding.Resolution)
	assert.Equal(t, accountingsync.DriftFixCreditMemo, finding.FixObjectType)
	assert.Equal(t, memo, finding.FixObjectID)
	assert.False(t, finding.Pushed())
}

func TestDriftFinding_DismissNeedsANoteAndAnOpenFinding(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	finding := accountingsync.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch),
	)
	require.ErrorIs(t, finding.Dismiss(actor, "  \n ", 1), accountingsync.ErrDriftNoteRequired)
	assert.True(t, finding.IsOpen())

	require.NoError(t, finding.Dismiss(actor, "  Rounding\n on the\tfuel line  ", 1_790_000_400))
	assert.Equal(t, accountingsync.DriftStatusDismissed, finding.Status)
	assert.Equal(t, accountingsync.DriftDismissed, finding.Resolution)
	assert.Equal(t, "Rounding on the fuel line", finding.ResolutionNote)
	assert.Equal(t, actor, finding.ResolvedByID)

	require.ErrorIs(t, finding.Dismiss(actor, "again", 2), accountingsync.ErrDriftClosed)
}

func TestDriftFinding_ObservingAClosedFindingChangesNothing(t *testing.T) {
	t.Parallel()

	obs := driftObservation(accountingsync.SyncObjectInvoice, accountingsync.DriftAmountMismatch)
	finding := accountingsync.NewAccountingDriftFinding(obs)

	later := *obs
	later.ProviderMinor = minor(140_000)
	later.ProviderModifiedBy = "J Doe"
	later.At = 1_790_000_900
	finding.Observe(&later)
	require.NotNil(t, finding.DifferenceMinor)
	assert.Equal(t, int64(-10_000), *finding.DifferenceMinor)
	assert.Equal(t, "J Doe", finding.ProviderModifiedBy)
	assert.Equal(t, int64(1_790_000_900), finding.LastSeenAt)
	assert.Equal(t, int64(1_790_000_000), finding.DetectedAt, "detection time stays the first sighting")

	require.True(t, finding.Clear(1_790_001_000))
	again := later
	again.ProviderMinor = minor(1)
	finding.Observe(&again)
	assert.Equal(t, int64(-10_000), *finding.DifferenceMinor)
}

func TestDriftFinding_DetailIsCapped(t *testing.T) {
	t.Parallel()

	obs := driftObservation(accountingsync.SyncObjectCustomer, accountingsync.DriftCustomerBalanceMismatch)
	for i := range 80 {
		obs.Detail = append(obs.Detail, accountingsync.DriftLine{
			ObjectType:    accountingsync.SyncObjectInvoice,
			ObjectNumber:  "INV",
			TrenovaMinor:  int64(i),
			ProviderMinor: 0,
		})
	}
	finding := accountingsync.NewAccountingDriftFinding(obs)
	assert.Len(t, finding.Detail, 50)
}
