package conversationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func threadPageRequest() *agentports.ListThreadsRequest {
	return &agentports.ListThreadsRequest{
		UserID: pulid.MustNew("usr_"),
	}
}

const (
	threadPosition = `ROW(at_."pinned", greatest(at_."last_message_at", at_."created_at"), at_."id")`
	threadTouched  = `greatest(at_."last_message_at", at_."created_at")`
)

func TestThreadCursorRoundTripsThroughItsEncoding(t *testing.T) {
	t.Parallel()

	for _, cursor := range []threadCursor{
		{pinned: true, touchedAt: 1_790_000_000, id: pulid.MustNew("athr_")},
		{pinned: false, touchedAt: 1, id: pulid.MustNew("athr_")},
	} {
		decoded, err := decodeThreadCursor(cursor.encode())
		require.NoError(t, err)
		assert.Equal(t, cursor, *decoded)
	}
}

func TestThreadCursorRejectsWhatNoPageHandedOut(t *testing.T) {
	t.Parallel()

	for _, bad := range []string{
		"not base64!",
		"MQ",
		"Mi4xLmF0aHJfMDFKMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA",
		"MS54LmF0aHJfMDFKMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA",
		"MS4xLnNob3J0",
		"MC40Mi43LmF0aHJfMDFKMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA",
	} {
		_, err := decodeThreadCursor(bad)
		require.Error(t, err, bad)

		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation, bad)
		assert.Equal(t, "cursor", validation.Field, bad)
		assert.Equal(t, errortypes.ErrInvalidFormat, validation.Code, bad)
	}
}

func TestThreadCursorIsTakenFromTheLastRowOfThePage(t *testing.T) {
	t.Parallel()

	thread := &conversation.Thread{
		ID:            pulid.MustNew("athr_"),
		Pinned:        true,
		LastMessageAt: 300,
		CreatedAt:     200,
	}

	assert.Equal(t, threadCursor{
		pinned:    true,
		touchedAt: 300,
		id:        thread.ID,
	}, threadCursorAt(thread))
}

func TestThreadCursorOfAConversationWithNoMessageIsWhenItWasCreated(t *testing.T) {
	t.Parallel()

	thread := &conversation.Thread{ID: pulid.MustNew("athr_"), CreatedAt: 200}

	assert.Equal(t, int64(200), threadCursorAt(thread).touchedAt)
}

func TestThreadPageQueryContinuesBelowTheCursorInRailOrder(t *testing.T) {
	t.Parallel()

	req := threadPageRequest()
	after := &threadCursor{
		pinned:    false,
		touchedAt: 42,
		id:        "athr_01J00000000000000000000000",
	}

	sql, args := keeltest.Render(t, threadPageQuery(req, threadWindow{after: after, limit: 50}))

	assert.Contains(t, sql, threadPosition+" < ROW($4, $5, $6)", sql)
	assert.Equal(t, []any{false, int64(42), "athr_01J00000000000000000000000"}, args[3:6])
	assert.Contains(
		t,
		sql,
		`ORDER BY at_."pinned" DESC, `+threadTouched+` DESC, at_."id" DESC`,
		sql,
	)
	assert.Contains(t, sql, "LIMIT 51", "one more than the page says whether another follows")
	assert.Contains(
		t,
		sql,
		`at_."origin" NOT IN ('Ask', 'Import', 'Formula')`,
		"unlisted conversations stay off the rail",
	)
}

func TestThreadPageQueryReadsTheFirstPageWithoutACursor(t *testing.T) {
	t.Parallel()

	req := threadPageRequest()
	req.IncludeUnlisted = true

	sql, _ := keeltest.Render(t, threadPageQuery(req, threadWindow{limit: 100}))

	assert.NotContains(t, sql, ") < ROW(", sql)
	assert.NotContains(t, sql, `"origin" NOT IN`, sql)
	assert.Contains(t, sql, "LIMIT 101", sql)
}

func TestThreadPageQueryReadsOneRangeBetweenTwoBoundaries(t *testing.T) {
	t.Parallel()

	req := threadPageRequest()
	window := threadWindow{
		after: &threadCursor{
			pinned:    true,
			touchedAt: 900,
			id:        "athr_01J00000000000000000000009",
		},
		until: &threadCursor{
			pinned:    false,
			touchedAt: 42,
			id:        "athr_01J00000000000000000000001",
		},
		limit: maxThreadRangeLimit,
	}

	sql, args := keeltest.Render(t, threadPageQuery(req, window))

	assert.Contains(t, sql, threadPosition+" < ROW($4, $5, $6)", sql)
	assert.Contains(
		t,
		sql,
		threadPosition+" >= ROW($7, $8, $9)",
		"the boundary row's place is in the range, whichever row now sits there",
	)
	assert.Equal(t, []any{
		true, int64(900), "athr_01J00000000000000000000009",
		false, int64(42), "athr_01J00000000000000000000001",
	}, args[3:9])
	assert.Contains(t, sql, "LIMIT 1001", sql)
}

func TestThreadWindowOfReadsARangeWithoutAPageLimit(t *testing.T) {
	t.Parallel()

	until := threadCursor{
		pinned:    false,
		touchedAt: 42,
		id:        pulid.MustNew("athr_"),
	}
	req := threadPageRequest()
	req.Limit = 50
	req.Until = until.encode()

	window, err := threadWindowOf(req)
	require.NoError(t, err)

	require.NotNil(t, window.until)
	assert.Equal(t, until, *window.until)
	assert.Nil(t, window.after)
	assert.Equal(t, maxThreadRangeLimit, window.limit)

	req.Until = "not a cursor!"
	_, err = threadWindowOf(req)
	require.Error(t, err)
}

func TestThreadWindowOfCapsAPage(t *testing.T) {
	t.Parallel()

	req := threadPageRequest()
	req.Limit = 5000

	window, err := threadWindowOf(req)
	require.NoError(t, err)
	assert.Equal(t, maxThreadLimit, window.limit)

	req.Limit = 0
	window, err = threadWindowOf(req)
	require.NoError(t, err)
	assert.Equal(t, defaultThreadLimit, window.limit)
}

func TestThreadPageQueryReadsOneShelf(t *testing.T) {
	t.Parallel()

	req := threadPageRequest()
	pinned := false
	req.Pinned = &pinned
	req.TouchedFrom = 1_000
	req.TouchedBefore = 2_000

	sql, args := keeltest.Render(t, threadPageQuery(req, threadWindow{limit: 50}))

	assert.Contains(t, sql, `at_."pinned" = $4`, sql)
	assert.Contains(t, sql, threadTouched+" >= $5", sql)
	assert.Contains(t, sql, threadTouched+" < $6", sql)
	assert.Equal(t, []any{false, int64(1_000), int64(2_000)}, args[3:6])
}

func TestThreadPageQueryReadsOnlyPinnedConversations(t *testing.T) {
	t.Parallel()

	req := threadPageRequest()
	pinned := true
	req.Pinned = &pinned

	sql, args := keeltest.Render(t, threadPageQuery(req, threadWindow{limit: 50}))

	assert.Contains(t, sql, `at_."pinned" = $4`, sql)
	assert.Equal(t, true, args[3])
	assert.NotContains(t, sql, threadTouched+" >=", sql)
	assert.NotContains(t, sql, threadTouched+" <", sql)
}

func TestThreadPageQueryLeavesAnOpenEndedShelfUnbounded(t *testing.T) {
	t.Parallel()

	req := threadPageRequest()
	req.TouchedBefore = 2_000

	sql, args := keeltest.Render(t, threadPageQuery(req, threadWindow{limit: 50}))

	assert.NotContains(t, sql, `at_."pinned" =`, sql)
	assert.NotContains(t, sql, threadTouched+" >=", sql)
	assert.Contains(t, sql, threadTouched+" < $4", sql)
	assert.Equal(t, int64(2_000), args[3])
}

func TestThreadShelfCountQueryBandsEveryConversationOnce(t *testing.T) {
	t.Parallel()

	req := &agentports.CountThreadShelvesRequest{
		UserID: pulid.MustNew("usr_"),
		Edges:  []int64{3_000, 2_000, 1_000},
	}

	sql, args := keeltest.Render(t, threadShelfCountQuery(req))

	band := `CASE WHEN at_."pinned" THEN $4::bigint` +
		" WHEN " + threadTouched + " >= $5 THEN $6::bigint" +
		" WHEN " + threadTouched + " >= $7 THEN $8::bigint" +
		" WHEN " + threadTouched + " >= $9 THEN $10::bigint" +
		" ELSE $11::bigint END"
	assert.Contains(t, sql, "SELECT "+band+", count(*)", sql)
	assert.Contains(t, sql, ") GROUP BY 1", sql)
	assert.Equal(t, []any{
		int64(-1), int64(3_000), int64(0), int64(2_000), int64(1), int64(1_000), int64(2), int64(3),
	}, args[3:11])
	assert.Contains(
		t,
		sql,
		`at_."origin" NOT IN ('Ask', 'Import', 'Formula')`,
		"unlisted conversations are not counted",
	)
}
