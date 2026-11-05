package driversettlementresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementworkspaceservice"
	"github.com/stretchr/testify/assert"
)

func TestBulkActionsNeedWhatTheSingleActionNeeds(t *testing.T) {
	t.Parallel()

	assert.Equal(t, permission.OpSubmit,
		bulkActionOperation(driversettlementworkspaceservice.BulkActionSubmit))
	assert.Equal(t, permission.OpApprove,
		bulkActionOperation(driversettlementworkspaceservice.BulkActionApprove))
	assert.Equal(t, permission.OpApprove,
		bulkActionOperation(driversettlementworkspaceservice.BulkActionPost))
	assert.Equal(t, permission.OpUpdate,
		bulkActionOperation(driversettlementworkspaceservice.BulkActionMarkPaid))
}
