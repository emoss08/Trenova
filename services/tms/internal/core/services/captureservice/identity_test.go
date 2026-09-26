package captureservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescribeDeviceNamesItsPersonAndOrganization(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	w.control.CaptureMinAgentVersion = "1.2.0"
	w.control.CaptureAllowAutoUpdate = false

	identity, err := s.DescribeDevice(t.Context(), principal)
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", identity.Updates.MinimumVersion)
	assert.False(t, identity.Updates.AllowAutoUpdate)
	assert.Equal(t, "Jordan Doe", identity.Person.Name)
	assert.Equal(t, "jordan@carrier.test", identity.Person.EmailAddress)
	assert.Equal(t, "Acme Freight", identity.Organization.Name)

	raw, err := sonic.Marshal(identity)
	require.NoError(t, err)
	body := map[string]any{}
	require.NoError(t, sonic.Unmarshal(raw, &body))
	device := body["device"].(map[string]any)
	assert.Equal(t, principal.Device.ID.String(), device["id"])
	assert.Equal(t, "DISPATCH-07", device["machineName"])
	assert.NotContains(t, device, "refreshTokenHash")
	assert.NotContains(t, device, "accessTokenHash")
	assert.Equal(t, "Jordan Doe", body["person"].(map[string]any)["name"])
	assert.Equal(t, map[string]any{"minimumVersion": "1.2.0", "allowAutoUpdate": false}, body["updates"])
}

func TestDeviceProfilesFollowTheCapturePermission(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))

	_, err := s.CreateProfile(t.Context(), &CreateProfileInput{
		TenantInfo: w.tenant,
		Settings:   paperwork("Paperwork", true),
	})
	require.NoError(t, err)

	profiles, err := s.DeviceProfiles(t.Context(), principal)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.True(t, profiles[0].IsDefault)

	w.denied[permission.ResourceCaptureBatch.String()+":create"] = true
	_, err = s.DeviceProfiles(t.Context(), principal)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestDeviceProfilesRefusedWhileCaptureIsOff(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	w.control.EnableCapture = false

	_, err := s.DeviceProfiles(t.Context(), principal)
	require.ErrorIs(t, err, ErrCaptureDisabled)

	var businessErr *errortypes.BusinessError
	require.ErrorAs(t, err, &businessErr)
	assert.Equal(t, DisabledReason, businessErr.Params[DisabledReasonParam])
}
