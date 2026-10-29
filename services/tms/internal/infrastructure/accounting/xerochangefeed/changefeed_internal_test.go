package xerochangefeed

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerorefsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerosession"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pollNow = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

var feedAuth = accountingconnectorports.AccountingDocumentAuth{
	RealmID:     "4c7e1f20-3b5a-4d8e-9f10-2a3b4c5d6e7f",
	AccessToken: "access-token",
}

func testFeed(t *testing.T) *Feed {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Type":"NotFound","Message":"not found"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	session := xerosession.NewSession(
		[]xero.Option{xero.WithBaseURL(server.URL), xero.WithRetry(restx.RetryConfig{})},
		server.Client(),
		xerosession.NewAccountCodes(time.Now),
	)
	return NewFeed(session, &xerorefsync.References{Session: session})
}

func TestChangeCursorRoundTripsAndRejectsForeignCursors(t *testing.T) {
	t.Parallel()

	paging := &changeCursor{
		since:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		paging:  true,
		started: pollNow,
		latest:  pollNow.Add(-time.Minute),
		targets: []changeTarget{targetPayments, targetContacts},
		index:   1,
		page:    3,
	}
	parsed, err := parseChangeCursor(paging.String(), pollNow)
	require.NoError(t, err)
	assert.Equal(t, paging, parsed)

	empty, err := parseChangeCursor("", pollNow)
	require.NoError(t, err)
	assert.Equal(t, pollNow, empty.since)

	for _, bad := range []string{
		"yesterday",
		"t:1790000000",
		"p|2026-09-01T00:00:00Z||||0|1",
		"p|2026-09-01T00:00:00Z|||journals|0|1",
		"p|2026-09-01T00:00:00Z|||payments|1|1",
		"p|2026-09-01T00:00:00Z|||payments|0|0",
	} {
		_, err = parseChangeCursor(bad, pollNow)
		require.ErrorIs(t, err, ErrChangeCursor, bad)
	}
}

func TestNextSinceIsTheLatestChangeLessOneSecond(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 9, 29, 11, 15, 30, 500, time.UTC)
	cases := []struct {
		name    string
		latest  time.Time
		started time.Time
		want    time.Time
	}{
		{"nothing seen", time.Time{}, time.Time{}, since},
		{
			"latest less a second",
			latest,
			time.Time{},
			time.Date(2026, 9, 29, 11, 15, 29, 0, time.UTC),
		},
		{"never before since", since, time.Time{}, since},
		{
			"paging bounded by its start",
			latest,
			since.Add(30 * time.Minute),
			since.Add(30*time.Minute - time.Second),
		},
		{
			"start after latest",
			latest,
			latest.Add(time.Hour),
			time.Date(2026, 9, 29, 11, 15, 29, 0, time.UTC),
		},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, nextSince(since, tc.latest, tc.started), tc.name)
	}
}

func TestReadChangesPagingAdvancesAcrossTargets(t *testing.T) {
	t.Parallel()

	conn := testFeed(t)
	cursor := &changeCursor{
		since:   time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
		paging:  true,
		started: pollNow,
		latest:  time.Date(2026, 9, 29, 11, 0, 0, 0, time.UTC),
		targets: []changeTarget{targetPayments, targetContacts},
		page:    2,
	}
	page, err := conn.ReadChanges(
		t.Context(),
		&accountingconnectorports.ReadAccountingChangesRequest{
			Auth: feedAuth, Cursor: cursor.String(), Now: pollNow, Payments: true,
		},
	)
	require.NoError(t, err)
	assert.True(t, page.More)
	next, err := parseChangeCursor(page.NextCursor, pollNow)
	require.NoError(t, err)
	assert.Equal(t, 1, next.index)
	assert.Equal(t, 2, next.page)

	done, err := conn.ReadChanges(
		t.Context(),
		&accountingconnectorports.ReadAccountingChangesRequest{
			Auth: feedAuth, Cursor: page.NextCursor, Now: pollNow, Payments: true,
		},
	)
	require.NoError(t, err)
	assert.False(t, done.More)
	assert.Equal(t, "2026-09-29T10:59:59Z", done.NextCursor)
}
