package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThreadApplyAttention(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		thread  Thread
		signals ThreadAttentionSignals
		want    ThreadAttention
	}{
		{
			name:   "quiet thread",
			thread: Thread{LastMessageAt: 100, LastReadAt: 100},
			want:   ThreadAttention{},
		},
		{
			name:   "message after the last read is unread",
			thread: Thread{LastMessageAt: 200, LastReadAt: 100},
			want:   ThreadAttention{Unread: true},
		},
		{
			name:    "pending decisions are carried through",
			thread:  Thread{LastMessageAt: 100, LastReadAt: 100},
			signals: ThreadAttentionSignals{PendingDecisions: 3},
			want:    ThreadAttention{PendingDecisions: 3},
		},
		{
			name:    "a failed last turn is flagged",
			thread:  Thread{},
			signals: ThreadAttentionSignals{LastTurnStatus: AssistantTurnStatusFailed},
			want:    ThreadAttention{LastTurnFailed: true},
		},
		{
			name:    "a stopped last turn is not a failure",
			thread:  Thread{},
			signals: ThreadAttentionSignals{LastTurnStatus: AssistantTurnStatusStopped},
			want:    ThreadAttention{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			thread := tc.thread
			thread.ApplyAttention(tc.signals)

			require.NotNil(t, thread.Attention)
			assert.Equal(t, tc.want, *thread.Attention)
		})
	}
}
