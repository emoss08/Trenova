package agentmemoryservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEditedContent_ChangesTheOnePlaceTheWordsAppear(t *testing.T) {
	t.Parallel()

	memory := &agentmemory.Memory{
		Kind:    agentmemory.MemoryKindInstruction,
		Content: "Copy dispatch on Acme emails. Quote Acme in dollars.",
	}

	got, err := EditedContent(memory, "in dollars", "in Canadian dollars")
	require.NoError(t, err)
	assert.Equal(t, "Copy dispatch on Acme emails. Quote Acme in Canadian dollars.", got)

	deleted, err := EditedContent(memory, " Quote Acme in dollars.", "")
	require.NoError(t, err)
	assert.Equal(t, "Copy dispatch on Acme emails.", deleted)
}

func TestEditedContent_RefusesWhatItCannotPlaceOrWouldBreak(t *testing.T) {
	t.Parallel()

	memory := &agentmemory.Memory{
		Kind:    agentmemory.MemoryKindInstruction,
		Content: "Copy dispatch on Acme emails. Quote Acme in dollars.",
	}
	cases := map[string]struct {
		old, replacement string
		says             string
	}{
		"no words":            {"", "x", "Say which words"},
		"words not there":     {"in euros", "in dollars", "not in the memory as it stands"},
		"words in two places": {"Acme", "Granite", "appear 2 times"},
		"nothing left":        {memory.Content, "", "empty; retire it"},
		"no change":           {"dollars", "dollars", "changes nothing"},
		"past the limit": {
			"dollars",
			strings.Repeat("d", agentmemory.MaxMemoryContentChars),
			"longer than",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := EditedContent(memory, tc.old, tc.replacement)
			require.Error(t, err)
			assert.Contains(t, errortypes.Summary(err), tc.says)
		})
	}
}

func TestEditedContent_LetsAProcedureRunPastTheUsualLimit(t *testing.T) {
	t.Parallel()

	procedure := &agentmemory.Memory{Kind: agentmemory.MemoryKindProcedure, Content: "1. Step."}
	long := strings.Repeat("s", agentmemory.MaxMemoryContentChars)

	got, err := EditedContent(procedure, "Step.", long)

	require.NoError(t, err)
	assert.Equal(t, "1. "+long, got)
}

func editingService(stored ...*agentmemory.Memory) *Service {
	repo := newReplacingRepo(stored...)

	return wired(&Service{l: zap.NewNop(), repo: repo, runs: &fakeRuns{}, labeler: &fakeLabeler{}})
}

func TestPlanEdit_IsAReplacementThatKeepsTheMemorysReadersAndShape(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	expires := int64(1_900_000_000)
	subject := pulid.MustNew("cus_")
	memory := &agentmemory.Memory{
		ID:           pulid.MustNew("amem_"),
		Kind:         agentmemory.MemoryKindInstruction,
		Status:       agentmemory.MemoryStatusActive,
		Scope:        agentmemory.MemoryScopeUser,
		OwnerUserID:  &owner,
		SubjectType:  agentmemory.MemorySubjectCustomer,
		SubjectID:    &subject,
		Content:      "Quote Acme in dollars.",
		ExpiresAt:    &expires,
		SubjectLabel: "Acme Foods",
	}
	svc := editingService(memory)
	runID := pulid.MustNew("arun_")

	req, err := svc.PlanEdit(t.Context(), &agentmemoryports.EditAgentMemoryRequest{
		TenantInfo:   tenant(),
		ID:           memory.ID,
		Old:          "in dollars",
		New:          "in Canadian dollars",
		RunID:        runID,
		PersonUserID: owner,
	})

	require.NoError(t, err)
	assert.Equal(t, &agentmemoryports.RememberRequest{
		TenantInfo:   req.TenantInfo,
		Kind:         agentmemory.MemoryKindInstruction,
		Content:      "Quote Acme in Canadian dollars.",
		SubjectType:  agentmemory.MemorySubjectCustomer,
		SubjectID:    subject,
		ExpiresAt:    &expires,
		RunID:        runID,
		Scope:        agentmemory.MemoryScopeUser,
		OwnerUserID:  owner,
		PersonUserID: owner,
		Replaces:     memory.ID,
	}, req)
}

func TestPlanEdit_FindsNoMemoryTheReaderCannotSee(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	someoneElses := &agentmemory.Memory{
		ID:          pulid.MustNew("amem_"),
		Kind:        agentmemory.MemoryKindFact,
		Status:      agentmemory.MemoryStatusActive,
		Scope:       agentmemory.MemoryScopeUser,
		OwnerUserID: &owner,
		Content:     "My kids are at practice on Tuesdays.",
	}
	offered := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agentmemory.MemoryKindFact,
		Status:  agentmemory.MemoryStatusSuggested,
		Scope:   agentmemory.MemoryScopeOrganization,
		Content: "Acme pays net 45.",
	}
	retired := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agentmemory.MemoryKindFact,
		Status:  agentmemory.MemoryStatusRetired,
		Scope:   agentmemory.MemoryScopeOrganization,
		Content: "Acme pays net 30.",
	}
	svc := editingService(someoneElses, offered, retired)

	for name, tc := range map[string]struct {
		id     pulid.ID
		person pulid.ID
		says   string
	}{
		"another person's":   {someoneElses.ID, pulid.MustNew("usr_"), "visible here"},
		"a run nobody is in": {someoneElses.ID, pulid.Nil, "visible here"},
		"a suggestion":       {offered.ID, owner, "visible here"},
		"a retired memory":   {retired.ID, owner, "no longer kept"},
		"no such memory":     {pulid.MustNew("amem_"), owner, "visible here"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.PlanEdit(t.Context(), &agentmemoryports.EditAgentMemoryRequest{
				TenantInfo:   tenant(),
				ID:           tc.id,
				Old:          "Acme",
				New:          "Granite",
				PersonUserID: tc.person,
			})
			require.Error(t, err)
			assert.Contains(t, errortypes.Summary(err), tc.says)
		})
	}
}

// An agent's own memories are its business: another agent cannot read one by
// guessing at its words, so the words are compared only once the memory is the
// editor's to see.
func TestPlanEdit_AnAgentsMemoryIsVisibleOnlyToThatAgent(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("agdef_")
	memory := &agentmemory.Memory{
		ID:                pulid.MustNew("amem_"),
		Kind:              agentmemory.MemoryKindProcedure,
		Status:            agentmemory.MemoryStatusActive,
		Scope:             agentmemory.MemoryScopeAgent,
		AgentDefinitionID: &owner,
		Content:           "1. Offer hazmat moves to a solo driver first.",
	}
	svc := editingService(memory)

	for name, agentID := range map[string]pulid.ID{
		"another agent": pulid.MustNew("agdef_"),
		"no agent":      pulid.Nil,
	} {
		_, err := svc.PlanEdit(t.Context(), &agentmemoryports.EditAgentMemoryRequest{
			TenantInfo:        tenant(),
			ID:                memory.ID,
			Old:               "words it never had",
			New:               "x",
			AgentDefinitionID: agentID,
		})
		require.Error(t, err, name)
		assert.Contains(t, errortypes.Summary(err), "visible here", name)
	}

	req, err := svc.PlanEdit(t.Context(), &agentmemoryports.EditAgentMemoryRequest{
		TenantInfo:        tenant(),
		ID:                memory.ID,
		Old:               "solo driver",
		New:               "solo driver with a hazmat endorsement",
		AgentDefinitionID: owner,
	})
	require.NoError(t, err)
	assert.Equal(t, "1. Offer hazmat moves to a solo driver with a hazmat endorsement first.", req.Content)
}

// Restoring a memory a newer one replaced would tell agents both. It is
// refused, naming the replacement, until that one is forgotten.
func TestSetStatus_RefusesToRestoreAMemoryANewerOneReplaced(t *testing.T) {
	t.Parallel()

	old := &agentmemory.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agentmemory.MemoryKindInstruction,
		Status:  agentmemory.MemoryStatusRetired,
		Scope:   agentmemory.MemoryScopeOrganization,
		Content: "Quote Acme in dollars.",
	}
	replacement := &agentmemory.Memory{
		ID:           pulid.MustNew("amem_"),
		Kind:         agentmemory.MemoryKindInstruction,
		Status:       agentmemory.MemoryStatusActive,
		Scope:        agentmemory.MemoryScopeOrganization,
		Content:      "Quote Acme in Canadian dollars.",
		SupersedesID: &old.ID,
	}
	repo := newReplacingRepo(old, replacement)
	svc := wired(&Service{l: zap.NewNop(), repo: repo, permissions: &fakePermissions{manages: true}})

	_, err := svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID:         old.ID,
		TenantInfo: tenant(),
		Status:     agentmemory.MemoryStatusActive,
	}, userActor())

	require.Error(t, err)
	assert.Contains(t, errortypes.Summary(err), "Quote Acme in Canadian dollars.")

	replacement.Status = agentmemory.MemoryStatusRetired
	restored, err := svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID:         old.ID,
		TenantInfo: tenant(),
		Status:     agentmemory.MemoryStatusActive,
	}, userActor())
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusActive, restored.Status)
}

func TestRewritable_APersonsOwnMemoryIsTheirsToWord(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	personal := &agentmemory.Memory{Scope: agentmemory.MemoryScopeUser, OwnerUserID: &owner}
	shared := &agentmemory.Memory{Scope: agentmemory.MemoryScopeOrganization}
	administrator := userActor()
	them := &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeUser,
		UserID:        owner,
	}

	require.NoError(t, rewritable(personal, them))
	require.NoError(t, rewritable(shared, administrator))
	err := rewritable(personal, administrator)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}
