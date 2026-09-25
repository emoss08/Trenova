package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

func TestEveryActionToolPreviewsWhatItWouldDo(t *testing.T) {
	t.Parallel()

	for _, tool := range buildRegistered(t).Actions {
		if tool.Policy().Kind != agent.ToolKindAction {
			continue
		}

		if _, previews := tool.(serviceports.ToolPreviewer); !previews {
			t.Errorf(
				"%s is an action tool without a previewer; a person would approve it blind",
				tool.Name(),
			)
		}
	}
}
