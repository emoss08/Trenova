package readscopetest

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerChildrenFollowTheirWorker(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../20261231024180_read_scope_worker_children.sql")
	require.NoError(t, err)

	children := []string{
		"worker_credentials",
		"worker_profiles",
		"worker_checklists",
		"worker_training_records",
	}

	policies := strings.Split(string(raw), "CREATE POLICY read_scope ON ")
	require.Len(t, policies, len(children)+1)

	for i, table := range children {
		policy := policies[i+1]
		assert.True(
			t,
			strings.HasPrefix(policy, "public."+table+" AS RESTRICTIVE FOR SELECT"),
			table,
		)
		assert.Contains(t, policy, "trenova_rls.read_scope('worker')) <> 'team'", table)
		assert.Contains(t, policy, "FROM public.workers w", table)
		for _, column := range []string{"worker_id", "organization_id", "business_unit_id"} {
			assert.Contains(t, policy, table+"."+column, table)
		}
	}
}
