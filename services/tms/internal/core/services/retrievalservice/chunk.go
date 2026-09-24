package retrievalservice

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/llmtokens"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MemoryChunkerVersion   = "memory/1"
	DocumentChunkerVersion = "document/1"
	EmailChunkerVersion    = "email/1"

	WindowTokens        = 350
	WindowOverlapPct    = 15
	MaxDocumentChunks   = 200
	MaxEmailChunks      = 20
	maxWindowMultiplier = 2
)

type Chunk struct {
	Index int
	Page  int
	Text  string
	Body  string
	Hash  string
}

func ChunkHash(version, text string) string {
	return hashutils.SHA256Hex(version + "\n" + text)
}

func sealChunks(version string, chunks []Chunk) []Chunk {
	for idx := range chunks {
		chunks[idx].Index = idx
		chunks[idx].Hash = ChunkHash(version, chunks[idx].Text)
	}

	return chunks
}

func windowRunes() int { return WindowTokens * llmtokens.RunesPerToken }

func overlapRunes() int { return windowRunes() * WindowOverlapPct / 100 }

func Windows(text string) []string {
	return windowsOf(text, windowRunes(), overlapRunes())
}

func windowsOf(text string, size, overlap int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	costs := make([]int, len(words))
	for idx, word := range words {
		costs[idx] = utf8.RuneCountInString(word) + 1
	}

	out := make([]string, 0, len(words)*2/max(1, size/8)+1)
	start := 0
	for start < len(words) {
		end, runes := start, 0
		for end < len(words) &&
			(end == start || llmtokens.FromRunes(runes+costs[end]) <= llmtokens.FromRunes(size)) {
			runes += costs[end]
			end++
		}

		window := strings.Join(words[start:end], " ")
		out = append(out, stringutils.TruncateRunes(window, size*maxWindowMultiplier))
		if end == len(words) {
			break
		}

		next, back := end, 0
		for next > start+1 && back+costs[next-1] <= overlap {
			back += costs[next-1]
			next--
		}
		start = next
	}

	return out
}
