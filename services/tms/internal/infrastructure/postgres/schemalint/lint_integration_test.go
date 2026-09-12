//go:build integration

package schemalint

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// domainDir is resolved from this file rather than the working directory:
// SetupTestDB chdirs to find the migrations, so a relative path would break
// depending on when it is read.
func domainDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot locate the schemalint source directory")

	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "core", "domain")
}

// knownUnfixed is the escape hatch for a boolean column that defaults to TRUE
// in the schema while its Go field still declares a bun default. It is empty,
// and the intent is that it stays that way.
//
// Adding an entry re-admits a column whose false is silently discarded on
// insert. Fix the field instead: drop `default:` from the tag, and name the
// value at every site that builds the row — a row provisioned from a struct
// that leaves the field alone stores false once the tag is gone, which is how
// a tenant ends up with a safety check quietly switched off.
var knownUnfixed = map[string]struct{}{}

// TestBooleanDefaultsAreNotSubstituted fails when a boolean column that
// defaults to TRUE is backed by a Go field declaring a bun default. See the
// package comment for why that combination silently discards false.
func TestBooleanDefaultsAreNotSubstituted(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	fields, err := BoolFieldsWithDefaultTag(domainDir(t))
	require.NoError(t, err)
	require.NotEmpty(t, fields, "found no bun tags to check - has the domain moved?")

	trueByDefault := columnsDefaultingTrue(t, ctx, db)
	require.NotEmpty(t, trueByDefault, "found no boolean columns - did migrations run?")

	offenders := make([]string, 0)
	stale := make(map[string]struct{}, len(knownUnfixed))
	for key := range knownUnfixed {
		stale[key] = struct{}{}
	}

	for i := range fields {
		field := &fields[i]
		if _, defaultsTrue := trueByDefault[field.Key()]; !defaultsTrue {
			continue
		}
		delete(stale, field.Key())
		if _, allowed := knownUnfixed[field.Key()]; allowed {
			continue
		}
		offenders = append(offenders, fmt.Sprintf("  %s  declared at %s", field.Key(), field.Location()))
	}

	sort.Strings(offenders)
	require.Empty(t, offenders, "these boolean columns default to TRUE and carry a bun default: tag, "+
		"so false is discarded when the row is created. Drop `default:` from the tag and set the "+
		"value where the row is built:\n%s", joinLines(offenders))

	// Keep the backlog honest: an entry whose tag is gone must not linger.
	left := make([]string, 0, len(stale))
	for key := range stale {
		left = append(left, "  "+key)
	}
	sort.Strings(left)
	require.Empty(t, left, "these are listed in knownUnfixed but no longer need to be - "+
		"delete them from the list:\n%s", joinLines(left))
}

func joinLines(lines []string) string {
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}

	return out
}

// columnsDefaultingTrue returns the "table.column" of every boolean column in
// the migrated schema whose column default is TRUE.
func columnsDefaultingTrue(t *testing.T, ctx context.Context, db *bun.DB) map[string]struct{} {
	t.Helper()

	var rows []struct {
		Table  string `bun:"table_name"`
		Column string `bun:"column_name"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("information_schema.columns").
		Column("table_name", "column_name").
		Where("table_schema = ?", "public").
		Where("data_type = ?", "boolean").
		Where("lower(column_default) LIKE ?", "true%").
		Scan(ctx, &rows))

	out := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		out[row.Table+"."+row.Column] = struct{}{}
	}

	return out
}
