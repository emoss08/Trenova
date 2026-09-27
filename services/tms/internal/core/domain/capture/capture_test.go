package capture_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func errorFields(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestRequestStatusCanMoveTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from capture.RequestStatus
		to   capture.RequestStatus
		want bool
	}{
		{capture.RequestPending, capture.RequestDelivered, true},
		{capture.RequestPending, capture.RequestInProgress, true},
		{capture.RequestDelivered, capture.RequestInProgress, true},
		{capture.RequestInProgress, capture.RequestCompleted, true},
		{capture.RequestDelivered, capture.RequestCompleted, true},
		{capture.RequestPending, capture.RequestCompleted, false},
		{capture.RequestInProgress, capture.RequestDelivered, false},
		{capture.RequestDelivered, capture.RequestPending, false},
		{capture.RequestPending, capture.RequestCanceled, true},
		{capture.RequestInProgress, capture.RequestFailed, true},
		{capture.RequestCanceled, capture.RequestInProgress, false},
		{capture.RequestCompleted, capture.RequestFailed, false},
		{capture.RequestExpired, capture.RequestDelivered, false},
		{capture.RequestPending, capture.RequestPending, false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.from.CanMoveTo(tt.to), "%s -> %s", tt.from, tt.to)
	}
}

func TestRequestTransitionStampsTimes(t *testing.T) {
	t.Parallel()

	req := &capture.CaptureRequest{Status: capture.RequestPending}
	require.True(t, req.Transition(capture.RequestDelivered, 100))
	require.NotNil(t, req.DeliveredAt)
	assert.Equal(t, int64(100), *req.DeliveredAt)

	require.True(t, req.Transition(capture.RequestCompleted, 200))
	require.NotNil(t, req.CompletedAt)
	assert.Equal(t, int64(200), *req.CompletedAt)

	assert.False(t, req.Transition(capture.RequestFailed, 300),
		"a finished request cannot be failed by a late report")
	assert.Equal(t, capture.RequestCompleted, req.Status)
}

func TestRequestExpiryOnlyWhileWaiting(t *testing.T) {
	t.Parallel()

	req := &capture.CaptureRequest{Status: capture.RequestPending, ExpiresAt: 100}
	assert.False(t, req.IsExpired(99))
	assert.True(t, req.IsExpired(100))

	req.Status = capture.RequestInProgress
	assert.False(t, req.IsExpired(10_000), "a scan in progress is not expired by the clock")
}

func TestRequestValidate(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("cprf_")
	req := &capture.CaptureRequest{
		UserID:     pulid.MustNew("usr_"),
		DeviceID:   pulid.MustNew("cdev_"),
		Mode:       capture.RequestModePrint,
		Status:     capture.RequestPending,
		TargetType: "invoice",
		TargetID:   pulid.MustNew("inv_"),
		ProfileID:  &profileID,
	}

	multiErr := errortypes.NewMultiError()
	req.Validate(multiErr)

	assert.ElementsMatch(t, []string{"targetType", "profileId"}, errorFields(multiErr))
}

func TestTargetValidate(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	docType := pulid.MustNew("dt_")

	tests := []struct {
		name     string
		target   capture.Target
		required bool
		fields   []string
	}{
		{name: "empty optional", target: capture.Target{}},
		{name: "empty required", target: capture.Target{}, required: true, fields: []string{"targetType"}},
		{
			name:   "document type without record",
			target: capture.Target{DocumentTypeID: &docType},
			fields: []string{"targetType"},
		},
		{name: "kind without id", target: capture.Target{ResourceType: "shipment"}, fields: []string{"targetId"}},
		{name: "id without kind", target: capture.Target{ResourceID: &id}, fields: []string{"targetType"}},
		{
			name:   "unfileable kind",
			target: capture.Target{ResourceType: "assistant_thread", ResourceID: &id},
			fields: []string{"targetType"},
		},
		{
			name:     "shipment",
			target:   capture.Target{ResourceType: "shipment", ResourceID: &id, DocumentTypeID: &docType},
			required: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.target.Validate(multiErr, capture.TargetFields{Type: "targetType", ID: "targetId"}, tt.required)
			assert.ElementsMatch(t, tt.fields, errorFields(multiErr))
		})
	}
}

func TestFileableResourcesAreRealResources(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	for _, resource := range capture.FileableResources() {
		_, ok := registry.Get(resource.String())
		assert.True(t, ok, "%s is fileable but not a registered resource", resource)
	}
}

func TestProfileValidate(t *testing.T) {
	t.Parallel()

	valid := func() *capture.CaptureProfile {
		profile := &capture.CaptureProfile{Name: "PODs"}
		profile.ApplyDefaults()

		return profile
	}

	multiErr := errortypes.NewMultiError()
	valid().Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "defaults must be a valid profile")

	fixed := valid()
	fixed.SeparatorStrategies = []capture.SeparatorStrategy{capture.SeparatorFixedPageCount}
	multiErr = errortypes.NewMultiError()
	fixed.Validate(multiErr)
	assert.Equal(t, []string{"fixedPageCount"}, errorFields(multiErr))

	stray := valid()
	stray.FixedPageCount = 3
	multiErr = errortypes.NewMultiError()
	stray.Validate(multiErr)
	assert.Equal(t, []string{"fixedPageCount"}, errorFields(multiErr))

	inactiveDefault := valid()
	inactiveDefault.IsDefault = true
	inactiveDefault.Status = capture.ProfileInactive
	multiErr = errortypes.NewMultiError()
	inactiveDefault.Validate(multiErr)
	assert.Equal(t, []string{"isDefault"}, errorFields(multiErr))

	bad := valid()
	bad.DPI = 1200
	bad.JPEGQuality = 10
	bad.SeparatorStrategies = []capture.SeparatorStrategy{"Staple"}
	multiErr = errortypes.NewMultiError()
	bad.Validate(multiErr)
	assert.ElementsMatch(t, []string{"dpi", "jpegQuality", "separatorStrategies"}, errorFields(multiErr))
}

func TestBatchSettleFiling(t *testing.T) {
	t.Parallel()

	batch := &capture.CaptureBatch{Status: capture.BatchReady, ItemCount: 3}
	batch.SettleFiling()
	assert.Equal(t, capture.BatchReady, batch.Status)

	batch.FiledItemCount = 1
	batch.SettleFiling()
	assert.Equal(t, capture.BatchPartiallyFiled, batch.Status)

	batch.FiledItemCount = 3
	batch.SettleFiling()
	assert.Equal(t, capture.BatchFiled, batch.Status)

	discarded := &capture.CaptureBatch{Status: capture.BatchDiscarded, ItemCount: 1, FiledItemCount: 1}
	discarded.SettleFiling()
	assert.Equal(t, capture.BatchDiscarded, discarded.Status, "a settled batch stays settled")
}

func TestDeviceRevokeClearsCredentials(t *testing.T) {
	t.Parallel()

	by := pulid.MustNew("usr_")
	device := &capture.CaptureDevice{
		ID:                  pulid.MustNew("cdev_"),
		Status:              capture.DeviceActive,
		AccessTokenHash:     "access",
		RefreshTokenHash:    "refresh",
		PreviousRefreshHash: "previous",
	}

	device.Revoke(&by, "Lost laptop", 500)

	assert.False(t, device.IsActive())
	assert.NotEqual(t, "access", device.AccessTokenHash)
	assert.NotEqual(t, "refresh", device.RefreshTokenHash)
	assert.NotEqual(t, device.AccessTokenHash, "", "the column is unique and not null")
	assert.Empty(t, device.PreviousRefreshHash)
	assert.False(t, device.IsOnline(500))
}

func TestDeviceOnline(t *testing.T) {
	t.Parallel()

	seen := int64(1000)
	device := &capture.CaptureDevice{Status: capture.DeviceActive, LastSeenAt: &seen}
	assert.True(t, device.IsOnline(1000+capture.OnlineWindowSeconds))
	assert.False(t, device.IsOnline(1001+capture.OnlineWindowSeconds))
}

func TestCoverSheetToken(t *testing.T) {
	t.Parallel()

	token, ok := capture.CoverSheetToken(capture.CoverSheetPayload("abc"))
	require.True(t, ok)
	assert.Equal(t, "abc", token)

	_, ok = capture.CoverSheetToken("https://tracking.example/abc")
	assert.False(t, ok)

	_, ok = capture.CoverSheetToken(capture.CoverSheetPrefix)
	assert.False(t, ok, "a prefix with no token is not a sheet")
}

func TestNormalizeRotation(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, capture.NormalizeRotation(360))
	assert.Equal(t, 270, capture.NormalizeRotation(-90))
	assert.Equal(t, 90, capture.NormalizeRotation(450))
	assert.Equal(t, 180, capture.NormalizeRotation(200))
}
