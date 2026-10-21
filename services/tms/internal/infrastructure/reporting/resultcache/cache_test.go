package resultcache

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scoped(ctx context.Context, grants readscope.Grants) context.Context {
	return readscope.With(ctx, func(context.Context) (readscope.Grants, error) {
		return grants, nil
	})
}

func TestKey_DiffersByTheReadScopeTheQueryRunsUnder(t *testing.T) {
	t.Parallel()

	cache := &Cache{}
	compiled := &reportports.CompiledReportQuery{SQL: "SELECT 1", Args: []any{"x"}}
	orgID := pulid.MustNew("org_")
	team := []pulid.ID{pulid.MustNew("usr_")}

	whole, err := cache.Key(scoped(t.Context(), readscope.Grants{}), compiled, report.FormatCSV, orgID)
	require.NoError(t, err)
	unscoped, err := cache.Key(t.Context(), compiled, report.FormatCSV, orgID)
	require.NoError(t, err)
	own, err := cache.Key(scoped(t.Context(), readscope.Grants{
		"shipment": {Scope: readscope.ScopeOwn},
	}), compiled, report.FormatCSV, orgID)
	require.NoError(t, err)
	teamKey, err := cache.Key(scoped(t.Context(), readscope.Grants{
		"worker": {Scope: readscope.ScopeTeam, Team: team},
	}), compiled, report.FormatCSV, orgID)
	require.NoError(t, err)
	sameTeam, err := cache.Key(scoped(t.Context(), readscope.Grants{
		"worker": {Scope: readscope.ScopeTeam, Team: team},
	}), compiled, report.FormatCSV, orgID)
	require.NoError(t, err)

	assert.Equal(t, whole, unscoped, "an unnarrowed scope reads what no scope reads")
	assert.NotEqual(t, whole, own)
	assert.NotEqual(t, whole, teamKey)
	assert.NotEqual(t, own, teamKey)
	assert.Equal(t, teamKey, sameTeam)
}

func TestKey_FailsWhenTheReadScopeCannotBeResolved(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return nil, errors.New("engine down")
	})

	_, err := (&Cache{}).Key(ctx, &reportports.CompiledReportQuery{SQL: "SELECT 1"},
		report.FormatCSV, pulid.MustNew("org_"))
	require.Error(t, err)
}
