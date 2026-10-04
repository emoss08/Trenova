package agent_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validMemory(content string) *agent.Memory {
	return &agent.Memory{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Kind:           agent.MemoryKindInstruction,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Content:        content,
	}
}

func TestMemoryValidate_ContentLimitComesFromTheConstant(t *testing.T) {
	t.Parallel()

	atLimit := validMemory(strings.Repeat("a", agent.MaxMemoryContentChars))
	me := errortypes.NewMultiError()
	atLimit.Validate(me)
	assert.False(t, me.HasErrors(), "a memory exactly at the limit is accepted")

	over := validMemory(strings.Repeat("a", agent.MaxMemoryContentChars+1))
	me = errortypes.NewMultiError()
	over.Validate(me)
	require.True(t, me.HasErrors())

	var content *errortypes.Error
	for _, err := range me.Errors {
		if err.Field == "content" {
			content = err
		}
	}
	require.NotNil(t, content)
	assert.Equal(t, "Content must be at most {0} characters", content.Message,
		"the catalog key carries the limit as an argument, never as a literal")
	assert.Equal(t, []any{agent.MaxMemoryContentChars}, content.Args)
	assert.Equal(t,
		"Content must be at most "+strconv.Itoa(agent.MaxMemoryContentChars)+" characters",
		content.Error(),
	)
}

func TestMemoryLimits_AreTheAgreedDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4000, agent.MaxMemoryContentChars)
	assert.Equal(t, 500, agent.MaxMemoryCandidates)
	assert.Equal(t, 5000, agent.MemoryActiveSoftCap)
	assert.Equal(t, 4000, agent.MemoryActiveWarnAt, "the warning shows at 80% of the cap")
	assert.Equal(t, 10, agent.DefaultMemoryRecallLimit)
	assert.Equal(t, 50, agent.MaxMemoryRecallLimit)
	assert.Less(t, agent.MemoryPromptExcerptChars, agent.MaxMemoryContentChars)
}

func TestMemoryReader_ReadsTheirOwnAndTheirRolesOnly(t *testing.T) {
	t.Parallel()

	avery, jordan := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	billing, dispatch := pulid.MustNew("rol_"), pulid.MustNew("rol_")
	reader := agent.MemoryReader{UserID: avery, RoleIDs: []pulid.ID{billing}}
	kept := func(scope agent.MemoryScope, owner, role pulid.ID) *agent.Memory {
		memory := validMemory("x")
		memory.SetAudience(scope, owner, role)

		return memory
	}

	assert.True(t, reader.Reads(kept(agent.MemoryScopeOrganization, pulid.Nil, pulid.Nil)))
	assert.True(t, reader.Reads(kept(agent.MemoryScopeUser, avery, pulid.Nil)))
	assert.False(t, reader.Reads(kept(agent.MemoryScopeUser, jordan, pulid.Nil)))
	assert.True(t, reader.Reads(kept(agent.MemoryScopeRole, pulid.Nil, billing)))
	assert.False(t, reader.Reads(kept(agent.MemoryScopeRole, pulid.Nil, dispatch)))
	assert.False(t, agent.MemoryReader{}.Reads(kept(agent.MemoryScopeUser, avery, pulid.Nil)),
		"nobody in particular reads nobody's own memories")
}

func TestMemoryValidate_APersonalScopeNamesItsReaders(t *testing.T) {
	t.Parallel()

	for _, scope := range []agent.MemoryScope{agent.MemoryScopeUser, agent.MemoryScopeRole} {
		memory := validMemory("x")
		memory.Scope = scope
		me := errortypes.NewMultiError()
		memory.Validate(me)
		assert.True(t, me.HasErrors(), scope)
	}

	memory := validMemory("x")
	memory.SetAudience(agent.MemoryScopeUser, pulid.MustNew("usr_"), pulid.MustNew("rol_"))
	assert.Nil(t, memory.RoleID, "a memory kept for a person names no role")
	me := errortypes.NewMultiError()
	memory.Validate(me)
	assert.False(t, me.HasErrors())

	offered := validMemory("x")
	offered.Source = agent.MemorySourceAgent
	offered.Status = agent.MemoryStatusSuggested
	me = errortypes.NewMultiError()
	offered.Validate(me)
	assert.False(t, me.HasErrors(), "an agent may offer a memory for a person to accept")

	written := validMemory("x")
	written.Status = agent.MemoryStatusSuggested
	me = errortypes.NewMultiError()
	written.Validate(me)
	assert.True(t, me.HasErrors(), "a person's own memory is never a suggestion")
}

// touchedAt is a memory kept for the organization, about nothing in
// particular, last touched the given number of days after day zero.
func touchedAt(days int64) *agent.Memory {
	memory := validMemory("touched on day " + strconv.FormatInt(days, 10))
	memory.Scope = agent.MemoryScopeOrganization
	memory.CreatedAt = 1_700_000_000 + days*24*60*60
	memory.UpdatedAt = memory.CreatedAt

	return memory
}

// A rule saved in spring that no prompt has carried since, while the rules
// beside it are read every day, stops riding along in autumn; the rules in
// use, and one used last week, stay.
func TestWithoutStaleMemories_LeavesOutWhatWentUnusedWhileItsNeighborsWereRead(t *testing.T) {
	t.Parallel()

	forgotten := touchedAt(0)
	recent := touchedAt(0)
	usedLastWeek := 1_700_000_000 + int64(93)*24*60*60
	recent.LastUsedAt = &usedLastWeek
	today := touchedAt(100)

	kept := agent.WithoutStaleMemories([]*agent.Memory{forgotten, recent, today})

	assert.Equal(t, []*agent.Memory{recent, today}, kept,
		"a hundred days unused is stale; a use last week makes an old memory fresh")
	assert.Equal(t, 90*24*60*60, agent.MemoryStaleAfterSeconds)
}

// An organization that paused its agents for half a year comes back to
// every memory it kept: nothing was used, so nothing fell behind.
func TestWithoutStaleMemories_MeasuresAgainstTheNewestNeighborNotTheClock(t *testing.T) {
	t.Parallel()

	first, second := touchedAt(0), touchedAt(10)

	kept := agent.WithoutStaleMemories([]*agent.Memory{first, second})

	assert.Equal(t, []*agent.Memory{first, second}, kept)
}

// Dana's own note from before her leave is measured against her other
// notes, not against what the rest of the organization used while she was
// away; and a memory about one customer stays however long that customer
// was quiet.
func TestWithoutStaleMemories_JudgesEachReaderAndSubjectOnItsOwn(t *testing.T) {
	t.Parallel()

	dana := pulid.MustNew("usr_")
	hers := touchedAt(0)
	hers.SetAudience(agent.MemoryScopeUser, dana, pulid.Nil)
	busyOrg := touchedAt(200)

	customer := pulid.MustNew("cus_")
	aboutAcme := touchedAt(0)
	aboutAcme.SubjectType = agent.MemorySubjectCustomer
	aboutAcme.SubjectID = &customer

	kept := agent.WithoutStaleMemories([]*agent.Memory{hers, busyOrg, aboutAcme})

	assert.Equal(t, []*agent.Memory{hers, busyOrg, aboutAcme}, kept)
}
