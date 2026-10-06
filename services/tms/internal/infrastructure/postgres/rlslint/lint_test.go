package rlslint

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const editionDir = "internal/cloud/"

func serviceRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
}

func TestEverySystemScopeIsReviewed(t *testing.T) {
	t.Parallel()

	sites, err := Scan(serviceRoot(t), "internal", "pkg", "cmd")
	require.NoError(t, err)

	seen := make(map[string]struct{}, len(sites))
	unreviewed := make([]string, 0)
	unexplained := make([]string, 0)
	for _, site := range sites {
		if strings.HasPrefix(site.Key, editionDir) {
			continue
		}
		seen[site.Key] = struct{}{}
		if _, ok := allowed[site.Key]; !ok {
			unreviewed = append(unreviewed, site.Position+" ("+site.Key+")")
		}
		if site.Reason == "" {
			unexplained = append(unexplained, site.Position)
		}
	}

	stale := make([]string, 0)
	for key, why := range allowed {
		if strings.TrimSpace(why) == "" {
			unexplained = append(unexplained, key+" (allowlist entry)")
		}
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}

	assert.Empty(t, unreviewed,
		"dbscope.WithSystem bypasses tenant isolation; add each new caller to allowed in "+
			"allowlist_test.go with why it must see every tenant:\n%s", strings.Join(unreviewed, "\n"))
	assert.Empty(t, unexplained,
		"dbscope.WithSystem needs a non-empty string literal or package constant as its reason:\n%s",
		strings.Join(unexplained, "\n"))
	assert.Empty(t, stale, "allowlist entries with no remaining caller:\n%s", strings.Join(stale, "\n"))
}

func TestRepositoryMethodsRunInScopedTransactions(t *testing.T) {
	t.Parallel()

	root := serviceRoot(t)
	violations, err := ScanRepositories(
		filepath.Join(root, "internal", "infrastructure", "postgres", "repositories"),
		root,
	)
	require.NoError(t, err)

	unexpected := make([]string, 0)
	seen := make(map[string]struct{}, len(violations))
	for _, v := range violations {
		seen[v.Key] = struct{}{}
		if _, ok := rawPoolAllowed[v.Key]; ok {
			continue
		}
		unexpected = append(unexpected, v.Position+" "+v.Key+": "+v.Problem)
	}

	stale := make([]string, 0)
	for key := range rawPoolAllowed {
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}

	assert.Empty(t, unexpected,
		"repository methods must run their statements in a tenant-scoped transaction "+
			"(see docs/engineering/row-level-security.md):\n%s", strings.Join(unexpected, "\n"))
	assert.Empty(t, stale, "rawPoolAllowed entries with no remaining violation:\n%s", strings.Join(stale, "\n"))
}
