package journalposting

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fixedNumbers struct{}

func (fixedNumbers) GenerateJournalBatchNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JB-1", nil
}

func (fixedNumbers) GenerateJournalEntryNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "JE-1", nil
}

var (
	orgID  = pulid.MustNew("org_")
	buID   = pulid.MustNew("bu_")
	yearID = pulid.MustNew("fy_")
)

func period(number int, status fiscalperiod.Status, start int64) *fiscalperiod.FiscalPeriod {
	return &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: yearID,
		PeriodNumber: number,
		Status:       status,
		StartDate:    start,
	}
}

func periodsReturning(t *testing.T, found *fiscalperiod.FiscalPeriod) *mocks.MockFiscalPeriodRepository {
	t.Helper()
	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  1_000,
		}).
		Return(found, nil).
		Once()
	return periods
}

func resolve(
	t *testing.T,
	periods PeriodReader,
	policy tenant.ClosedPeriodPostingPolicy,
) (*ResolvedPeriod, error) {
	t.Helper()
	return ResolvePeriod(t.Context(), periods, &ResolvePeriodRequest{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Date:           1_000,
		Policy:         policy,
		Subject:        "driver settlement payment",
	})
}

func TestResolvePeriodKeepsTheDateInAnOpenOrLockedPeriod(t *testing.T) {
	t.Parallel()

	for _, status := range []fiscalperiod.Status{fiscalperiod.StatusOpen, fiscalperiod.StatusLocked} {
		found := period(3, status, 900)
		resolved, err := resolve(t, periodsReturning(t, found), tenant.ClosedPeriodPostingPolicyRequireReopen)
		require.NoError(t, err)
		assert.Equal(t, found.ID, resolved.Period.ID)
		assert.Equal(t, int64(1_000), resolved.AccountingDate)
	}
}

func TestResolvePeriodRefusesAClosedPeriodUnlessThePolicyMovesIt(t *testing.T) {
	t.Parallel()

	for _, status := range []fiscalperiod.Status{fiscalperiod.StatusClosed, fiscalperiod.StatusPermanentlyClosed} {
		_, err := resolve(t, periodsReturning(t, period(3, status, 900)), tenant.ClosedPeriodPostingPolicyRequireReopen)
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
		assert.Contains(t, err.Error(), "closed fiscal period")
	}
}

func TestResolvePeriodMovesToTheEarliestLaterOpenPeriod(t *testing.T) {
	t.Parallel()

	closed := period(3, fiscalperiod.StatusClosed, 900)
	later := period(6, fiscalperiod.StatusOpen, 6_000)
	next := period(5, fiscalperiod.StatusLocked, 5_000)
	periods := periodsReturning(t, closed)
	periods.EXPECT().
		ListByFiscalYearID(mock.Anything, repositories.ListByFiscalYearIDRequest{
			FiscalYearID: yearID,
			OrgID:        orgID,
			BuID:         buID,
		}).
		Return([]*fiscalperiod.FiscalPeriod{
			period(2, fiscalperiod.StatusOpen, 500),
			later,
			period(4, fiscalperiod.StatusClosed, 4_000),
			next,
		}, nil).
		Once()

	resolved, err := resolve(t, periods, tenant.ClosedPeriodPostingPolicyPostToNextOpen)
	require.NoError(t, err)
	assert.Equal(t, next.ID, resolved.Period.ID)
	assert.Equal(t, int64(5_000), resolved.AccountingDate)
}

func TestResolvePeriodFailsWhenNoLaterPeriodIsOpen(t *testing.T) {
	t.Parallel()

	periods := periodsReturning(t, period(3, fiscalperiod.StatusClosed, 900))
	periods.EXPECT().
		ListByFiscalYearID(mock.Anything, mock.Anything).
		Return([]*fiscalperiod.FiscalPeriod{period(4, fiscalperiod.StatusClosed, 4_000)}, nil).
		Once()

	_, err := resolve(t, periods, tenant.ClosedPeriodPostingPolicyPostToNextOpen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No next open fiscal period")
}

func TestResolvePeriodRefusesAnInactivePeriodAndAMissingOne(t *testing.T) {
	t.Parallel()

	_, err := resolve(t, periodsReturning(t, period(3, fiscalperiod.StatusInactive, 900)),
		tenant.ClosedPeriodPostingPolicyPostToNextOpen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inactive fiscal period")

	missing := mocks.NewMockFiscalPeriodRepository(t)
	missing.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("fiscal period not found")).
		Once()
	_, err = resolve(t, missing, tenant.ClosedPeriodPostingPolicyPostToNextOpen)
	require.Error(t, err)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "accountingDate", validation.Field)
}

func TestResolveWorkflowFollowsThePostingMode(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	const now = int64(42)

	automatic := ResolveWorkflow(&tenant.AccountingControl{
		JournalPostingMode: tenant.JournalPostingModeAutomatic,
	}, userID, now)
	assert.Equal(t, StatusPosted, automatic.EntryStatus)
	require.True(t, automatic.Posted())
	assert.Equal(t, now, *automatic.PostedAt)
	assert.Equal(t, userID, automatic.PostedByID)

	review := ResolveWorkflow(&tenant.AccountingControl{
		JournalPostingMode:      tenant.JournalPostingModeManual,
		RequireManualJEApproval: true,
	}, userID, now)
	assert.Equal(t, StatusPending, review.EntryStatus)
	assert.False(t, review.Posted())
	assert.True(t, review.RequiresApproval)
	assert.False(t, review.IsApproved)
	assert.True(t, review.ApprovedByID.IsNil())

	approved := ResolveWorkflow(&tenant.AccountingControl{
		JournalPostingMode: tenant.JournalPostingModeManual,
	}, userID, now)
	assert.Equal(t, StatusApproved, approved.EntryStatus)
	assert.False(t, approved.Posted())
	assert.True(t, approved.IsApproved)
	assert.Equal(t, userID, approved.ApprovedByID)
}

func writeRequest(control *tenant.AccountingControl, lines ...Line) *WriteRequest {
	return &WriteRequest{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Control:        control,
		ActorID:        pulid.MustNew("usr_"),
		AccountingDate: 1_000,
		Now:            2_000,
		Subject:        "driver settlement",
		Description:    "Payment of driver settlement DS-1",
		Source: Source{
			ObjectType:     "DriverSettlement",
			ObjectID:       pulid.MustNew("dstl_"),
			DocumentNumber: "DS-1",
			Event:          tenant.JournalSourceEventDriverSettlementPaid,
			IdempotencyKey: "driver-settlement-paid:1",
		},
		Lines: lines,
	}
}

func TestWriterPostsABalancedJournalInTheResolvedPeriod(t *testing.T) {
	t.Parallel()

	open := period(3, fiscalperiod.StatusOpen, 900)
	payable, cash := pulid.MustNew("gla_"), pulid.MustNew("gla_")
	var written repositories.CreateJournalPostingParams
	journals := mocks.NewMockJournalPostingRepository(t)
	journals.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			written = params
			return nil
		}).
		Once()

	req := writeRequest(&tenant.AccountingControl{JournalPostingMode: tenant.JournalPostingModeAutomatic},
		Line{AccountID: payable, Debit: 700},
		Line{AccountID: cash, Credit: 700},
	)
	result, err := Writer{Periods: periodsReturning(t, open), Numbers: fixedNumbers{}, Journals: journals}.
		Write(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, result.BatchID, written.BatchID)
	assert.Equal(t, result.EntryID, written.EntryID)
	assert.Equal(t, open.ID, written.FiscalPeriodID)
	assert.Equal(t, yearID, written.FiscalYearID)
	assert.Equal(t, int64(1_000), written.AccountingDate)
	assert.Equal(t, StatusPosted, written.EntryStatus)
	assert.True(t, written.IsPosted)
	assert.Equal(t, int64(700), written.TotalDebit)
	assert.Equal(t, int64(700), written.TotalCredit)
	assert.Equal(t, "DriverSettlementPaid", written.SourceEventType)
	assert.Equal(t, "driver-settlement-paid:1", written.SourceIdempotencyKey)
	assert.Equal(t, req.Source.ObjectID.String(), written.SourceObjectID)
	require.Len(t, written.Lines, 2)
	assert.Equal(t, payable, written.Lines[0].GLAccountID)
	assert.Equal(t, int64(700), written.Lines[0].NetAmount)
	assert.Equal(t, cash, written.Lines[1].GLAccountID)
	assert.Equal(t, int64(-700), written.Lines[1].NetAmount)
	assert.Equal(t, int16(2), written.Lines[1].LineNumber)
}

func TestWriterLeavesAManualModeJournalUnposted(t *testing.T) {
	t.Parallel()

	var written repositories.CreateJournalPostingParams
	journals := mocks.NewMockJournalPostingRepository(t)
	journals.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			written = params
			return nil
		}).
		Once()

	req := writeRequest(&tenant.AccountingControl{
		JournalPostingMode:      tenant.JournalPostingModeManual,
		RequireManualJEApproval: true,
	},
		Line{AccountID: pulid.MustNew("gla_"), Debit: 10},
		Line{AccountID: pulid.MustNew("gla_"), Credit: 10},
	)
	_, err := Writer{
		Periods:  periodsReturning(t, period(3, fiscalperiod.StatusOpen, 900)),
		Numbers:  fixedNumbers{},
		Journals: journals,
	}.Write(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, StatusPending, written.EntryStatus)
	assert.Equal(t, StatusPending, written.SourceStatus)
	assert.False(t, written.IsPosted)
	assert.Nil(t, written.PostedAt)
	assert.True(t, written.RequiresApproval)
}

func TestWriterRefusesJournalsThatDoNotBalanceOrHaveBadLines(t *testing.T) {
	t.Parallel()

	account := pulid.MustNew("gla_")
	cases := map[string][]Line{
		"unbalanced":   {{AccountID: account, Debit: 10}, {AccountID: account, Credit: 9}},
		"one line":     {{AccountID: account, Debit: 10}},
		"no account":   {{Debit: 10}, {AccountID: account, Credit: 10}},
		"both sides":   {{AccountID: account, Debit: 10, Credit: 10}, {AccountID: account, Credit: 0}},
		"zero line":    {{AccountID: account}, {AccountID: account, Debit: 5}, {AccountID: account, Credit: 5}},
		"negative leg": {{AccountID: account, Debit: -10}, {AccountID: account, Credit: -10}},
	}
	for name, lines := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := Writer{
				Periods:  mocks.NewMockFiscalPeriodRepository(t),
				Numbers:  fixedNumbers{},
				Journals: mocks.NewMockJournalPostingRepository(t),
			}.Write(t.Context(), writeRequest(&tenant.AccountingControl{}, lines...))
			require.Error(t, err)
			assert.True(t, errortypes.IsBusinessError(err))
		})
	}
}
