package cloudsignup_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudsignup"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	"github.com/stretchr/testify/require"
)

func TestStatusCheckConstraintAcceptsEveryDeclaredStatus(t *testing.T) {
	t.Parallel()

	definition, err := migrations.LatestConstraintDefinition("ck_cloud_signups_status")
	require.NoError(t, err)

	for _, status := range cloudsignup.AllStatuses() {
		require.Contains(t, definition, "'"+string(status)+"'",
			"ck_cloud_signups_status does not accept %q; add a migration that widens it", status)
	}
}
