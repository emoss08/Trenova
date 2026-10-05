package cloudlifecyclejobs

import (
	"github.com/emoss08/trenova/internal/cloud/lifecycle/cloudlifecycleservice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	CloudSubscriptionSweepWorkflowName = "CloudSubscriptionSweepWorkflow"
	CloudTenantPurgeWorkflowName       = "CloudTenantPurgeWorkflow"

	purgeWorkflowIDPrefix = "cloud-tenant-purge"
	sweepScheduleID       = "cloud-subscription-sweep"
)

type SweepInput struct {
	Now int64 `json:"now"`
}

type SweepResult struct {
	cloudlifecycleservice.SweepResult

	PurgesStarted int `json:"purgesStarted"`
	PurgesRunning int `json:"purgesRunning"`
}

type PurgePayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func (p *PurgePayload) ref() cloudlifecycleservice.TenantRef {
	return cloudlifecycleservice.TenantRef{
		OrganizationID: p.OrganizationID,
		BusinessUnitID: p.BusinessUnitID,
	}
}

type PurgeEligibility struct {
	Eligible bool                         `json:"eligible"`
	Members  []*repositories.TenantMember `json:"members"`
}

type PurgeRowsResult struct {
	Deleted  int64    `json:"deleted"`
	Passes   int      `json:"passes"`
	Retained []string `json:"retained"`
	Blocked  []string `json:"blocked"`
	Complete bool     `json:"complete"`
}

type PurgeUsersInput struct {
	PurgePayload

	UserIDs []pulid.ID `json:"userIds"`
}

type PurgeStorageResult struct {
	Deleted int64 `json:"deleted"`
	Skipped bool  `json:"skipped"`
}

type PurgeResult struct {
	OrganizationID pulid.ID                                `json:"organizationId"`
	Skipped        bool                                    `json:"skipped"`
	Rows           *PurgeRowsResult                        `json:"rows,omitempty"`
	Storage        *PurgeStorageResult                     `json:"storage,omitempty"`
	Users          *cloudlifecycleservice.PurgeUsersResult `json:"users,omitempty"`
	Tenant         *repositories.DeleteTenantResult        `json:"tenant,omitempty"`
}
