package ai

import (
	"bytes"
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	value, err := parseDate("from", "2026-01-02")
	require.NoError(t, err)
	assert.Equal(t, int64(1767312000), value)

	_, err = parseDate("from", "")
	require.ErrorContains(t, err, "--from is required")

	_, err = parseDate("to", "01/02/2026")
	require.ErrorContains(t, err, "--to must be a date")
}

type pagedOperator struct {
	services.AITrainingExportOperator
	pages [][]repositories.WithdrawnTrainingExample
	calls []pulid.ID
}

func (o *pagedOperator) WithdrawnExamples(
	_ context.Context,
	req repositories.ListWithdrawnTrainingExamplesRequest,
) ([]repositories.WithdrawnTrainingExample, error) {
	o.calls = append(o.calls, req.AfterID)
	if len(o.pages) == 0 {
		return []repositories.WithdrawnTrainingExample{}, nil
	}
	page := o.pages[0]
	o.pages = o.pages[1:]
	return page, nil
}

func TestWriteWithdrawnPagesThroughEveryExample(t *testing.T) {
	t.Parallel()

	full := make([]repositories.WithdrawnTrainingExample, 0, withdrawnPage)
	for i := range withdrawnPage {
		full = append(full, repositories.WithdrawnTrainingExample{
			ID:        pulid.ID("aitr_" + string(rune('a'+i%26))),
			ExampleID: "ex",
			Split:     aitraining.SplitTrain.String(),
		})
	}
	operator := &pagedOperator{pages: [][]repositories.WithdrawnTrainingExample{
		full,
		{{ID: "aitr_last", ExampleID: "last", Split: aitraining.SplitValidation.String()}},
	}}

	var out bytes.Buffer
	count, err := writeWithdrawn(t.Context(), operator, "aitx_1", &out)
	require.NoError(t, err)

	assert.Equal(t, withdrawnPage+1, count)
	assert.Len(t, operator.calls, 2)
	assert.Equal(t, full[len(full)-1].ID, operator.calls[1])
	assert.Contains(t, out.String(), "last\tvalidation\n")
}

func TestExportIDRejectsGarbage(t *testing.T) {
	t.Parallel()

	_, err := exportID("not an id")
	require.Error(t, err)
}
