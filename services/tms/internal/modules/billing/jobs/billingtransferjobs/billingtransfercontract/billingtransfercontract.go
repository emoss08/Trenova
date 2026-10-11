package billingtransfercontract

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const BulkBillingTransferWorkflowName = "BulkBillingTransferWorkflow"

// CancelSignalName shortens how long a cancel waits to take effect. It is
// never the authority for one: the run row's cancel_requested_at is, because
// that survives a worker restart and a Temporal outage and is visible to the
// dialog the instant it is written.
const CancelSignalName = "billing-transfer-cancel"

type TransferRunPayload struct {
	temporaltype.BasePayload

	RunID pulid.ID `json:"runId"`
}

func (p *TransferRunPayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}
