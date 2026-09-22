package agent_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/require"
)

/*
The registries here are slices and switches rather than anything the compiler
checks, and both fail quietly: an EventKind missing from knownEvents reports
itself invalid and resolves no subject, and a SubjectType missing from the
Postgres enum only fails at the insert, in production. These tests read the
declarations out of the source so a constant added without its registration is
caught here rather than by a run that never wakes.
*/

const (
	eventsSource   = "events.go"
	enumsSource    = "enums.go"
	migrationsPath = "../../../infrastructure/postgres/migrations"
	subjectEnum    = "agent_subject_type_enum"
)

func readSource(t *testing.T, name string) string {
	t.Helper()

	contents, err := os.ReadFile(name)
	require.NoError(t, err)

	return string(contents)
}

func TestEveryDeclaredEventKindIsRegistered(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`Event\w+\s+=\s+EventKind\("([^"]+)"\)`)
	matches := pattern.FindAllStringSubmatch(readSource(t, eventsSource), -1)
	require.NotEmpty(t, matches)

	registered := make(map[agent.EventKind]agent.SubjectType, len(matches))
	for _, event := range agent.KnownEvents() {
		registered[event.Kind] = event.SubjectType
	}

	for _, match := range matches {
		kind := agent.EventKind(match[1])
		subject, ok := registered[kind]
		require.Truef(t, ok, "event kind %q is declared but missing from knownEvents", kind)
		require.Truef(t, subject.IsValid(), "event kind %q resolves an unknown subject", kind)
		require.True(t, kind.IsValid(), kind)
	}
	require.Len(t, registered, len(matches))
}

func declaredSubjectTypes(t *testing.T) []agent.SubjectType {
	t.Helper()

	pattern := regexp.MustCompile(`Subject\w+\s+=\s+SubjectType\("([^"]+)"\)`)
	matches := pattern.FindAllStringSubmatch(readSource(t, enumsSource), -1)
	require.NotEmpty(t, matches)

	out := make([]agent.SubjectType, 0, len(matches))
	for _, match := range matches {
		out = append(out, agent.SubjectType(match[1]))
	}

	return out
}

func TestEveryDeclaredSubjectTypeIsListedAndValid(t *testing.T) {
	t.Parallel()

	listed := make(map[agent.SubjectType]struct{}, len(agent.AllSubjectTypes()))
	for _, subject := range agent.AllSubjectTypes() {
		listed[subject] = struct{}{}
	}

	declared := declaredSubjectTypes(t)
	for _, subject := range declared {
		require.Truef(t, subject.IsValid(), "subject type %q is declared but not valid", subject)
		_, ok := listed[subject]
		require.Truef(t, ok, "subject type %q is missing from AllSubjectTypes", subject)
	}
	require.Len(t, listed, len(declared))
}

// The column is a Postgres enum, so a subject type without a migration is a
// row the database refuses the first time an agent runs on it.
func TestEverySubjectTypeReachesThePostgresEnum(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(migrationsPath)
	require.NoError(t, err)

	var corpus strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		contents, readErr := os.ReadFile(filepath.Join(migrationsPath, entry.Name()))
		require.NoError(t, readErr)
		if strings.Contains(string(contents), subjectEnum) {
			corpus.Write(contents)
		}
	}
	require.NotEmpty(t, corpus.String())

	// The assertion is on membership rather than on the corpus, so a failure
	// names the missing value instead of printing every agent migration.
	sql := corpus.String()
	for _, subject := range declaredSubjectTypes(t) {
		require.Truef(t, strings.Contains(sql, "'"+string(subject)+"'"),
			"subject type %q has no value in %s; add an enum migration for it",
			subject, subjectEnum,
		)
	}
}
