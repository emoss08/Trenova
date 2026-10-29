package completionengine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStopped(t *testing.T) {
	t.Parallel()

	live := t.Context()
	gone, cancel := context.WithCancel(t.Context())
	cancel()

	cause := errors.New("unexpected EOF")
	assert.NoError(t, Stopped(gone, nil))
	assert.Same(t, cause, Stopped(live, cause), "a live caller's error is left alone")

	wrapped := Stopped(gone, cause)
	assert.ErrorIs(t, wrapped, context.Canceled)
	assert.ErrorIs(t, wrapped, cause, "the provider's own account is kept")
	assert.Same(t, context.Canceled, Stopped(gone, context.Canceled))
}
