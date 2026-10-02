package agentruntime

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

// The reader folds a turn's repeated reads of one tool into one artifact, so
// the turn must tell it about every earlier list and search of the tool, not
// only every earlier get.
func TestTurn_RemembersEarlierListsAndSearchesForTheReader(t *testing.T) {
	t.Parallel()

	turn := &Turn{}
	turn.noteShown(&serviceports.ToolCall{ID: "call_1", Name: "search_shipments"})
	turn.noteShown(&serviceports.ToolCall{ID: "call_2", Name: "list_invoices"})
	turn.noteShown(&serviceports.ToolCall{ID: "call_3", Name: "get_invoice"})
	turn.noteShown(&serviceports.ToolCall{ID: "call_4", Name: "approve_invoice"})
	turn.noteShown(&serviceports.ToolCall{ID: "call_5", Name: "search_shipments"})

	assert.Equal(t, []string{"call_1", "call_5"}, turn.earlier("search_shipments"))
	assert.Equal(t, []string{"call_2"}, turn.earlier("list_invoices"))
	assert.Equal(t, []string{"call_3"}, turn.earlier("get_invoice"))
	assert.Nil(t, turn.earlier("approve_invoice"), "a write is never bunched")
}
