package ptocontract

import (
	"github.com/emoss08/trenova/shared/pulid"
)

const PTOAccrualWorkflowName = "PTOAccrualWorkflow"

type PTOAccrualWorkflowInput struct {
	OrganizationID pulid.ID `json:"organizationId,omitempty"`
	BusinessUnitID pulid.ID `json:"businessUnitId,omitempty"`
	WorkerID       pulid.ID `json:"workerId,omitempty"`
	AsOf           int64    `json:"asOf,omitempty"`
}
