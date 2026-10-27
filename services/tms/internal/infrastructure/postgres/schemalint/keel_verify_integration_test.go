//go:build integration

package schemalint

import (
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/stretchr/testify/require"
)

var vectorTables = []string{"ai_embeddings", "ai_catalog_embeddings"}

func TestEveryKeelModelMatchesTheMigratedSchema(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	drifts, err := keel.Verify(ctx, db, keeltables.All())
	require.NoError(t, err)

	vector := postgres.NewTestCapabilityProbe(postgres.NewTestConnection(db)).VectorReady(ctx)
	lines := make([]string, 0, len(drifts))
	for _, d := range drifts {
		if !vector && d.Kind == keel.DriftMissingTable && slices.Contains(vectorTables, d.Table) {
			continue
		}
		lines = append(lines, d.String())
	}
	require.Empty(t, lines, "models and migrations disagree; fix the model's keel tag or "+
		"add a migration, then run task keel-gen:\n%s", strings.Join(lines, "\n"))
}

func TestEveryDeclaredDatabaseFunctionMatchesTheMigratedSchema(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	funcs := append(postgres.RLSFuncs(), listquery.SearchFuncs()...)
	drifts, err := keel.VerifyFuncs(ctx, db, funcs...)
	require.NoError(t, err)

	lines := make([]string, 0, len(drifts))
	for _, d := range drifts {
		lines = append(lines, d.String())
	}
	require.Empty(t, lines, "declared database functions and migrations disagree; fix the "+
		"keel.DeclareFunc declaration or add a migration:\n%s", strings.Join(lines, "\n"))
}
