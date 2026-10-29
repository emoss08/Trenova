package delegation

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/stretchr/testify/assert"
)

func TestDirectedReply_SaysWhatCameOfTheTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		report assistantstreamports.AssistantDelegateFinishedEvent
		want   string
	}{
		"completed": {
			report: assistantstreamports.AssistantDelegateFinishedEvent{
				Status: assistantstreamports.DelegateStatusCompleted, Reply: "  Marked ready.  ",
			},
			want: "Marked ready.",
		},
		"completed in silence": {
			report: assistantstreamports.AssistantDelegateFinishedEvent{
				Status: assistantstreamports.DelegateStatusCompleted, AgentName: "Billing Assistant",
			},
			want: "Billing Assistant finished without saying anything more.",
		},
		"out of tool calls": {
			report: assistantstreamports.AssistantDelegateFinishedEvent{
				Status: assistantstreamports.DelegateStatusExhausted,
				Reply:  "I checked the paperwork.",
				Reason: "It used every tool call.",
			},
			want: "I checked the paperwork.\n\nIt used every tool call.",
		},
		"failed": {
			report: assistantstreamports.AssistantDelegateFinishedEvent{
				Status: assistantstreamports.DelegateStatusFailed, Reason: "The model stopped answering.",
			},
			want: "The model stopped answering.",
		},
		"failed without a reason": {
			report: assistantstreamports.AssistantDelegateFinishedEvent{
				Status: assistantstreamports.DelegateStatusFailed, AgentName: "Billing Assistant",
			},
			want: "Billing Assistant did not report back.",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, DirectedReply(&tt.report))
		})
	}
}
