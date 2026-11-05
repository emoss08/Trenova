package reportjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestMayRunAs(t *testing.T) {
	t.Parallel()

	assert.True(t, mayRunAs(&tenant.User{Status: domaintypes.StatusActive}, nil))
	assert.False(t, mayRunAs(&tenant.User{Status: domaintypes.StatusInactive}, nil))
	assert.False(t, mayRunAs(&tenant.User{Status: domaintypes.StatusActive, IsLocked: true}, nil))
	assert.False(t, mayRunAs(nil, errortypes.NewNotFoundError("User not found")))
	assert.True(t, mayRunAs(nil, errors.New("database unavailable")), "an outage is retried, not disabled")
}
