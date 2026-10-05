package cloud_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/tenantbindlint"
	"github.com/stretchr/testify/require"
)

var tenantBindAllowed = map[string]string{}

func TestCloudHandlersNeverBindATenantFieldFromTheRequest(t *testing.T) {
	t.Parallel()

	violations, err := tenantbindlint.Check(serviceRoot(t), "./internal/cloud/...")
	require.NoError(t, err)

	unexpected := make([]string, 0)
	seen := make(map[string]struct{}, len(violations))
	for _, v := range violations {
		seen[v.Key()] = struct{}{}
		if _, ok := tenantBindAllowed[v.Key()]; ok {
			continue
		}
		unexpected = append(unexpected, v.String())
	}
	require.Empty(t, unexpected, "\n"+strings.Join(unexpected, "\n"))

	stale := make([]string, 0)
	for key := range tenantBindAllowed {
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}
	require.Empty(t, stale, "remove allowlist entries that no longer match a bind site")
}
