package agenttoolschema_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateHelpers_DeclareTheirFormatAnExampleAndOneSentence(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]any{
		toolschema.FormatDate:          agenttoolschema.Date("When it starts."),
		toolschema.FormatDateTime:      agenttoolschema.DateTime("When it happened."),
		toolschema.FormatLocalDateTime: agenttoolschema.LocalDateTime("When pickup opens."),
	}

	for name, property := range cases {
		format, ok := toolschema.DateFormatNamed(name)
		require.True(t, ok)
		assert.Equal(t, name, toolschema.FormatOf(property))
		assert.Equal(t, []string{format.Example}, property[toolschema.KeyExamples])
		assert.Contains(t, property[toolschema.KeyDescription], format.Sentence)
		assert.NoError(t, toolschema.Validate(map[string]any{
			toolschema.KeyType:       toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{"at": property},
		}, map[string]any{"at": format.Example}), "its own example fits its format")
	}
}

func TestDate_WithoutADescriptionIsTheSentenceAlone(t *testing.T) {
	t.Parallel()

	format, _ := toolschema.DateFormatNamed(toolschema.FormatDate)

	assert.Equal(t, format.Sentence, agenttoolschema.Date("")[toolschema.KeyDescription])
}
