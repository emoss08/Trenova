package agentmemoryrank

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func relevanceMemory(content string) *agentmemory.Memory {
	return &agentmemory.Memory{ID: pulid.MustNew("amem_"), Content: content}
}

func TestRelevance_NothingToJudgeByLeavesEveryMemoryBearing(t *testing.T) {
	t.Parallel()

	memory := relevanceMemory("Assess ELD applicability for each driver.")

	relevance := Relevance(
		"",
		"",
		agentmemoryports.RankedMemories{Memories: []*agentmemory.Memory{memory}},
	)

	assert.False(t, relevance.Judged)
	assert.True(t, relevance.Bears(memory.ID))
}

func TestRelevance_SharedWordsBear(t *testing.T) {
	t.Parallel()

	eld := relevanceMemory("Assess ELD applicability for each driver and operation.")
	report := relevanceMemory("When I run a report, include the actual report results.")
	notify := relevanceMemory("Copy dispatch when you tell a customer a load delivered.")

	relevance := Relevance(
		"",
		"Let the customer know shipment SEED-PAY-001 has delivered.",
		agentmemoryports.RankedMemories{Memories: []*agentmemory.Memory{eld, report, notify}},
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
		"",
		"Which drivers can take this load?",
		agentmemoryports.RankedMemories{Memories: []*agentmemory.Memory{memory}},
	)

	assert.True(t, relevance.Bears(memory.ID))
}

func TestRelevance_AMemoryNearInMeaningBearsWithoutSharedWords(t *testing.T) {
	t.Parallel()

	near := relevanceMemory("Net 45 for Acme.")
	far := relevanceMemory("The yard closes at six.")

	relevance := Relevance("", "When is the invoice due?", agentmemoryports.RankedMemories{
		Memories: []*agentmemory.Memory{near, far},
		Similar:  []pulid.ID{near.ID},
		Semantic: true,
	})

	assert.True(t, relevance.Judged)
	assert.True(t, relevance.Bears(near.ID))
	assert.False(t, relevance.Bears(far.ID))
}

func TestRelevance_ASubjectOrToolNameIsReadToo(t *testing.T) {
	t.Parallel()

	about := &agentmemory.Memory{
		ID: pulid.MustNew("amem_"), Content: "Pays on the 15th.", SubjectLabel: "Acme Foods",
	}
	tool := &agentmemory.Memory{
		ID: pulid.MustNew(
			"amem_",
		), Content: "Prefer the closest tractor.", ToolName: "dispatch_assign_move",
	}

	relevance := Relevance("", "Assign a move for Acme", agentmemoryports.RankedMemories{
		Memories: []*agentmemory.Memory{about, tool},
	})

	assert.True(t, relevance.Bears(about.ID))
	assert.True(t, relevance.Bears(tool.ID))
}

func TestRelevance_ACommonWordAloneDoesNotBear(t *testing.T) {
	t.Parallel()

	pool := make([]*agentmemory.Memory, 0, 12)
	for range 10 {
		pool = append(
			pool,
			relevanceMemory("Every shipment for a new customer needs a credit check."),
		)
	}
	hazmat := relevanceMemory("A shipment with hazmat goes to a driver with the endorsement.")
	billing := relevanceMemory("Bill the shipment when the POD is in.")
	pool = append(pool, hazmat, billing)

	relevance := Relevance(
		"Where is shipment 4417 and which hazmat driver has it?",
		"",
		agentmemoryports.RankedMemories{Memories: pool},
	)

	assert.True(t, relevance.Bears(hazmat.ID), "a rare word bears on its own")
	assert.False(t, relevance.Bears(billing.ID),
		"a word every memory shares says what kind of work it is, not which memory")
	assert.False(t, relevance.Bears(pool[0].ID))
}

func TestRelevance_TwoSharedWordsBearEvenWhenBothAreCommon(t *testing.T) {
	t.Parallel()

	pool := make([]*agentmemory.Memory, 0, 21)
	for range 20 {
		pool = append(pool, relevanceMemory("Each customer shipment is checked before dispatch."))
	}
	both := relevanceMemory("Customer shipment notices go out by email.")
	pool = append(pool, both)

	relevance := Relevance("Tell the customer the shipment arrived", "",
		agentmemoryports.RankedMemories{Memories: pool})

	assert.True(t, relevance.Bears(both.ID))
}

func TestRelevance_TheTurnsOwnMessageIsReadBeforeTheOneBeforeIt(t *testing.T) {
	t.Parallel()

	earlier := relevanceMemory("Acme invoices go to the Dallas office.")
	current := relevanceMemory("Tractors due for inspection go to the Fort Worth shop.")

	relevance := Relevance(
		"Which tractors are due for inspection?",
		"Which tractors are due for inspection?\nSend the Acme invoices out.",
		agentmemoryports.RankedMemories{Memories: []*agentmemory.Memory{earlier, current}},
	)

	assert.True(t, relevance.Bears(current.ID))
	assert.False(t, relevance.Bears(earlier.ID),
		"the earlier topic does not leak into a message that says what it is about")
}

func TestRelevance_AShortReplyIsReadWithTheMessageBeforeIt(t *testing.T) {
	t.Parallel()

	acme := relevanceMemory("Acme invoices go to the Dallas office.")
	other := relevanceMemory("Tractors due for inspection go to the Fort Worth shop.")

	relevance := Relevance(
		"yes, go ahead and send it",
		"yes, go ahead and send it\nSend the Acme invoices out.",
		agentmemoryports.RankedMemories{Memories: []*agentmemory.Memory{acme, other}},
	)

	assert.True(t, relevance.Bears(acme.ID))
	assert.False(t, relevance.Bears(other.ID))
}

func TestRelevance_ConversationalWordsAreNotATopic(t *testing.T) {
	t.Parallel()

	memory := relevanceMemory("Now that the yard is full, send drivers to the overflow lot.")

	relevance := Relevance("ok thanks, now what?", "", agentmemoryports.RankedMemories{
		Memories: []*agentmemory.Memory{memory},
		Semantic: true,
	})

	assert.True(t, relevance.Judged)
	assert.False(t, relevance.Bears(memory.ID))
}

func TestRecallTerms_AreTheWordsAsTyped(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"detention", "policies", "acme"},
		RecallTerms("the detention policies for acme"),
		"a prefix search finds \"policies\" by \"policies\", not by \"policy\"")
}

func TestRelevance_AWordMostOfAFewMemoriesShareIsNotRare(t *testing.T) {
	t.Parallel()

	first := relevanceMemory("Every shipment for Acme is checked twice.")
	second := relevanceMemory("A shipment to Dallas needs a lumper receipt.")
	third := relevanceMemory("Pay drivers on Friday.")

	relevance := Relevance("Where is my shipment to Dallas?", "", agentmemoryports.RankedMemories{
		Memories: []*agentmemory.Memory{first, second, third},
	})

	assert.True(t, relevance.Judged)
	assert.False(t, relevance.Bears(first.ID), "two of three say shipment; the word picks neither")
	assert.True(t, relevance.Bears(second.ID), "Dallas does")
	assert.False(t, relevance.Bears(third.ID))
}
