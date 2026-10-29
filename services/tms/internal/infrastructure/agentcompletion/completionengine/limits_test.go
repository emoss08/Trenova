package completionengine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyRefused_ReadsOnlyAuthStatuses(t *testing.T) {
	t.Parallel()

	assert.False(t, keyRefused(errors.New("dial tcp: refused")))
}
