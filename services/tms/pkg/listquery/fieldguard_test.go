package listquery

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
)

func TestFieldGuard_AWithheldFieldCannotBeResolved(t *testing.T) {
	t.Parallel()

	guarded := WithFieldGuard(t.Context(), func(_, field string) bool { return field == "status" })

	_, resolved := newPlan(guarded, testTrucks.Query(), nil).resolve("status")
	assert.False(t, resolved)

	_, resolved = newPlan(t.Context(), testTrucks.Query(), nil).resolve("status")
	assert.True(t, resolved, "with no guard the field resolves")
}

func TestFieldGuard_AWithheldFieldCannotOrderAKeyset(t *testing.T) {
	t.Parallel()

	guarded := WithFieldGuard(t.Context(), func(_, field string) bool { return field == "status" })

	_, _, err := newPlan(guarded, testTrucks.Query(), nil).keysetOrder([]domaintypes.SortField{
		{Field: "status", Direction: dbtype.SortDirectionAsc},
	})

	assert.Error(t, err)
}
