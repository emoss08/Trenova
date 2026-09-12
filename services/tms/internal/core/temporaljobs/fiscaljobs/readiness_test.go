package fiscaljobs

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	accountingcontrol "github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func yearEndedDaysAgo(days int, now int64) *fiscalyear.FiscalYear {
	return &fiscalyear.FiscalYear{
		ID:      pulid.MustNew("fyr_"),
		Name:    "FY 2031",
		Year:    2031,
		EndDate: now - int64(days)*86400,
	}
}

func blockers(messages ...string) *fiscalclose.Result {
	result := &fiscalclose.Result{
		CanClose: len(messages) == 0,
		Blockers: make([]*fiscalclose.Blocker, 0, len(messages)),
	}
	for _, message := range messages {
		result.Blockers = append(result.Blockers, &fiscalclose.Blocker{
			Field:    "status",
			Code:     errortypes.ErrInvalid,
			Message:  message,
			Category: "year",
		})
	}

	return result
}

func TestReadinessWordingReportsAReadyYear(t *testing.T) {
	t.Parallel()

	now := time.Date(2032, time.February, 10, 0, 0, 0, 0, time.UTC).Unix()
	fy := yearEndedDaysAgo(41, now)

	title, message := readinessWording(fy, blockers(), now)

	assert.Equal(t, "FY 2031 is ready to close", title)
	assert.Contains(t, message, "41 days ago")
	assert.Contains(t, message, "nothing is blocking")
}

func TestReadinessWordingListsWhatIsOutstanding(t *testing.T) {
	t.Parallel()

	now := time.Date(2032, time.February, 10, 0, 0, 0, 0, time.UTC).Unix()
	fy := yearEndedDaysAgo(10, now)

	title, message := readinessWording(
		fy,
		blockers("2 period(s) are still open.", "Set a retained earnings account."),
		now,
	)

	assert.Equal(t, "FY 2031 cannot be closed yet", title)
	assert.Contains(t, message, "2 period(s) are still open.")
	assert.Contains(t, message, "Set a retained earnings account.")
}

func TestReadinessWordingTruncatesALongBlockerList(t *testing.T) {
	t.Parallel()

	now := time.Now().Unix()
	messages := []string{"one", "two", "three", "four", "five", "six", "seven"}

	_, message := readinessWording(yearEndedDaysAgo(3, now), blockers(messages...), now)

	assert.Contains(t, message, "five")
	assert.NotContains(t, message, "six")
	assert.True(t, strings.HasSuffix(message, "…"), "expected the list to be elided")
}

func TestReadinessPriorityEscalatesWithAge(t *testing.T) {
	t.Parallel()

	now := time.Now().Unix()

	assert.Equal(
		t,
		notification.PriorityLow,
		readinessPriority(blockers(), yearEndedDaysAgo(5, now), now),
		"a year that is ready and barely overdue is a nudge",
	)
	assert.Equal(
		t,
		notification.PriorityMedium,
		readinessPriority(blockers("still open"), yearEndedDaysAgo(5, now), now),
	)
	assert.Equal(
		t,
		notification.PriorityHigh,
		readinessPriority(blockers(), yearEndedDaysAgo(60, now), now),
		"two months past year end is a reporting problem, not a scheduling one",
	)
}

func TestDaysPastEndNeverGoesNegative(t *testing.T) {
	t.Parallel()

	now := time.Now().Unix()

	assert.Equal(t, 0, daysPastEnd(&fiscalyear.FiscalYear{EndDate: now + 86400}, now))
	assert.Equal(t, 3, daysPastEnd(&fiscalyear.FiscalYear{EndDate: now - 3*86400}, now))
}

func TestISOWeekKeyIsStableWithinAWeek(t *testing.T) {
	t.Parallel()

	monday := time.Date(2032, time.February, 9, 6, 0, 0, 0, time.UTC).Unix()
	friday := time.Date(2032, time.February, 13, 23, 0, 0, 0, time.UTC).Unix()
	nextMonday := time.Date(2032, time.February, 16, 6, 0, 0, 0, time.UTC).Unix()

	assert.Equal(t, isoWeekKey(monday), isoWeekKey(friday),
		"a daily schedule must resolve to one key all week")
	assert.NotEqual(t, isoWeekKey(friday), isoWeekKey(nextMonday),
		"a year left open must be raised again the following week")
}

// The fiscal calendar has to run ahead for every tenant: a close cannot carry
// balances into a year that does not exist, and that is not something a tenant
// opts into by enabling scheduled period close.
func TestGetFiscalCalendarTenantsIgnoresTheScheduledCloseOptIn(t *testing.T) {
	t.Parallel()

	acRepo := mocks.NewMockAccountingControlRepository(t)
	acRepo.EXPECT().ListAll(mock.Anything).Return([]*accountingcontrol.AccountingControl{
		{OrganizationID: pulid.MustNew("org_"), BusinessUnitID: pulid.MustNew("bu_")},
		{OrganizationID: pulid.MustNew("org_"), BusinessUnitID: pulid.MustNew("bu_")},
	}, nil).Once()

	activities := &Activities{acRepo: acRepo}

	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivity(activities.GetFiscalCalendarTenantsActivity)

	encoded, err := env.ExecuteActivity(activities.GetFiscalCalendarTenantsActivity)
	require.NoError(t, err)

	var result GetAutoCloseTenantsResult
	require.NoError(t, encoded.Get(&result))
	assert.Len(t, result.Tenants, 2)
}
