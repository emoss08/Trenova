package accountingsynccontract

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DrainAccountingOutboxWorkflowName    = "DrainAccountingOutboxWorkflow"
	BackfillAccountingWorkflowName       = "BackfillAccountingWorkflow"
	ReconcileAccountingDriftWorkflowName = "ReconcileAccountingDriftWorkflow"
	DrainWorkflowIDPrefix                = "accounting-sync:"
	BackfillWorkflowIDPrefix             = "accounting-backfill:"
	DriftWorkflowIDPrefix                = "accounting-drift:"
	DrainSignalName                      = "accounting-sync-kick"
)

type DrainPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ConnectionID   pulid.ID `json:"connectionId"`
	Batches        int      `json:"batches"`
}

func (p *DrainPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type DrainSignal struct{}

type BackfillPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	BackfillID     pulid.ID `json:"backfillId"`
	Steps          int      `json:"steps"`
}

func (p *BackfillPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type DriftPayload struct {
	OrganizationID  pulid.ID       `json:"organizationId"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"`
	ConnectionID    pulid.ID       `json:"connectionId"`
	Started         bool           `json:"started"`
	Balances        bool           `json:"balances"`
	AfterID         pulid.ID       `json:"afterId"`
	AfterCustomerID pulid.ID       `json:"afterCustomerId"`
	EventsLeft      int            `json:"eventsLeft"`
	Totals          DriftRunResult `json:"totals"`
}

func (p *DriftPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type DriftRunResult struct {
	Batches   int  `json:"batches"`
	Compared  int  `json:"compared"`
	Customers int  `json:"customers"`
	Opened    int  `json:"opened"`
	Resolved  int  `json:"resolved"`
	Events    int  `json:"events"`
	Held      bool `json:"held"`
}

func DrainWorkflowID(connectionID pulid.ID) string {
	return DrainWorkflowIDPrefix + connectionID.String()
}

func BackfillWorkflowID(backfillID pulid.ID) string {
	return BackfillWorkflowIDPrefix + backfillID.String()
}

func DriftWorkflowID(connectionID pulid.ID) string {
	return DriftWorkflowIDPrefix + connectionID.String()
}
