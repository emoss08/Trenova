package captureservice

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPagesArrivingAreAnnouncedOncePerWindowAndOnceAtItsEnd(t *testing.T) {
	t.Parallel()

	announcer := newReceivedAnnouncer(80 * time.Millisecond)
	batch := pulid.MustNew("cbat_")
	var published atomic.Int32
	publish := func() { published.Add(1) }

	for range 20 {
		announcer.announce(batch, publish)
	}
	assert.Equal(t, int32(1), published.Load(), "the first page is announced at once")

	assert.Eventually(t, func() bool { return published.Load() == 2 },
		time.Second, 10*time.Millisecond, "the pages after it are announced once, at the window's end")

	time.Sleep(120 * time.Millisecond)
	assert.Equal(t, int32(2), published.Load(), "nothing more without more pages")

	other := pulid.MustNew("cbat_")
	announcer.announce(other, publish)
	assert.Equal(t, int32(3), published.Load(), "each batch has its own window")
}

func TestAnAnnouncerForgetsASealedBatch(t *testing.T) {
	t.Parallel()

	announcer := newReceivedAnnouncer(time.Hour)
	batch := pulid.MustNew("cbat_")
	var published atomic.Int32
	announcer.announce(batch, func() { published.Add(1) })
	announcer.forget(batch)
	announcer.announce(batch, func() { published.Add(1) })
	assert.Equal(t, int32(2), published.Load())

	var unset *receivedAnnouncer
	unset.announce(batch, func() { published.Add(1) })
	assert.Equal(t, int32(3), published.Load(), "without an announcer every page is announced")
}
