package chaptertitles

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/assert"
)

func sequencesOf(messages []conversation.Message) []int {
	out := make([]int, 0, len(messages))
	for idx := range messages {
		out = append(out, messages[idx].Sequence)
	}

	return out
}

func TestSnapToChapter_EndsTheSummaryWhereTheLatestChapterInsideItStarts(t *testing.T) {
	t.Parallel()

	older := assistanttest.SpokenStretch(0, 9)

	snapped, through := SnapToChapter(
		older,
		9,
		map[int]string{0: "Late loads", 6: "Acme invoice dispute"},
	)

	assert.Equal(t, 5, through)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, sequencesOf(snapped),
		"the chapter that starts inside the stretch is kept whole for the agent")
}

func TestSnapToChapter_TakesTheLatestOfSeveralStarts(t *testing.T) {
	t.Parallel()

	older := assistanttest.SpokenStretch(0, 9)

	snapped, through := SnapToChapter(older, 9, map[int]string{4: "B", 7: "C"})

	assert.Equal(t, 6, through)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6}, sequencesOf(snapped))
}

func TestSnapToChapter_KeepsTheCutWhenTheStartWouldSummarizeTooLittle(t *testing.T) {
	t.Parallel()

	older := assistanttest.SpokenStretch(0, 9)

	snapped, through := SnapToChapter(older, 9, map[int]string{0: "Late loads", 3: "Acme"})

	assert.Equal(t, 9, through, "three of ten messages is less than half the stretch")
	assert.Equal(t, older, snapped)
}

func TestSnapToChapter_SnapsAtExactlyHalf(t *testing.T) {
	t.Parallel()

	older := assistanttest.SpokenStretch(0, 9)

	snapped, through := SnapToChapter(older, 9, map[int]string{5: "Acme"})

	assert.Equal(t, 4, through)
	assert.Len(t, snapped, 5)
}

func TestSnapToChapter_LeavesTheCutWithoutAChapterStartInside(t *testing.T) {
	t.Parallel()

	older := assistanttest.SpokenStretch(0, 9)
	cases := map[string]map[int]string{
		"no chapters":                        nil,
		"empty titles":                       {},
		"a chapter at the stretch's start":   {0: "Late loads"},
		"a chapter after the stretch":        {12: "Acme"},
		"chapters at the start and after it": {0: "Late loads", 10: "Acme"},
	}
	for name, titles := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			snapped, through := SnapToChapter(older, 9, titles)

			assert.Equal(t, 9, through)
			assert.Equal(t, older, snapped)
		})
	}
}

func TestSnapToChapter_LeavesAStretchOfOnlyAnEarlierSummary(t *testing.T) {
	t.Parallel()

	older := []conversation.Message{assistanttest.EarlierSummary(20, 9)}

	snapped, through := SnapToChapter(older, 9, map[int]string{20: "Late loads", 5: "Acme"})

	assert.Equal(t, 9, through)
	assert.Equal(t, older, snapped)
}

func TestSnapToChapter_KeepsAnEarlierSummaryAndCountsOnlyWhatWasSaid(t *testing.T) {
	t.Parallel()

	older := append(
		[]conversation.Message{assistanttest.EarlierSummary(30, 9)},
		assistanttest.SpokenStretch(10, 19)...)

	snapped, through := SnapToChapter(older, 19, map[int]string{30: "Old", 16: "Acme"})

	assert.Equal(t, 15, through)
	assert.Equal(t, []int{30, 10, 11, 12, 13, 14, 15}, sequencesOf(snapped),
		"the earlier summary stays first and the six messages before the start are summarized")
}

func TestSnapToChapter_LeavesAnEmptyStretch(t *testing.T) {
	t.Parallel()

	snapped, through := SnapToChapter(nil, 4, map[int]string{2: "Acme"})

	assert.Empty(t, snapped)
	assert.Equal(t, 4, through)
}
