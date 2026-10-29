package agenttrigger_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/stretchr/testify/require"
)

const eventsSource = "events.go"

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

	registered := make(map[agenttrigger.EventKind]agent.SubjectType, len(matches))
	for _, event := range agenttrigger.KnownEvents() {
		registered[event.Kind] = event.SubjectType
	}

	for _, match := range matches {
		kind := agenttrigger.EventKind(match[1])
		subject, ok := registered[kind]
		require.Truef(t, ok, "event kind %q is declared but missing from knownEvents", kind)
		require.Truef(t, subject.IsValid(), "event kind %q resolves an unknown subject", kind)
		require.True(t, kind.IsValid(), kind)
	}
	require.Len(t, registered, len(matches))
}
