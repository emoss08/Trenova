package aitrace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestEmitRoot_UsesTheAnchoredIdsAndTheRecordedTimes(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_emit_root")
	start := time.Unix(1_790_000_000, 0)
	end := start.Add(42 * time.Second)
	origin := AnchorFor(AnchorAssistantTurn, "atrn_emit_origin")

	ctx, other := tracer().Start(t.Context(), "unrelated request")
	EmitRoot(ctx, a, &RootSpec{
		Name:              "invoke_agent Billing desk",
		Start:             start,
		End:               end,
		Attrs:             []attribute.KeyValue{GenAIOperationName.String(OperationInvokeAgent)},
		Links:             []trace.Link{origin.Link()},
		Status:            codes.Error,
		StatusDescription: "timeout",
	})
	other.End()

	root := spanNamed(t, a.TraceID, "invoke_agent Billing desk")
	assert.Equal(t, a.RootSpanID, root.SpanContext.SpanID())
	assert.False(t, root.Parent.IsValid(), "the root has no parent")
	assert.Equal(t, trace.SpanKindInternal, root.SpanKind)
	assert.True(t, root.StartTime.Equal(start))
	assert.True(t, root.EndTime.Equal(end))
	assert.Contains(t, root.Attributes, AIAnchor.Bool(true))
	assert.Contains(t, root.Attributes, GenAIOperationName.String(OperationInvokeAgent))
	require.Len(t, root.Links, 1)
	assert.Equal(t, origin.TraceID, root.Links[0].SpanContext.TraceID())
	assert.Equal(t, codes.Error, root.Status.Code)
	assert.Equal(t, "timeout", root.Status.Description)
}

func TestEmitRoot_ChildrenStartedEarlierHangUnderIt(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAssistantTurn, "atrn_emit_children")
	_, child := tracer().Start(ContextWithAnchor(t.Context(), a), "activity")
	child.End()

	EmitRoot(t.Context(), a, &RootSpec{Name: "invoke_agent Assistant", Start: time.Now()})

	activity := spanNamed(t, a.TraceID, "activity")
	root := spanNamed(t, a.TraceID, "invoke_agent Assistant")
	assert.Equal(t, root.SpanContext.SpanID(), activity.Parent.SpanID())
}

func TestEmitRoot_NeverStartsAfterItEnds(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_emit_inverted")
	end := time.Unix(1_790_000_000, 0)

	EmitRoot(t.Context(), a, &RootSpec{Name: "inverted", Start: end.Add(time.Minute), End: end})

	root := spanNamed(t, a.TraceID, "inverted")
	assert.True(t, root.StartTime.Equal(end))
	assert.True(t, root.EndTime.Equal(end))

	b := AnchorFor(AnchorAgentRun, "ar_emit_unstarted")
	EmitRoot(t.Context(), b, &RootSpec{Name: "unstarted", End: end})
	unstarted := spanNamed(t, b.TraceID, "unstarted")
	assert.True(t, unstarted.StartTime.Equal(end))
}

func TestEmitRoot_IgnoresAnInvalidAnchor(t *testing.T) {
	t.Parallel()

	before := len(exporter.GetSpans())
	EmitRoot(t.Context(), Anchor{}, &RootSpec{Name: "nothing"})

	for _, span := range exporter.GetSpans()[before:] {
		assert.NotEqual(t, "nothing", span.Name)
	}
}

func TestEmitRoot_DropsAnUnsampledAnchor(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_emit_unsampled")
	a.Sampled = false

	EmitRoot(t.Context(), a, &RootSpec{Name: "unsampled", Start: time.Now()})

	assert.Empty(t, spansIn(a.TraceID))
}
