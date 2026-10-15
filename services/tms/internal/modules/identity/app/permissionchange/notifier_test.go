package permissionchange

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingRealtime struct {
	published []*agentports.PublishResourceInvalidationRequest
}

func (r *recordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *agentports.PublishResourceInvalidationRequest,
) error {
	r.published = append(r.published, req)
	return nil
}

func newNotifier(t *testing.T) (
	*Notifier,
	*permissionportsmock.MockPermissionCacheRepository,
	*recordingRealtime,
) {
	t.Helper()

	cache := permissionportsmock.NewMockPermissionCacheRepository(t)
	realtime := &recordingRealtime{}
	n := New(Params{Cache: cache, Realtime: realtime, Logger: zap.NewNop()}).(*Notifier)
	return n, cache, realtime
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestOrganizationChanged_AdvancesTheEpochAndTellsEveryone(t *testing.T) {
	t.Parallel()

	n, cache, realtime := newNotifier(t)
	tn := tenant()
	cache.EXPECT().InvalidateOrganization(t.Context(), tn.OrgID).Return(nil).Once()

	n.OrganizationChanged(t.Context(), tn)

	require.Len(t, realtime.published, 1)
	event := realtime.published[0]
	assert.Equal(t, Resource, event.Resource)
	assert.Equal(t, ActionChanged, event.Action)
	assert.Equal(t, tn.OrgID, event.OrganizationID)
	assert.Equal(t, tn.BuID, event.BusinessUnitID)
	assert.True(t, event.AudienceUserID.IsNil())
}

func TestOrganizationChanged_StillAnnouncesWhenTheCacheIsDown(t *testing.T) {
	t.Parallel()

	n, cache, realtime := newNotifier(t)
	tn := tenant()
	cache.EXPECT().InvalidateOrganization(t.Context(), tn.OrgID).
		Return(errors.New("redis unavailable")).Once()

	n.OrganizationChanged(t.Context(), tn)

	assert.Len(t, realtime.published, 1)
}

func TestUserChanged_DropsTheUsersEntryAndTellsOnlyThem(t *testing.T) {
	t.Parallel()

	n, cache, realtime := newNotifier(t)
	tn := tenant()
	userID := pulid.MustNew("usr_")
	cache.EXPECT().Delete(t.Context(), userID, tn.OrgID).Return(nil).Once()

	n.UserChanged(t.Context(), tn, userID)

	require.Len(t, realtime.published, 1)
	assert.Equal(t, userID, realtime.published[0].AudienceUserID)
}

func TestAnnounce_NeedsAFullTenant(t *testing.T) {
	t.Parallel()

	n, cache, realtime := newNotifier(t)
	orgID := pulid.MustNew("org_")
	cache.EXPECT().InvalidateOrganization(t.Context(), orgID).Return(nil).Once()

	n.OrganizationChanged(t.Context(), pagination.TenantInfo{OrgID: orgID})

	assert.Empty(t, realtime.published)
}
