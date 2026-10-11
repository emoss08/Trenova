package reportcontract

import (
	"github.com/emoss08/trenova/shared/pulid"
)

const RunReportWorkflowName = "RunReportWorkflow"

type RunReportPayload struct {
	RunID          pulid.ID `json:"runId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}
