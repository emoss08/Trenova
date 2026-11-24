package runtimetools

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
)

func unmadeLinks(t Turn, linked []string) []string {
	made := t.Artifacts()
	unmade := make([]string, 0, len(linked))
	for _, id := range linked {
		if !slices.ContainsFunc(made, func(shown agentruntimeports.ShownArtifact) bool {
			return shown.ID.String() == id
		}) {
			unmade = append(unmade, id)
		}
	}

	return unmade
}

func linkRefusal(tool string, unmade []string) toolcall.Result {
	return toolcall.Failed(
		"Tool %q was not run: its words link %s, which no call in this turn made. Words may "+
			"link only an artifact this turn made, as [words](artifact:<id>) with the id its "+
			"result gave; take the link out, or make the artifact first.",
		tool, strings.Join(unmade, ", "),
	)
}
