package assistanthandler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListThreadsReadsOneShelf(t *testing.T) {
	t.Parallel()

	service, status := call(
		t,
		http.MethodGet,
		"/assistant/threads/?pinned=false&touchedFrom=1000&touchedBefore=2000&cursor=abc",
		"",
	)

	assert.Equal(t, http.StatusOK, status)
	require.NotNil(t, service.listed)
	require.NotNil(t, service.listed.Pinned)
	assert.False(t, *service.listed.Pinned)
	assert.Equal(t, int64(1000), service.listed.TouchedFrom)
	assert.Equal(t, int64(2000), service.listed.TouchedBefore)
	assert.Equal(t, "abc", service.listed.Cursor)
}

func TestListThreadsReadsEveryShelfWithoutAFilter(t *testing.T) {
	t.Parallel()

	service, status := call(t, http.MethodGet, "/assistant/threads/", "")

	assert.Equal(t, http.StatusOK, status)
	require.NotNil(t, service.listed)
	assert.Nil(t, service.listed.Pinned)
	assert.Zero(t, service.listed.TouchedFrom)
	assert.Zero(t, service.listed.TouchedBefore)
}

func TestCountThreadShelvesReadsEachEdgeInOrder(t *testing.T) {
	t.Parallel()

	service, status := call(
		t,
		http.MethodGet,
		"/assistant/threads/shelves/?edge=3000&edge=2000&edge=1000",
		"",
	)

	assert.Equal(t, http.StatusOK, status)
	require.NotNil(t, service.counted)
	assert.Equal(t, []int64{3000, 2000, 1000}, service.counted.Edges)
}

func TestCountThreadShelvesIsNotTakenForAConversation(t *testing.T) {
	t.Parallel()

	service, status := call(t, http.MethodGet, "/assistant/threads/shelves/", "")

	assert.Equal(t, http.StatusOK, status)
	require.NotNil(t, service.counted, "the shelves route, not the conversation route")
	assert.Empty(t, service.counted.Edges)
}

func TestCountThreadShelvesRefusesAnEdgeThatIsNotANumber(t *testing.T) {
	t.Parallel()

	service, status := call(t, http.MethodGet, "/assistant/threads/shelves/?edge=soon", "")

	assert.GreaterOrEqual(t, status, http.StatusBadRequest)
	assert.Nil(t, service.counted)
}
