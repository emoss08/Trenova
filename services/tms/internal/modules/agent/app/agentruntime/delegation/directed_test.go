package delegation

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/stretchr/testify/assert"
)

func TestDirectedReply_SaysWhatCameOfTheTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		report assistantports.AssistantDelegateFinishedEvent
		want   string
	}{
		"completed": {
			report: assistantports.AssistantDelegateFinishedEvent{
				Status: assistantports.DelegateStatusCompleted, Reply: "  Marked ready.  ",
			},
			want: "Marked ready.",
		},
		"completed in silence": {
			report: assistantports.AssistantDelegateFinishedEvent{
				Status: assistantports.DelegateStatusCompleted, AgentName: "Billing Assistant",
			},
			want: "Billing Assistant finished without saying anything more.",
		},
		"out of tool calls": {
			report: assistantports.AssistantDelegateFinishedEvent{
				Status: assistantports.DelegateStatusExhausted,
				Reply:  "I checked the paperwork.",
				Reason: "It used every tool call.",
			},
			want: "I checked the paperwork.\n\nIt used every tool call.",
		},
		"failed": {
			report: assistantports.AssistantDelegateFinishedEvent{
				Status: assistantports.DelegateStatusFailed, Reason: "The model stopped answering.",
			},
			want: "The model stopped answering.",
		},
		"failed without a reason": {
			report: assistantports.AssistantDelegateFinishedEvent{
				Status: assistantports.DelegateStatusFailed, AgentName: "Billing Assistant",
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
