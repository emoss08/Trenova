package agentmemoryservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func relevanceMemory(content string) *agent.Memory {
	return &agent.Memory{ID: pulid.MustNew("amem_"), Content: content}
}

func TestRelevance_NothingToJudgeByLeavesEveryMemoryBearing(t *testing.T) {
	t.Parallel()

	memory := relevanceMemory("Assess ELD applicability for each driver.")

	relevance := Relevance("", services.RankedMemories{Memories: []*agent.Memory{memory}})

	assert.False(t, relevance.Judged)
	assert.True(t, relevance.Bears(memory.ID))
}

func TestRelevance_SharedWordsBear(t *testing.T) {
	t.Parallel()

	eld := relevanceMemory("Assess ELD applicability for each driver and operation.")
	report := relevanceMemory("When I run a report, include the actual report results.")
	notify := relevanceMemory("Copy dispatch when you tell a customer a load delivered.")

	relevance := Relevance(
		"Let the customer know shipment SEED-PAY-001 has delivered.",
		services.RankedMemories{Memories: []*agent.Memory{eld, report, notify}},
	)

	assert.True(t, relevance.Judged)
	assert.False(t, relevance.Bears(eld.ID))
	assert.False(t, relevance.Bears(report.ID))
	assert.True(t, relevance.Bears(notify.ID))
}

func TestRelevance_TheOperatorsWordReachesTheSchemasWord(t *testing.T) {
	t.Parallel()

	memory := relevanceMemory("A worker on probation never runs hazmat.")

	relevance := Relevance(
		"Which drivers can take this load?",
		services.RankedMemories{Memories: []*agent.Memory{memory}},
	)

	assert.True(t, relevance.Bears(memory.ID))
}

func TestRelevance_AMemoryNearInMeaningBearsWithoutSharedWords(t *testing.T) {
	t.Parallel()

	near := relevanceMemory("Net 45 for Acme.")
	far := relevanceMemory("The yard closes at six.")

	relevance := Relevance("When is the invoice due?", services.RankedMemories{
		Memories: []*agent.Memory{near, far},
		Similar:  []pulid.ID{near.ID},
		Semantic: true,
	})

	assert.True(t, relevance.Judged)
	assert.True(t, relevance.Bears(near.ID))
	assert.False(t, relevance.Bears(far.ID))
}

func TestRelevance_ASubjectOrToolNameIsReadToo(t *testing.T) {
	t.Parallel()

	about := &agent.Memory{
		ID: pulid.MustNew("amem_"), Content: "Pays on the 15th.", SubjectLabel: "Acme Foods",
	}
	tool := &agent.Memory{
		ID: pulid.MustNew("amem_"), Content: "Prefer the closest tractor.", ToolName: "assign_move",
	}

	relevance := Relevance("Assign a move for Acme", services.RankedMemories{
		Memories: []*agent.Memory{about, tool},
	})

	assert.True(t, relevance.Bears(about.ID))
	assert.True(t, relevance.Bears(tool.ID))
}
