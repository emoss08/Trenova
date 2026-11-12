package agentmemory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemory_UpkeepReason(t *testing.T) {
	t.Parallel()

	const now = int64(1_800_000_000)
	window := UpkeepWindow(now)
	daysAgo := func(days int64) int64 { return now - days*secondsPerDay }
	at := func(v int64) *int64 { return &v }

	for name, tc := range map[string]struct {
		memory Memory
		want   MemoryUpkeepReason
	}{
		"a fact nobody has confirmed in six months is aging, however often it is used": {
			memory: Memory{Kind: MemoryKindFact, CreatedAt: daysAgo(181), UseCount: 40},
			want:   MemoryUpkeepAging,
		},
		"a fact confirmed since is not asked about again": {
			memory: Memory{
				Kind: MemoryKindFact, CreatedAt: daysAgo(400), ConfirmedAt: at(daysAgo(10)),
				UseCount: 40,
			},
		},
		"a fact with an end date expires on its own": {
			memory: Memory{
				Kind: MemoryKindFact, CreatedAt: daysAgo(181), ExpiresAt: at(now + 1),
				UseCount: 40,
			},
		},
		"an instruction no turn has used in a month is unused": {
			memory: Memory{Kind: MemoryKindInstruction, CreatedAt: daysAgo(31)},
			want:   MemoryUpkeepNeverUsed,
		},
		"an unused memory recorded this month is given time": {
			memory: Memory{Kind: MemoryKindInstruction, CreatedAt: daysAgo(29)},
		},
		"an instruction in use is left alone however old": {
			memory: Memory{Kind: MemoryKindInstruction, CreatedAt: daysAgo(900), UseCount: 1},
		},
		"a fact both aging and unused is asked whether it is still true": {
			memory: Memory{Kind: MemoryKindFact, CreatedAt: daysAgo(200)},
			want:   MemoryUpkeepAging,
		},
		"one person's own memory is theirs to keep up": {
			memory: Memory{
				Kind: MemoryKindFact, Scope: MemoryScopeUser, CreatedAt: daysAgo(400),
			},
		},
		"a retired memory is read by nobody": {
			memory: Memory{
				Kind: MemoryKindFact, Status: MemoryStatusRetired, CreatedAt: daysAgo(400),
			},
		},
	} {
		memory := tc.memory
		if memory.Status == "" {
			memory.Status = MemoryStatusActive
		}
		if memory.Scope == "" {
			memory.Scope = MemoryScopeOrganization
		}
		assert.Equal(t, tc.want, memory.UpkeepReason(window), name)
	}
}
