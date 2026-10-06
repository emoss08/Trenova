package migrations

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

const upMigrationSuffix = ".up.sql"

var ErrConstraintNotFound = errors.New("no up migration adds the constraint")

func LatestConstraintDefinition(name string) (string, error) {
	entries, err := fs.ReadDir(sqlMigrations, ".")
	if err != nil {
		return "", fmt.Errorf("read embedded migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), upMigrationSuffix) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	quoted := regexp.QuoteMeta(name)
	adds := regexp.MustCompile(
		`(?s)ADD CONSTRAINT "` + quoted + `"(.*?);|CONSTRAINT "` + quoted + `" CHECK(.*?)\)\s*,?\s*\n`,
	)

	var latest []byte
	for _, file := range names {
		body, readErr := fs.ReadFile(sqlMigrations, file)
		if readErr != nil {
			return "", fmt.Errorf("read migration %q: %w", file, readErr)
		}
		if match := adds.Find(body); match != nil {
			latest = match
		}
	}
	if latest == nil {
		return "", fmt.Errorf("%w: %s", ErrConstraintNotFound, name)
	}

	return string(latest), nil
}
