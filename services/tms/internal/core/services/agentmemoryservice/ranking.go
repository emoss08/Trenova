package agentmemoryservice

import (
	"cmp"
	"context"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	recencyHalfLifeSeconds = 30 * 24 * 60 * 60
	recencyWeight          = 0.6
	useWeight              = 0.4
)

type RecencyRanker struct{}

func NewRecencyRanker() services.MemoryRanker { return RecencyRanker{} }

func (RecencyRanker) RankMemories(
	_ context.Context,
	req services.RankMemoriesRequest,
) ([]*agent.Memory, error) {
	return RankByRecencyAndUse(req.Memories, req.Now), nil
}

type scoredMemory struct {
	memory *agent.Memory
	score  float64
}

func RankByRecencyAndUse(memories []*agent.Memory, now int64) []*agent.Memory {
	mostUsed := 0
	for _, memory := range memories {
		if memory != nil {
			mostUsed = max(mostUsed, memory.UseCount)
		}
	}

	scored := make([]scoredMemory, 0, len(memories))
	for _, memory := range memories {
		if memory == nil {
			continue
		}
		scored = append(scored, scoredMemory{
			memory: memory,
			score: recencyWeight*recencyScore(memory, now) +
				useWeight*useScore(memory.UseCount, mostUsed),
		})
	}

	slices.SortStableFunc(scored, func(a, b scoredMemory) int {
		if byScore := cmp.Compare(b.score, a.score); byScore != 0 {
			return byScore
		}
		if byAge := cmp.Compare(b.memory.CreatedAt, a.memory.CreatedAt); byAge != 0 {
			return byAge
		}

		return cmp.Compare(a.memory.ID.String(), b.memory.ID.String())
	})

	ranked := make([]*agent.Memory, 0, len(scored))
	for _, entry := range scored {
		ranked = append(ranked, entry.memory)
	}

	return ranked
}

func recencyScore(memory *agent.Memory, now int64) float64 {
	touched := max(memory.CreatedAt, memory.UpdatedAt)
	age := max(now-touched, 0)

	return 1 / (1 + float64(age)/recencyHalfLifeSeconds)
}

func useScore(uses, mostUsed int) float64 {
	if uses <= 0 || mostUsed <= 0 {
		return 0
	}

	return math.Log1p(float64(uses)) / math.Log1p(float64(mostUsed))
}
