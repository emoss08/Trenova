package agentmemoryservice

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeIndexer struct {
	err    error
	marked []pulid.ID
	types  []airetrieval.SourceType
}

func (f *fakeIndexer) MarkStale(
	_ context.Context,
	_ pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	ids ...pulid.ID,
) error {
	f.marked = append(f.marked, ids...)
	f.types = append(f.types, sourceType)

	return f.err
}

func (f *fakeIndexer) DeleteSource(
	context.Context,
	pagination.TenantInfo,
	airetrieval.SourceType,
	pulid.ID,
) error {
	return nil
}

func (f *fakeIndexer) Reindex(
	context.Context,
	pagination.TenantInfo,
	airetrieval.SourceType,
) error {
	return nil
}

func TestRememberQueuesTheNewMemoryForIndexing(t *testing.T) {
	t.Parallel()

	indexer := &fakeIndexer{}
	svc := newService(&fakeMemoryRepo{}, &fakeRuns{}, &fakeLabeler{})
	svc.indexer = indexer

	memory, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo: tenant(),
		Kind:       agent.MemoryKindFact,
		Content:    "Acme closes at 3 PM on Fridays.",
	}, userActor())
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{memory.ID}, indexer.marked)
	assert.Equal(t, []airetrieval.SourceType{airetrieval.SourceTypeMemory}, indexer.types)
}

func TestRememberDoesNotQueueAMemoryItAlreadyHad(t *testing.T) {
	t.Parallel()

	indexer := &fakeIndexer{}
	existing := &agent.Memory{ID: pulid.MustNew("amem_"), Kind: agent.MemoryKindFact}
	svc := newService(&fakeMemoryRepo{found: existing}, &fakeRuns{}, &fakeLabeler{})
	svc.indexer = indexer

	_, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo: tenant(),
		Kind:       agent.MemoryKindFact,
		Content:    "Acme closes at 3 PM on Fridays.",
	}, userActor())
	require.NoError(t, err)
	assert.Empty(t, indexer.marked)
}

func TestAnIndexingFailureNeverFailsTheWrite(t *testing.T) {
	t.Parallel()

	indexer := &fakeIndexer{err: errors.New("outbox unavailable")}
	svc := newService(&fakeMemoryRepo{}, &fakeRuns{}, &fakeLabeler{})
	svc.indexer = indexer

	memory, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo: tenant(),
		Kind:       agent.MemoryKindInstruction,
		Content:    "Always book Globex with an appointment.",
	}, userActor())
	require.NoError(t, err)
	assert.NotNil(t, memory)
	assert.Len(t, indexer.marked, 1)
}

func TestRecordCorrectionQueuesTheCorrection(t *testing.T) {
	t.Parallel()

	indexer := &fakeIndexer{}
	svc := newService(&fakeMemoryRepo{}, &fakeRuns{}, &fakeLabeler{})
	svc.indexer = indexer

	memory, err := svc.RecordCorrection(
		t.Context(),
		proposal("assign_move", map[string]any{"workerId": "wrk_a"}),
		&agent.AgentDecision{
			Decision:        agent.DecisionModified,
			Modifications:   map[string]any{"workerId": "wrk_b"},
			DecidedByUserID: pulid.MustNew("usr_"),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, memory)
	assert.Equal(t, []pulid.ID{memory.ID}, indexer.marked)
}

type searchingMemoryRepo struct {
	fakeMemoryRepo

	byWords  []*agent.Memory
	eligible []*agent.Memory
	searches []repositories.SearchAgentMemoriesRequest
}

func (f *searchingMemoryRepo) Search(
	_ context.Context,
	req repositories.SearchAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	f.searches = append(f.searches, req)
	if req.Query != "" {
		return f.byWords, nil
	}

	return slices.DeleteFunc(slices.Clone(f.eligible), func(memory *agent.Memory) bool {
		return !slices.Contains(req.IDs, memory.ID)
	}), nil
}

type fakeVectors struct {
	similar services.SimilarMemories
	err     error
	asked   []services.SimilarMemoriesRequest
}

func (f *fakeVectors) SimilarMemories(
	_ context.Context,
	req *services.SimilarMemoriesRequest,
) (services.SimilarMemories, error) {
	f.asked = append(f.asked, *req)

	return f.similar, f.err
}

func memoryNamed(content string) *agent.Memory {
	return &agent.Memory{ID: pulid.MustNew("amem_"), Kind: agent.MemoryKindFact, Content: content}
}

func TestRecallFusesWordsAndMeaning(t *testing.T) {
	t.Parallel()

	both := memoryNamed("Globex allows two hours of free time before detention.")
	words := memoryNamed("Detention at Globex needs a manager's approval to waive.")
	meaning := memoryNamed("The customer's dock gives trucks a grace period.")
	weak := memoryNamed("Acme closes early on Fridays.")
	hidden := memoryNamed("Kept for another agent.")

	repo := &searchingMemoryRepo{
		byWords:  []*agent.Memory{both, words},
		eligible: []*agent.Memory{both, words, meaning, weak},
	}
	vectors := &fakeVectors{similar: services.SimilarMemories{
		Memories: []services.MemorySimilarity{
			{MemoryID: meaning.ID, Similarity: 0.82},
			{MemoryID: both.ID, Similarity: 0.8},
			{MemoryID: hidden.ID, Similarity: 0.79},
			{MemoryID: weak.ID, Similarity: 0.2},
		},
		Semantics: services.RetrievalSemantics{Used: true},
		Floor:     0.5,
	}}
	svc := newService(&repo.fakeMemoryRepo, &fakeRuns{}, &fakeLabeler{})
	svc.repo = repo
	svc.vectors = vectors

	recalled, err := svc.Recall(t.Context(), services.RecallAgentMemoriesRequest{
		TenantInfo:  tenant(),
		Query:       "free time before detention",
		Limit:       10,
		Attribution: services.AIUsageAttribution{UserID: pulid.MustNew("usr_")},
	})
	require.NoError(t, err)
	require.Len(t, recalled, 3, "a memory the filters refuse or under the floor is left out")

	assert.Equal(t, both.ID, recalled[0].Memory.ID)
	assert.Equal(t, agent.MemoryMatchBoth, recalled[0].Match)

	matches := make(map[pulid.ID]agent.MemoryMatch, len(recalled))
	for _, memory := range recalled {
		matches[memory.Memory.ID] = memory.Match
	}
	assert.Equal(t, agent.MemoryMatchWords, matches[words.ID])
	assert.Equal(t, agent.MemoryMatchMeaning, matches[meaning.ID])

	require.Len(t, repo.searches, 2)
	assert.ElementsMatch(t, []pulid.ID{meaning.ID, hidden.ID}, repo.searches[1].IDs,
		"hits found only by meaning are read back through the same filters")
	assert.Empty(t, repo.searches[1].Query)
	require.Len(t, vectors.asked, 1)
	assert.Equal(t, "free time before detention", vectors.asked[0].Text)
	assert.Equal(t, 40, vectors.asked[0].Limit)
}

func TestRecallStaysOnWordsWithoutMeaning(t *testing.T) {
	t.Parallel()

	found := memoryNamed("Globex allows two hours of free time.")
	repo := &searchingMemoryRepo{byWords: []*agent.Memory{found}}

	cases := map[string]*fakeVectors{
		"meaning unavailable": {similar: services.SimilarMemories{
			Semantics: services.RetrievalSemantics{Reason: airetrieval.UnavailableReasonNoProvider},
		}},
		"meaning failed": {err: errors.New("boom")},
	}
	for name, vectors := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svc := newService(&repo.fakeMemoryRepo, &fakeRuns{}, &fakeLabeler{})
			svc.repo = &searchingMemoryRepo{byWords: repo.byWords}
			svc.vectors = vectors

			recalled, err := svc.Recall(t.Context(), services.RecallAgentMemoriesRequest{
				TenantInfo: tenant(),
				Query:      "free time",
			})
			require.NoError(t, err)
			require.Len(t, recalled, 1)
			assert.Equal(t, agent.MemoryMatchWords, recalled[0].Match)
		})
	}

	svc := newService(&repo.fakeMemoryRepo, &fakeRuns{}, &fakeLabeler{})
	svc.repo = &searchingMemoryRepo{eligible: []*agent.Memory{found}}
	vectors := &fakeVectors{}
	svc.vectors = vectors
	listed, err := svc.Recall(t.Context(), services.RecallAgentMemoriesRequest{
		TenantInfo: tenant(),
		IDs:        []pulid.ID{found.ID},
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Empty(t, listed[0].Match, "a recall that searched for nothing matched nothing")
	assert.Empty(t, vectors.asked)
}

func TestFuseRecallHonoursTheLimit(t *testing.T) {
	t.Parallel()

	first, second, third := memoryNamed("a"), memoryNamed("b"), memoryNamed("c")
	byID := map[pulid.ID]*agent.Memory{first.ID: first, second.ID: second, third.ID: third}

	recalled := FuseRecall(byID,
		[]pulid.ID{first.ID, second.ID},
		[]pulid.ID{third.ID, second.ID},
		2,
	)
	require.Len(t, recalled, 2)
	assert.Equal(t, second.ID, recalled[0].Memory.ID)
	assert.Equal(t, agent.MemoryMatchBoth, recalled[0].Match)
}
