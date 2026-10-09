package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func surfaceErrors(surface agent.Surface) []string {
	multiErr := errortypes.NewMultiError()
	surface.Validate("surface", multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}

	return fields
}

func TestSurfaceValidateAcceptsTheDeskTheAssistantAndNothing(t *testing.T) {
	t.Parallel()

	for _, surface := range []agent.Surface{"", agent.SurfaceDesk, agent.SurfaceAssistant} {
		assert.Empty(t, surfaceErrors(surface), "surface %q", surface)
		assert.True(t, surface.IsValid(), "surface %q", surface)
	}
}

func TestSurfaceValidateRefusesAnythingElseOnTheField(t *testing.T) {
	t.Parallel()

	for _, surface := range []agent.Surface{"desk", "DESK", "Inbox", " Desk"} {
		assert.Equal(t, []string{"surface"}, surfaceErrors(surface), "surface %q", surface)
		assert.False(t, surface.IsValid(), "surface %q", surface)
	}
}
