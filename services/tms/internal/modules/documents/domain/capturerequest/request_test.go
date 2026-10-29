package capturerequest_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capturerequest"
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
		from capturerequest.RequestStatus
		to   capturerequest.RequestStatus
		want bool
	}{
		{capturerequest.RequestPending, capturerequest.RequestDelivered, true},
		{capturerequest.RequestPending, capturerequest.RequestInProgress, true},
		{capturerequest.RequestDelivered, capturerequest.RequestInProgress, true},
		{capturerequest.RequestInProgress, capturerequest.RequestCompleted, true},
		{capturerequest.RequestDelivered, capturerequest.RequestCompleted, true},
		{capturerequest.RequestPending, capturerequest.RequestCompleted, false},
		{capturerequest.RequestInProgress, capturerequest.RequestDelivered, false},
		{capturerequest.RequestDelivered, capturerequest.RequestPending, false},
		{capturerequest.RequestPending, capturerequest.RequestCanceled, true},
		{capturerequest.RequestInProgress, capturerequest.RequestFailed, true},
		{capturerequest.RequestCanceled, capturerequest.RequestInProgress, false},
		{capturerequest.RequestCompleted, capturerequest.RequestFailed, false},
		{capturerequest.RequestExpired, capturerequest.RequestDelivered, false},
		{capturerequest.RequestPending, capturerequest.RequestPending, false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.from.CanMoveTo(tt.to), "%s -> %s", tt.from, tt.to)
	}
}

func TestRequestTransitionStampsTimes(t *testing.T) {
	t.Parallel()

	req := &capturerequest.CaptureRequest{Status: capturerequest.RequestPending}
	require.True(t, req.Transition(capturerequest.RequestDelivered, 100))
	require.NotNil(t, req.DeliveredAt)
	assert.Equal(t, int64(100), *req.DeliveredAt)

	require.True(t, req.Transition(capturerequest.RequestCompleted, 200))
	require.NotNil(t, req.CompletedAt)
	assert.Equal(t, int64(200), *req.CompletedAt)

	assert.False(t, req.Transition(capturerequest.RequestFailed, 300),
		"a finished request cannot be failed by a late report")
	assert.Equal(t, capturerequest.RequestCompleted, req.Status)
}

func TestRequestExpiryOnlyWhileWaiting(t *testing.T) {
	t.Parallel()

	req := &capturerequest.CaptureRequest{Status: capturerequest.RequestPending, ExpiresAt: 100}
	assert.False(t, req.IsExpired(99))
	assert.True(t, req.IsExpired(100))

	req.Status = capturerequest.RequestInProgress
	assert.False(t, req.IsExpired(10_000), "a scan in progress is not expired by the clock")
}

func TestRequestValidate(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("cprf_")
	req := &capturerequest.CaptureRequest{
		UserID:     pulid.MustNew("usr_"),
		DeviceID:   pulid.MustNew("cdev_"),
		Mode:       capturerequest.RequestModePrint,
		Status:     capturerequest.RequestPending,
		TargetType: "invoice",
		TargetID:   pulid.MustNew("inv_"),
		ProfileID:  &profileID,
	}

	multiErr := errortypes.NewMultiError()
	req.Validate(multiErr)

	assert.ElementsMatch(t, []string{"targetType", "profileId"}, errorFields(multiErr))
}
