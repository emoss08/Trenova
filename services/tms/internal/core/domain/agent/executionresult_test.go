package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolExecutionResult_BoundedKeepsAPointerNotAPayload(t *testing.T) {
	t.Parallel()

	ids := make(map[string]string, 12)
	for _, key := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"} {
		ids[key+"Id"] = strings.Repeat("x", 500)
	}
	ids["blankId"] = "   "

	bounded := (&ToolExecutionResult{
		Action: "created",
		Kind:   "report",
		Name:   "Shipments\nfor  Peak\t" + strings.Repeat("n", 5000),
		IDs:    ids,
	}).Bounded()

	require.NotNil(t, bounded)
	assert.True(t, strings.HasPrefix(bounded.Name, "Shipments for Peak n"), "one line")
	assert.Len(t, []rune(bounded.Name), maxResultNameChars)
	assert.Len(t, bounded.IDs, maxResultIDs)
	assert.NotContains(t, bounded.IDs, "blankId", "an empty id is not kept")
	assert.Contains(t, bounded.IDs, "aId", "the first ids in key order are kept")
	assert.NotContains(t, bounded.IDs, "lId")

	size := len(bounded.Action) + len(bounded.Kind) + len(bounded.Name)
	for key, id := range bounded.IDs {
		size += len(key) + len(id)
	}
	assert.LessOrEqual(t, size, 2048)
}

func TestToolExecutionResult_BoundedDropsAnEmptyResult(t *testing.T) {
	t.Parallel()

	var missing *ToolExecutionResult
	assert.Nil(t, missing.Bounded())
	assert.Nil(t, (&ToolExecutionResult{Name: " \n ", IDs: map[string]string{"x": ""}}).Bounded())
}

func TestToolExecutionResult_DescribesWhatWasMade(t *testing.T) {
	t.Parallel()

	result := &ToolExecutionResult{
		Action: "created",
		Kind:   "report",
		Name:   "Lanes",
		IDs:    map[string]string{"definitionId": "rd_01", "folderId": "fld_01"},
	}

	assert.Equal(t, `It created the report "Lanes".`, result.Describe())
	assert.Equal(t, "definitionId rd_01, folderId fld_01", result.References())
	assert.Equal(t,
		"It produced definitionId rd_01, folderId fld_01; use those ids, not the "+
			"proposal's, when you refer to what it made.",
		result.IDNote(),
	)

	assert.Equal(t, "It created the report.", (&ToolExecutionResult{
		Action: "created", Kind: "report",
	}).Describe())
	assert.Equal(t, `It made "Lanes".`, (&ToolExecutionResult{Name: "Lanes"}).Describe())

	var missing *ToolExecutionResult
	assert.Empty(t, missing.Describe())
	assert.Empty(t, missing.IDNote())
	assert.Empty(t, (&ToolExecutionResult{IDs: map[string]string{"id": "x"}}).Describe())
}

// A name is quoted as a person wrote it, not escaped the way Go writes one.
func TestToolExecutionResult_QuotesANameAsWritten(t *testing.T) {
	t.Parallel()

	result := &ToolExecutionResult{Action: "created", Kind: "report", Name: `Peak "PD" \ lanes`}

	assert.Equal(t, `It created the report "Peak "PD" \ lanes".`, result.Describe())
}
