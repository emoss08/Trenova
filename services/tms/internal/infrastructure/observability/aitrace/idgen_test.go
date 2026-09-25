package aitrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIDGenerator_ForcesTheAnchoredIds(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_forced")
	generator := NewIDGenerator()

	traceID, spanID := generator.NewIDs(withForcedAnchor(t.Context(), a))

	assert.Equal(t, a.TraceID, traceID)
	assert.Equal(t, a.RootSpanID, spanID)
}

func TestIDGenerator_IsRandomWithoutAForcedAnchor(t *testing.T) {
	t.Parallel()

	generator := NewIDGenerator()

	firstTrace, firstSpan := generator.NewIDs(t.Context())
	secondTrace, secondSpan := generator.NewIDs(t.Context())

	assert.True(t, firstTrace.IsValid())
	assert.True(t, firstSpan.IsValid())
	assert.NotEqual(t, firstTrace, secondTrace)
	assert.NotEqual(t, firstSpan, secondSpan)

	invalidTrace, invalidSpan := generator.NewIDs(withForcedAnchor(t.Context(), Anchor{}))
	assert.True(t, invalidTrace.IsValid(), "an invalid anchor is never forced")
	assert.True(t, invalidSpan.IsValid())
}

func TestIDGenerator_NeverForcesAChildSpanID(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_forced_child")
	spanID := NewIDGenerator().NewSpanID(withForcedAnchor(t.Context(), a), a.TraceID)

	assert.True(t, spanID.IsValid())
	assert.NotEqual(t, a.RootSpanID, spanID)
}
