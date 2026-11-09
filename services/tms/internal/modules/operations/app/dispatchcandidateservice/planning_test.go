package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/stretchr/testify/assert"
)

func TestPlannedCompletion(t *testing.T) {
	t.Parallel()

	end := testNow + 22*3600

	tests := []struct {
		name  string
		move  *dispatchboardports.BoardMove
		score *dispatchplanports.CandidateScore
		want  int64
	}{
		{
			name:  "the projected finish wins",
			move:  &dispatchboardports.BoardMove{DestinationWindowStart: testNow + 20*3600},
			score: &dispatchplanports.CandidateScore{ProjectedFreeAt: testNow + 9*3600},
			want:  testNow + 9*3600,
		},
		{
			name: "without a projection the delivery window closes it",
			move: &dispatchboardports.BoardMove{
				DestinationWindowStart: testNow + 20*3600,
				DestinationWindowEnd:   &end,
			},
			score: &dispatchplanports.CandidateScore{},
			want:  end,
		},
		{name: "nil move", score: &dispatchplanports.CandidateScore{}, want: 0},
		{name: "nil score", move: &dispatchboardports.BoardMove{}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, PlannedCompletion(tt.move, tt.score))
		})
	}
}
