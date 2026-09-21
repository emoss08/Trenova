package agentruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func closedSchema(fields ...string) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		properties[field] = map[string]any{"type": "string"}
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
}

// The schema says additionalProperties: false and the model sends an argument
// the tool never declared anyway — a runId it made up, most often. The tool
// ignores it, but the proposal used to carry it to the card as though it were
// part of the request.
func TestDeclaredArguments_DropsWhatAClosedSchemaNeverDeclared(t *testing.T) {
	t.Parallel()

	kept := declaredArguments(
		closedSchema("subjectId", "attemptSummary"),
		map[string]any{"subjectId": "shp_1", "attemptSummary": "no BOL", "runId": "ar_made_up"},
	)

	assert.Equal(t, map[string]any{"subjectId": "shp_1", "attemptSummary": "no BOL"}, kept)
}

func TestDeclaredArguments_LeavesAnOpenSchemaAlone(t *testing.T) {
	t.Parallel()

	args := map[string]any{"subjectId": "shp_1", "runId": "ar_1"}

	open := closedSchema("subjectId")
	open["additionalProperties"] = true
	assert.Equal(t, args, declaredArguments(open, args))

	unspecified := closedSchema("subjectId")
	delete(unspecified, "additionalProperties")
	assert.Equal(t, args, declaredArguments(unspecified, args))

	assert.Equal(t, args, declaredArguments(map[string]any{"type": "object"}, args),
		"a schema that declares nothing cannot say what is undeclared")
	assert.Nil(t, declaredArguments(closedSchema("subjectId"), nil))
}
