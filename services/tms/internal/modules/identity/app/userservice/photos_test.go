package userservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestURLsByUser_ReadsTheKeysOnceAndSignsOnlyUsersWithAPicture(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	withPicture, without := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	key := tenantInfo.OrgID.String() + "/user/profile-picture/a.jpg"
	thumbnail := tenantInfo.OrgID.String() + "/user/profile-picture/b.jpg"
	deps.repo.EXPECT().
		GetPhotoKeys(mock.Anything, mock.MatchedBy(func(req tenantports.GetUsersByIDsRequest) bool {
			return len(req.UserIDs) == 2 && req.TenantInfo.OrgID == tenantInfo.OrgID
		})).
		Return([]tenantports.UserPhotoKeys{
			{UserID: withPicture, Key: key, ThumbnailKey: thumbnail},
		}, nil).
		Once()

	photos, err := deps.svc.URLsByUser(
		t.Context(),
		tenantInfo,
		[]pulid.ID{withPicture, without, withPicture, pulid.Nil},
	)

	require.NoError(t, err)
	require.Len(t, photos, 1)
	assert.Contains(t, photos[withPicture].URL, key)
	assert.Contains(t, photos[withPicture].ThumbnailURL, thumbnail)
	assert.Nil(t, photos[without])
}

func TestURLsByUser_NoUsersReadsNothing(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	photos, err := deps.svc.URLsByUser(t.Context(), pagination.TenantInfo{}, nil)

	require.NoError(t, err)
	assert.Empty(t, photos)
	deps.repo.AssertNotCalled(t, "GetPhotoKeys", mock.Anything, mock.Anything)
}
