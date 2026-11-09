package agentmemoryservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func keptFor(owner pulid.ID, kind agentmemory.MemoryKind, content string) *agentmemory.Memory {
	return &agentmemory.Memory{
		ID:          pulid.MustNew("amem_"),
		Kind:        kind,
		Status:      agentmemory.MemoryStatusActive,
		Scope:       agentmemory.MemoryScopeUser,
		OwnerUserID: &owner,
		Content:     content,
	}
}

func personal(actor *permissionports.RequestActor, kind agentmemory.MemoryKind, content string) *agentmemoryports.RememberRequest {
	return &agentmemoryports.RememberRequest{
		TenantInfo:  tenant(),
		Kind:        kind,
		Content:     content,
		Scope:       agentmemory.MemoryScopeUser,
		OwnerUserID: actor.UserID,
	}
}

func consolidating(neighbors []*agentmemory.Memory, judge *fakeJudge) (*Service, *fakeMemoryRepo) {
	repo := &fakeMemoryRepo{neighbors: neighbors}
	svc := newService(repo, &fakeRuns{}, &fakeLabeler{})
	svc.judge = judge

	return svc, repo
}

// Five near-identical memories said capacity postings refuse "TX". A save
// whose words are nearly all a kept memory's refreshes that memory, with no
// model call, rather than keeping a sixth.
func TestRemember_AParaphraseOfAKeptMemoryRefreshesIt(t *testing.T) {
	t.Parallel()

	actor := userActor()
	kept := keptFor(actor.UserID, agentmemory.MemoryKindInstruction,
		"Quote Acme Foods in dollars, never in euros.")
	judge := &fakeJudge{}
	svc, repo := consolidating([]*agentmemory.Memory{kept}, judge)

	got, err := svc.Remember(t.Context(), personal(actor, agentmemory.MemoryKindInstruction,
		"Quote Acme Foods in dollars and never in euros."), actor)

	require.NoError(t, err)
	assert.Equal(t, kept.ID, got.ID)
	assert.True(t, got.Refreshed)
	assert.Empty(t, repo.created)
	assert.Equal(t, []pulid.ID{kept.ID}, repo.used)
	assert.Empty(t, judge.asked, "an obvious repeat needs no model")
}

func TestRemember_AskTheJudgeOnlyAboutTheNeighborsOfTheSameReaders(t *testing.T) {
	t.Parallel()

	actor := userActor()
	neighbor := keptFor(actor.UserID, agentmemory.MemoryKindFact, "Acme's dock closes at 3pm.")
	judge := &fakeJudge{}
	svc, repo := consolidating([]*agentmemory.Memory{neighbor}, judge)

	_, err := svc.Remember(t.Context(), personal(actor, agentmemory.MemoryKindFact,
		"Acme's receiving window ends at 15:00 on Fridays."), actor)

	require.NoError(t, err)
	require.Len(t, repo.neighborAsks, 1)
	ask := repo.neighborAsks[0]
	assert.Equal(t, agentmemory.MemoryScopeUser, ask.Same.Scope)
	assert.Equal(t, actor.UserID, ask.Same.OwnerUserID)
	assert.Contains(t, ask.Terms, "receiving")
	require.Len(t, judge.asked, 1)
	assert.Equal(t, []*agentmemory.Memory{neighbor}, judge.asked[0].Candidates)
	assert.Equal(t, aiusage.FeatureAgentMemoryConsolidation, judge.asked[0].Attribution.Feature)
	require.Len(t, repo.created, 1)
	assert.Nil(t, repo.created[0].SupersedesID, "unrelated memories are both kept")
}

func TestRemember_WhatTheJudgeCallsTheSameRefreshesTheKeptMemory(t *testing.T) {
	t.Parallel()

	actor := userActor()
	kept := keptFor(actor.UserID, agentmemory.MemoryKindFact, "Acme's dock closes at 3pm.")
	svc, repo := consolidating([]*agentmemory.Memory{kept}, &fakeJudge{
		verdicts: map[pulid.ID]agentmemoryports.MemoryVerdict{
			kept.ID: agentmemoryports.MemoryVerdictSame,
		},
	})

	got, err := svc.Remember(t.Context(), personal(actor, agentmemory.MemoryKindFact,
		"Acme stops receiving at 15:00."), actor)

	require.NoError(t, err)
	assert.Equal(t, kept.ID, got.ID)
	assert.True(t, got.Refreshed)
	assert.Empty(t, repo.created)
}

// Saying as an Instruction what was kept as a Fact is the same thing, made a
// rule: the Instruction replaces the Fact so agents follow it, not weigh it.
func TestRemember_TheSameThingAsAStrongerKindReplacesTheWeakerOne(t *testing.T) {
	t.Parallel()

	actor := userActor()
	kept := keptFor(actor.UserID, agentmemory.MemoryKindFact, "Acme prefers morning appointments.")
	svc, repo := consolidating([]*agentmemory.Memory{kept}, &fakeJudge{
		verdicts: map[pulid.ID]agentmemoryports.MemoryVerdict{
			kept.ID: agentmemoryports.MemoryVerdictSame,
		},
	})

	got, err := svc.Remember(t.Context(), personal(actor, agentmemory.MemoryKindInstruction,
		"Book Acme appointments in the morning."), actor)

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, agentmemory.MemoryStatusActive, got.Status)
	require.NotNil(t, got.SupersedesID)
	assert.Equal(t, kept.ID, *got.SupersedesID)
}

func TestRemember_AContradictionReplacesAMemoryTheWriterMayChange(t *testing.T) {
	t.Parallel()

	actor := userActor()
	kept := keptFor(actor.UserID, agentmemory.MemoryKindInstruction, "Quote Acme in dollars.")
	svc, repo := consolidating([]*agentmemory.Memory{kept}, &fakeJudge{
		verdicts: map[pulid.ID]agentmemoryports.MemoryVerdict{
			kept.ID: agentmemoryports.MemoryVerdictContradicts,
		},
	})

	got, err := svc.Remember(t.Context(), personal(actor, agentmemory.MemoryKindInstruction,
		"Quote Acme in Canadian dollars from now on."), actor)

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, agentmemory.MemoryStatusActive, got.Status)
	require.NotNil(t, got.SupersedesID)
	assert.Equal(t, kept.ID, *got.SupersedesID, "agents are never told both")
}

func TestRemember_AContradictionOfASharedMemoryWaitsForAnAdministrator(t *testing.T) {
	t.Parallel()

	kept := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agentmemory.MemoryKindInstruction,
		Status:  agentmemory.MemoryStatusActive,
		Scope:   agentmemory.MemoryScopeOrganization,
		Content: "Copy dispatch on every customer email.",
	}
	svc, repo := consolidating([]*agentmemory.Memory{kept}, &fakeJudge{
		verdicts: map[pulid.ID]agentmemoryports.MemoryVerdict{
			kept.ID: agentmemoryports.MemoryVerdictContradicts,
		},
	})

	got, err := svc.Remember(t.Context(), &agentmemoryports.RememberRequest{
		TenantInfo: tenant(),
		Kind:       agentmemory.MemoryKindInstruction,
		Content:    "Never copy dispatch on customer emails.",
		Scope:      agentmemory.MemoryScopeOrganization,
	}, userActor())

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, agentmemory.MemoryStatusSuggested, got.Status)
	require.NotNil(t, got.SupersedesID)
	assert.Equal(t, kept.ID, *got.SupersedesID, "approving the offer retires what it contradicts")
}

func TestRemember_WorkThatReadOutsideContentNeverReplacesOnItsOwn(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	kept := keptFor(person, agentmemory.MemoryKindFact, "Acme pays net 30.")
	svc, repo := consolidating([]*agentmemory.Memory{kept}, &fakeJudge{
		verdicts: map[pulid.ID]agentmemoryports.MemoryVerdict{
			kept.ID: agentmemoryports.MemoryVerdictContradicts,
		},
	})

	runID := pulid.MustNew("arun_")
	got, err := svc.Remember(t.Context(), &agentmemoryports.RememberRequest{
		TenantInfo:   tenant(),
		Kind:         agentmemory.MemoryKindFact,
		Content:      "Acme now pays net 60, per the document they sent.",
		Scope:        agentmemory.MemoryScopeUser,
		OwnerUserID:  person,
		PersonUserID: person,
		RunID:        runID,
		Taint:        &agent.RunTaint{Seen: []agent.TaintSource{agent.TaintSourceDocument}},
	}, agentActor())

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, agentmemory.MemoryStatusSuggested, got.Status)
	assert.Equal(t, agentmemory.MemorySourceAgent, got.Source)
	assert.True(t, got.Tainted)
	require.NotNil(t, got.SupersedesID)
	assert.Equal(t, kept.ID, *got.SupersedesID, "a person decides whether it replaces theirs")
}

func TestRemember_AJudgeThatCannotAnswerLeavesTheSaveAsItWas(t *testing.T) {
	t.Parallel()

	actor := userActor()
	kept := keptFor(actor.UserID, agentmemory.MemoryKindInstruction, "Quote Acme in dollars.")
	svc, repo := consolidating([]*agentmemory.Memory{kept}, &fakeJudge{
		err: errors.New("every provider is resting"),
	})

	got, err := svc.Remember(t.Context(), personal(actor, agentmemory.MemoryKindInstruction,
		"Quote Acme in Canadian dollars."), actor)

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, agentmemory.MemoryStatusActive, got.Status)
	assert.Nil(t, got.SupersedesID)
}

func TestPreviewRemember_NeverSpendsAModelCall(t *testing.T) {
	t.Parallel()

	actor := userActor()
	kept := keptFor(actor.UserID, agentmemory.MemoryKindInstruction, "Quote Acme in dollars.")
	judge := &fakeJudge{}
	svc, _ := consolidating([]*agentmemory.Memory{kept}, judge)

	plan, err := svc.PreviewRemember(t.Context(), personal(actor, agentmemory.MemoryKindInstruction,
		"Quote Acme in Canadian dollars."), actor)

	require.NoError(t, err)
	assert.Nil(t, plan.Existing)
	assert.Empty(t, judge.asked)
}

type racingRepo struct {
	fakeMemoryRepo
	winner *agentmemory.Memory
	tried  bool
}

func (r *racingRepo) Create(context.Context, *agentmemory.Memory) (*agentmemory.Memory, error) {
	r.tried = true

	return nil, agentmemoryports.ErrMemoryAlreadyKept
}

func (r *racingRepo) FindActive(
	context.Context,
	*agentmemoryports.FindActiveAgentMemoryRequest,
) (*agentmemory.Memory, error) {
	if !r.tried {
		return nil, nil //nolint:nilnil // nothing kept yet, as the repository answers
	}

	return r.winner, nil
}

func TestRemember_TwoIdenticalSavesAtOnceKeepOneMemory(t *testing.T) {
	t.Parallel()

	actor := userActor()
	winner := keptFor(actor.UserID, agentmemory.MemoryKindInstruction, "Quote Acme in dollars.")
	repo := &racingRepo{winner: winner}
	svc := wired(&Service{l: zap.NewNop(), repo: repo, runs: &fakeRuns{}, labeler: &fakeLabeler{}})

	got, err := svc.Remember(t.Context(), personal(actor, agentmemory.MemoryKindInstruction,
		"Quote Acme in dollars."), actor)

	require.NoError(t, err)
	assert.Equal(t, winner.ID, got.ID)
	assert.True(t, got.Refreshed)
}

func TestDuplicateOf_NeedsTheSameKindAndNearlyAllTheWords(t *testing.T) {
	t.Parallel()

	fact := &agentmemory.Memory{Kind: agentmemory.MemoryKindFact, Content: "Acme pays net 45 days"}
	instruction := &agentmemory.Memory{
		Kind:    agentmemory.MemoryKindInstruction,
		Content: "Acme pays net 45 days",
	}
	other := &agentmemory.Memory{Kind: agentmemory.MemoryKindFact, Content: "Granite pays by ACH"}

	assert.Same(t, fact, DuplicateOf(
		&agentmemory.Memory{Kind: agentmemory.MemoryKindFact, Content: "Acme pays net 45 days."},
		[]*agentmemory.Memory{other, instruction, fact},
	))
	assert.Nil(t, DuplicateOf(
		&agentmemory.Memory{Kind: agentmemory.MemoryKindFact, Content: "Acme pays net 30 days, by check"},
		[]*agentmemory.Memory{fact},
	))
	assert.Nil(t, DuplicateOf(&agentmemory.Memory{Kind: agentmemory.MemoryKindFact}, []*agentmemory.Memory{fact}))
}

// "Quote Acme in dollars" and "Quote Acme in Canadian dollars" share four of
// five words, enough for the overlap alone, and say opposite things. A word
// that carries meaning is never waved through as a repeat; the judge decides.
func TestDuplicateOf_OneMeaningfulWordMakesItANewMemory(t *testing.T) {
	t.Parallel()

	kept := &agentmemory.Memory{Kind: agentmemory.MemoryKindInstruction, Content: "Quote Acme in dollars."}

	assert.Nil(t, DuplicateOf(
		&agentmemory.Memory{Kind: agentmemory.MemoryKindInstruction, Content: "Quote Acme in Canadian dollars."},
		[]*agentmemory.Memory{kept},
	))
	assert.Same(t, kept, DuplicateOf(
		&agentmemory.Memory{Kind: agentmemory.MemoryKindInstruction, Content: "Quote Acme in the dollars."},
		[]*agentmemory.Memory{kept},
	))
}
