package customertracking

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"
	"github.com/stretchr/testify/assert"
)

func TestNotifiesReadsTheRuleForEachEvent(t *testing.T) {
	profile := &trackingprofile.Profile{NotifyPickedUp: true, NotifyPodReady: true}

	assert.True(t, Notifies(profile, EventPickedUp))
	assert.False(t, Notifies(profile, EventDelayed))
	assert.False(t, Notifies(profile, EventDelivered))
	assert.True(t, Notifies(profile, EventPodReady))
	assert.False(t, Notifies(profile, Event("Bogus")))
	assert.False(t, Notifies(nil, EventPickedUp))
}

func TestRetryDelaySecondsBacksOffToAnHour(t *testing.T) {
	assert.Equal(t, int64(60), RetryDelaySeconds(1))
	assert.Equal(t, int64(120), RetryDelaySeconds(2))
	assert.Equal(t, int64(3600), RetryDelaySeconds(8))
}
