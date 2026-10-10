package journalreversal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusTransitions(t *testing.T) {
	t.Parallel()

	assert.True(t, StatusRequested.CanApprove())
	assert.True(t, StatusPendingApproval.CanApprove())
	assert.False(t, StatusApproved.CanApprove())

	assert.True(t, StatusRequested.CanReject())
	assert.True(t, StatusPendingApproval.CanReject())
	assert.False(t, StatusPosted.CanReject())

	assert.True(t, StatusRequested.CanCancel())
	assert.True(t, StatusPendingApproval.CanCancel())
	assert.True(t, StatusApproved.CanCancel())
	assert.False(t, StatusPosted.CanCancel())

	assert.True(t, StatusApproved.CanPost())
	assert.False(t, StatusRequested.CanPost())
	assert.Equal(t, "Approved", StatusApproved.String())
}
