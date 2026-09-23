package assistantartifactrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

/*
An upsert names the row it reads its version from.

On a conflict Postgres sees two rows, the one already there and EXCLUDED, so a
bare "version + 1" is ambiguous and the statement fails outright. The recorder
only warns on a failed save, so every artifact a turn produced was dropped and
the Desk's pane stayed empty through the turns that should have filled it.
*/
func TestUpsertBumpsTheStoredRowsVersion(t *testing.T) {
	t.Parallel()

	artifact := &assistantartifact.Artifact{
		ThreadID:         pulid.MustNew("athr_"),
		Kind:             assistantartifact.KindTableView,
		Status:           assistantartifact.StatusReady,
		Title:            "Shipments",
		SourceToolCallID: "call_1",
	}
	target, err := conflictTarget(artifact)
	require.NoError(t, err)

	sql := buildUpsert(bun.NewDB(nil, pgdialect.New()), artifact, target).String()

	assert.Contains(t, sql, "version = aart.version + 1", sql)
	assert.NotContains(t, sql, "version = version + 1", sql)
	assert.Contains(t, sql, "payload = EXCLUDED.payload", sql)
}
