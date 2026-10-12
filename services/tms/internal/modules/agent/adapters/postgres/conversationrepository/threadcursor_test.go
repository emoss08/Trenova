package conversationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func threadPageRequest() *agentports.ListThreadsRequest {
	return &agentports.ListThreadsRequest{
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
}

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

	db := bun.NewDB(nil, pgdialect.New())
	req := threadPageRequest()
	after := &threadCursor{
		pinned:    false,
		touchedAt: 42,
		id:        "athr_01J00000000000000000000000",
	}
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, threadWindow{after: after, limit: 50}).String()

	assert.Contains(
		t,
		sql,
		"(athr.pinned, greatest(athr.last_message_at, athr.created_at), athr.id) < (FALSE, 42, 'athr_01J00000000000000000000000')",
	)
	assert.Contains(
		t,
		sql,
		`ORDER BY "athr"."pinned" DESC, greatest(athr.last_message_at, athr.created_at) DESC, "athr"."id" DESC`,
	)
	assert.Contains(t, sql, "LIMIT 51", "one more than the page says whether another follows")
	assert.Contains(t, sql, "athr.origin NOT IN", "unlisted conversations stay off the rail")
}

func TestThreadPageQueryReadsTheFirstPageWithoutACursor(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	req := threadPageRequest()
	req.IncludeUnlisted = true
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, threadWindow{limit: 100}).String()

	assert.NotContains(t, sql, ") < (")
	assert.NotContains(t, sql, "athr.origin NOT IN")
	assert.Contains(t, sql, "LIMIT 101")
}

func TestThreadPageQueryReadsOneRangeBetweenTwoBoundaries(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
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
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, window).String()

	assert.Contains(
		t,
		sql,
		"(athr.pinned, greatest(athr.last_message_at, athr.created_at), athr.id) < (TRUE, 900, 'athr_01J00000000000000000000009')",
	)
	assert.Contains(
		t,
		sql,
		"(athr.pinned, greatest(athr.last_message_at, athr.created_at), athr.id) >= (FALSE, 42, 'athr_01J00000000000000000000001')",
		"the boundary row's place is in the range, whichever row now sits there",
	)
	assert.Contains(t, sql, "LIMIT 1001")
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

	db := bun.NewDB(nil, pgdialect.New())
	req := threadPageRequest()
	pinned := false
	req.Pinned = &pinned
	req.TouchedFrom = 1_000
	req.TouchedBefore = 2_000
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, threadWindow{limit: 50}).String()

	assert.Contains(t, sql, "athr.pinned = FALSE")
	assert.Contains(t, sql, "greatest(athr.last_message_at, athr.created_at) >= 1000")
	assert.Contains(t, sql, "greatest(athr.last_message_at, athr.created_at) < 2000")
}

func TestThreadPageQueryReadsOnlyPinnedConversations(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	req := threadPageRequest()
	pinned := true
	req.Pinned = &pinned
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, threadWindow{limit: 50}).String()

	assert.Contains(t, sql, "athr.pinned = TRUE")
	assert.NotContains(t, sql, "greatest(athr.last_message_at, athr.created_at) >=")
	assert.NotContains(t, sql, "greatest(athr.last_message_at, athr.created_at) <")
}

func TestThreadPageQueryLeavesAnOpenEndedShelfUnbounded(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	req := threadPageRequest()
	req.TouchedBefore = 2_000
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, threadWindow{limit: 50}).String()

	assert.NotContains(t, sql, "athr.pinned =")
	assert.NotContains(t, sql, "greatest(athr.last_message_at, athr.created_at) >=")
	assert.Contains(t, sql, "greatest(athr.last_message_at, athr.created_at) < 2000")
}

func TestThreadShelfCountQueryBandsEveryConversationOnce(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	req := &agentports.CountThreadShelvesRequest{
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Edges:      []int64{3_000, 2_000, 1_000},
	}

	sql := threadShelfCountQuery(db, req).String()

	assert.Contains(
		t,
		sql,
		"CASE WHEN athr.pinned THEN -1"+
			" WHEN greatest(athr.last_message_at, athr.created_at) >= 3000 THEN 0"+
			" WHEN greatest(athr.last_message_at, athr.created_at) >= 2000 THEN 1"+
			" WHEN greatest(athr.last_message_at, athr.created_at) >= 1000 THEN 2"+
			" ELSE 3 END AS band",
	)
	assert.Contains(t, sql, "COUNT(*) AS total")
	assert.Contains(t, sql, "GROUP BY band")
	assert.Contains(t, sql, "athr.origin NOT IN", "unlisted conversations are not counted")
}
