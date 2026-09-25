package aitrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceparent_RoundTripsIntoALink(t *testing.T) {
	t.Parallel()

	ctx, span := tracer().Start(t.Context(), "POST /assistant/threads/:id/turns/")
	defer span.End()

	traceparent := Traceparent(ctx)
	require.NotEmpty(t, traceparent)
	assert.Equal(t,
		"00-"+span.SpanContext().TraceID().String()+"-"+span.SpanContext().SpanID().String()+"-01",
		traceparent)

	link, ok := LinkFromTraceparent(traceparent)
	require.True(t, ok)
	assert.Equal(t, span.SpanContext().TraceID(), link.SpanContext.TraceID())
	assert.Equal(t, span.SpanContext().SpanID(), link.SpanContext.SpanID())
}

func TestTraceparent_IsEmptyWithoutASpan(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Traceparent(t.Context()))

	_, ok := LinkFromTraceparent("")
	assert.False(t, ok)
	_, ok = LinkFromTraceparent("00-not-a-trace-01")
	assert.False(t, ok)
}

func TestIDs_ReadsTheCurrentSpan(t *testing.T) {
	t.Parallel()

	traceID, spanID := IDs(t.Context())
	assert.Empty(t, traceID)
	assert.Empty(t, spanID)

	a := AnchorFor(AnchorAgentRun, "ar_ids")
	traceID, spanID = IDs(ContextWithAnchor(t.Context(), a))
	assert.Equal(t, a.TraceID.String(), traceID)
	assert.Equal(t, a.RootSpanID.String(), spanID)
}

func TestLinkTo_AcceptsOnlyStoredW3CIds(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_link")
	link, ok := LinkTo(a.TraceID.String(), a.RootSpanID.String())
	require.True(t, ok)
	assert.Equal(t, a.TraceID, link.SpanContext.TraceID())
	assert.Equal(t, a.RootSpanID, link.SpanContext.SpanID())
	assert.True(t, link.SpanContext.IsRemote())

	for _, ids := range [][2]string{
		{"", ""},
		{a.TraceID.String(), ""},
		{"00000000000000000000000000000000", a.RootSpanID.String()},
		{a.TraceID.String(), "zzzzzzzzzzzzzzzz"},
	} {
		_, ok = LinkTo(ids[0], ids[1])
		assert.False(t, ok, ids)
	}
}

func TestAnchor_LinkPointsAtTheRoot(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorDelegate, "atrn_1:call_1")
	assert.Equal(t, trace.Link{SpanContext: a.SpanContext()}, a.Link())
}
