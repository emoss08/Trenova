package supportlog_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_800_000_000)

func requested() *supportlog.LogRequest {
	return supportlog.New(&supportlog.NewRequest{
		RequestedByID:  pulid.MustNew("usr_"),
		RequesterName:  " Ada Lovelace ",
		RequesterEmail: "ada@example.com",
		Period:         supportlog.PeriodDay,
		Now:            now,
	})
}

func TestNew_ReachesBackOverThePeriod(t *testing.T) {
	t.Parallel()

	for period, seconds := range map[supportlog.Period]int64{
		supportlog.PeriodHour: 3600,
		supportlog.PeriodDay:  86_400,
		supportlog.PeriodWeek: 604_800,
	} {
		req := supportlog.New(&supportlog.NewRequest{Period: period, Now: now})
		assert.Equal(t, now-seconds, req.Since, period)
		assert.Equal(t, now, req.Until, period)
		assert.Equal(t, supportlog.StatusRequested, req.Status, period)
	}
	assert.Equal(t, "Ada Lovelace", requested().RequesterName)
}

func TestValidate_RefusesAnUnknownPeriodAndAMissingRequester(t *testing.T) {
	t.Parallel()

	req := supportlog.New(&supportlog.NewRequest{Period: supportlog.Period("month"), Now: now})
	multiErr := errortypes.NewMultiError()
	req.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	multiErr = errortypes.NewMultiError()
	requested().Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestApply_MovesThroughCollectingToSent(t *testing.T) {
	t.Parallel()

	req := requested()
	changed, err := req.Apply(&supportlog.Report{Status: supportlog.StatusCollecting, BundleID: "lb_1"}, now+10)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, supportlog.StatusCollecting, req.Status)
	assert.Equal(t, "lb_1", req.BundleID)
	require.NotNil(t, req.CollectingAt)
	assert.Equal(t, now+10, *req.CollectingAt)

	changed, err = req.Apply(&supportlog.Report{Status: supportlog.StatusCollecting}, now+20)
	require.NoError(t, err)
	assert.False(t, changed, "a repeated report changes nothing")
	assert.Equal(t, now+10, *req.CollectingAt)

	changed, err = req.Apply(&supportlog.Report{Status: supportlog.StatusSent}, now+30)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, supportlog.StatusSent, req.Status)
	require.NotNil(t, req.FinishedAt)

	changed, err = req.Apply(&supportlog.Report{Status: supportlog.StatusSent, BundleID: "lb_1"}, now+40)
	require.NoError(t, err)
	assert.False(t, changed, "the operator may repeat its last report")
}

func TestApply_SentNeedsABundle(t *testing.T) {
	t.Parallel()

	_, err := requested().Apply(&supportlog.Report{Status: supportlog.StatusSent}, now)
	require.Error(t, err)
	_, isRefusal := errortypes.RefusalOf(err)
	assert.True(t, isRefusal)
}

func TestApply_FailedKeepsTheSentenceOrSaysWhy(t *testing.T) {
	t.Parallel()

	req := requested()
	_, err := req.Apply(&supportlog.Report{Status: supportlog.StatusFailed, Message: " Loki did not answer. "}, now)
	require.NoError(t, err)
	assert.Equal(t, "Loki did not answer.", req.FailureMessage)

	req = requested()
	_, err = req.Apply(&supportlog.Report{Status: supportlog.StatusFailed}, now)
	require.NoError(t, err)
	assert.Equal(t, supportlog.MessageFailedWithoutReason, req.FailureMessage)
}

func TestApply_RefusesWhatCannotFollow(t *testing.T) {
	t.Parallel()

	sent := requested()
	_, err := sent.Apply(&supportlog.Report{Status: supportlog.StatusSent, BundleID: "lb_1"}, now)
	require.NoError(t, err)

	_, err = sent.Apply(&supportlog.Report{Status: supportlog.StatusFailed}, now)
	require.ErrorIs(t, err, supportlog.ErrFinished)

	_, err = sent.Apply(&supportlog.Report{Status: supportlog.StatusSent, BundleID: "lb_2"}, now)
	require.ErrorIs(t, err, supportlog.ErrBundleMismatch)

	for name, report := range map[string]*supportlog.Report{
		"back to requested": {Status: supportlog.StatusRequested},
		"an unknown status": {Status: supportlog.Status("stored")},
		"a bundle id with a slash": {
			Status:   supportlog.StatusCollecting,
			BundleID: "../etc",
		},
		"a message too long": {
			Status:  supportlog.StatusFailed,
			Message: string(make([]byte, supportlog.MaxFailureMessageLength+1)),
		},
	} {
		_, err = requested().Apply(report, now)
		require.Error(t, err, name)
		_, isRefusal := errortypes.RefusalOf(err)
		assert.True(t, isRefusal, name)
	}
}

func TestOverdue_FailsAWaitingRequestAfterThirtyMinutesAndCollectingAfterTwoHours(t *testing.T) {
	t.Parallel()

	waiting := requested()
	assert.False(t, waiting.FailOverdue(now+29*60))
	assert.True(t, waiting.FailOverdue(now+30*60))
	assert.Equal(t, supportlog.StatusFailed, waiting.Status)
	assert.Equal(t, supportlog.MessageNotPickedUp, waiting.FailureMessage)

	collecting := requested()
	_, err := collecting.Apply(&supportlog.Report{Status: supportlog.StatusCollecting}, now+60)
	require.NoError(t, err)
	assert.False(t, collecting.FailOverdue(now+60+2*3600-1))
	assert.True(t, collecting.FailOverdue(now+60+2*3600))
	assert.Equal(t, supportlog.MessageNotFinished, collecting.FailureMessage)

	sent := requested()
	_, err = sent.Apply(&supportlog.Report{Status: supportlog.StatusSent, BundleID: "lb_1"}, now)
	require.NoError(t, err)
	assert.False(t, sent.FailOverdue(now+30*24*3600), "a finished request is never failed later")
}

func TestLoad_Remaining(t *testing.T) {
	t.Parallel()

	assert.Equal(t, supportlog.DailyLimit, (&supportlog.Load{}).Remaining())
	assert.Equal(t, 0, (&supportlog.Load{Recent: supportlog.DailyLimit + 2}).Remaining())
}
