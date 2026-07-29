/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var migrationFileRE = regexp.MustCompile(`^(\d{14})_([0-9a-z_\-]+)\.(?:tx\.)?(up|down)\.sql$`)

type migrationFile struct {
	version   string
	comment   string
	direction string
	name      string
}

func embeddedMigrationFiles(t *testing.T) []migrationFile {
	t.Helper()

	entries, err := fs.ReadDir(sqlMigrations, ".")
	require.NoError(t, err, "read embedded migrations")

	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		matches := migrationFileRE.FindStringSubmatch(name)
		require.NotNil(
			t,
			matches,
			"migration %q does not match <14-digit version>_<name>[.tx].(up|down).sql",
			name,
		)

		files = append(files, migrationFile{
			version:   matches[1],
			comment:   matches[2],
			direction: matches[3],
			name:      name,
		})
	}

	return files
}

// TestMigrationVersionsAreUnique guards against the silent-skip failure mode in
// bun's discovery: migrations are keyed only by their numeric prefix, and two
// files sharing a prefix collapse into one migration where the last file walked
// wins. The losing migration never runs and is never reported as pending, so the
// schema drifts from the models with no error anywhere.
func TestMigrationVersionsAreUnique(t *testing.T) {
	t.Parallel()

	owners := make(map[string]migrationFile)
	for _, file := range embeddedMigrationFiles(t) {
		key := file.version + "." + file.direction
		if existing, ok := owners[key]; ok {
			t.Errorf(
				"migrations %q and %q share version %s; bun keys migrations by version alone, so only one of them would ever run",
				existing.name,
				file.name,
				file.version,
			)
			continue
		}
		owners[key] = file
	}
}

// TestMigrationsHaveMatchingDirections keeps every migration reversible so
// migrator.Reset and a rollback of the latest group behave predictably.
func TestMigrationsHaveMatchingDirections(t *testing.T) {
	t.Parallel()

	directions := make(map[string]map[string]string)
	for _, file := range embeddedMigrationFiles(t) {
		key := file.version + "_" + file.comment
		if _, ok := directions[key]; !ok {
			directions[key] = make(map[string]string, 2)
		}
		directions[key][file.direction] = file.name
	}

	for key, found := range directions {
		require.Containsf(t, found, "up", "migration %q has no .up.sql", key)
		require.Containsf(t, found, "down", "migration %q has no .down.sql", key)

		up := strings.HasSuffix(found["up"], ".tx.up.sql")
		down := strings.HasSuffix(found["down"], ".tx.down.sql")
		require.Equalf(
			t,
			up,
			down,
			"migration %q mixes transactional and non-transactional files (%s, %s)",
			key,
			found["up"],
			found["down"],
		)
	}
}

// TestDiscoverYieldsEveryMigration asserts the count bun actually discovers
// matches the number of distinct migrations on disk, so a discovery-level merge
// fails the build rather than a database.
func TestDiscoverYieldsEveryMigration(t *testing.T) {
	t.Parallel()

	versions := make(map[string]struct{})
	for _, file := range embeddedMigrationFiles(t) {
		versions[file.version] = struct{}{}
	}

	require.Len(t, Setup().Sorted(), len(versions))
}
