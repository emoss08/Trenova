package supportlognotice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportlogports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type created struct {
	items []*notification.Notification
}

func (c *created) Create(_ context.Context, n *notification.Notification) (*notification.Notification, error) {
	c.items = append(c.items, n)
	return n, nil
}

func finished(status supportlog.Status) *supportlog.LogRequest {
	finishedAt := int64(1_800_000_000)
	return &supportlog.LogRequest{
		ID:             pulid.MustNew(supportlog.RequestIDPrefix),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		RequestedByID:  pulid.MustNew("usr_"),
		Period:         supportlog.PeriodWeek,
		Status:         status,
		BundleID:       "lb_1",
		FailureMessage: "Loki did not answer.",
		FinishedAt:     &finishedAt,
	}
}

func TestLogRequestFinished_TellsOnlyWhoAsked(t *testing.T) {
	t.Parallel()

	sink := &created{}
	n := &Notifier{notifications: sink}
	req := finished(supportlog.StatusSent)

	require.NoError(t, n.LogRequestFinished(t.Context(), &supportlogports.FinishedNotice{Request: req}))
	require.Len(t, sink.items, 1)
	got := sink.items[0]
	assert.Equal(t, req.RequestedByID, *got.TargetUserID)
	assert.Equal(t, req.OrganizationID, got.OrganizationID)
	assert.Equal(t, EventSent, got.EventType)
	assert.Contains(t, got.Message, "last 7 days")
	assert.Contains(t, got.Message, "30 days")
	assert.Equal(t, SupportLogPath, got.Data["link"])
	assert.NotContains(t, got.Message, req.ID.String(), "no record id reaches the reader")
}

func TestLogRequestFinished_AFailureCarriesItsSentence(t *testing.T) {
	t.Parallel()

	sink := &created{}
	n := &Notifier{notifications: sink}

	require.NoError(t, n.LogRequestFinished(t.Context(), &supportlogports.FinishedNotice{
		Request: finished(supportlog.StatusFailed),
	}))
	require.Len(t, sink.items, 1)
	assert.Equal(t, EventFailed, sink.items[0].EventType)
	assert.Equal(t, "Loki did not answer.", sink.items[0].Message)
}

func TestLogRequestFinished_SaysNothingBeforeTheEnd(t *testing.T) {
	t.Parallel()

	sink := &created{}
	n := &Notifier{notifications: sink}
	require.NoError(t, n.LogRequestFinished(t.Context(), &supportlogports.FinishedNotice{
		Request: finished(supportlog.StatusCollecting),
	}))
	assert.Empty(t, sink.items)
}
