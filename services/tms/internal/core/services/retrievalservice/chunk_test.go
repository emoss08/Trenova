package retrievalservice

import (
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/llmtokens"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func numberedWords(count int) string {
	words := make([]string, 0, count)
	for idx := range count {
		words = append(words, "word"+strconv.Itoa(idx))
	}

	return strings.Join(words, " ")
}

func TestWindowsStayWithinTheTokenBudgetAndOverlap(t *testing.T) {
	t.Parallel()

	windows := Windows(numberedWords(2000))
	require.Greater(t, len(windows), 3)

	for idx, window := range windows {
		assert.LessOrEqual(t, llmtokens.Estimate(window), WindowTokens+1, "window %d", idx)
		if idx == 0 {
			continue
		}

		previous := strings.Fields(windows[idx-1])
		current := strings.Fields(window)
		overlap := 0
		for overlap < len(previous) && overlap < len(current) &&
			previous[len(previous)-1-overlap] != current[0] {
			overlap++
		}
		shared := strings.Join(previous[len(previous)-1-overlap:], " ")
		assert.True(t, strings.HasPrefix(window, shared),
			"window %d starts where the previous one's tail began", idx)
		sharedTokens := llmtokens.Estimate(shared)
		assert.GreaterOrEqual(t, sharedTokens, WindowTokens*WindowOverlapPct/100-8,
			"window %d overlaps the previous one by about 15%%", idx)
		assert.LessOrEqual(t, sharedTokens, WindowTokens*WindowOverlapPct/100+4)
	}

	last := strings.Fields(windows[len(windows)-1])
	assert.Equal(t, "word1999", last[len(last)-1], "the text is covered to its end")
}

func TestWindowsOfShortTextIsOneWindow(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"a short page"}, Windows("  a short\n page "))
	assert.Empty(t, Windows(" \n\t "))
}

func TestWindowsCutAWordLongerThanAWindow(t *testing.T) {
	t.Parallel()

	windows := Windows(strings.Repeat("x", windowRunes()*5))
	require.Len(t, windows, 1)
	assert.Len(t, []rune(windows[0]), windowRunes()*maxWindowMultiplier)
}

func documentSource(pages ...string) *repositories.RetrievalDocumentSource {
	docID := pulid.MustNew("doc_")
	source := &repositories.RetrievalDocumentSource{
		Document: &document.Document{
			ID:               docID,
			OriginalName:     "acme-ratecon.pdf",
			ResourceType:     "shipment",
			IsCurrentVersion: true,
			Status:           document.StatusActive,
		},
		Content: &documentcontent.Content{
			ID:                   pulid.MustNew("dc_"),
			DocumentID:           docID,
			Status:               documentcontent.StatusIndexed,
			DetectedDocumentKind: "RateConfirmation",
			PageCount:            len(pages),
		},
	}
	for idx, text := range pages {
		source.Pages = append(source.Pages, &documentcontent.Page{
			PageNumber:    idx + 1,
			ExtractedText: text,
		})
	}

	return source
}

func TestChunkDocumentWindowsEachPageUnderAHeader(t *testing.T) {
	t.Parallel()

	source := documentSource("Linehaul $2,450 for load 4471.", "Detention $75 per hour.")
	source.Content.StructuredData = map[string]any{
		"loadNumber": "4471",
		"stops":      []any{map[string]any{"city": "Dallas"}, map[string]any{"city": "Chicago"}},
		"hazmat":     false,
		"weight":     42000.5,
	}

	chunks := ChunkDocument(source)
	require.Len(t, chunks, 3)

	fields := chunks[0]
	assert.Equal(t, 0, fields.Page)
	assert.Contains(t, fields.Text, "Document: acme-ratecon.pdf")
	assert.Contains(t, fields.Text, "Kind: RateConfirmation")
	assert.Contains(t, fields.Text, "Attached to: shipment")
	assert.Equal(
		t,
		"hazmat: false\nloadNumber: 4471\nstops[0].city: Dallas\nstops[1].city: Chicago\nweight: 42000.5",
		fields.Body,
	)

	assert.Equal(t, 1, chunks[1].Page)
	assert.Contains(t, chunks[1].Text, "Page 1 of 2")
	assert.Equal(t, "Linehaul $2,450 for load 4471.", chunks[1].Body)
	assert.Equal(t, 2, chunks[2].Page)

	for idx, chunk := range chunks {
		assert.Equal(t, idx, chunk.Index)
		assert.Equal(t, ChunkHash(DocumentChunkerVersion, chunk.Text), chunk.Hash)
	}
}

func TestChunkDocumentFallsBackToTheContentText(t *testing.T) {
	t.Parallel()

	source := documentSource()
	source.Content.ContentText = "A text file with no pages."

	chunks := ChunkDocument(source)
	require.Len(t, chunks, 1)
	assert.Equal(t, 1, chunks[0].Page)
	assert.Equal(t, "A text file with no pages.", chunks[0].Body)
}

func TestChunkDocumentStopsAtTheChunkCap(t *testing.T) {
	t.Parallel()

	pages := make([]string, 0, 120)
	for range 120 {
		pages = append(pages, numberedWords(600))
	}

	chunks := ChunkDocument(documentSource(pages...))
	assert.Len(t, chunks, MaxDocumentChunks)
	assert.Equal(t, MaxDocumentChunks-1, chunks[len(chunks)-1].Index)
}

func TestChunkEmailStripsQuotesAndSignatures(t *testing.T) {
	t.Parallel()

	message := &inboundmessage.InboundMessage{
		FromName:    "Dana",
		FromAddress: "dana@globex.test",
		Subject:     "Driver never showed",
		TextBody: "Your driver missed the 7 AM pickup.\n> earlier quoted line\nPlease call me.\n" +
			"-- \nDana Whitfield\nGlobex\n\nOn Mon, Sep 21, 2026 Dispatch wrote:\n> old thread",
	}

	chunks := ChunkEmail(message)
	require.Len(t, chunks, 1)
	assert.Contains(t, chunks[0].Text, "Email from Dana <dana@globex.test>")
	assert.Contains(t, chunks[0].Text, "Subject: Driver never showed")
	assert.Equal(t, "Your driver missed the 7 AM pickup. Please call me.", chunks[0].Body)
	assert.NotContains(t, chunks[0].Text, "quoted")
	assert.NotContains(t, chunks[0].Text, "Whitfield")
	assert.Equal(t, ChunkHash(EmailChunkerVersion, chunks[0].Text), chunks[0].Hash)
}

func TestChunkEmailWithoutABodyIsItsHeader(t *testing.T) {
	t.Parallel()

	chunks := ChunkEmail(&inboundmessage.InboundMessage{
		FromAddress: "ops@acme.test",
		Subject:     "Rate sheet",
		TextBody:    "> only a quote",
	})
	require.Len(t, chunks, 1)
	assert.Equal(t, "Email from ops@acme.test\nSubject: Rate sheet", chunks[0].Text)
	assert.Empty(t, chunks[0].Body)
}

func TestChunkEmailStopsAtTheChunkCap(t *testing.T) {
	t.Parallel()

	chunks := ChunkEmail(&inboundmessage.InboundMessage{
		FromAddress: "ops@acme.test",
		TextBody:    numberedWords(20000),
	})
	assert.Len(t, chunks, MaxEmailChunks)
}

func TestChunkMemoryIsOneLine(t *testing.T) {
	t.Parallel()

	correction := &agent.Memory{
		Kind:     agent.MemoryKindCorrection,
		ToolName: "assign_move",
		Content:  " Loads over 900 miles need a team. ",
	}
	chunks := ChunkMemory(correction)
	require.Len(t, chunks, 1)
	assert.Equal(
		t,
		"[Correction] tool assign_move: Loads over 900 miles need a team.",
		chunks[0].Text,
	)
	assert.Equal(t, ChunkHash(MemoryChunkerVersion, chunks[0].Text), chunks[0].Hash)

	fact := ChunkMemory(&agent.Memory{Kind: agent.MemoryKindFact, Content: "Acme closes at 3."})
	assert.Equal(t, "[Fact] Acme closes at 3.", fact[0].Text)

	assert.Empty(t, ChunkMemory(&agent.Memory{Kind: agent.MemoryKindFact, Content: "  "}))
}

func TestChunkHashFoldsInTheChunkerVersion(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t,
		ChunkHash(DocumentChunkerVersion, "same text"),
		ChunkHash(EmailChunkerVersion, "same text"))
	assert.NotEqual(t,
		ChunkHash(DocumentChunkerVersion, "same text"),
		ChunkHash(DocumentChunkerVersion, "same text."))
	assert.Len(t, ChunkHash(MemoryChunkerVersion, "x"), 64)
}

func TestChangedChunksDiffsByIndexAndHash(t *testing.T) {
	t.Parallel()

	chunks := sealChunks(DocumentChunkerVersion, []Chunk{
		{Text: "unchanged"}, {Text: "edited"}, {Text: "new"},
	})
	stored := []repositories.EmbeddingChunkHash{
		{ChunkIndex: 0, ContentHash: chunks[0].Hash},
		{ChunkIndex: 1, ContentHash: ChunkHash(DocumentChunkerVersion, "before the edit")},
		{ChunkIndex: 7, ContentHash: "dropped"},
	}

	assert.Equal(t, []int{1, 2}, ChangedChunks(chunks, stored))
	assert.Empty(t, ChangedChunks(chunks[:1], stored))
}
