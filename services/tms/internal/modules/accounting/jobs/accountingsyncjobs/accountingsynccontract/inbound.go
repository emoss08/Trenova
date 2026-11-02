package accountingsynccontract

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	RefreshAccountingReferenceWorkflowName = "RefreshAccountingReferenceWorkflow"
	ReferenceWorkflowIDPrefix              = "accounting-reference:"
	PollAccountingChangesWorkflowName      = "PollAccountingChangesWorkflow"
	ChangesWorkflowIDPrefix                = "accounting-changes:"
	ChangesSignalName                      = "accounting-changes-kick"
)

type RefreshReferencePayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ConnectionID   pulid.ID `json:"connectionId"`
}

func (p *RefreshReferencePayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

func ReferenceWorkflowID(connectionID pulid.ID) string {
	return ReferenceWorkflowIDPrefix + connectionID.String()
}

type ChangesPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ConnectionID   pulid.ID `json:"connectionId"`
	Reads          int      `json:"reads"`
}

func (p *ChangesPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type ChangesSignal struct{}

func ChangesWorkflowID(connectionID pulid.ID) string {
	return ChangesWorkflowIDPrefix + connectionID.String()
}
