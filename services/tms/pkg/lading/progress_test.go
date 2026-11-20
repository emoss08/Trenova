package lading

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlowStarted_IsWhetherTheConversationLeftAnythingToKeep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{name: "no state", state: "", want: false},
		{name: "the host's empty state", state: `{}`, want: false},
		{
			name: "a rejected message leaves the default state",
			state: `{"key":"","documents":{},"load":{"exists":false,"state":""},"pending":[],` +
				`"batches":[],"finishedSends":[],"sends":0,"sent":{}}`,
			want: false,
		},
		{name: "a received document", state: `{"documents":{"status":{"key":"BH-1"}}}`, want: true},
		{name: "a send", state: `{"sends":1}`, want: true},
		{name: "a load Trenova has", state: `{"load":{"exists":true,"state":"open"}}`, want: true},
		{name: "a send awaiting its ack", state: `{"pending":[{"id":"s1","due":10}]}`, want: true},
		{name: "a batch being gathered", state: `{"batches":[{"clause":0,"due":10}]}`, want: true},
		{name: "state the host cannot read is kept", state: `[`, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, FlowStarted([]byte(tt.state)))
		})
	}
}
