package tenantbindlint

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	handlerViolations    []Violation
	handlerViolationsErr error
	handlerViolationOnce sync.Once
)

func violationsInHandlers(t *testing.T) []Violation {
	t.Helper()

	handlerViolationOnce.Do(func() {
		handlerViolations, handlerViolationsErr = Check("../../..", "./internal/api/handlers/...")
	})
	require.NoError(t, handlerViolationsErr)

	return handlerViolations
}

func TestHandlersNeverBindATenantFieldFromTheRequest(t *testing.T) {
	unexpected := make([]string, 0)
	for _, v := range violationsInHandlers(t) {
		if _, ok := allowed[v.Key()]; ok {
			continue
		}
		unexpected = append(unexpected, v.String())
	}

	require.Empty(t, unexpected, "\n"+strings.Join(unexpected, "\n"))
}

func TestAllowlistHasNoStaleEntries(t *testing.T) {
	seen := make(map[string]struct{})
	for _, v := range violationsInHandlers(t) {
		seen[v.Key()] = struct{}{}
	}

	stale := make([]string, 0)
	for key := range allowed {
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}

	require.Empty(t, stale, "remove allowlist entries that no longer match a bind site")
}
