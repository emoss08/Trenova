package rlslint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const helperFixture = `package fixture

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
)

type repository struct {
	db *postgres.Connection
}

func (r *repository) Unscoped(ctx context.Context) (int, error) {
	return r.count(ctx)
}

func (r *repository) Scoped(ctx context.Context) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		return r.count(ctx)
	})
}

func (r *repository) ThroughTwoHelpers(ctx context.Context) (int, error) {
	return r.outer(ctx)
}

func (r *repository) HelperScopesItself(ctx context.Context) (int, error) {
	return r.scopedCount(ctx)
}

func (r *repository) outer(ctx context.Context) (int, error) {
	return r.count(ctx)
}

func (r *repository) count(ctx context.Context) (int, error) {
	return r.db.DBForContext(ctx).NewSelect().Count(ctx)
}

func (r *repository) scopedCount(ctx context.Context) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		return r.db.DBForContext(ctx).NewSelect().Count(ctx)
	})
}
`

func TestScanRepositories_FollowsHelpersThatTouchTheConnection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "fixture")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "repo.go"), []byte(helperFixture), 0o600))

	violations, err := ScanRepositories(dir, root)
	require.NoError(t, err)

	flagged := make([]string, 0, len(violations))
	for _, violation := range violations {
		flagged = append(flagged, violation.Key)
	}
	assert.ElementsMatch(t, []string{
		"fixture/repo.go:repository.Unscoped",
		"fixture/repo.go:repository.ThroughTwoHelpers",
	}, flagged)
}
