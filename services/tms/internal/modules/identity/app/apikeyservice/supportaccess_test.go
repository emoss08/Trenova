package apikeyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

// Support access lets someone outside the organization in; only a person grants it, so no API
// key may be given it, and saving a key that asks for it is refused.
func TestAnAPIKeyCanNeverHoldSupportAccess(t *testing.T) {
	t.Parallel()

	_, offered := runtimePolicy[permission.ResourceSupportAccess.String()]
	assert.False(t, offered)
}
