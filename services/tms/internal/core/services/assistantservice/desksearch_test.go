package assistantservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deskSearchConversations struct {
	repositories.ConversationRepository

	calls []repositories.SearchDeskRequest
	rows  map[string][]repositories.DeskSearchRow
}

func (c *deskSearchConversations) SearchDesk(
	_ context.Context,
	req repositories.SearchDeskRequest,
) ([]repositories.DeskSearchRow, error) {
	c.calls = append(c.calls, req)
	out := []repositories.DeskSearchRow{}
	for _, kind := range req.Kinds {
		out = append(out, c.rows[kind]...)
	}

	return out, nil
}

func TestSearchDesk_NothingTypedListsRecentChatsThenArtifacts(t *testing.T) {
	t.Parallel()

	repo := &deskSearchConversations{rows: map[string][]repositories.DeskSearchRow{
		"chat": {{Kind: "chat", ID: "athr_1", Title: "Storm loads"}},
		"art":  {{Kind: "art", ID: "aart_1", Title: "Workers"}},
	}}
	service := &Service{conversations: repo}
	actor := providerActor()

	results, err := service.SearchDesk(t.Context(), actor, serviceports.DeskSearchRequest{})
	require.NoError(t, err)

	require.Len(t, repo.calls, 2)
	assert.Equal(t, []string{"chat"}, repo.calls[0].Kinds)
	assert.Equal(t, deskSearchRecentChats, repo.calls[0].LimitPerKind)
	assert.Equal(t, []string{"art"}, repo.calls[1].Kinds)
	assert.Equal(t, actor.UserID, repo.calls[0].UserID)
	require.Len(t, results, 2)
	assert.Equal(t, "chat", results[0].Kind)
	assert.Equal(t, "art", results[1].Kind)
}

func TestSearchDesk_QuerySearchesEveryKindAndTrimsMessages(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("filler words ", 20) + "the missing biller on SEED-INV-7 " +
		strings.Repeat("more text ", 30)
	repo := &deskSearchConversations{rows: map[string][]repositories.DeskSearchRow{
		"msg": {{Kind: "msg", ID: "amsg_1", Title: long}},
	}}
	service := &Service{conversations: repo}

	results, err := service.SearchDesk(
		t.Context(),
		providerActor(),
		serviceports.DeskSearchRequest{Query: "  missing   biller "},
	)
	require.NoError(t, err)

	require.Len(t, repo.calls, 1)
	assert.Equal(t, deskSearchKinds, repo.calls[0].Kinds)
	assert.Equal(t, "missing biller", repo.calls[0].Query)
	assert.Equal(t, deskSearchLimitAllKinds, repo.calls[0].LimitPerKind)
	require.Len(t, results, 1)
	assert.True(t, strings.HasPrefix(results[0].Title, "…"))
	assert.True(t, strings.HasSuffix(results[0].Title, "…"))
	assert.Contains(t, results[0].Title, "missing biller")
}

func TestSearchDesk_OneKindAndUnknownKinds(t *testing.T) {
	t.Parallel()

	repo := &deskSearchConversations{}
	service := &Service{conversations: repo}

	_, err := service.SearchDesk(t.Context(), providerActor(), serviceports.DeskSearchRequest{Kind: "dec"})
	require.NoError(t, err)
	require.Len(t, repo.calls, 1)
	assert.Equal(t, []string{"dec"}, repo.calls[0].Kinds)
	assert.Equal(t, deskSearchLimitOneKind, repo.calls[0].LimitPerKind)

	results, err := service.SearchDesk(t.Context(), providerActor(), serviceports.DeskSearchRequest{Kind: "nope"})
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Len(t, repo.calls, 1)
}

func TestDeskSnippet_ShortMessageStaysWhole(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Two loads late.", deskSnippet("Two  loads\nlate.", "loads"))
}
