package captureservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paperwork(name string, isDefault bool) ProfileSettings {
	return ProfileSettings{
		Name:              name,
		Status:            capture.ProfileActive,
		IsDefault:         isDefault,
		DPI:               300,
		PixelType:         capture.PixelBlackWhite,
		Duplex:            true,
		UseFeeder:         true,
		DiscardBlankPages: true,
		JPEGQuality:       80,
	}
}

func TestProfileAdminNeedsProfilePermission(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	w.denied[permission.ResourceCaptureProfile.String()+":create"] = true

	_, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   paperwork("Paperwork", false),
	})
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, w.profiles)
}

func TestMakingAProfileTheDefaultTakesTheFlag(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()

	first, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   paperwork("Paperwork", true),
	})
	require.NoError(t, err)

	photos := paperwork("Photos", true)
	photos.PixelType = capture.PixelColor
	second, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   photos,
	})
	require.NoError(t, err)

	retired := paperwork("Legal", false)
	retired.Status = capture.ProfileInactive
	_, err = s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   retired,
	})
	require.NoError(t, err)

	assert.False(t, w.profiles[first.ID].IsDefault)
	assert.True(t, w.profiles[second.ID].IsDefault)

	available, err := s.AvailableProfiles(t.Context(), w.tenant)
	require.NoError(t, err)
	require.Len(t, available, 2, "an inactive profile is not offered")
	assert.Equal(t, second.ID, available[0].ID, "the default comes first")

	all, err := s.ListProfiles(t.Context(), &ListProfilesInput{TenantInfo: w.tenant})
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestAvailableProfilesNeedOnlyCapture(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	w.denied[permission.ResourceCaptureProfile.String()+":read"] = true

	_, err := s.ListProfiles(t.Context(), &ListProfilesInput{TenantInfo: w.tenant})
	assert.True(t, errortypes.IsAuthorizationError(err))

	_, err = s.AvailableProfiles(t.Context(), w.tenant)
	require.NoError(t, err)
}

func TestProfileUpdateRefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()

	created, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   paperwork("Paperwork", false),
	})
	require.NoError(t, err)

	grey := paperwork("Paperwork", false)
	grey.PixelType = capture.PixelGrayscale
	updated, err := s.UpdateProfile(t.Context(), &UpdateProfileInput{
		TenantInfo: w.tenant,
		ProfileID:  created.ID,
		Version:    created.Version,
		Settings:   grey,
	})
	require.NoError(t, err)
	assert.Equal(t, capture.PixelGrayscale, updated.PixelType)

	_, err = s.UpdateProfile(t.Context(), &UpdateProfileInput{
		TenantInfo: w.tenant,
		ProfileID:  created.ID,
		Version:    created.Version,
		Settings:   paperwork("Paperwork", false),
	})
	assert.True(t, errortypes.IsConflictError(err))
	assert.Equal(t, capture.PixelGrayscale, w.profiles[created.ID].PixelType)
}

func TestProfileSettingsAreCheckedAndSeparatorsDeduplicated(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()

	stray := paperwork("Paperwork", false)
	stray.FixedPageCount = 2
	_, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   stray,
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	inactiveDefault := paperwork("Paperwork", true)
	inactiveDefault.Status = capture.ProfileInactive
	_, err = s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   inactiveDefault,
	})
	require.ErrorAs(t, err, &multiErr)

	split := paperwork("  Split by patch  ", false)
	split.SeparatorStrategies = []capture.SeparatorStrategy{
		capture.SeparatorPatchCode,
		capture.SeparatorCoverSheet,
		capture.SeparatorPatchCode,
	}
	created, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   split,
	})
	require.NoError(t, err)
	assert.Equal(t, "Split by patch", created.Name)
	assert.Equal(t, []capture.SeparatorStrategy{
		capture.SeparatorPatchCode,
		capture.SeparatorCoverSheet,
	}, created.SeparatorStrategies)
}

func TestDeletingAProfileOfAnotherTenantFindsNothing(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()

	created, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   paperwork("Paperwork", false),
	})
	require.NoError(t, err)

	other := w.tenant
	other.OrgID = pulid.MustNew("org_")
	err = s.DeleteProfile(t.Context(), other, created.ID)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Contains(t, w.profiles, created.ID)

	require.NoError(t, s.DeleteProfile(t.Context(), w.tenant, created.ID))
	assert.NotContains(t, w.profiles, created.ID)
	assert.Contains(t, w.published, "capture_profile:deleted")
}

func TestRevokingMineRefusesAnotherPersonsDevice(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)

	other := w.tenant
	other.UserID = pulid.MustNew("usr_")
	_, err := s.RevokeDevice(t.Context(), &RevokeDeviceRequest{
		TenantInfo: other,
		DeviceID:   tokens.DeviceID,
		Mine:       true,
	})
	assert.True(t, errortypes.IsNotFoundError(err),
		"somebody else's device reads as missing, even to a person allowed to revoke it")
	assert.Equal(t, capture.DeviceActive, w.devices[tokens.DeviceID].Status)

	device, err := s.RevokeDevice(t.Context(), &RevokeDeviceRequest{
		TenantInfo: w.tenant,
		DeviceID:   tokens.DeviceID,
		Mine:       true,
	})
	require.NoError(t, err)
	assert.Equal(t, capture.DeviceRevoked, device.Status)
}
