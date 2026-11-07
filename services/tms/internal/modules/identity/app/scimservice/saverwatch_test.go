package scimservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newWatch(repo *fakeRepo, requester *fakeRequester) *SaverWatch {
	return NewSaverWatch(SaverWatchParams{Logger: zap.NewNop(), Repo: repo, Requester: requester})
}

func TestAPersonsPermissionChangeRechecksTheGrantsTheirMappingsMade(t *testing.T) {
	t.Parallel()

	directory := scimports.PendingDirectory{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		DirectoryID:    pulid.MustNew("scimd_"),
	}
	repo := &fakeRepo{marked: []scimports.PendingDirectory{directory}}
	requester := &fakeRequester{}
	saver := pulid.MustNew("usr_")

	newWatch(repo, requester).PermissionsChanged(
		t.Context(),
		pagination.TenantInfo{OrgID: directory.OrganizationID, BuID: directory.BusinessUnitID},
		saver,
	)

	require.Len(t, repo.filters, 1)
	assert.Equal(t, scimports.MappingFilter{
		OrganizationID: directory.OrganizationID,
		SavedByID:      saver,
	}, repo.filters[0])
	assert.Equal(t, []pulid.ID{directory.DirectoryID}, requester.requested)
}

func TestAnOrganizationWidePermissionChangeRechecksEveryMappingThere(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	orgID := pulid.MustNew("org_")

	newWatch(repo, &fakeRequester{}).PermissionsChanged(
		t.Context(),
		pagination.TenantInfo{OrgID: orgID},
		pulid.Nil,
	)

	require.Len(t, repo.filters, 1)
	assert.Equal(t, scimports.MappingFilter{OrganizationID: orgID}, repo.filters[0])
}

func TestLosingAccessRechecksTheGrantsAPersonsMappingsMade(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	requester := &fakeRequester{}
	saver := pulid.MustNew("usr_")

	require.NoError(t, newWatch(repo, requester).RevokeUserAccess(
		t.Context(),
		tenantports.RevokeUserAccessRequest{UserID: saver},
	))

	require.Len(t, repo.filters, 1)
	assert.Equal(t, scimports.MappingFilter{SavedByID: saver}, repo.filters[0],
		"access lost everywhere rechecks the person's mappings in every organization")
	assert.Empty(t, requester.requested, "nothing was marked, so nothing is reconciled")
}

func TestTheSweepRechecksGrantsWhoseSaversRoleJustExpired(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.svc.RequestPendingReconciles(t.Context())

	require.NoError(t, err)
	require.Len(t, h.repo.expired, 1)
	window := h.repo.expired[0]
	assert.Equal(t, int64(ExpiredSaverWindow.Seconds()), window[1]-window[0])
}
