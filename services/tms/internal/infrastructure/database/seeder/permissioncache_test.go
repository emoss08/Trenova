package seeder

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingPermissionCache struct {
	invalidated []pulid.ID
	failOn      pulid.ID
}

func (c *recordingPermissionCache) InvalidateOrganization(_ context.Context, orgID pulid.ID) error {
	if orgID == c.failOn {
		return errors.New("redis unavailable")
	}
	c.invalidated = append(c.invalidated, orgID)
	return nil
}

func TestInvalidatePermissionCachesRefreshesEveryChangedOrganization(t *testing.T) {
	cache := &recordingPermissionCache{}
	orgs := []pulid.ID{pulid.MustNew("org_"), pulid.MustNew("org_")}

	require.NoError(t, invalidatePermissionCaches(t.Context(), cache, orgs))

	assert.Equal(t, orgs, cache.invalidated)
}

func TestInvalidatePermissionCachesStopsAtTheFirstFailure(t *testing.T) {
	failing := pulid.MustNew("org_")
	cache := &recordingPermissionCache{failOn: failing}
	orgs := []pulid.ID{pulid.MustNew("org_"), failing, pulid.MustNew("org_")}

	err := invalidatePermissionCaches(t.Context(), cache, orgs)

	require.ErrorContains(t, err, failing.String())
	assert.Equal(t, orgs[:1], cache.invalidated)
}

func TestInvalidatePermissionCachesWithoutACacheDoesNothing(t *testing.T) {
	require.NoError(t, invalidatePermissionCaches(t.Context(), nil, []pulid.ID{pulid.MustNew("org_")}))
}
