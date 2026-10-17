package extractionevalcontract_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/extractionevaljobs/extractionevalcontract"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/api/enums/v1"
)

func TestRefreezeWorkflowIDIsOnePerTenant(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}
	sameTenantOtherUser := tenant
	sameTenantOtherUser.UserID = pulid.MustNew("usr_")

	assert.Equal(
		t,
		extractionevalcontract.RefreezeWorkflowID(tenant),
		extractionevalcontract.RefreezeWorkflowID(sameTenantOtherUser),
	)
	assert.NotEqual(
		t,
		extractionevalcontract.RefreezeWorkflowID(tenant),
		extractionevalcontract.RefreezeWorkflowID(
			pagination.TenantInfo{OrgID: "org_1", BuID: "bu_2"},
		),
	)
}

func TestRefreezeStatusOf(t *testing.T) {
	t.Parallel()

	for status, want := range map[enums.WorkflowExecutionStatus]extractioneval.RefreezeStatus{
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING:    extractioneval.RefreezeStatusRunning,
		enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:  extractioneval.RefreezeStatusCompleted,
		enums.WORKFLOW_EXECUTION_STATUS_FAILED:     extractioneval.RefreezeStatusFailed,
		enums.WORKFLOW_EXECUTION_STATUS_TERMINATED: extractioneval.RefreezeStatusFailed,
		enums.WORKFLOW_EXECUTION_STATUS_CANCELED:   extractioneval.RefreezeStatusFailed,
	} {
		assert.Equal(t, want, extractionevalcontract.RefreezeStatusOf(status), status.String())
	}
}
