package assistantservice

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	deskSearchLimitOneKind  = 14
	deskSearchLimitAllKinds = 5
	deskSearchRecentChats   = 5
	deskSearchRecentArts    = 3
	maxDeskSearchQueryLen   = 100
	// How much of a message shows around what matched.
	deskSnippetLead = 36
	deskSnippetLen  = 180
)

var deskSearchKinds = []string{"chat", "msg", "art", "dec"}

// SearchDesk finds what a person has in the Desk: conversations by title,
// messages by what was said, and artifacts and decisions by title. Only their
// own conversations are read, so nothing here needs a further permission.
// With nothing typed it lists recent conversations and artifacts.
func (s *Service) SearchDesk(
	ctx context.Context,
	actor serviceports.RequestActor,
	req serviceports.DeskSearchRequest,
) ([]serviceports.DeskSearchResult, error) {
	kind := strings.TrimSpace(req.Kind)
	if kind == "all" {
		kind = ""
	}
	query := strings.Join(strings.Fields(req.Query), " ")
	if utf8.RuneCountInString(query) > maxDeskSearchQueryLen {
		query = string([]rune(query)[:maxDeskSearchQueryLen])
	}
	if kind != "" && !slices.Contains(deskSearchKinds, kind) {
		return []serviceports.DeskSearchResult{}, nil
	}

	search := func(kinds []string, limit int) ([]repositories.DeskSearchRow, error) {
		return s.conversations.SearchDesk(ctx, repositories.SearchDeskRequest{
			TenantInfo:   actor.TenantInfo(),
			UserID:       actor.UserID,
			Query:        query,
			Kinds:        kinds,
			LimitPerKind: limit,
		})
	}

	var rows []repositories.DeskSearchRow
	var err error
	switch {
	case kind != "":
		rows, err = search([]string{kind}, deskSearchLimitOneKind)
	case query != "":
		rows, err = search(deskSearchKinds, deskSearchLimitAllKinds)
	default:
		var chats, arts []repositories.DeskSearchRow
		if chats, err = search([]string{"chat"}, deskSearchRecentChats); err == nil {
			arts, err = search([]string{"art"}, deskSearchRecentArts)
		}
		rows = append(chats, arts...)
	}
	if err != nil {
		return nil, err
	}

	out := make([]serviceports.DeskSearchResult, 0, len(rows))
	for _, row := range rows {
		title := row.Title
		if row.Kind == "msg" {
			title = deskSnippet(row.Title, query)
		}
		out = append(out, serviceports.DeskSearchResult{
			Kind:         row.Kind,
			ID:           row.ID,
			ThreadID:     row.ThreadID,
			AgentID:      row.AgentID,
			Title:        title,
			ThreadTitle:  row.ThreadTitle,
			ArtifactKind: row.ArtifactKind,
			Status:       row.Status,
			At:           row.At,
		})
	}

	return out, nil
}

// deskSnippet is a message on one line, starting a little before the first
// place it matches so the match is in view.
func deskSnippet(content, query string) string {
	text := []rune(strings.Join(strings.Fields(content), " "))
	start := 0
	if query != "" {
		lower := []rune(strings.ToLower(string(text)))
		if at := strings.Index(string(lower), strings.ToLower(query)); at >= 0 {
			start = max(0, utf8.RuneCountInString(string(lower)[:at])-deskSnippetLead)
		}
	}
	end := min(len(text), start+deskSnippetLen)
	snippet := string(text[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(text) {
		snippet += "…"
	}

	return snippet
}
