//go:build integration

package schemalint

import (
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/require"
)

func TestEveryKeelModelMatchesTheMigratedSchema(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	drifts, err := keel.Verify(ctx, db, keeltables.All())
	require.NoError(t, err)

	lines := make([]string, 0, len(drifts))
	for _, d := range drifts {
		lines = append(lines, d.String())
	}
	require.Empty(t, drifts, "models and migrations disagree; fix the model's keel tag or "+
		"add a migration, then run task keel-gen:\n%s", strings.Join(lines, "\n"))
}
