package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/stretchr/testify/assert"
)

func TestAContinuationKeepsTheNameOfTheScriptItContinues(t *testing.T) {
	t.Parallel()

	earlier := map[string]any{
		codemode.ParamScriptTitle:   "Lane performance analysis",
		codemode.ParamScriptPurpose: "Totals and the weakest lanes.",
		codemode.ParamScriptCode:    "total = len(lanes)",
	}
	sent := map[string]any{
		codemode.ParamScriptContinues: "call_printed",
		codemode.ParamScriptCode:      "show_figure('Lanes', total)",
	}

	inherited := inheritScriptName(sent, earlier)
	assert.Equal(t, "Lane performance analysis", inherited[codemode.ParamScriptTitle])
	assert.Equal(t, "Totals and the weakest lanes.", inherited[codemode.ParamScriptPurpose])
	assert.NotContains(t, sent, codemode.ParamScriptTitle, "the call as sent is left alone")

	sent[codemode.ParamScriptTitle] = "Top lanes"
	assert.Equal(t, "Top lanes", inheritScriptName(sent, earlier)[codemode.ParamScriptTitle],
		"a title of its own is kept")
	assert.Equal(t, sent, inheritScriptName(sent, nil), "a script of its own is untouched")
}
