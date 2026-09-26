/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package migrations

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEveryMigrationVersionNamesOneMigration(t *testing.T) {
	commentsByVersion := make(map[string]map[string]struct{})
	for _, file := range embeddedMigrationFiles(t) {
		comments, ok := commentsByVersion[file.version]
		if !ok {
			comments = make(map[string]struct{}, 1)
			commentsByVersion[file.version] = comments
		}
		comments[file.comment] = struct{}{}
	}

	clashes := make([]string, 0)
	for version, comments := range commentsByVersion {
		if len(comments) < 2 {
			continue
		}
		names := make([]string, 0, len(comments))
		for comment := range comments {
			names = append(names, comment)
		}
		sort.Strings(names)
		clashes = append(clashes, fmt.Sprintf("  %s: %v", version, names))
	}
	sort.Strings(clashes)

	require.Empty(
		t,
		clashes,
		"bun applies migrations by version, so two files with the same version "+
			"are one migration to it and the second is silently skipped. "+
			"Renumber one of each pair:\n%s",
		joinLines(clashes),
	)
}

func joinLines(lines []string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	return out
}
