package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestPrepareDryRunRefusesWhatCannotBeTried(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	for name, req := range map[string]*DryRunRequest{
		"empty":    {Definition: &agentdefinition.Definition{}, Prompt: "  "},
		"too long": {Definition: &agentdefinition.Definition{}, Prompt: strings.Repeat("a", agentdefinition.MaxTestPromptRunes+1)},
		"no agent": {Prompt: "Which invoices are on hold?"},
	} {
		_, err := svc.PrepareDryRun(t.Context(), req, &services.RequestActor{})

		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr, name)
	}
}
