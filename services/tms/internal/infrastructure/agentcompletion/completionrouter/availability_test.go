package completionrouter

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func availabilityTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestNovaReady_ReportsARepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	svc.Repo = &fakeRepo{Err: errors.New("database unavailable")}

	_, err := svc.NovaReady(t.Context(), availabilityTenant())
	require.Error(t, err, "a failure to look is not the same as nothing found")
}
