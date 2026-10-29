package benchresult

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaseErrors_NamesTheStepAFireFailedIn(t *testing.T) {
	t.Parallel()

	result := &CaseResult{
		Error: "setup 2: duplicate key value violates unique constraint",
		Steps: []*StepResult{
			{Index: 1, Kind: "fire", Error: "the subject query returned 0 rows, not one"},
			{Index: 2, Kind: "say"},
			{Index: 3, Kind: "decide", Error: "no pending proposal to approve"},
		},
	}

	assert.Equal(t, []string{
		"setup 2: duplicate key value violates unique constraint",
		"step 1: the subject query returned 0 rows, not one",
		"step 3: no pending proposal to approve",
	}, result.Errors())
	assert.Empty(t, (&CaseResult{Steps: []*StepResult{{Index: 1}}}).Errors())
}
