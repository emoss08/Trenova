package agentpagecontext_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func surfaceErrors(surface agentpagecontext.Surface) []string {
	multiErr := errortypes.NewMultiError()
	surface.Validate("surface", multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}

	return fields
}

func TestSurfaceValidateAcceptsTheDeskTheAssistantThePaletteAndNothing(t *testing.T) {
	t.Parallel()

	for _, surface := range []agentpagecontext.Surface{
		"",
		agentpagecontext.SurfaceDesk,
		agentpagecontext.SurfaceAssistant,
		agentpagecontext.SurfacePalette,
	} {
		assert.Empty(t, surfaceErrors(surface), "surface %q", surface)
		assert.True(t, surface.IsValid(), "surface %q", surface)
	}
}

func TestSurfaceValidateRefusesAnythingElseOnTheField(t *testing.T) {
	t.Parallel()

	for _, surface := range []agentpagecontext.Surface{"desk", "DESK", "Inbox", " Desk", "palette"} {
		assert.Equal(t, []string{"surface"}, surfaceErrors(surface), "surface %q", surface)
		assert.False(t, surface.IsValid(), "surface %q", surface)
	}
}
