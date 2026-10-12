package aifeedback

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTurnSnapshot_BoundsWhatItKeeps(t *testing.T) {
	t.Parallel()

	tools := make([]ToolLine, 0, MaxSnapshotToolLines+5)
	for index := range MaxSnapshotToolLines + 5 {
		tools = append(tools, ToolLine{
			Name:    fmt.Sprintf("tool_%d", index),
			Summary: "Found\n3   loads",
		})
	}

	snapshot := NewTurnSnapshot(SnapshotParams{
		Question: strings.Repeat("q", MaxSnapshotTextRunes+50),
		Answer:   strings.Repeat("ü", MaxSnapshotTextRunes+50),
		Tools:    tools,
	})

	assert.Equal(t, MaxSnapshotTextRunes+1, utf8.RuneCountInString(snapshot.Question))
	assert.True(t, strings.HasSuffix(snapshot.Question, "…"))
	assert.Equal(t, MaxSnapshotTextRunes+1, utf8.RuneCountInString(snapshot.Answer))
	assert.Len(t, snapshot.Tools, MaxSnapshotToolLines)
	assert.Equal(t, 5, snapshot.OmittedTools)
	assert.Equal(t, "Found 3 loads", snapshot.Tools[0].Summary, "a summary is one line")
	assert.False(t, snapshot.Redacted)
}

func TestNewTurnSnapshot_KeepsShortTextAsItIs(t *testing.T) {
	t.Parallel()

	snapshot := NewTurnSnapshot(SnapshotParams{Question: "Where is load 12?", Answer: "In Dallas."})

	assert.Equal(t, "Where is load 12?", snapshot.Question)
	assert.Equal(t, "In Dallas.", snapshot.Answer)
	assert.Empty(t, snapshot.Tools)
	assert.Zero(t, snapshot.OmittedTools)
}

func TestNewTurnSnapshot_RedactsRestrictedValuesEverywhere(t *testing.T) {
	t.Parallel()

	snapshot := NewTurnSnapshot(SnapshotParams{
		Question: "What is Dana Reyes paid per mile?",
		Answer:   "Dana Reyes is paid 0.62 per mile; her license D1234567 expires soon.",
		Tools: []ToolLine{
			{Name: "get_worker", Summary: "Dana Reyes, license D1234567"},
		},
		Redact: []string{"D1234567", "0.62", "Dana Reyes", "  ", "ab"},
	})

	require.True(t, snapshot.Redacted)
	for _, text := range []string{snapshot.Question, snapshot.Answer, snapshot.Tools[0].Summary} {
		assert.NotContains(t, text, "D1234567")
		assert.NotContains(t, text, "0.62")
		assert.NotContains(t, text, "Dana Reyes")
	}
	assert.Equal(
		t,
		"[redacted] is paid [redacted] per mile; her license [redacted] expires soon.",
		snapshot.Answer,
	)
	assert.Equal(t, "get_worker", snapshot.Tools[0].Name)
}

func TestNewTurnSnapshot_RedactsTheLongerValueFirst(t *testing.T) {
	t.Parallel()

	snapshot := NewTurnSnapshot(SnapshotParams{
		Answer: "Account 12345678 belongs to 1234.",
		Redact: []string{"1234", "12345678"},
	})

	assert.Equal(t, "Account [redacted] belongs to [redacted].", snapshot.Answer,
		"a shorter value inside a longer one does not leave the rest of it behind")
}

func TestNewTurnSnapshot_RedactsBeforeCutting(t *testing.T) {
	t.Parallel()

	secret := "SSN-000-11-2222"
	answer := strings.Repeat("a", MaxSnapshotTextRunes-5) + secret

	snapshot := NewTurnSnapshot(SnapshotParams{Answer: answer, Redact: []string{secret}})

	assert.NotContains(t, snapshot.Answer, "SSN-0",
		"a value cut at the boundary does not leak its start")
}
