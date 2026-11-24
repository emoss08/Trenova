package toolwords_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolwords"
	"github.com/stretchr/testify/assert"
)

func TestNamedReadsAToolNamedInWords(t *testing.T) {
	t.Parallel()

	assert.Contains(t, toolwords.Named("customer list to find Peak Distributing's id"),
		"customer_list")
	assert.Contains(t, toolwords.Named("list customers by name"), "customer_list")
	assert.Contains(t, toolwords.Named("receivable list open items"),
		"receivable_list_open_items")
	assert.Contains(t, toolwords.Named("the open items, receivable_list_open_items"),
		"receivable_list_open_items")
	assert.Contains(t, toolwords.Named("Worker_List please"), "worker_list")
}

func TestNamedIsBounded(t *testing.T) {
	t.Parallel()

	words := toolwords.Named("a b c d e")
	assert.LessOrEqual(t, len(words), 5*8)
	assert.Empty(t, toolwords.Named(""))
}
