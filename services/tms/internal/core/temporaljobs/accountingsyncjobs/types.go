package accountingsyncjobs

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	CheckAccountingConnectionsWorkflowName          = "CheckAccountingConnectionsWorkflow"
	RefreshAccountingReferenceWorkflowName          = "RefreshAccountingReferenceWorkflow"
	RefreshAllAccountingReferenceWorkflowName       = "RefreshAllAccountingReferenceWorkflow"
	ReferenceWorkflowIDPrefix                       = "accounting-reference:"
	healthPageSize                                  = 25
	healthMaxPages                                  = 40
	referenceSweepPageSize                          = 100
	modelPassAttempts                         int32 = 3
)

type HealthSweepResult struct {
	Listed  int `json:"listed"`
	Checked int `json:"checked"`
	Failed  int `json:"failed"`
	Pages   int `json:"pages"`
}

type RefreshReferencePayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ConnectionID   pulid.ID `json:"connectionId"`
}

func (p *RefreshReferencePayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type ReferenceKindResult struct {
	Kind    string `json:"kind"`
	Fetched int    `json:"fetched"`
	Removed int64  `json:"removed"`
}

type RefreshReferenceResult struct {
	Kinds         []ReferenceKindResult `json:"kinds"`
	Targets       int                   `json:"targets"`
	Created       int64                 `json:"created"`
	Updated       int64                 `json:"updated"`
	Proposed      int                   `json:"proposed"`
	ModelProposed int                   `json:"modelProposed"`
}

type RescoreReferenceResult struct {
	Targets    int        `json:"targets"`
	Created    int64      `json:"created"`
	Updated    int64      `json:"updated"`
	Proposed   int        `json:"proposed"`
	NeedsModel []pulid.ID `json:"needsModel"`
}

type ReferenceSweepPayload struct {
	AfterID pulid.ID `json:"afterId"`
	Started int      `json:"started"`
}

type ReferenceSweepPage struct {
	Connections []RefreshReferencePayload `json:"connections"`
	LastID      pulid.ID                  `json:"lastId"`
	More        bool                      `json:"more"`
}

type ReferenceSweepResult struct {
	Started int `json:"started"`
	Skipped int `json:"skipped"`
}

func ReferenceWorkflowID(connectionID pulid.ID) string {
	return ReferenceWorkflowIDPrefix + connectionID.String()
}
