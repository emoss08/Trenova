package agentmemoryservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentsearch"
	"github.com/emoss08/trenova/shared/pulid"
)

const minRelevantTermLen = 3

func Relevance(text string, ranked services.RankedMemories) agent.MemoryRelevance {
	terms := queryTerms(text)
	if !ranked.Semantic && len(terms) == 0 {
		return agent.MemoryRelevance{}
	}

	ids := make(map[pulid.ID]struct{}, len(ranked.Similar)+len(ranked.Memories)/4)
	for _, id := range ranked.Similar {
		ids[id] = struct{}{}
	}
	if len(terms) > 0 {
		for _, memory := range ranked.Memories {
			if memory == nil {
				continue
			}
			if _, ok := ids[memory.ID]; ok {
				continue
			}
			if sharesTerm(terms, memory) {
				ids[memory.ID] = struct{}{}
			}
		}
	}

	return agent.MemoryRelevance{Judged: true, IDs: ids}
}

func queryTerms(text string) map[string]struct{} {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	terms := agentsearch.Terms(text)
	for term := range terms {
		if len(term) < minRelevantTermLen {
			delete(terms, term)
		}
	}

	return terms
}

func sharesTerm(terms map[string]struct{}, memory *agent.Memory) bool {
	for _, text := range [...]string{
		memory.Content,
		memory.SubjectLabel,
		strings.ReplaceAll(memory.ToolName, "_", " "),
	} {
		if text == "" {
			continue
		}
		for token := range agentsearch.TokenSet(text) {
			if _, ok := terms[token]; ok {
				return true
			}
		}
	}

	return false
}
