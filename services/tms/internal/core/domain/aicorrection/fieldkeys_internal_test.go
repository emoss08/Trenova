package aicorrection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveryPredictedFieldHasOneLabel(t *testing.T) {
	t.Parallel()

	assert.Len(t, PredictedFieldKeys, len(predictedFields))
	seen := make(map[string]struct{}, len(predictedFields))
	for _, field := range predictedFields {
		assert.NotContains(t, seen, field.Key, "field key %s is listed twice", field.Key)
		seen[field.Key] = struct{}{}
		assert.NotEmpty(t, field.Label, "field key %s has no label", field.Key)
		assert.Equal(t, field.Label, FieldLabel(field.Key))
	}
}

func TestFieldLabelsFollowTheCanonicalKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Pickup Window", FieldLabel("pickupwindow"))
	assert.Equal(t, "BOL", FieldLabel(" bol "))
	assert.Equal(t, "PO Number", FieldLabel("poNumber"))
	assert.Equal(t, "brokerNotes", FieldLabel(" brokerNotes "))
	assert.Empty(t, FieldLabel(""))
}
