package scriptquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/stretchr/testify/assert"
)

// A script that names a read it may not call is told the reads it may, the
// nearest first, and how a tool's name becomes a script's: told only that
// '_Trenova' had no attribute 'fuel', Haiku guessed two more spellings.
func TestMissingReadNote_NamesTheReadsAScriptMayCall(t *testing.T) {
	t.Parallel()

	index := []codemode.Read{
		{Name: "shipment_list"},
		{Name: "fuel_list_purchases"},
		{Name: "worker_list"},
	}

	note := missingReadNote(&codemode.ScriptError{
		Name:  "AttributeError",
		Value: "'_Trenova' object has no attribute 'fuel_purchase'",
	}, index)
	assert.Contains(t, note, `No read is named "fuel_purchase" here.`)
	assert.Contains(t, note,
		"trenova.shipment.list, trenova.fuel.list_purchases, trenova.worker.list")

	near := missingReadNote(&codemode.ScriptError{
		Name:  "AttributeError",
		Value: "'_Namespace' object has no attribute 'purchases'",
	}, index)
	assert.Contains(t, near, "are trenova.fuel.list_purchases, trenova.shipment.list",
		"the read whose name holds the missing word comes first")

	assert.Empty(t, missingReadNote(&codemode.ScriptError{
		Name: "AttributeError", Value: "'dict' object has no attribute 'items2'",
	}, index), "an attribute error in the script's own code says nothing about reads")
	assert.Empty(t, missingReadNote(&codemode.ScriptError{
		Name: "KeyError", Value: "'fuel'",
	}, index))
	assert.Empty(t, missingReadNote(nil, index))
}

// A script that stopped before its first read is what the turn's count
// leaves out; one that read and then failed still counts.
func TestScriptAnswer_ReadNothingOnlyBeforeItsFirstRead(t *testing.T) {
	t.Parallel()

	assert.True(t, (&scriptAnswer{Status: "Errored"}).ScriptReadNothing())
	assert.False(t, (&scriptAnswer{
		Status: "Errored", Calls: []scriptCallAnswer{{Seq: 1, Tool: "worker_list"}},
	}).ScriptReadNothing())
	assert.False(t, (&scriptAnswer{Status: "Succeeded"}).ScriptReadNothing())
}
