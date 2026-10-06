package aitraininghistoryservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/aitraininghistoryservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyHistoryListsNothing(t *testing.T) {
	t.Parallel()

	history, err := aitraininghistoryservice.NewEmpty().ListHistory(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Empty(t, history)
}
