package fieldgatetest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/stretchr/testify/require"
)

func ListedRows[T any](t *testing.T, result any) ([]T, []string) {
	t.Helper()

	var outcome querytool.SearchOutcome
	var withheld []string
	switch typed := result.(type) {
	case querytool.SearchOutcome:
		outcome = typed
	case *fieldgate.Outcome:
		outcome = typed.SearchOutcome
		withheld = typed.Withheld
	default:
		require.Failf(t, "unexpected result", "%T", result)
	}

	rows := make([]T, 0, len(outcome.Items.([]any)))
	for _, item := range outcome.Items.([]any) {
		row, ok := item.(T)
		require.True(t, ok, "%T", item)
		rows = append(rows, row)
	}

	return rows, withheld
}
