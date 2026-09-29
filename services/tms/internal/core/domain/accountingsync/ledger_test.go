package accountingsync

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChooseModeMovesSetupOnAndRemembersGranularity(t *testing.T) {
	conn := &AccountingConnection{SetupStep: SetupStepMode}

	require.NoError(t, conn.ChooseMode(SyncModeLedger, LedgerDailySummary))
	assert.Equal(t, SyncModeLedger, conn.Mode())
	assert.Equal(t, LedgerDailySummary, conn.Granularity())
	assert.True(t, conn.SumsByDay())
	assert.Equal(t, SetupStepMappings, conn.SetupStep)

	require.NoError(t, conn.ChooseMode(SyncModeDocument, LedgerDetailed))
	assert.Equal(t, SyncModeDocument, conn.Mode())
	assert.Empty(t, conn.LedgerGranularity)
	assert.Empty(t, conn.Granularity())
	assert.False(t, conn.SumsByDay())
}

func TestChooseModeKeepsALaterStepOnlyWhenNothingChanges(t *testing.T) {
	for _, step := range []SetupStep{SetupStepMappings, SetupStepStartDate} {
		conn := &AccountingConnection{
			SetupStep:         step,
			SyncMode:          SyncModeLedger,
			LedgerGranularity: LedgerDetailed,
		}
		require.NoError(t, conn.ChooseMode(SyncModeLedger, LedgerDetailed))
		assert.Equal(t, step, conn.SetupStep, "same choice at %s", step)
	}

	document := &AccountingConnection{SetupStep: SetupStepStartDate}
	require.NoError(t, document.ChooseMode(SyncModeDocument, LedgerDailySummary))
	assert.Equal(t, SetupStepStartDate, document.SetupStep, "documents are the default")
}

func TestChooseModeReturnsToMappingsWhenTheChoiceChanges(t *testing.T) {
	cases := []struct {
		name        string
		from        AccountingConnection
		mode        SyncMode
		granularity LedgerGranularity
	}{
		{
			name: "documents to journal entries",
			from: AccountingConnection{SetupStep: SetupStepStartDate},
			mode: SyncModeLedger, granularity: LedgerDetailed,
		},
		{
			name: "journal entries to documents",
			from: AccountingConnection{
				SetupStep: SetupStepStartDate, SyncMode: SyncModeLedger,
				LedgerGranularity: LedgerDailySummary,
			},
			mode: SyncModeDocument,
		},
		{
			name: "detailed to a daily summary",
			from: AccountingConnection{
				SetupStep: SetupStepStartDate, SyncMode: SyncModeLedger,
				LedgerGranularity: LedgerDetailed,
			},
			mode: SyncModeLedger, granularity: LedgerDailySummary,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn := tc.from
			require.NoError(t, conn.ChooseMode(tc.mode, tc.granularity))
			assert.Equal(t, SetupStepMappings, conn.SetupStep)
		})
	}
}

func TestStartDateIsFixedOnceOpeningBalancesAreSent(t *testing.T) {
	start := int64(1_782_864_000)
	sent := int64(1_782_900_000)
	conn := &AccountingConnection{
		SyncMode:                    SyncModeLedger,
		SyncStartDate:               &start,
		LedgerOpeningBalancesSentAt: &sent,
	}
	require.ErrorIs(t, conn.CanMoveStartDate(start-86_400), ErrStartDateFixed)
	require.ErrorIs(t, conn.CanMoveStartDate(start+86_400), ErrStartDateFixed)
	require.NoError(t, conn.CanMoveStartDate(start))

	withoutOpening := &AccountingConnection{SyncMode: SyncModeLedger, SyncStartDate: &start}
	require.NoError(t, withoutOpening.CanMoveStartDate(start-86_400))
	require.NoError(t, (&AccountingConnection{}).CanMoveStartDate(start))
}

func TestJournalSendableLeavesOutTheFiscalCloseAndItsReversal(t *testing.T) {
	cases := []struct {
		entryType journalentry.EntryType
		reverses  journalentry.EntryType
		want      bool
	}{
		{journalentry.EntryTypeStandard, "", true},
		{journalentry.EntryTypeReversal, journalentry.EntryTypeStandard, true},
		{journalentry.EntryTypeClosing, "", false},
		{journalentry.EntryTypeOpening, "", false},
		{journalentry.EntryTypeReversal, journalentry.EntryTypeClosing, false},
		{journalentry.EntryTypeReversal, journalentry.EntryTypeOpening, false},
		{"", "", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, JournalSendable(tc.entryType, tc.reverses), "%s reversing %q",
			tc.entryType, tc.reverses)
	}
}

func TestChooseModeRefusals(t *testing.T) {
	enabled := int64(1_790_000_000)
	cases := []struct {
		name        string
		conn        *AccountingConnection
		mode        SyncMode
		granularity LedgerGranularity
		want        error
	}{
		{
			name: "after sync is enabled",
			conn: &AccountingConnection{SetupStep: SetupStepStartDate, SyncEnabledAt: &enabled},
			mode: SyncModeLedger, granularity: LedgerDetailed, want: ErrModeFixed,
		},
		{
			name: "once setup is complete",
			conn: &AccountingConnection{SetupStep: SetupStepComplete},
			mode: SyncModeDocument, want: ErrModeFixed,
		},
		{
			name: "an unknown mode",
			conn: &AccountingConnection{SetupStep: SetupStepMode},
			mode: SyncMode("Hybrid"), want: ErrModeNotRecognized,
		},
		{
			name: "ledger without a granularity",
			conn: &AccountingConnection{SetupStep: SetupStepMode},
			mode: SyncModeLedger, granularity: LedgerGranularity("Weekly"), want: ErrGranularityRequired,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := *tc.conn
			require.ErrorIs(t, tc.conn.ChooseMode(tc.mode, tc.granularity), tc.want)
			assert.Equal(t, before, *tc.conn)
		})
	}
}

func TestModeDefaultsToDocumentsAndDetailedJournals(t *testing.T) {
	conn := &AccountingConnection{}
	assert.Equal(t, SyncModeDocument, conn.Mode())
	assert.False(t, conn.SendsLedger())

	conn.SyncMode = SyncModeLedger
	assert.Equal(t, LedgerDetailed, conn.Granularity())
	assert.False(t, conn.SumsByDay())
}

func TestSendsKeepsTheTwoModesApart(t *testing.T) {
	documents := &AccountingConnection{SyncMode: SyncModeDocument}
	ledger := &AccountingConnection{SyncMode: SyncModeLedger, LedgerGranularity: LedgerDetailed}

	for _, objectType := range AllSyncObjectTypes() {
		switch {
		case objectType.IsParty():
			assert.True(t, documents.Sends(objectType), objectType)
			assert.True(t, ledger.Sends(objectType), objectType)
		case objectType.IsLedger():
			assert.False(t, documents.Sends(objectType), objectType)
			assert.True(t, ledger.Sends(objectType), objectType)
		default:
			assert.True(t, documents.Sends(objectType), objectType)
			assert.False(t, ledger.Sends(objectType), objectType)
		}
	}
	daily := &AccountingConnection{SyncMode: SyncModeLedger, LedgerGranularity: LedgerDailySummary}
	assert.False(t, daily.Sends(SyncObjectJournalEntry))
	assert.True(t, daily.Sends(SyncObjectJournalSummary))
	assert.False(t, daily.Sends(SyncObjectInvoice))
	assert.True(t, ledger.Sends(SyncObjectCustomer))
	assert.True(t, ledger.Sends(SyncObjectJournalSummary))
	assert.False(t, ledger.Sends(SyncObjectInvoice))
	assert.False(t, documents.Sends(SyncObjectJournalEntry))
}

func TestEnableSyncMarksOpeningBalancesOnlyForTheLedgerAndOnlyOnce(t *testing.T) {
	settings := SyncSettings{StartDate: 1_790_000_000, OpeningBalances: true}

	documents := &AccountingConnection{SyncMode: SyncModeDocument}
	documents.EnableSync(settings, 100)
	assert.False(t, documents.SentOpeningBalances())

	ledger := &AccountingConnection{SyncMode: SyncModeLedger, LedgerGranularity: LedgerDetailed}
	ledger.EnableSync(settings, 100)
	require.True(t, ledger.SentOpeningBalances())
	assert.Equal(t, int64(100), *ledger.LedgerOpeningBalancesSentAt)

	ledger.EnableSync(settings, 200)
	assert.Equal(t, int64(100), *ledger.LedgerOpeningBalancesSentAt)

	without := &AccountingConnection{SyncMode: SyncModeLedger, LedgerGranularity: LedgerDetailed}
	without.EnableSync(SyncSettings{StartDate: 1_790_000_000}, 100)
	assert.False(t, without.SentOpeningBalances())
}

func TestFiscalYearStartFollowsTheProvidersFirstMonth(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	at := func(year int, month time.Month, day int) int64 {
		return time.Date(year, month, day, 12, 0, 0, 0, chicago).Unix()
	}
	first := func(year int, month time.Month) int64 {
		return time.Date(year, month, 1, 0, 0, 0, 0, chicago).Unix()
	}

	cases := []struct {
		name  string
		month int
		asOf  int64
		want  int64
	}{
		{name: "calendar year", month: 1, asOf: at(2026, time.September, 27), want: first(2026, time.January)},
		{name: "april year after april", month: 4, asOf: at(2026, time.September, 27), want: first(2026, time.April)},
		{name: "april year before april", month: 4, asOf: at(2026, time.February, 3), want: first(2025, time.April)},
		{name: "on the first day", month: 4, asOf: first(2026, time.April), want: first(2026, time.April)},
		{name: "unset month is january", month: 0, asOf: at(2026, time.March, 1), want: first(2026, time.January)},
		{name: "out of range month is january", month: 13, asOf: at(2026, time.March, 1), want: first(2026, time.January)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &AccountingConnection{ExternalFiscalYearStartMonth: tc.month}
			assert.Equal(t, tc.want, conn.FiscalYearStart(tc.asOf, chicago))
		})
	}
}

func TestJournalIDsNameTheLocalDay(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	lateEvening := time.Date(2026, time.September, 27, 22, 30, 0, 0, chicago).Unix()

	assert.Equal(t, "jday_20260927", JournalDayID(lateEvening, chicago))
	assert.Equal(t, "jday_20260928", JournalDayID(lateEvening, time.UTC))
	assert.Equal(t, "jday_20260928", JournalDayID(lateEvening, nil))
	assert.Equal(t, "jopen_20260927", JournalOpeningID(lateEvening, chicago))
	assert.NotEqual(t, JournalDayID(lateEvening, chicago), JournalOpeningID(lateEvening, chicago))
}

func TestLedgerPartyFollowsTheProviderAccountType(t *testing.T) {
	cases := []struct {
		name string
		ref  *AccountingReferenceObject
		want LedgerPartyNeed
	}{
		{name: "receivable", ref: &AccountingReferenceObject{Kind: ReferenceKindAccount, AccountClass: AccountClassReceivable}, want: LedgerPartyCustomer},
		{name: "payable", ref: &AccountingReferenceObject{Kind: ReferenceKindAccount, AccountClass: AccountClassPayable}, want: LedgerPartyVendor},
		{name: "income", ref: &AccountingReferenceObject{Kind: ReferenceKindAccount, AccountClass: AccountClassIncome}, want: LedgerPartyNone},
		{name: "not an account", ref: &AccountingReferenceObject{Kind: ReferenceKindCustomer, AccountClass: AccountClassReceivable}, want: LedgerPartyNone},
		{name: "unknown", want: LedgerPartyNone},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.ref.LedgerParty(), tc.name)
	}
}

func TestBackfillsOnlyWhatTheModeSendsAndDaysOnlyWhenSummed(t *testing.T) {
	documents := &AccountingConnection{SyncMode: SyncModeDocument}
	detailed := &AccountingConnection{SyncMode: SyncModeLedger, LedgerGranularity: LedgerDetailed}
	daily := &AccountingConnection{SyncMode: SyncModeLedger, LedgerGranularity: LedgerDailySummary}

	assert.True(t, documents.Backfills(SyncObjectInvoice))
	assert.False(t, documents.Backfills(SyncObjectJournalEntry))
	assert.False(t, documents.Backfills(SyncObjectJournalSummary))
	assert.True(t, detailed.Backfills(SyncObjectJournalEntry))
	assert.False(t, detailed.Backfills(SyncObjectJournalSummary))
	assert.False(t, detailed.Backfills(SyncObjectCarrierBill))
	assert.False(t, daily.Backfills(SyncObjectJournalEntry))
	assert.True(t, daily.Backfills(SyncObjectJournalSummary))
}

func TestLedgerAccountRoleFollowsTheDefaultAccounts(t *testing.T) {
	control := &tenant.AccountingControl{
		DefaultARAccountID:                      pulid.MustNew("gla_"),
		DefaultRevenueAccountID:                 pulid.MustNew("gla_"),
		DefaultCashAccountID:                    pulid.MustNew("gla_"),
		DefaultWriteOffAccountID:                pulid.MustNew("gla_"),
		DefaultAPAccountID:                      pulid.MustNew("gla_"),
		DefaultSettlementsPayableAccountID:      pulid.MustNew("gla_"),
		DefaultPurchasedTransportationAccountID: pulid.MustNew("gla_"),
	}
	assert.Equal(t, AccountRoleAR, LedgerAccountRole(control.DefaultARAccountID, control))
	assert.Equal(t, AccountRoleRevenue, LedgerAccountRole(control.DefaultRevenueAccountID, control))
	assert.Equal(t, AccountRoleDeposit, LedgerAccountRole(control.DefaultCashAccountID, control))
	assert.Equal(t, AccountRoleWriteOff, LedgerAccountRole(control.DefaultWriteOffAccountID, control))
	assert.Equal(t, AccountRoleAP, LedgerAccountRole(control.DefaultAPAccountID, control))
	assert.Equal(t, AccountRoleAP, LedgerAccountRole(control.DefaultSettlementsPayableAccountID, control))
	assert.Equal(t, AccountRolePurchasedTransportation,
		LedgerAccountRole(control.DefaultPurchasedTransportationAccountID, control))
	assert.Empty(t, LedgerAccountRole(pulid.MustNew("gla_"), control))
	assert.Empty(t, LedgerAccountRole(pulid.Nil, &tenant.AccountingControl{}),
		"an unset default does not match an unset account")
	assert.Empty(t, LedgerAccountRole(control.DefaultARAccountID, nil))
}
