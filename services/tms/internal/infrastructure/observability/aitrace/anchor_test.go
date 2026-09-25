package aitrace

import (
	"crypto/sha256"
	"strconv"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestAnchorFor_DerivesIdsFromTheVersionedHash(t *testing.T) {
	t.Parallel()

	traceSum := sha256.Sum256([]byte("trenova/ai-trace/v1|AgentRun|ar_01J0000000000000000000000"))
	spanSum := sha256.Sum256(
		[]byte("trenova/ai-trace/v1|AgentRun|ar_01J0000000000000000000000|root"),
	)

	a := AnchorFor(AnchorAgentRun, "ar_01J0000000000000000000000")

	var wantTrace trace.TraceID
	copy(wantTrace[:], traceSum[:16])
	var wantSpan trace.SpanID
	copy(wantSpan[:], spanSum[:8])
	assert.Equal(t, wantTrace, a.TraceID)
	assert.Equal(t, wantSpan, a.RootSpanID)
	assert.True(t, a.IsValid())
}

func TestAnchorFor_IsDeterministic(t *testing.T) {
	t.Parallel()

	for _, kind := range []AnchorKind{
		AnchorAgentRun, AnchorAssistantTurn, AnchorDelegate, AnchorEvaluation,
	} {
		first := AnchorFor(kind, "key-1")
		second := AnchorFor(kind, "key-1")
		assert.Equal(t, first, second, kind)
	}
}

func TestAnchorFor_DistinctKindsAndKeysGiveDistinctIds(t *testing.T) {
	t.Parallel()

	traces := make(map[trace.TraceID]string)
	spans := make(map[trace.SpanID]string)
	for _, kind := range []AnchorKind{
		AnchorAgentRun, AnchorAssistantTurn, AnchorDelegate, AnchorEvaluation,
	} {
		for i := range 64 {
			label := string(kind) + "/" + strconv.Itoa(i)
			a := AnchorFor(kind, "key-"+strconv.Itoa(i))
			require.True(t, a.IsValid(), label)

			if previous, seen := traces[a.TraceID]; seen {
				t.Fatalf("%s and %s share a trace id", previous, label)
			}
			if previous, seen := spans[a.RootSpanID]; seen {
				t.Fatalf("%s and %s share a root span id", previous, label)
			}
			traces[a.TraceID] = label
			spans[a.RootSpanID] = label

			var traceHead trace.SpanID
			copy(traceHead[:], a.TraceID[:8])
			assert.NotEqual(t, traceHead, a.RootSpanID,
				"the root span id is hashed separately, not cut from the trace id")
		}
	}
}

func TestAnchorFor_RefusesWhatItCannotName(t *testing.T) {
	t.Parallel()

	assert.False(t, AnchorFor(AnchorAgentRun, "").IsValid())
	assert.False(t, AnchorFor(AnchorKind(""), "ar_1").IsValid())
	assert.False(t, AnchorFor(AnchorKind("Plan"), "ar_1").IsValid())
	assert.Equal(t, Anchor{}, AnchorFor(AnchorKind("Plan"), "ar_1"))
	assert.False(t, Anchor{}.IsValid())
	assert.False(t, Anchor{}.SpanContext().IsValid())
}

func TestForRun_NamesTheUnitOfWork(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("ar_")
	turnID := pulid.MustNew("atrn_")
	evaluationID := pulid.MustNew(agent.EvaluationIDPrefix)

	tests := []struct {
		name       string
		owner      serviceports.RunStepOwner
		delegation *serviceports.Delegation
		want       Anchor
	}{
		{
			name:  "an agent run",
			owner: serviceports.RunStepOwner{Kind: serviceports.RunStepOwnerAgentRun, ID: runID},
			want:  AnchorFor(AnchorAgentRun, runID.String()),
		},
		{
			name: "an assistant turn",
			owner: serviceports.RunStepOwner{
				Kind: serviceports.RunStepOwnerAssistantTurn,
				ID:   turnID,
			},
			want: AnchorFor(AnchorAssistantTurn, turnID.String()),
		},
		{
			name: "an evaluation replay",
			owner: serviceports.RunStepOwner{
				Kind: serviceports.RunStepOwnerAgentRun,
				ID:   evaluationID,
			},
			want: AnchorFor(AnchorEvaluation, evaluationID.String()),
		},
		{
			name: "a delegate working for a turn",
			owner: serviceports.RunStepOwner{
				Kind: serviceports.RunStepOwnerAssistantTurn,
				ID:   turnID,
			},
			delegation: &serviceports.Delegation{CallID: "call_7", StepScope: "scope"},
			want:       AnchorFor(AnchorDelegate, turnID.String()+":call_7"),
		},
		{
			name: "a delegation without a call is the owner's own work",
			owner: serviceports.RunStepOwner{
				Kind: serviceports.RunStepOwnerAssistantTurn,
				ID:   turnID,
			},
			delegation: &serviceports.Delegation{},
			want:       AnchorFor(AnchorAssistantTurn, turnID.String()),
		},
		{
			name:  "no owner",
			owner: serviceports.RunStepOwner{Kind: serviceports.RunStepOwnerAgentRun},
		},
		{
			name:  "an unknown owner kind",
			owner: serviceports.RunStepOwner{Kind: "Plan", ID: runID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ForRun(tt.owner, tt.delegation))
		})
	}
}

func TestForDelegate_MatchesTheDelegateAnchor(t *testing.T) {
	t.Parallel()

	turnID := pulid.MustNew("atrn_")
	assert.Equal(t,
		AnchorFor(AnchorDelegate, turnID.String()+":call_1"),
		ForDelegate(turnID, "call_1"))
	assert.False(t, ForDelegate(turnID, "").IsValid())
	assert.False(t, ForDelegate(pulid.Nil, "call_1").IsValid())
}

func TestForAttribution_AgreesWithForRun(t *testing.T) {
	t.Parallel()

	turnID := pulid.MustNew("atrn_")
	runID := pulid.MustNew("ar_")

	assert.Equal(
		t,
		ForRun(
			serviceports.RunStepOwner{Kind: serviceports.RunStepOwnerAssistantTurn, ID: turnID},
			nil,
		),
		ForAttribution(&serviceports.AIUsageAttribution{
			OwnerKind: serviceports.RunStepOwnerAssistantTurn,
			OwnerID:   turnID,
		}),
	)
	assert.Equal(t,
		ForRun(
			serviceports.RunStepOwner{Kind: serviceports.RunStepOwnerAssistantTurn, ID: turnID},
			&serviceports.Delegation{CallID: "call_2"},
		),
		ForAttribution(&serviceports.AIUsageAttribution{
			OwnerKind:      serviceports.RunStepOwnerAssistantTurn,
			OwnerID:        turnID,
			DelegateCallID: "call_2",
		}))
	assert.Equal(t,
		AnchorFor(AnchorAgentRun, runID.String()),
		ForAttribution(&serviceports.AIUsageAttribution{RunID: runID}),
		"a caller that names only the run is traced to the run")
	assert.False(t, ForAttribution(&serviceports.AIUsageAttribution{
		ThreadID: pulid.MustNew("ath_"),
	}).IsValid())
}

func TestAnchor_SpanContextIsARemoteParent(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAssistantTurn, "atrn_1")
	a.Sampled = true
	sc := a.SpanContext()
	assert.True(t, sc.IsRemote())
	assert.True(t, sc.IsSampled())
	assert.Equal(t, a.TraceID, sc.TraceID())
	assert.Equal(t, a.RootSpanID, sc.SpanID())

	a.Sampled = false
	assert.False(t, a.SpanContext().IsSampled())

	ctx := ContextWithAnchor(t.Context(), a)
	assert.Equal(t, a.SpanContext(), trace.SpanContextFromContext(ctx))
	assert.Equal(t, t.Context(), ContextWithAnchor(t.Context(), Anchor{}))
}

func TestParent_KeepsASpanAlreadyInTheAnchoredTrace(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_parent_keep")
	ctx, span := tracer().Start(ContextWithAnchor(t.Context(), a), "activity")
	defer span.End()

	parented, links := Parent(ctx, a)

	assert.Equal(t, ctx, parented)
	assert.Empty(t, links)
}

func TestParent_ReparentsAForeignSpanAndLinksIt(t *testing.T) {
	t.Parallel()

	turn := AnchorFor(AnchorAssistantTurn, "atrn_parent_move")
	delegate := ForDelegate(pulid.ID("atrn_parent_move"), "call_1")
	ctx, span := tracer().Start(ContextWithAnchor(t.Context(), turn), "run-activity")
	defer span.End()

	parented, links := Parent(ctx, delegate)

	assert.Equal(t, delegate.SpanContext(), trace.SpanContextFromContext(parented))
	require.Len(t, links, 1)
	assert.Equal(t, span.SpanContext(), links[0].SpanContext)
}

func TestParent_AnchorsAContextWithNoSpan(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_parent_empty")
	parented, links := Parent(t.Context(), a)

	assert.Equal(t, a.SpanContext(), trace.SpanContextFromContext(parented))
	assert.Empty(t, links)

	unchanged, links := Parent(t.Context(), Anchor{})
	assert.Equal(t, t.Context(), unchanged)
	assert.Empty(t, links)
}
