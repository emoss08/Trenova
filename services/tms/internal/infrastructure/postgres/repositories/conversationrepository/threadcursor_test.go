package conversationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func threadPageRequest() *repositories.ListThreadsRequest {
	return &repositories.ListThreadsRequest{
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
}

func TestThreadCursorRoundTripsThroughItsEncoding(t *testing.T) {
	t.Parallel()

	for _, cursor := range []threadCursor{
		{
			pinned:        true,
			lastMessageAt: 1_790_000_000,
			createdAt:     1_789_000_000,
			id:            pulid.MustNew("athr_"),
		},
		{pinned: false, lastMessageAt: 0, createdAt: 1, id: pulid.MustNew("athr_")},
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
		"Mi4xLjIuYXRocl8wMUowMDAwMDAwMDAwMDAwMDAwMDAwMDAw",
		"MS54LjIuYXRocl8wMUowMDAwMDAwMDAwMDAwMDAwMDAwMDAw",
		"MS4xLjIuc2hvcnQ",
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
		pinned:        true,
		lastMessageAt: 300,
		createdAt:     200,
		id:            thread.ID,
	}, threadCursorAt(thread))
}

func TestThreadPageQueryContinuesBelowTheCursorInRailOrder(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	req := threadPageRequest()
	after := &threadCursor{
		pinned:        false,
		lastMessageAt: 42,
		createdAt:     7,
		id:            "athr_01J00000000000000000000000",
	}
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, threadWindow{after: after, limit: 50}).String()

	assert.Contains(
		t,
		sql,
		"(athr.pinned, athr.last_message_at, athr.created_at, athr.id) < (FALSE, 42, 7, 'athr_01J00000000000000000000000')",
	)
	assert.Contains(
		t,
		sql,
		"ORDER BY athr.pinned DESC, athr.last_message_at DESC, athr.created_at DESC, athr.id DESC",
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
			pinned:        true,
			lastMessageAt: 900,
			createdAt:     800,
			id:            "athr_01J00000000000000000000009",
		},
		until: &threadCursor{
			pinned:        false,
			lastMessageAt: 42,
			createdAt:     7,
			id:            "athr_01J00000000000000000000001",
		},
		limit: maxThreadRangeLimit,
	}
	var dest []*conversation.Thread

	sql := threadPageQuery(db, &dest, req, window).String()

	assert.Contains(
		t,
		sql,
		"(athr.pinned, athr.last_message_at, athr.created_at, athr.id) < (TRUE, 900, 800, 'athr_01J00000000000000000000009')",
	)
	assert.Contains(
		t,
		sql,
		"(athr.pinned, athr.last_message_at, athr.created_at, athr.id) >= (FALSE, 42, 7, 'athr_01J00000000000000000000001')",
		"the boundary row's place is in the range, whichever row now sits there",
	)
	assert.Contains(t, sql, "LIMIT 1001")
}

func TestThreadWindowOfReadsARangeWithoutAPageLimit(t *testing.T) {
	t.Parallel()

	until := threadCursor{
		pinned:        false,
		lastMessageAt: 42,
		createdAt:     7,
		id:            pulid.MustNew("athr_"),
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
