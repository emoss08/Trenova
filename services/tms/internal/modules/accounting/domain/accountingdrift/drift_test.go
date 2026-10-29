package accountingdrift_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func minor(v int64) *int64 { return &v }

func driftObservation(
	objectType accountingsync.SyncObjectType,
	kind accountingdrift.DriftKind,
) *accountingdrift.DriftObservation {
	return &accountingdrift.DriftObservation{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
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

	finding := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch),
	)

	assert.Equal(t, accountingdrift.DriftStatusOpen, finding.Status)
	assert.Equal(t, "USD", finding.CurrencyCode)
	require.NotNil(t, finding.DifferenceMinor)
	assert.Equal(t, int64(-5_000), *finding.DifferenceMinor, "the provider holds 50.00 less")
	assert.Equal(t, int64(1_790_000_000), finding.DetectedAt)
	assert.Equal(t, finding.DetectedAt, finding.LastSeenAt)
}

func TestDriftFinding_NoDifferenceWithoutBothValues(t *testing.T) {
	t.Parallel()

	obs := driftObservation(
		accountingsync.SyncObjectInvoice,
		accountingdrift.DriftDeletedInProvider,
	)
	obs.ProviderMinor = nil
	finding := accountingdrift.NewAccountingDriftFinding(obs)

	assert.Nil(t, finding.DifferenceMinor)
	assert.False(
		t,
		finding.WithinTolerance(1_000_000),
		"a deleted document is never within tolerance",
	)
}

func TestDriftFinding_OffersTheFixesTrenovaCanMake(t *testing.T) {
	t.Parallel()

	both := []accountingdrift.DriftDirection{
		accountingdrift.DriftPushTrenovaValue,
		accountingdrift.DriftAdjustTrenova,
	}
	pushOnly := []accountingdrift.DriftDirection{accountingdrift.DriftPushTrenovaValue}

	cases := []struct {
		objectType accountingsync.SyncObjectType
		kind       accountingdrift.DriftKind
		want       []accountingdrift.DriftDirection
	}{
		{accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch, both},
		{accountingsync.SyncObjectDebitMemo, accountingdrift.DriftVoidedInProvider, both},
		{accountingsync.SyncObjectInvoice, accountingdrift.DriftDeletedInProvider, both},
		{accountingsync.SyncObjectInvoice, accountingdrift.DriftStatusMismatch, pushOnly},
		{accountingsync.SyncObjectCustomerPayment, accountingdrift.DriftVoidedInProvider, both},
		{accountingsync.SyncObjectCustomerPayment, accountingdrift.DriftAmountMismatch, pushOnly},
		{accountingsync.SyncObjectCreditMemo, accountingdrift.DriftAmountMismatch, pushOnly},
		{accountingsync.SyncObjectCarrierBill, accountingdrift.DriftAmountMismatch, pushOnly},
		{accountingsync.SyncObjectDriverBillPay, accountingdrift.DriftVoidedInProvider, pushOnly},
		{
			accountingsync.SyncObjectCustomer,
			accountingdrift.DriftCustomerBalanceMismatch,
			[]accountingdrift.DriftDirection{},
		},
		{
			accountingsync.DriftObjectGLAccount,
			accountingdrift.DriftTrialBalanceMismatch,
			[]accountingdrift.DriftDirection{},
		},
	}
	for _, tc := range cases {
		finding := accountingdrift.NewAccountingDriftFinding(
			driftObservation(tc.objectType, tc.kind),
		)
		assert.Equal(t, tc.want, finding.Directions(), "%s %s", tc.objectType, tc.kind)
	}
}

func TestDriftFinding_CanFixRefusesWhatIsNotOffered(t *testing.T) {
	t.Parallel()

	finding := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectCarrierBill, accountingdrift.DriftAmountMismatch),
	)
	require.NoError(t, finding.CanFix(accountingdrift.DriftPushTrenovaValue))
	require.ErrorIs(
		t,
		finding.CanFix(accountingdrift.DriftAdjustTrenova),
		accountingdrift.ErrDriftFixUnavailable,
	)

	require.True(t, finding.Clear(1_790_000_100))
	require.ErrorIs(
		t,
		finding.CanFix(accountingdrift.DriftPushTrenovaValue),
		accountingdrift.ErrDriftClosed,
	)
}

func TestDriftFinding_ToleranceComparesTheSizeOfTheDifference(t *testing.T) {
	t.Parallel()

	finding := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch),
	)
	assert.True(t, finding.WithinTolerance(5_000))
	assert.False(t, finding.WithinTolerance(4_999))
	assert.False(t, finding.WithinTolerance(0))

	status := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftStatusMismatch),
	)
	assert.False(t, status.WithinTolerance(1_000_000), "only money findings have a tolerance")
}

func TestDriftFinding_ClearingAfterAPushCreditsThePush(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	record := pulid.MustNew("acctsr_")
	pushed := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch),
	)
	pushed.MarkPushed(record, actor)
	assert.True(t, pushed.Pushed())
	assert.True(t, pushed.IsOpen(), "a push resolves only once a compare matches")

	require.True(t, pushed.Clear(1_790_000_200))
	assert.Equal(t, accountingdrift.DriftStatusResolved, pushed.Status)
	assert.Equal(t, accountingdrift.DriftPushedTrenovaValue, pushed.Resolution)
	assert.Equal(t, record, pushed.FixObjectID)
	assert.Equal(t, actor, pushed.ResolvedByID)
	require.NotNil(t, pushed.ResolvedAt)
	assert.Equal(t, int64(1_790_000_200), *pushed.ResolvedAt)

	untouched := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch),
	)
	require.True(t, untouched.Clear(1_790_000_200))
	assert.Equal(t, accountingdrift.DriftNoLongerDiffers, untouched.Resolution)
	assert.False(t, untouched.Clear(1_790_000_300), "a closed finding clears once")
}

func TestDriftFinding_AdjustingResolvesWithWhatWasPosted(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	memo := pulid.MustNew("inv_")
	finding := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch),
	)
	finding.MarkAdjusted(accountingdrift.DriftFixCreditMemo, memo, actor, 1_790_000_300)

	assert.Equal(t, accountingdrift.DriftStatusResolved, finding.Status)
	assert.Equal(t, accountingdrift.DriftAdjustedTrenova, finding.Resolution)
	assert.Equal(t, accountingdrift.DriftFixCreditMemo, finding.FixObjectType)
	assert.Equal(t, memo, finding.FixObjectID)
	assert.False(t, finding.Pushed())
}

func TestDriftFinding_DismissNeedsANoteAndAnOpenFinding(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	finding := accountingdrift.NewAccountingDriftFinding(
		driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch),
	)
	require.ErrorIs(t, finding.Dismiss(actor, "  \n ", 1), accountingdrift.ErrDriftNoteRequired)
	assert.True(t, finding.IsOpen())

	require.NoError(t, finding.Dismiss(actor, "  Rounding\n on the\tfuel line  ", 1_790_000_400))
	assert.Equal(t, accountingdrift.DriftStatusDismissed, finding.Status)
	assert.Equal(t, accountingdrift.DriftDismissed, finding.Resolution)
	assert.Equal(t, "Rounding on the fuel line", finding.ResolutionNote)
	assert.Equal(t, actor, finding.ResolvedByID)

	require.ErrorIs(t, finding.Dismiss(actor, "again", 2), accountingdrift.ErrDriftClosed)
}

func TestDriftFinding_ObservingAClosedFindingChangesNothing(t *testing.T) {
	t.Parallel()

	obs := driftObservation(accountingsync.SyncObjectInvoice, accountingdrift.DriftAmountMismatch)
	finding := accountingdrift.NewAccountingDriftFinding(obs)

	later := *obs
	later.ProviderMinor = minor(140_000)
	later.ProviderModifiedBy = "J Doe"
	later.At = 1_790_000_900
	finding.Observe(&later)
	require.NotNil(t, finding.DifferenceMinor)
	assert.Equal(t, int64(-10_000), *finding.DifferenceMinor)
	assert.Equal(t, "J Doe", finding.ProviderModifiedBy)
	assert.Equal(t, int64(1_790_000_900), finding.LastSeenAt)
	assert.Equal(
		t,
		int64(1_790_000_000),
		finding.DetectedAt,
		"detection time stays the first sighting",
	)

	require.True(t, finding.Clear(1_790_001_000))
	again := later
	again.ProviderMinor = minor(1)
	finding.Observe(&again)
	assert.Equal(t, int64(-10_000), *finding.DifferenceMinor)
}

func TestDriftFinding_DetailIsCapped(t *testing.T) {
	t.Parallel()

	obs := driftObservation(
		accountingsync.SyncObjectCustomer,
		accountingdrift.DriftCustomerBalanceMismatch,
	)
	for i := range 80 {
		obs.Detail = append(obs.Detail, accountingdrift.DriftLine{
			ObjectType:    accountingsync.SyncObjectInvoice,
			ObjectNumber:  "INV",
			TrenovaMinor:  int64(i),
			ProviderMinor: 0,
		})
	}
	finding := accountingdrift.NewAccountingDriftFinding(obs)
	assert.Len(t, finding.Detail, 50)
}

func TestTrialBalanceFindingsAreExplainedNotFixed(t *testing.T) {
	t.Parallel()

	finding := accountingdrift.NewAccountingDriftFinding(
		driftObservation(
			accountingsync.DriftObjectGLAccount,
			accountingdrift.DriftTrialBalanceMismatch,
		),
	)
	for _, direction := range accountingdrift.AllDriftDirections() {
		require.ErrorIs(t, finding.CanFix(direction), accountingdrift.ErrDriftExplainOnly)
	}
	assert.True(t, finding.Kind.IsMoney())
	assert.True(t, finding.Kind.IsBalance())
}
