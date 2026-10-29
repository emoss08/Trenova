package agentreflection

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validReflection() *Reflection {
	return &Reflection{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		SubjectType:       ReflectionSubjectThread,
		ThreadID:          pulid.Must("athr_"),
		Status:            ReflectionStatusRunning,
		FromSequence:      4,
		ThroughSequence:   9,
	}
}

func TestReflectionValidate(t *testing.T) {
	t.Parallel()

	me := errortypes.NewMultiError()
	validReflection().Validate(me)
	assert.False(t, me.HasErrors())

	cases := map[string]func(*Reflection){
		"a conversation look back without its conversation": func(r *Reflection) { r.ThreadID = nil },
		"a run look back without its run": func(r *Reflection) {
			r.SubjectType = ReflectionSubjectRun
		},
		"a skip reason on a look back that ran": func(r *Reflection) {
			r.SkipReason = ReflectionSkipNoSignal
		},
		"a skip without a reason":      func(r *Reflection) { r.Status = ReflectionStatusSkipped },
		"a stretch that runs backward": func(r *Reflection) { r.ThroughSequence = 2 },
		"an unknown status":            func(r *Reflection) { r.Status = "Paused" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reflection := validReflection()
			change(reflection)
			me := errortypes.NewMultiError()
			reflection.Validate(me)
			assert.True(t, me.HasErrors())
		})
	}
}

func TestReflectionEndings(t *testing.T) {
	t.Parallel()

	skipped := validReflection()
	skipped.Skip(ReflectionSkipNoSignal, 100)
	assert.Equal(t, ReflectionStatusSkipped, skipped.Status)
	assert.True(t, skipped.Status.Settled())
	assert.Equal(t, int64(100), *skipped.FinishedAt)

	failed := validReflection()
	failed.Fail("  the provider timed out  ", 200)
	assert.Equal(t, ReflectionStatusFailed, failed.Status)
	assert.False(t, failed.Status.Settled())
	assert.Equal(t, "the provider timed out", failed.ErrorMessage)

	kept := pulid.MustNew("amem_")
	offered := pulid.MustNew("amem_")
	refreshed := pulid.MustNew("amem_")
	completed := validReflection()
	completed.Complete([]ReflectionChange{
		{Action: ReflectionActionSaved, MemoryID: &kept},
		{Action: ReflectionActionSuggested, MemoryID: &offered},
		{Action: ReflectionActionRefreshed, MemoryID: &refreshed},
		{Action: ReflectionActionRefused, Reason: "It names a tool the work did not use"},
	}, "Kept one, offered one.", 300)
	assert.Equal(t, ReflectionStatusCompleted, completed.Status)
	assert.Equal(t, []pulid.ID{kept, offered}, completed.KeptMemoryIDs())
}

func TestReflectionSignals(t *testing.T) {
	t.Parallel()

	signals := ReflectionSignals{
		{Kind: ReflectionSignalToolRecovered, Count: 1},
		{Kind: ReflectionSignalLongTask, Count: 0},
	}
	assert.True(t, signals.Has(ReflectionSignalToolRecovered))
	assert.False(t, signals.Has(ReflectionSignalLongTask))
	assert.Equal(t, []string{"ToolRecovered"}, signals.Kinds())
}

func TestMemoryKindsAndSourcesForLearning(t *testing.T) {
	t.Parallel()

	assert.True(t, agentmemory.MemoryKindProcedure.IsValid())
	assert.True(t, agentmemory.MemoryKindProcedure.Followed())
	assert.False(t, agentmemory.MemoryKindFact.Followed())
	assert.Less(t, agentmemory.MemoryKindCorrection.Rank(), agentmemory.MemoryKindProcedure.Rank())
	assert.Less(t, agentmemory.MemoryKindProcedure.Rank(), agentmemory.MemoryKindFact.Rank())

	assert.True(t, agentmemory.MemorySourceReflection.OfferedByAgent())
	assert.True(t, agentmemory.MemorySourceAgent.OfferedByAgent())
	assert.False(t, agentmemory.MemorySourceUser.OfferedByAgent())
}

func TestMemoryValidate_ALearnedMemoryNamesItsLookBack(t *testing.T) {
	t.Parallel()

	memory := &agentmemory.Memory{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Kind:           agentmemory.MemoryKindProcedure,
		Source:         agentmemory.MemorySourceReflection,
		Status:         agentmemory.MemoryStatusSuggested,
		Scope:          agentmemory.MemoryScopeOrganization,
		Content:        "Read the move before assigning it.",
	}

	me := errortypes.NewMultiError()
	memory.Validate(me)
	assert.True(t, me.HasErrors(), "a learned memory without its look back")

	memory.ReflectionID = pulid.Must("arfl_")
	me = errortypes.NewMultiError()
	memory.Validate(me)
	assert.False(t, me.HasErrors(), "a learned memory may wait as a suggestion")

	memory.ID = pulid.MustNew("amem_")
	memory.SupersedesID = &memory.ID
	me = errortypes.NewMultiError()
	memory.Validate(me)
	assert.True(t, me.HasErrors(), "a memory cannot replace itself")
}

func TestMemoryRecordKindOfID(t *testing.T) {
	t.Parallel()

	kind, ok := agentmemory.MemoryRecordKindOfID(pulid.MustNew("cus_"))
	assert.True(t, ok)
	assert.Equal(t, agentmemory.MemoryRecordCustomer, kind)

	kind, ok = agentmemory.MemoryRecordKindOfID(pulid.MustNew("sm_"))
	assert.True(t, ok)
	assert.Equal(t, agentmemory.MemoryRecordShipmentMove, kind)

	_, ok = agentmemory.MemoryRecordKindOfID(pulid.MustNew("invl_"))
	assert.False(t, ok)

	_, ok = agentmemory.MemoryRecordKindOfID(pulid.ID("cus_not-an-id"))
	assert.False(t, ok)
}

func TestNewestReplacements_KeepsTheNewestThatTookEffect(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("amem_")
	second := pulid.MustNew("amem_")
	retired := &agentmemory.Memory{
		ID:           pulid.MustNew("amem_"),
		Status:       agentmemory.MemoryStatusRetired,
		SupersedesID: &first,
		CreatedAt:    10,
	}
	active := &agentmemory.Memory{
		ID:           pulid.MustNew("amem_"),
		Status:       agentmemory.MemoryStatusActive,
		SupersedesID: &first,
		CreatedAt:    20,
	}
	offered := &agentmemory.Memory{
		ID:           pulid.MustNew("amem_"),
		Status:       agentmemory.MemoryStatusSuggested,
		SupersedesID: &second,
		CreatedAt:    30,
	}
	dismissed := &agentmemory.Memory{
		ID:           pulid.MustNew("amem_"),
		Status:       agentmemory.MemoryStatusDismissed,
		SupersedesID: &first,
		CreatedAt:    40,
	}

	newest := agentmemory.NewestReplacements(
		[]*agentmemory.Memory{active, nil, offered, retired, dismissed, {ID: first}},
	)

	assert.Equal(t, map[pulid.ID]*agentmemory.Memory{first: active}, newest)
}
