package aiaudit_test

import (
	"testing"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit/aiauditdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIAuditEvent_StoresAZeroCostAsZero(t *testing.T) {
	t.Parallel()

	table := aiauditdb.AIAuditEventsSchema()
	index := table.Column("cost_usd")
	require.GreaterOrEqual(t, index, int32(0))
	assert.False(t, table.Columns[index].Flags.Has(schema.FlagNullZero),
		"nullzero writes a zero cost as NULL, which no longer hashes as it was sealed")
}
