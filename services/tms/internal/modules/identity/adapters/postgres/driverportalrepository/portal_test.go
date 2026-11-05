package driverportalrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireInvitable_RefusesAWorkerWhoLeftAfterTheInvitationWasSent(t *testing.T) {
	t.Parallel()

	err := requireInvitable(&worker.Worker{Status: domaintypes.StatusInactive})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer active")
}

func TestRequireInvitable_RefusesAWorkerWhoAlreadyHasAccess(t *testing.T) {
	t.Parallel()

	err := requireInvitable(&worker.Worker{
		Status: domaintypes.StatusActive,
		UserID: pulid.MustNew("usr_"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has portal access")
}

func TestRequireInvitable_LetsAnActiveWorkerAccept(t *testing.T) {
	t.Parallel()

	require.NoError(t, requireInvitable(&worker.Worker{Status: domaintypes.StatusActive}))
}
