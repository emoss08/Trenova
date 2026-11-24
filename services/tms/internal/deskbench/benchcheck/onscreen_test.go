package benchcheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/stretchr/testify/assert"
)

func TestOnScreenReadsTheReplyAndTheBoardItDrew(t *testing.T) {
	t.Parallel()

	evidence := &Evidence{Turns: []*benchrecord.TurnRecord{{
		Reply: "6 credentials need attention: 2 has already lapsed and 4 run out within 30 days.",
		Tools: []*benchrecord.ToolCallRecord{
			{
				Name: "show_answer", Failed: true,
				Result: `Tool "show_answer" was not run: the words type "8 days".`,
			},
			{
				Name: "show_answer",
				Result: "# Credentials expiring within 30 days\n\n## 2. Expiring within 30 days\n\n" +
					"_Robert Brown's medical card and physical expire on December 1, and Sarah " +
					"Williams's on December 9._",
			},
			{Name: "worker_list_credentials", Result: "Ines Ortega"},
		},
	}}}

	shown := evidence.onScreen()

	assert.True(t, replyStates(shown, "Robert"), "a name in a group's note is on screen")
	assert.True(t, replyStates(shown, "Sarah"))
	assert.False(t, replyStates(shown, "8 days"), "a refused board was never on screen")
	assert.False(t, replyStates(shown, "Ines Ortega"), "what a read returned is not on screen")
}
