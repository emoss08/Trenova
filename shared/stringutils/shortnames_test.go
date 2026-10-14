package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
)

func TestDistinctShortNamesKeepsTheLastWordWhenItIsUnique(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"Bedrock", "Ollama", "Groq", "Gemini"},
		stringutils.DistinctShortNames(
			[]string{"AWS Bedrock", "Local Ollama", "Groq", "Google Gemini"},
		),
	)
}

func TestDistinctShortNamesFallsBackToTheFirstWordWhenTwoShareTheLast(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"Workstation", "Office", "Groq"},
		stringutils.DistinctShortNames([]string{"Workstation vLLM", "Office vLLM", "Groq"}),
	)
}

func TestDistinctShortNamesYieldsToASingleWordName(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"Azure", "OpenAI"},
		stringutils.DistinctShortNames([]string{"Azure OpenAI", "OpenAI"}),
	)
}

func TestDistinctShortNamesUsesTheWholeNameWhenNoWordIsDistinct(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"Prod vLLM", "Prod vllm"},
		stringutils.DistinctShortNames([]string{"Prod vLLM", "Prod vllm"}),
	)
}

func TestDistinctShortNamesOfNothingIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, stringutils.DistinctShortNames(nil))
}
